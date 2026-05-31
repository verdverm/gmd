package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/indexer"
	"github.com/verdverm/gmd/pkg/llm"
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
		fmt.Printf("OK     typesense connected (%s), %d total chunks\n", cfg.Typesense.Host, count)

		if len(cfg.Collections) > 0 {
			fmt.Printf("OK     %d collection(s) configured\n", len(cfg.Collections))
			for name := range cfg.Collections {
				key := cfg.CollectionKey(name)
				cnt, err := r.TSClient().CountByCollection(context.Background(), []string{key})
				if err != nil {
					fmt.Printf("  %s: (error: %v)\n", name, err)
				} else {
					fmt.Printf("  %s: %d chunks\n", name, cnt[key])
				}
			}
		} else {
			fmt.Println("WARN   no collections configured")
		}

		fmt.Println()
		fmt.Println("LLM Endpoints:")
		l := llm.New(llm.Config{
			APIKey:         cfg.LLM.APIKey,
			EmbeddingModel: cfg.LLM.EmbeddingModel,
			ExpansionModel: cfg.LLM.ExpansionModel,
			RerankModel:    cfg.LLM.RerankModel,
			EmbedURL:       cfg.LLM.EmbeddingBaseURL,
			ExpandURL:      cfg.LLM.ExpansionBaseURL,
			RerankURL:      cfg.LLM.RerankBaseURL,
		})
		statuses := l.CheckAll(context.Background())
		for _, s := range statuses {
			if !s.OK {
				fmt.Printf("FAIL   %-10s %s  (%s)\n", s.Label, s.URL, s.Err)
				continue
			}
			models := strings.Join(s.Models, ", ")
			fmt.Printf("OK     %-10s %s  [%s]\n", s.Label, s.URL, models)
		}
		modelCheck := func(name, model string) {
			if model == "" {
				return
			}
			for _, s := range statuses {
				for _, m := range s.Models {
					if m == model {
						return
					}
				}
			}
			fmt.Printf("WARN   %s model not found: %s\n", name, model)
		}
		modelCheck("embedding", cfg.LLM.EmbeddingModel)
		modelCheck("expansion", cfg.LLM.ExpansionModel)
		modelCheck("rerank", cfg.LLM.RerankModel)

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
