package main

import (
	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var collAddPath string
var collAddPatterns []string

var collectionCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new collection in the config",
	Long: `Creates a new collection entry in the project config with the given name,
root path, and file glob patterns.

Example:
  gmd collection create mydocs --path ./docs --patterns "**/*.md"

After creating, run 'gmd update' to index the collection's files.`,
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
	collectionCreateCmd.Flags().StringVarP(&collAddPath, "path", "p", ".", "Collection root path")
	collectionCreateCmd.Flags().StringSliceVarP(&collAddPatterns, "patterns", "P", []string{"**/*.md"}, "File glob patterns")
	collectionCmd.AddCommand(collectionCreateCmd)
}
