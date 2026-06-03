package main

import (
	"github.com/spf13/cobra"
)

var (
	wikiName   string
	wikiPath   string
	wikiTarget string
)

var wikiCmd = &cobra.Command{
	Use:   "wiki",
	Short: "Manage a Karpathy-style LLM Wiki",
	Long: `Wiki commands for creating and managing a compounding knowledge base.

A Karpathy-style LLM wiki: ingest source material, let an AI agent write
structured wiki pages with wikilinks, then query the growing knowledge base
with citations, contradiction detection, and gap analysis.

Workflow:
  1. gmd wiki init --name mywiki        # scaffold + config
  2. gmd wiki ingest paper.pdf          # agent reads → writes wiki pages
  3. gmd wiki query "key findings"      # search + LLM synthesis
  4. gmd wiki lint                       # check health
  5. gmd wiki graph --format mermaid    # visualize wikilinks`,
}

func init() {
	wikiCmd.PersistentFlags().StringVar(&wikiName, "name", "", "Wiki name (collection name)")
	wikiCmd.PersistentFlags().StringVar(&wikiPath, "path", "", "Wiki directory path")
	wikiCmd.PersistentFlags().StringVar(&wikiTarget, "target", "", "Target agent for skills")

	rootCmd.AddCommand(wikiCmd)
}
