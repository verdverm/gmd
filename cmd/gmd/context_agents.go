package main

import (
	"github.com/spf13/cobra"
)

var contextAgentsCmd = &cobra.Command{
	Use:   "agents [list|show]",
	Short: "Agent role definitions",
	Long: `View agent role definition directories.

Agent roles are directories of files (prompt, config, etc.) stored
in .gmd/agents/ (project-local) or ~/.config/gmd/agents/ (global).

Examples:
  gmd context agents list
  gmd context agents show wiki-writer
  gmd context agents list --global`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	contextAgentsCmd.AddCommand(contextAgentsListCmd)
	contextAgentsCmd.AddCommand(contextAgentsShowCmd)
}
