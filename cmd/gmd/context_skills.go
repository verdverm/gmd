package main

import (
	"github.com/spf13/cobra"
)

var contextSkillsCmd = &cobra.Command{
	Use:   "skills [list|show]",
	Short: "Agent skill templates",
	Long: `View embedded agent skill templates for AI coding assistants.

Skill templates provide specialized instructions and workflows for
AI assistants operating on GMD wikis.

Examples:
  gmd context skills list
  gmd context skills show gmd-wiki`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	contextSkillsCmd.AddCommand(contextSkillsListCmd)
	contextSkillsCmd.AddCommand(contextSkillsShowCmd)
}
