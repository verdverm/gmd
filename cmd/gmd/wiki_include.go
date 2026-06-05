package main

import (
	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var wikiIncludeReplaceAll bool

var wikiIncludeCmd = &cobra.Command{
	Use:   "include <name> <patterns...>",
	Short: "Add file glob patterns to a wiki",
	Long: `Adds file-matching patterns to a wiki (e.g. "**/*.md").
By default patterns are appended; use --replace-all to replace existing.

Example:
  gmd wiki include mywiki "**/*.md" "**/*.txt"
  gmd wiki include mywiki --replace-all "**/*.go"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		return config.AddCollectionPatterns(r.Config(), args[0], args[1:], wikiIncludeReplaceAll)
	},
}

func init() {
	wikiIncludeCmd.Flags().BoolVar(&wikiIncludeReplaceAll, "replace-all", false, "Replace all existing patterns instead of appending")
	wikiCmd.AddCommand(wikiIncludeCmd)
}
