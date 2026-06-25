package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/indexer"
)

func runIndex(msg string) error {
	r, err := getRuntime()
	if err != nil {
		return err
	}
	cfg := r.Config()
	registry, err := newRegistry(cfg)
	if err != nil {
		return err
	}
	idx := indexer.New(r.Config(), r.TSClient(), registry.Embedder())
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
	Short: "Scan, chunk, embed, and index all collections",
	Long: `Scans all configured collections for markdown files, chunks them with
heading-aware breakpoints, generates embeddings via the configured LLM,
and upserts everything into Typesense.

Unchanged files are skipped via SHA-256 dedup. Run this after editing
config or changing files to keep the index in sync.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runIndex("Update")
	},
}
