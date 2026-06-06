package main

import (
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context [add|list|rm]",
	Short: "Manage context documents attached to collections",
	Long: `Context documents provide additional text that is injected alongside
search results to give AI assistants domain knowledge about a collection.

The content is stored in the config file and served with every search
result from that collection — useful for adding project overviews,
glossaries, or usage guidelines.

Workflow:
  gmd context add docs ./CONTEXT.md
  gmd context list
  gmd context rm docs`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}
