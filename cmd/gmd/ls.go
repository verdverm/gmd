package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/output"
)

func collectionNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Collections))
	for n := range cfg.Collections {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

var lsCmd = &cobra.Command{
	Use:   "ls [collection]",
	Short: "List indexed documents grouped by collection",
	Long: `Lists all indexed documents grouped by collection, with one file
path per line.

Optionally filter by collection name. Useful for verifying what has been
indexed and browsing available content.

Examples:
  gmd ls
  gmd ls docs`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}

		cfg := r.Config()
		for _, c := range args {
			if _, ok := cfg.Collections[c]; !ok {
				return fmt.Errorf("unknown collection %q; available: %s", c, strings.Join(collectionNames(cfg), ", "))
			}
		}
		cols := make([]string, len(args))
		for i, c := range args {
			cols[i] = cfg.CollectionKey(c)
		}

		results, err := r.TSClient().ListDocuments(context.Background(), cols)
		if err != nil {
			return fmt.Errorf("listing documents: %w", err)
		}

		// Reverse-map stored collection keys back to user-facing names
		revCol := make(map[string]string)
		for name := range cfg.Collections {
			revCol[cfg.CollectionKey(name)] = name
		}
		for i := range results {
			if n, ok := revCol[results[i].Collection]; ok {
				results[i].Collection = n
			}
		}

		fmt.Print(output.FormatLS(results))
		return nil
	},
}
