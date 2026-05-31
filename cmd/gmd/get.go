package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/ts"
)

var getPath string

var getCmd = &cobra.Command{
	Use:   "get <path>",
	Short: "Retrieve full document content by file path",
	Long: `Fetches the complete content of a document from the index by its relative
file path. Results are displayed with collection name and relevance score.

Example:
  gmd get docs/guides/deployment.md`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}

		path := args[0]
		results, err := r.TSClient().TextSearch(context.Background(), ts.HybridSearchParams{
			Query:      "",
			FilterBy:   fmt.Sprintf("path:=%s", path),
			Limit:      10,
			GroupLimit: 10,
		})
		if err != nil {
			return fmt.Errorf("searching for %q: %w", path, err)
		}

		if len(results) == 0 {
			fmt.Printf("no documents found matching %q\n", path)
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

var multiGetCmd = &cobra.Command{
	Use:   "multi-get <pattern>",
	Short: "Batch fetch documents by path pattern",
	Long: `Batch fetch documents by path pattern.
The pattern is a Typesense filter expression on the path field,
e.g. "path:docs/notes" or "path:docs/*".`,
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
