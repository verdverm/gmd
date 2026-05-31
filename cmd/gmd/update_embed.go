package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/indexer"
	"github.com/verdverm/gmd/pkg/llm"
)

func makeLLMClient() *llm.Client {
	cfg := globalRuntime.Config()
	return llm.New(llm.Config{
		APIKey:         cfg.LLM.APIKey,
		EmbeddingModel: cfg.LLM.EmbeddingModel,
		ExpansionModel: cfg.LLM.ExpansionModel,
		RerankModel:    cfg.LLM.RerankModel,
		EmbedURL:       cfg.LLM.EmbeddingBaseURL,
		ExpandURL:      cfg.LLM.ExpansionBaseURL,
		RerankURL:      cfg.LLM.RerankBaseURL,
	})
}

func runIndex(msg string) error {
	r, err := getRuntime()
	if err != nil {
		return err
	}
	idx := indexer.New(r.Config(), r.TSClient(), makeLLMClient())
	ctx := context.Background()
	progress := func(m string) { fmt.Println(m) }
	result, err := idx.UpdateAll(ctx, progress)
	if err != nil {
		return fmt.Errorf("%s failed: %w", msg, err)
	}
	fmt.Println()
	fmt.Printf("%s complete: %d indexed, %d skipped, %d chunks\n", msg, result.Indexed, result.Skipped, result.ChunkCount)
	if len(result.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "Errors (%d):\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  %s\n", e)
		}
	}
	return nil
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Index or re-index all collections",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runIndex("Update")
	},
}

var embedCmd = &cobra.Command{
	Use:   "embed",
	Short: "Re-embed all documents (when embedding model changes)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runIndex("Embed")
	},
}
