package main

import (
	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var collIncludeReplaceAll bool

var collectionIncludeCmd = &cobra.Command{
	Use:   "include <name> <patterns...>",
	Short: "Add file glob patterns to a collection",
	Long: `Adds file-matching patterns to a collection (e.g. "**/*.md").
By default patterns are appended; use --replace-all to replace existing.

Example:
  gmd collection include mydocs "**/*.md" "**/*.txt"
  gmd collection include mydocs --replace-all "**/*.go"

Run 'gmd update' after changing patterns to re-index with the new
matching rules.`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		return config.AddCollectionPatterns(r.Config(), args[0], args[1:], collIncludeReplaceAll)
	},
}

func init() {
	collectionIncludeCmd.Flags().BoolVar(&collIncludeReplaceAll, "replace-all", false, "Replace all existing patterns instead of appending")
	collectionCmd.AddCommand(collectionIncludeCmd)
}
