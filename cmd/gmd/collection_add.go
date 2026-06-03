package main

import (
	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var collAddPath string
var collAddPatterns []string

var collectionAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new collection to the config",
	Long: `Creates a new collection entry in the project config with the given name,
root path, and file glob patterns.

Example:
  gmd collection add mydocs --path ./docs --patterns "**/*.md"

After adding, run 'gmd update' to index the collection's files.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		name := args[0]
		path := collAddPath
		patterns := collAddPatterns
		if len(patterns) == 0 {
			patterns = []string{"**/*.md"}
		}
		return config.AddCollection(r.Config(), name, path, patterns)
	},
}

func init() {
	collectionAddCmd.Flags().StringVarP(&collAddPath, "path", "p", ".", "Collection root path")
	collectionAddCmd.Flags().StringSliceVarP(&collAddPatterns, "patterns", "P", []string{"**/*.md"}, "File glob patterns")
	collectionCmd.AddCommand(collectionAddCmd)
}
