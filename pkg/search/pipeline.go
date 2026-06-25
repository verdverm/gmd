package search

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/ts"
)

//go:embed embeds
var searchEmbedsFS embed.FS

func expansionPrompt() string {
	data, _ := searchEmbedsFS.ReadFile("embeds/expansion.md")
	return string(data)
}

type Mode int

const (
	ModeText Mode = iota
	ModeVector
	ModeHybrid
)

type Params struct {
	Query       string
	Collections []string
	Limit       int
	Format      string
}

type Result struct {
	Collection string  `json:"collection"`
	Path       string  `json:"path"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	ChunkSeq   int     `json:"chunk_seq"`
	Score      float64 `json:"score"`
}

type Pipeline struct {
	cfg      *config.Config
	ts       *ts.Client
	embedder llm.Embedder
	expander llm.ChatModel
	reranker llm.Reranker
}

func New(cfg *config.Config, tsClient *ts.Client, embedder llm.Embedder, expander llm.ChatModel, reranker llm.Reranker) *Pipeline {
	return &Pipeline{cfg: cfg, ts: tsClient, embedder: embedder, expander: expander, reranker: reranker}
}

func (p *Pipeline) Search(ctx context.Context, params Params, mode Mode) ([]Result, error) {
	switch mode {
	case ModeText:
		return p.textSearch(ctx, params)
	case ModeVector:
		return p.vectorSearch(ctx, params)
	case ModeHybrid:
		return p.fullPipeline(ctx, params)
	default:
		return p.textSearch(ctx, params)
	}
}

func (p *Pipeline) textSearch(ctx context.Context, params Params) ([]Result, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = p.cfg.Pipeline.Output.MaxResults
	}
	results, err := p.ts.TextSearch(ctx, ts.HybridSearchParams{
		Query:       params.Query,
		Collections: params.Collections,
		Limit:       limit,
		GroupLimit:  1,
	})
	if err != nil {
		return nil, err
	}
	return tsResultsToResults(results), nil
}

func (p *Pipeline) vectorSearch(ctx context.Context, params Params) ([]Result, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = p.cfg.Pipeline.Output.MaxResults
	}
	embedding, err := llm.EmbedSingle(ctx, p.embedder, params.Query)
	if err != nil {
		return nil, fmt.Errorf("embedding query: %w", err)
	}
	results, err := p.ts.VectorSearch(ctx, ts.HybridSearchParams{
		QueryVector: embedding,
		Collections: params.Collections,
		Limit:       limit,
		GroupLimit:  1,
	})
	if err != nil {
		return nil, err
	}
	return tsResultsToResults(results), nil
}

func (p *Pipeline) fullPipeline(ctx context.Context, params Params) ([]Result, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = p.cfg.Pipeline.Output.MaxResults
	}

	isStrong, _ := p.checkStrongSignal(ctx, params.Query, params.Collections)

	variants, err := p.generateVariants(ctx, params.Query, isStrong)
	if err != nil {
		variants = []variant{{
			text:    params.Query,
			vecText: params.Query,
			weight:  p.cfg.Pipeline.RRF.OriginalWeight,
		}}
	}

	variantResults, err := p.searchVariants(ctx, variants, params.Collections)
	if err != nil {
		return nil, err
	}

	fused := p.rrfFuse(variantResults, p.cfg.Pipeline.RRF.K)

	if len(fused) == 0 {
		return nil, nil
	}

	candidateLimit := p.cfg.Pipeline.Rerank.CandidateLimit
	if len(fused) > candidateLimit {
		fused = fused[:candidateLimit]
	}

	rerankErr := p.applyRerank(ctx, params.Query, fused)
	if rerankErr != nil {
		for i := range fused {
			fused[i].finalScore = fused[i].rrfScore
		}
	} else {
		p.blend(fused)
	}

	if len(fused) > limit {
		fused = fused[:limit]
	}

	results := make([]Result, len(fused))
	for i, f := range fused {
		results[i] = Result{
			Collection: f.result.Collection,
			Path:       f.result.Path,
			Title:      f.result.Title,
			Content:    f.result.Content,
			ChunkSeq:   f.result.ChunkSeq,
			Score:      f.finalScore,
		}
	}

	return results, nil
}

func (p *Pipeline) checkStrongSignal(ctx context.Context, query string, collections []string) (bool, error) {
	results, err := p.ts.TextSearch(ctx, ts.HybridSearchParams{
		Query:       query,
		Collections: collections,
		Limit:       2,
		GroupLimit:  1,
	})
	if err != nil {
		return false, err
	}
	if len(results) < 2 {
		return false, nil
	}
	gap := results[0].Score - results[1].Score
	return results[0].Score >= p.cfg.Pipeline.StrongSignal.MinScore &&
		gap >= p.cfg.Pipeline.StrongSignal.MinGap, nil
}

type variant struct {
	text    string
	vecText string
	weight  float64
}

func (p *Pipeline) generateVariants(ctx context.Context, originalQuery string, isStrong bool) ([]variant, error) {
	if isStrong {
		return []variant{{
			text:    originalQuery,
			vecText: originalQuery,
			weight:  p.cfg.Pipeline.RRF.OriginalWeight,
		}}, nil
	}

	expansion, err := p.expandQuery(ctx, originalQuery)
	if err != nil {
		return nil, err
	}

	variants := []variant{{
		text:    originalQuery,
		vecText: originalQuery,
		weight:  p.cfg.Pipeline.RRF.OriginalWeight,
	}}

	for _, line := range strings.Split(expansion, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		typ := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch typ {
		case "lex":
			variants = append(variants, variant{
				text:    val,
				vecText: val,
				weight:  p.cfg.Pipeline.RRF.ExpansionWeight,
			})
		case "vec":
			variants = append(variants, variant{
				text:    originalQuery,
				vecText: val,
				weight:  p.cfg.Pipeline.RRF.ExpansionWeight,
			})
		case "hyde":
			variants = append(variants, variant{
				text:    val,
				vecText: val,
				weight:  p.cfg.Pipeline.RRF.ExpansionWeight,
			})
		}
	}

	return variants, nil
}

func (p *Pipeline) expandQuery(ctx context.Context, query string) (string, error) {
	prompt := fmt.Sprintf(expansionPrompt(), query)
	return p.expander.Chat(ctx,
		"You are a search query expansion assistant. Generate precise, focused variants.",
		prompt,
	)
}

type variantResult struct {
	weight  float64
	results []ts.HybridSearchResult
}

func (p *Pipeline) searchVariants(ctx context.Context, variants []variant, collections []string) ([]variantResult, error) {
	results := make([]variantResult, len(variants))

	for i, v := range variants {
		var queryVector []float64
		if v.vecText != "" {
			emb, err := llm.EmbedSingle(ctx, p.embedder, v.vecText)
			if err != nil {
				return nil, fmt.Errorf("embedding variant %d: %w", i, err)
			}
			queryVector = emb
		}

		hits, err := p.ts.HybridSearch(ctx, ts.HybridSearchParams{
			Query:       v.text,
			QueryVector: queryVector,
			Collections: collections,
			Limit:       p.cfg.Pipeline.Rerank.CandidateLimit,
			GroupLimit:  1,
		})
		if err != nil {
			return nil, fmt.Errorf("searching variant %d: %w", i, err)
		}

		results[i] = variantResult{
			weight:  v.weight,
			results: hits,
		}
	}

	return results, nil
}

type fusedDoc struct {
	key         string
	result      ts.HybridSearchResult
	rrfScore    float64
	rerankScore float64
	finalScore  float64
}

const topRankBonus = 1.0

func (p *Pipeline) rrfFuse(variantResults []variantResult, k int) []fusedDoc {
	type docEntry struct {
		result ts.HybridSearchResult
		ranks  []int
	}

	docMap := make(map[string]*docEntry)

	for vi, vr := range variantResults {
		for ri, r := range vr.results {
			key := r.Collection + ":" + r.Path
			if entry, ok := docMap[key]; ok {
				entry.ranks[vi] = ri + 1
			} else {
				entry = &docEntry{
					result: r,
					ranks:  make([]int, len(variantResults)),
				}
				for j := range entry.ranks {
					entry.ranks[j] = -1
				}
				entry.ranks[vi] = ri + 1
				docMap[key] = entry
			}
		}
	}

	fused := make([]fusedDoc, 0, len(docMap))
	for key, entry := range docMap {
		var score float64
		hasTopRank := false
		for vi, rank := range entry.ranks {
			if rank > 0 {
				w := variantResults[vi].weight
				score += w / float64(k+rank)
				if rank == 1 {
					hasTopRank = true
				}
			}
		}
		if hasTopRank {
			score += topRankBonus
		}
		fused = append(fused, fusedDoc{
			key:      key,
			result:   entry.result,
			rrfScore: score,
		})
	}

	sort.Slice(fused, func(i, j int) bool {
		return fused[i].rrfScore > fused[j].rrfScore
	})

	return fused
}

func (p *Pipeline) applyRerank(ctx context.Context, query string, candidates []fusedDoc) error {
	if len(candidates) == 0 {
		return nil
	}

	documents := make([]string, len(candidates))
	for i, c := range candidates {
		documents[i] = c.result.Content
	}

	rerankResults, err := p.reranker.Rerank(ctx, query, documents)
	if err != nil {
		return err
	}

	for _, rr := range rerankResults {
		if rr.Index >= 0 && rr.Index < len(candidates) {
			candidates[rr.Index].rerankScore = rr.Score
		}
	}

	return nil
}

func (p *Pipeline) blend(candidates []fusedDoc) {
	if len(candidates) == 0 {
		return
	}

	maxRRF := candidates[0].rrfScore
	for i := range candidates {
		if candidates[i].rrfScore > maxRRF {
			maxRRF = candidates[i].rrfScore
		}
	}

	maxRerank := 0.0
	for _, c := range candidates {
		if c.rerankScore > maxRerank {
			maxRerank = c.rerankScore
		}
	}

	cfg := p.cfg.Pipeline.Blending

	for i := range candidates {
		rrfNorm := candidates[i].rrfScore
		if maxRRF > 0 {
			rrfNorm /= maxRRF
		}

		rerankNorm := candidates[i].rerankScore
		if maxRerank > 0 {
			rerankNorm /= maxRerank
		}

		rank := i + 1
		var rrfWeight float64
		switch {
		case rank <= cfg.Thresholds.Top:
			rrfWeight = cfg.Weights.Top
		case rank <= cfg.Thresholds.Middle:
			rrfWeight = cfg.Weights.Middle
		default:
			rrfWeight = cfg.Weights.Bottom
		}

		candidates[i].finalScore = rrfWeight*rrfNorm + (1-rrfWeight)*rerankNorm
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].finalScore > candidates[j].finalScore
	})
}

func tsResultsToResults(results []ts.HybridSearchResult) []Result {
	out := make([]Result, len(results))
	for i, r := range results {
		out[i] = Result{
			Collection: r.Collection,
			Path:       r.Path,
			Title:      r.Title,
			Content:    r.Content,
			ChunkSeq:   r.ChunkSeq,
			Score:      r.Score,
		}
	}
	return out
}
