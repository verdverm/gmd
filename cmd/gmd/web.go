package main

import (
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Web search and content retrieval via EXA",
	Long: `Web commands for searching and fetching web content using the EXA neural search API.

EXA indexes and embeds the entire web, enabling semantic search (not just
keyword matching) and clean content extraction from any URL.

Workflows:
  1. Search:   gmd web search "your query"
  2. Fetch:    gmd web fetch https://example.com
  3. Agent:    gmd web agent "your research question" --steps 5

Requires EXA_API_KEY environment variable to be set.`,
}

func init() {
	webCmd.AddCommand(webFetchCmd)
	webCmd.AddCommand(webSearchCmd)
	webCmd.AddCommand(webAgentCmd)
	rootCmd.AddCommand(webCmd)
}
