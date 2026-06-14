package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/context/agents"
)

var contextAgentsShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show files within an agent role directory",
	Long: `Displays all files within a named agent role directory.

Example:
  gmd context agents show wiki-writer
  gmd context agents show wiki-writer --global`,
	Args: cobra.ExactArgs(1),
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

		files, err := agents.ShowAgent(args[0], isGlobal, baseDir)
		if err != nil {
			return err
		}
		for fname, content := range files {
			fmt.Printf("=== %s ===\n%s\n", fname, content)
		}
		return nil
	},
}

func init() {
	contextAgentsCmd.AddCommand(contextAgentsShowCmd)
}
