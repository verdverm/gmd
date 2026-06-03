package main

import (
	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var collectionExcludeCmd = &cobra.Command{
	Use:   "exclude <name> <pattern>",
	Short: "Add an ignore pattern to exclude files from a collection",
	Long: `Adds a glob pattern to the collection's ignore list. Matching files will
be skipped during indexing. Multiple patterns can be added.

Example:
  gmd collection exclude docs "node_modules/**"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		return config.AddIgnorePattern(r.Config(), args[0], args[1])
	},
}

func init() {
	collectionCmd.AddCommand(collectionExcludeCmd)
}
