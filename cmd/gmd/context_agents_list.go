package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/context/agents"
)

var contextAgentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available agent role definitions",
	Long: `Shows all agent role directories available in the current scope.

Project-local agents live in .gmd/agents/.
Global agents live in ~/.config/gmd/agents/.

Examples:
  gmd context agents list
  gmd context agents list --global`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}
		baseDir := cfg.ProjectRoot
		isGlobal := contextGlobal
		if isGlobal || baseDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			baseDir = home
			isGlobal = true
		}

		names, err := agents.ListAgents(isGlobal, baseDir)
		if err != nil {
			return err
		}
		if len(names) == 0 {
			fmt.Println("No agent role definitions found.")
			return nil
		}
		for _, name := range names {
			fmt.Printf("  %s\n", name)
		}
		return nil
	},
}

func init() {
	contextAgentsCmd.AddCommand(contextAgentsListCmd)
}
