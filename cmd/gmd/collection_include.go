package main

import (
	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var collectionIncludeCmd = &cobra.Command{
	Use:   "include <name> <patterns...>",
	Short: "Set the file glob patterns for a collection",
	Long: `Sets the file-matching patterns for a collection (e.g. "**/*.md").

Example:
  gmd collection include mydocs "**/*.md" "**/*.txt"

Run 'gmd update' after changing patterns to re-index with the new
matching rules.`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		return config.SetCollectionPatterns(r.Config(), args[0], args[1:])
	},
}

func init() {
	collectionCmd.AddCommand(collectionIncludeCmd)
}
