package main

import (
	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/search"
)

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
	queryCmd.Flags().IntVarP(&searchLimit, "limit", "n", 5, "max results")
	queryCmd.Flags().StringVarP(&searchFormat, "format", "f", "cli", "output format")
	queryCmd.Flags().StringSliceVarP(&searchCollections, "collection", "c", nil, "collection(s) to search (default: auto-detect from CWD)")
}
