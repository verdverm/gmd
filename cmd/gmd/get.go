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
	Short: "Get document content by path",
	Args:  cobra.ExactArgs(1),
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
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("multi-get (not yet implemented, needs glob-to-Typesense-search mapping in pkg)")
		return nil
	},
}
