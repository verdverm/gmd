package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run diagnostics on config, Typesense, and LLM endpoints",
	Long: `Checks the health of all GMD dependencies:

  - Config loading and project root detection
  - Typesense connectivity and chunk counts
  - LLM endpoint reachability and model availability

Reports OK, WARN, or FAIL for each check. Use this to troubleshoot
when search returns no results or indexing fails.`,
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

		if len(cfg.Collections) > 0 || len(cfg.Wikis) > 0 {
			sourceCount := len(cfg.Collections) + len(cfg.Wikis)
			fmt.Printf("OK     %d source(s) configured\n", sourceCount)
			for name := range cfg.Collections {
				key := cfg.CollectionKey(name)
				cnt, err := r.TSClient().CountByCollection(context.Background(), []string{key})
				if err != nil {
					fmt.Printf("  %s: (error: %v)\n", name, err)
				} else {
					fmt.Printf("  %s: %d chunks\n", name, cnt[key])
				}
			}
			for name := range cfg.Wikis {
				key := cfg.CollectionKey(name)
				cnt, err := r.TSClient().CountByCollection(context.Background(), []string{key})
				if err != nil {
					fmt.Printf("  %s (wiki): (error: %v)\n", name, err)
				} else {
					fmt.Printf("  %s (wiki): %d chunks\n", name, cnt[key])
				}
			}
		} else {
			fmt.Println("WARN   no collections or wikis configured")
		}

		// Schema validation: compare configured fields against Typesense
		tsFields, err := r.TSClient().GetSchemaFields(context.Background())
		if err != nil {
			fmt.Printf("WARN   schema: could not fetch Typesense schema (%v)\n", err)
		} else {
			tsFieldSet := make(map[string]string)
			for _, f := range tsFields {
				tsFieldSet[f.Name] = f.Type
			}
			baseFields := map[string]string{
				"collection": "string", "path": "string", "title": "string",
				"content": "string", "hash": "string", "chunk_seq": "int32",
				"total_chunks": "int32", "embedding": "float[]", "links": "string[]",
			}
			hasIssues := false
			for _, col := range cfg.Collections {
				for fname, f := range col.Fields {
					tsType, inTS := tsFieldSet[fname]
					if !inTS {
						fmt.Printf("PENDING %-20s %-8s  (not yet in Typesense, run update)\n", fname, f.Type)
						hasIssues = true
					} else if tsType != f.Type {
						fmt.Printf("WARN   %-20s config says %q but Typesense has %q\n", fname, f.Type, tsType)
						hasIssues = true
					}
				}
			}
			for _, wc := range cfg.Wikis {
				for fname, f := range wc.Fields {
					tsType, inTS := tsFieldSet[fname]
					if !inTS {
						fmt.Printf("PENDING %-20s %-8s  (not yet in Typesense, run update)\n", fname, f.Type)
						hasIssues = true
					} else if tsType != f.Type {
						fmt.Printf("WARN   %-20s config says %q but Typesense has %q\n", fname, f.Type, tsType)
						hasIssues = true
					}
				}
			}
			// Check for orphaned fields (in TS but not in any config or base)
			allConfigFields := make(map[string]bool)
			for _, col := range cfg.Collections {
				for fname := range col.Fields {
					allConfigFields[fname] = true
				}
			}
			for _, wc := range cfg.Wikis {
				for fname := range wc.Fields {
					allConfigFields[fname] = true
				}
			}
			for _, f := range tsFields {
				if _, isBase := baseFields[f.Name]; isBase {
					continue
				}
				if !allConfigFields[f.Name] {
					fmt.Printf("ORPHAN  %-20s %-8s  (in Typesense but no collection configures it)\n", f.Name, f.Type)
					hasIssues = true
				}
			}
			if !hasIssues {
				fmt.Println("OK     schema: all fields in sync")
			}
		}

		fmt.Println()
		fmt.Println("LLM Endpoints:")
		l, err := llmConfigFromConfig(cfg)
		if err != nil {
			fmt.Printf("FAIL   LLM config: %v\n", err)
			return nil
		}
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
