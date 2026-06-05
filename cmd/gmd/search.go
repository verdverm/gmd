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
	searchNoExpansion bool
	searchPreset      string
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

	if searchPreset != "" {
		// Use named search preset
		preset, ok := cfg.SearchDefaults[searchPreset]
		if !ok {
			return fmt.Errorf("search preset %q not found in searchDefaults", searchPreset)
		}
		collections = preset
	}

	if len(collections) == 0 {
		// Auto-detect from CWD, then collect all searchable sources
		cwd, err := os.Getwd()
		if err == nil {
			collections = config.MatchSourcesByCWD(cfg, cwd)
		}
		if len(collections) == 0 {
			collections = cfg.AllSearchableSources()
		}
	}

	// Expand sourceRefs for wikis
	var expandedCollections []string
	for _, c := range collections {
		if searchNoExpansion {
			expandedCollections = append(expandedCollections, cfg.CollectionKey(c))
		} else {
			keys, err := cfg.SourceKeysForSearch(c)
			if err != nil {
				return fmt.Errorf("resolving sources: %w", err)
			}
			expandedCollections = append(expandedCollections, keys...)
		}
	}

	params := search.SearchParams{
		Query:       args[0],
		Collections: expandedCollections,
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
	searchCmd.Flags().BoolVar(&searchNoExpansion, "no-expansion", false, "Do not expand wiki sourceRefs")
	searchCmd.Flags().StringVarP(&searchPreset, "search", "s", "", "Named search preset from searchDefaults")
}
