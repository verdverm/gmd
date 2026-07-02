package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/ts"
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
			hasIssues := false

			// Per-source field checks
			for _, col := range cfg.Collections {
				for _, d := range ts.DiffSchemaFields(col.Fields, tsFields) {
					switch d.Status {
					case "PENDING":
						fmt.Printf("PENDING %-20s %-8s  (not yet in Typesense, run update)\n", d.Name, d.ConfigType)
						hasIssues = true
					case "TYPE_MISMATCH":
						fmt.Printf("WARN   %-20s config says %q but Typesense has %q\n", d.Name, d.ConfigType, d.TSType)
						hasIssues = true
					case "ORPHANED":
						fmt.Printf("ORPHAN  %-20s %-8s  (in Typesense but no collection configures it)\n", d.Name, d.TSType)
						hasIssues = true
					}
				}
			}
			for _, wc := range cfg.Wikis {
				for _, d := range ts.DiffSchemaFields(wc.Fields, tsFields) {
					switch d.Status {
					case "PENDING":
						fmt.Printf("PENDING %-20s %-8s  (not yet in Typesense, run update)\n", d.Name, d.ConfigType)
						hasIssues = true
					case "TYPE_MISMATCH":
						fmt.Printf("WARN   %-20s config says %q but Typesense has %q\n", d.Name, d.ConfigType, d.TSType)
						hasIssues = true
					}
				}
			}

			if !hasIssues {
				fmt.Println("OK     schema: all fields in sync")
			}
		}

		fmt.Println()
		fmt.Println("LLM Endpoints:")
		registry, err := newRegistry(cfg)
		if err != nil {
			fmt.Printf("FAIL   LLM config: %v\n", err)
			return nil
		}
		statuses := registry.CheckProviders(context.Background())
		for _, s := range statuses {
			if !s.OK {
				fmt.Printf("FAIL   %-15s %s  (%s)\n", s.Label, s.Provider, s.Err)
				continue
			}
			fmt.Printf("OK     %-15s %s  model=%s\n", s.Label, s.Provider, s.Model)
		}

		return nil
	},
}
