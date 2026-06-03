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
	llmClient := llm.New(llmConfigFromConfig(cfg))
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

func init() {
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 5, "max results")
	searchCmd.Flags().StringVarP(&searchFormat, "format", "f", "cli", "output format")
	searchCmd.Flags().StringSliceVarP(&searchCollections, "collection", "c", nil, "collection(s) to search (default: auto-detect from CWD)")
}
