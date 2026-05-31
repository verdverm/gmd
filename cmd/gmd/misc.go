package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/indexer"
	"github.com/verdverm/gmd/pkg/ts"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List indexed documents",
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}

		results, err := r.TSClient().TextSearch(context.Background(), ts.HybridSearchParams{
			Query:      "",
			Limit:      1000,
			GroupLimit: 1,
		})
		if err != nil {
			return fmt.Errorf("listing documents: %w", err)
		}

		if len(results) == 0 {
			fmt.Println("No indexed documents.")
			return nil
		}

		for _, res := range results {
			title := res.Title
			if title == "" {
				title = res.Path
			}
			fmt.Printf("  %-40s  %s  (score: %.4f)\n", title, res.Collection, res.Score)
		}
		fmt.Printf("\n%d document(s) indexed\n", len(results))
		return nil
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run diagnostics on GMD configuration and index",
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			fmt.Printf("FAIL  config: %v\n", err)
			return nil
		}

		fmt.Println("OK     config loaded")

		cfg := r.Config()
		if cfg.ProjectRoot != "" {
			fmt.Printf("OK     project root: %s\n", cfg.ProjectRoot)
		} else {
			fmt.Println("INFO   no project root detected (not in a .gmd directory)")
		}

		count, err := r.TSClient().CollectionCount(context.Background())
		if err != nil {
			fmt.Printf("FAIL  typesense: %v\n", err)
			return nil
		}
		fmt.Printf("OK     typesense connected, %d total chunks\n", count)

		if len(cfg.Collections) > 0 {
			fmt.Printf("OK     %d collection(s) configured\n", len(cfg.Collections))
			for name := range cfg.Collections {
				cnt, err := r.TSClient().CountByCollection(context.Background(), []string{name})
				if err != nil {
					fmt.Printf("  %s: (error: %v)\n", name, err)
				} else {
					fmt.Printf("  %s: %d chunks\n", name, cnt[name])
				}
			}
		} else {
			fmt.Println("WARN   no collections configured")
		}

		if cfg.LLM.BaseURL != "" {
			fmt.Printf("OK     llm endpoint: %s\n", cfg.LLM.BaseURL)
		}
		if cfg.LLM.EmbeddingModel != "" {
			fmt.Printf("OK     embedding model: %s\n", cfg.LLM.EmbeddingModel)
		}
		if cfg.LLM.ExpansionModel != "" {
			fmt.Printf("OK     expansion model: %s\n", cfg.LLM.ExpansionModel)
		}
		if cfg.LLM.RerankModel != "" {
			fmt.Printf("OK     rerank model: %s\n", cfg.LLM.RerankModel)
		}

		return nil
	},
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove stale chunks for deleted or changed files",
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}

		idx := indexer.New(r.Config(), r.TSClient(), nil)
		ctx := context.Background()
		progress := func(m string) { fmt.Println(m) }
		results := idx.CleanupAllCollections(ctx, progress)

		total := 0
		for name, count := range results {
			fmt.Printf("[%s] Removed %d stale chunks\n", name, count)
			total += count
		}
		fmt.Printf("\nCleanup complete: %d total stale chunks removed.\n", total)
		return nil
	},
}
