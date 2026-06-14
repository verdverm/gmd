package main

import (
	"github.com/spf13/cobra"
)

var (
	contextGlobal bool
	contextTarget string
)

var contextCmd = &cobra.Command{
	Use:   "context [status|install|uninstall|list|show|agentsmd|skills|agents]",
	Short: "Manage agent context: skills, AGENTS.md docs, and agent definitions",
	Long: `Consolidated agent context management for AI coding assistants.

Manage embedded AGENTS.md reference content, agent skill templates,
and agent role definitions from a single command.

Categories:
  agentsmd    AGENTS.md reference documents for AI assistants
  skills      agent skill templates (install/uninstall to harness paths)
  agents      agent role definitions (directory of directories)

Top-level operations:
  status      show installed skills and available context items
  install     write skill templates to agent discovery paths
  uninstall   remove skill templates from agent discovery paths
  list        flat list of all context items across categories
  show        output full content of a named context item

Use --global to target global (home directory) scope instead of project-local.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	contextCmd.PersistentFlags().BoolVar(&contextGlobal, "global", false, "Target global (home directory) scope instead of project-local")

	contextCmd.AddCommand(contextStatusCmd)
	contextCmd.AddCommand(contextInstallCmd)
	contextCmd.AddCommand(contextUninstallCmd)
	contextCmd.AddCommand(contextListCmd)
	contextCmd.AddCommand(contextShowCmd)
	contextCmd.AddCommand(contextAgentsmdCmd)
	contextCmd.AddCommand(contextSkillsCmd)
	contextCmd.AddCommand(contextAgentsCmd)
}
