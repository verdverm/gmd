package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var multiGetCmd = &cobra.Command{
	Use:   "multi-get <pattern>",
	Short: "Batch fetch documents by path pattern",
	Long: `Batch fetch documents by path pattern using a Typesense filter expression.

Examples:
  gmd multi-get path:docs/notes
  gmd multi-get path:docs/*`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}

		filter := fmt.Sprintf("path:%s", args[0])
		results, err := r.TSClient().SearchChunksByPath(context.Background(), filter, 1000)
		if err != nil {
			return fmt.Errorf("searching by pattern %q: %w", args[0], err)
		}

		if len(results) == 0 {
			fmt.Printf("no documents found matching %q\n", args[0])
			return nil
		}

		for _, res := range results {
			fmt.Printf("=== %s (%s) [score: %.4f] ===\n", res.Path, res.Collection, res.Score)
			if res.Title != "" {
				fmt.Printf("Title: %s\n\n", res.Title)
			}
			fmt.Println(res.Content)
			fmt.Println()
		}
		return nil
	},
}
