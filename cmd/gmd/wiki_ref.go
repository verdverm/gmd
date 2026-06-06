package main

import (
	"github.com/spf13/cobra"
)

var wikiRefCmd = &cobra.Command{
	Use:   "ref [add|rm|list]",
	Short: "Manage wiki source references",
	Long: `Manage source references for a wiki. Source references allow a wiki to
aggregate content from collections or other wikis for search.

Workflow:
  gmd wiki ref add mywiki docs
  gmd wiki ref list mywiki
  gmd wiki ref rm mywiki docs`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	wikiCmd.AddCommand(wikiRefCmd)
}
