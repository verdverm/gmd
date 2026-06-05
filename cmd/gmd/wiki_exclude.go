package main

import (
	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var wikiExcludeReplaceAll bool

var wikiExcludeCmd = &cobra.Command{
	Use:   "exclude <name> <patterns...>",
	Short: "Add ignore patterns to a wiki",
	Long: `Adds glob patterns to the wiki's ignore list. Matching files will
be skipped during indexing. By default patterns are appended; use
--replace-all to replace existing.

Example:
  gmd wiki exclude mywiki "node_modules/**" "tmp/**"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		return config.AddIgnorePatterns(r.Config(), args[0], args[1:], wikiExcludeReplaceAll)
	},
}

func init() {
	wikiExcludeCmd.Flags().BoolVar(&wikiExcludeReplaceAll, "replace-all", false, "Replace all existing ignore patterns instead of appending")
	wikiCmd.AddCommand(wikiExcludeCmd)
}
