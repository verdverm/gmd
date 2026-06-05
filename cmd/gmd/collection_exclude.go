package main

import (
	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var collExcludeReplaceAll bool

var collectionExcludeCmd = &cobra.Command{
	Use:   "exclude <name> <patterns...>",
	Short: "Add ignore patterns to a collection",
	Long: `Adds glob patterns to the collection's ignore list. Matching files will
be skipped during indexing. By default patterns are appended; use
--replace-all to replace existing.

Example:
  gmd collection exclude docs "node_modules/**" "tmp/**"
  gmd collection exclude docs --replace-all "build/**"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		return config.AddIgnorePatterns(r.Config(), args[0], args[1:], collExcludeReplaceAll)
	},
}

func init() {
	collectionExcludeCmd.Flags().BoolVar(&collExcludeReplaceAll, "replace-all", false, "Replace all existing ignore patterns instead of appending")
	collectionCmd.AddCommand(collectionExcludeCmd)
}
