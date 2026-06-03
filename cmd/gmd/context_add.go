package main

import (
	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var contextAddCmd = &cobra.Command{
	Use:   "add <collection> <path>",
	Short: "Attach a text file as context to a collection",
	Long: `Associates a text file with a collection. The file's content is stored in
the config and served alongside search results to provide AI assistants
with domain-specific knowledge about the collection.

Example:
  gmd context add docs ./CONTEXT.md`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		return config.AddContextDoc(r.Config(), args[0], args[1])
	},
}

func init() {
	contextCmd.AddCommand(contextAddCmd)
}
