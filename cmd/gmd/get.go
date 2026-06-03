package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <path-or-pattern>",
	Short: "Retrieve document content by path or glob pattern",
	Long: `Fetches document content from the index. Accepts an exact file path
or a glob pattern. Pattern matching uses Typesense filter syntax with
* ? and [ ] wildcards.

Examples:
  gmd get README.md
  gmd get docs/configuration.md
  gmd get "docs/*"
  gmd get "*.md"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}

		docs, err := r.TSClient().FetchDocs(context.Background(), args[0])
		if err != nil {
			return fmt.Errorf("fetching %q: %w", args[0], err)
		}

		if len(docs) == 0 {
			fmt.Printf("no documents found matching %q\n", args[0])
			return nil
		}

		for _, doc := range docs {
			fmt.Printf("=== %s (%s) ===\n", doc.Path, doc.Collection)
			if doc.Title != "" {
				fmt.Printf("Title: %s\n\n", doc.Title)
			}
			fmt.Println(doc.Content)
			fmt.Println()
		}
		return nil
	},
}
