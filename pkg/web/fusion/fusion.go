package fusion

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/web"
)

// Config controls multi-provider search fusion.
type Config struct {
	Dedup           string // "heuristic", "llm", "none"
	Synthesize      bool
	SynthesisPrompt string // custom system prompt (overrides default)
	LLMClient       *llm.Client
}

// Result is the output of a fused multi-provider search.
type Result struct {
	Answer  string
	Results []web.SearchResult
	Costs   []web.CostSummary
}

// Run executes a multi-provider search with parallel fan-out, dedup, and optional synthesis.
func Run(ctx context.Context, query string, providers []web.SearchProvider, opts web.SearchOptions, cfg Config) (*Result, error) {
	allResults, err := MultiSearch(ctx, query, providers, opts)
	if err != nil {
		return nil, err
	}

	deduped, err := Dedup(ctx, allResults, cfg)
	if err != nil {
		return nil, err
	}

	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].Score > deduped[j].Score
	})

	var answer string
	if cfg.Synthesize && cfg.LLMClient != nil {
		answer, err = Synthesize(ctx, query, deduped, cfg)
		if err != nil {
			return nil, fmt.Errorf("synthesizing: %w", err)
		}
	}

	var costs []web.CostSummary
	for _, r := range deduped {
		if r.Cost != nil {
			costs = append(costs, *r.Cost)
		}
	}

	return &Result{
		Answer:  answer,
		Results: deduped,
		Costs:   costs,
	}, nil
}

// MultiSearch fans out a query to multiple providers in parallel and collects all results.
// Each result is tagged with its provider name in Extra["_provider"].
// Partial failures are tolerated: if some providers fail, results from successful
// providers are still returned. An error is returned only if all providers fail.
func MultiSearch(ctx context.Context, query string, providers []web.SearchProvider, opts web.SearchOptions) ([]web.SearchResult, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no search providers configured")
	}

	type providerResult struct {
		results []web.SearchResult
		err     error
		name    string
	}

	ch := make(chan providerResult, len(providers))
	var wg sync.WaitGroup

	for _, p := range providers {
		wg.Add(1)
		go func(sp web.SearchProvider) {
			defer wg.Done()
			results, err := sp.Search(ctx, opts)
			name := providerName(sp)
			ch <- providerResult{results: results, err: err, name: name}
		}(p)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var all []web.SearchResult
	var errs []string
	for pr := range ch {
		if pr.err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", pr.name, pr.err))
			continue
		}
		for _, r := range pr.results {
			if r.Extra == nil {
				r.Extra = make(map[string]any)
			}
			r.Extra["_provider"] = pr.name
			all = append(all, r)
		}
	}

	if len(all) == 0 {
		if len(errs) > 0 {
			return nil, fmt.Errorf("all providers failed: %s", strings.Join(errs, "; "))
		}
		return nil, fmt.Errorf("no results from any provider")
	}

	return all, nil
}

// Dedup removes duplicate results based on the configured method.
func Dedup(ctx context.Context, results []web.SearchResult, cfg Config) ([]web.SearchResult, error) {
	switch cfg.Dedup {
	case "llm":
		return dedupLLM(ctx, results, cfg)
	case "none":
		return results, nil
	default:
		return dedupHeuristic(results), nil
	}
}

func dedupHeuristic(results []web.SearchResult) []web.SearchResult {
	seen := make(map[string]int)
	var out []web.SearchResult
	for _, r := range results {
		key := strings.TrimRight(r.URL, "/")
		key = strings.ToLower(key)
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(r.Title))
		}
		if key == "" {
			out = append(out, r)
			continue
		}
		if idx, exists := seen[key]; exists {
			if r.Score > out[idx].Score {
				if r.Content == "" {
					r.Content = out[idx].Content
				}
				out[idx] = r
			}
			continue
		}
		seen[key] = len(out)
		out = append(out, r)
	}
	return out
}

func dedupLLM(ctx context.Context, results []web.SearchResult, cfg Config) ([]web.SearchResult, error) {
	if cfg.LLMClient == nil {
		return results, nil
	}

	var sb strings.Builder
	sb.WriteString("Identify duplicate search results. For each group of duplicates, keep only the best one.\n\n")
	for i, r := range results {
		fmt.Fprintf(&sb, "[%d] %s (%s) — %s\n", i, r.Title, r.URL, truncateStr(r.Content, 300))
	}

	messages := []llm.ChatMessage{
		{
			Role:    "system",
			Content: "You deduplicate search results. Output a JSON array of indices to KEEP. Example: [0, 2, 5]. Only output the array, no other text.",
		},
		{Role: "user", Content: sb.String()},
	}

	resp, err := cfg.LLMClient.Summarize(ctx, messages)
	if err != nil {
		return dedupHeuristic(results), nil
	}

	indices := parseKeepIndices(resp, len(results))
	if len(indices) == 0 {
		return dedupHeuristic(results), nil
	}

	var out []web.SearchResult
	for _, idx := range indices {
		if idx >= 0 && idx < len(results) {
			out = append(out, results[idx])
		}
	}
	return out, nil
}

// Synthesize produces a unified answer from search results.
func Synthesize(ctx context.Context, query string, results []web.SearchResult, cfg Config) (string, error) {
	if cfg.LLMClient == nil {
		return "", fmt.Errorf("LLM client is required for synthesis")
	}

	var sb strings.Builder
	for i, r := range results {
		fmt.Fprintf(&sb, "[%d] %s\n", i+1, r.Title)
		fmt.Fprintf(&sb, "    URL: %s\n", r.URL)
		if r.Content != "" {
			fmt.Fprintf(&sb, "    Content: %s\n", truncateStr(r.Content, 1500))
		}
		if provider, ok := r.Extra["_provider"].(string); ok {
			fmt.Fprintf(&sb, "    Provider: %s\n", provider)
		}
		sb.WriteString("\n")
	}

	systemPrompt := cfg.SynthesisPrompt
	if systemPrompt == "" {
		systemPrompt = searchSynthesisPrompt()
	}

	messages := []llm.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Query: %s\n\nSearch results:\n%s\nSynthesize a comprehensive answer.", query, sb.String())},
	}

	return cfg.LLMClient.Summarize(ctx, messages)
}

func providerName(sp web.SearchProvider) string {
	if named, ok := sp.(interface{ Name() string }); ok {
		return named.Name()
	}
	return fmt.Sprintf("%T", sp)
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func parseKeepIndices(s string, maxIdx int) []int {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	if !strings.HasPrefix(s, "[") {
		return nil
	}
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	parts := strings.Split(s, ",")
	var indices []int
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var idx int
		if _, err := fmt.Sscanf(p, "%d", &idx); err == nil {
			if idx >= 0 && idx < maxIdx {
				indices = append(indices, idx)
			}
		}
	}
	return indices
}
