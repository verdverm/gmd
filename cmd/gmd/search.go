package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/config"
	"github.com/verdverm/gmd/llm"
	"github.com/verdverm/gmd/output"
	"github.com/verdverm/gmd/search"
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
		BaseURL:        cfg.LLM.BaseURL,
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
	Short: "Full-text keyword search (no vector, no expansion)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return searchRun(args, search.ModeText)
	},
}

var vsearchCmd = &cobra.Command{
	Use:   "vsearch <query>",
	Short: "Vector similarity search (no text)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return searchRun(args, search.ModeVector)
	},
}

var queryCmd = &cobra.Command{
	Use:   "query <query>",
	Short: "Full hybrid pipeline (expansion, RRF, rerank, blend)",
	Args:  cobra.ExactArgs(1),
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
