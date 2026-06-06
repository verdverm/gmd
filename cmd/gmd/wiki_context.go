package main

import (
	"github.com/spf13/cobra"
)

var wikiContextCmd = &cobra.Command{
	Use:   "context [add|list|rm]",
	Short: "Manage context documents attached to wikis",
	Long: `Context documents provide additional text that is injected alongside
search results to give AI assistants domain knowledge about a wiki.

Workflow:
  gmd wiki context add mywiki ./CONTEXT.md
  gmd wiki context list
  gmd wiki context rm mywiki`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	wikiCmd.AddCommand(wikiContextCmd)
}
