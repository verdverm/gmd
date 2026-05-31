package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show index and collection health",
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		cfg := r.Config()

		fmt.Println("GMD Status")
		fmt.Println("==========")
		if cfg.ProjectRoot != "" {
			fmt.Printf("Project Root:  %s\n", cfg.ProjectRoot)
		} else {
			fmt.Println("Project Root:  (none - no project config found)")
		}

		ctx := context.Background()

		totalDocs, err := r.TSClient().CollectionCount(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot get collection count: %v\n", err)
		}

		fmt.Printf("Total Chunks:  %d\n", totalDocs)
		fmt.Println()

		colKeys := make([]string, 0, len(cfg.Collections))
		colNameForKey := make(map[string]string, len(cfg.Collections))
		for name := range cfg.Collections {
			key := cfg.CollectionKey(name)
			colKeys = append(colKeys, key)
			colNameForKey[key] = name
		}

		if len(colKeys) > 0 {
			counts, err := r.TSClient().CountByCollection(ctx, colKeys)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cannot get collection counts: %v\n", err)
			}
			fmt.Println("Collections:")
			for _, key := range colKeys {
				name := colNameForKey[key]
				col := cfg.Collections[name]
				count := counts[key]
				fmt.Printf("  %s:\n", name)
				fmt.Printf("    Path:    %s\n", col.Path)
				if col.Pattern != "" {
					fmt.Printf("    Pattern: %s\n", col.Pattern)
				}
				fmt.Printf("    Chunks:  %d\n", count)
			}
		}

		return nil
	},
}
