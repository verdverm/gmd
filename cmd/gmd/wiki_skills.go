package main

import (
	"github.com/spf13/cobra"
)

var wikiSkillsCmd = &cobra.Command{
	Use:   "skills [list|show|write]",
	Short: "Manage wiki agent skill templates",
	Long: `Manage embedded agent skill templates for AI coding assistants.

Workflow:
  gmd wiki skills list                # see available templates
  gmd wiki skills show <name>         # view a template
  gmd wiki skills write --target all  # install to agent discovery paths`,
}

func init() {
	wikiCmd.AddCommand(wikiSkillsCmd)
}
