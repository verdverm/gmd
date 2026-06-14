package main

import (
	"github.com/spf13/cobra"
)

var contextAgentsmdCmd = &cobra.Command{
	Use:   "agentsmd [list|show]",
	Short: "AGENTS.md reference documents for AI assistants",
	Long: `View embedded AGENTS.md reference content at different detail levels.

Detail levels:
  oneline   single-line description of GMD
  summary   essential commands and usage guidelines
  detailed  full command reference, config, and pipeline details
  full      complete reference with architecture and design decisions

Examples:
  gmd context agentsmd list
  gmd context agentsmd show summary`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	contextCmd.AddCommand(contextAgentsmdCmd)
}
