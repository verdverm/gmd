package main

import (
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Web search and content retrieval via EXA",
	Long: `Web commands for searching and fetching web content using the EXA neural search API.

Commands:
  fetch     Fetch clean content from one or more URLs
  search    Neural web search
  agent     LLM-orchestrated search agent with synthesis

Getting started:
  Set EXA_API_KEY environment variable to your EXA API key.
  Then: gmd web search "your query"`,
}

func init() {
	webCmd.AddCommand(webFetchCmd)
	webCmd.AddCommand(webSearchCmd)
	webCmd.AddCommand(webAgentCmd)
	rootCmd.AddCommand(webCmd)
}
