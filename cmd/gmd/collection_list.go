package main

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var collectionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured collections and wikis with paths and patterns",
	Long: `Displays every collection's and wiki's name, root path, file pattern, and context
description as configured in .gmd/config.cue. Does not query Typesense
— shows config only.

Example:
  gmd collection list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}

		cfg := r.Config()

		// Build reverse index: for each collection/wiki, which wikis reference it
		referencedBy := make(map[string][]string)
		for wname, wc := range cfg.Wikis {
			for _, ref := range wc.SourceRefs {
				referencedBy[ref] = append(referencedBy[ref], wname)
			}
		}

		collections := cfg.Collections
		colNames := make([]string, 0, len(collections))
		for name := range collections {
			colNames = append(colNames, name)
		}
		sort.Strings(colNames)

		wikis := cfg.Wikis
		wikiNames := make([]string, 0, len(wikis))
		for name := range wikis {
			wikiNames = append(wikiNames, name)
		}
		sort.Strings(wikiNames)

		if len(colNames) == 0 && len(wikiNames) == 0 {
			fmt.Println("No collections or wikis configured.")
			return nil
		}

		for _, name := range colNames {
			col := collections[name]
			fmt.Printf("  %s\n", name)
			fmt.Printf("    path:     %s\n", col.Path)
			fmt.Printf("    patterns: %v\n", col.Patterns)
			if len(col.Ignore) > 0 {
				fmt.Printf("    ignore:   %v\n", col.Ignore)
			}
			if col.Context != "" {
				fmt.Printf("    context:  %s\n", col.Context)
			}
			if refs, ok := referencedBy[name]; ok {
				fmt.Printf("    referenced by: %v\n", refs)
			}
		}

		for _, name := range wikiNames {
			wc := wikis[name]
			fmt.Printf("  %s (wiki)\n", name)
			fmt.Printf("    path:       %s\n", wc.Path)
			fmt.Printf("    wikiDir:    %s\n", wc.WikiDir)
			fmt.Printf("    rawDir:     %s\n", wc.RawDir)
			fmt.Printf("    patterns:   %v\n", wc.Patterns)
			if len(wc.Ignore) > 0 {
				fmt.Printf("    ignore:     %v\n", wc.Ignore)
			}
			if wc.Context != "" {
				fmt.Printf("    context:    %s\n", wc.Context)
			}
			if len(wc.SourceRefs) > 0 {
				fmt.Printf("    sourceRefs: %v\n", wc.SourceRefs)
			}
			if refs, ok := referencedBy[name]; ok {
				fmt.Printf("    referenced by: %v\n", refs)
			}
		}
		return nil
	},
}

func init() {
	collectionCmd.AddCommand(collectionListCmd)
}
