package main

import (
	"github.com/spf13/cobra"
)

var (
	wikiPath   string
	wikiTarget string
)

var wikiCmd = &cobra.Command{
	Use:   "wiki [create|list|show|remove|rename|include|exclude|ref|ingest|query|graph|lint|doctor]",
	Short: "Manage LLM wikis — knowledge bases with agent-driven operations",
	Long: `Wiki commands for creating and managing a compounding knowledge base.

A Karpathy-style LLM wiki: ingest source material, let an AI agent write
structured wiki pages with wikilinks, then query the growing knowledge base
with citations, contradiction detection, and gap analysis.

Lifecycle:
  1. gmd wiki create mywiki                # scaffold + config
  2. gmd wiki ingest mywiki paper.pdf      # agent reads → writes wiki pages
  3. gmd wiki query mywiki "key findings"  # search + LLM synthesis
  4. gmd wiki lint mywiki                  # check health
  5. gmd wiki graph mywiki --format mermaid # visualize wikilinks
  6. gmd wiki list                         # list all wikis

Wikis are first-class config entities parallel to collections. They share
Typesense chunk storage and can reference collections via sourceRefs for
aggregated search.`,
}

func init() {
	wikiCmd.PersistentFlags().StringVar(&wikiPath, "path", "", "Wiki directory path")
	wikiCmd.PersistentFlags().StringVar(&wikiTarget, "target", "", "Target agent for skills")

	rootCmd.AddCommand(wikiCmd)
}
