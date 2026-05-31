package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/output"
	"github.com/verdverm/gmd/pkg/search"
)

var (
	searchLimit       int
	searchFormat      string
	searchCollections []string
)

func searchRun(args []string, mode search.SearchMode) error {
	r, err := getRuntime()
	if err != nil {
		return err
	}
	cfg := r.Config()
	llmClient := llm.New(llm.Config{
		APIKey:         cfg.LLM.APIKey,
		EmbeddingModel: cfg.LLM.EmbeddingModel,
		ExpansionModel: cfg.LLM.ExpansionModel,
		RerankModel:    cfg.LLM.RerankModel,
		EmbedURL:       cfg.LLM.EmbeddingBaseURL,
		ExpandURL:      cfg.LLM.ExpansionBaseURL,
		RerankURL:      cfg.LLM.RerankBaseURL,
	})
	p := search.New(cfg, r.TSClient(), llmClient)
	ctx := context.Background()

	collections := searchCollections
	if len(collections) == 0 {
		cwd, err := os.Getwd()
		if err == nil {
			collections = config.MatchCollectionsByCWD(cfg, cwd)
		}
	}
	for i, c := range collections {
		collections[i] = cfg.CollectionKey(c)
	}

	params := search.SearchParams{
		Query:       args[0],
		Collections: collections,
		Limit:       searchLimit,
		Format:      searchFormat,
	}

	results, err := p.Search(ctx, params, mode)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	out, err := output.FormatResults(results, searchFormat)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Full-text keyword search only — fast, no LLM overhead",
	Long: `Performs a text-only keyword search against indexed documents using
Typesense full-text matching. No embeddings, vector search, or LLM
calls are involved — this is the fastest search mode.

Use this for quick lookups by exact terms or phrases. For semantic
understanding and relevance-ranked results, use 'gmd query' instead.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return searchRun(args, search.ModeText)
	},
}

var vsearchCmd = &cobra.Command{
	Use:   "vsearch <query>",
	Short: "Vector similarity search — semantic matching via embeddings",
	Long: `Embeds the query and performs a vector similarity search using cosine
distance against document chunk embeddings in Typesense.

This finds semantically related content even when exact keywords don't
match. No query expansion or reranking is performed — for the full
pipeline with those features, use 'gmd query' instead.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return searchRun(args, search.ModeVector)
	},
}

var queryCmd = &cobra.Command{
	Use:   "query <query>",
	Short: "Full hybrid search pipeline — expansion, RRF fusion, rerank, blend",
	Long: `Runs the complete search pipeline for best-quality results:

  1. Strong signal detection — fast path if a top result is obvious
  2. LLM query expansion — generates lex, vec, and HyDE variants
  3. Hybrid search — text + vector for each variant, grouped by doc
  4. RRF fusion — merges results across all variants with weights
  5. LLM reranking — re-scores candidates for relevance
  6. Position blending — tiers results by position with configurable weights

This is the recommended command for general-purpose search. Use
'gmd search' for fast keyword lookups without LLM overhead.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return searchRun(args, search.ModeHybrid)
	},
}

func init() {
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 5, "max results")
	searchCmd.Flags().StringVarP(&searchFormat, "format", "f", "cli", "output format")
	searchCmd.Flags().StringSliceVarP(&searchCollections, "collection", "c", nil, "collection(s) to search (default: auto-detect from CWD)")
	vsearchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 5, "max results")
	vsearchCmd.Flags().StringVarP(&searchFormat, "format", "f", "cli", "output format")
	vsearchCmd.Flags().StringSliceVarP(&searchCollections, "collection", "c", nil, "collection(s) to search (default: auto-detect from CWD)")
	queryCmd.Flags().IntVarP(&searchLimit, "limit", "n", 5, "max results")
	queryCmd.Flags().StringVarP(&searchFormat, "format", "f", "cli", "output format")
	queryCmd.Flags().StringSliceVarP(&searchCollections, "collection", "c", nil, "collection(s) to search (default: auto-detect from CWD)")
}
