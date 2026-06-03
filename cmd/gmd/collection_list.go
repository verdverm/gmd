package main

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var collectionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured collections with paths and patterns",
	Long: `Displays every collection's name, root path, file pattern, and context
description as configured in .gmd/config.cue. Does not query Typesense
— shows config only.

Example:
  gmd collection list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}

		collections := r.Config().Collections
		names := make([]string, 0, len(collections))
		for name := range collections {
			names = append(names, name)
		}
		sort.Strings(names)

		if len(names) == 0 {
			fmt.Println("No collections configured.")
			return nil
		}

		for _, name := range names {
			col := collections[name]
			fmt.Printf("  %s\n", name)
			fmt.Printf("    path:    %s\n", col.Path)
			fmt.Printf("    patterns: %v\n", col.Patterns)
			if len(col.Ignore) > 0 {
				fmt.Printf("    ignore:  %v\n", col.Ignore)
			}
			if col.Context != "" {
				fmt.Printf("    context: %s\n", col.Context)
			}
		}
		return nil
	},
}

func init() {
	collectionCmd.AddCommand(collectionListCmd)
}
