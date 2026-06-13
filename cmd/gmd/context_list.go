package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/context/agents"
	"github.com/verdverm/gmd/pkg/context/agentsmd"
	"github.com/verdverm/gmd/pkg/context/skills"
)

var contextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all context items across all categories",
	Long: `Flat list of all available context items: AGENTS.md detail levels,
skills, and agent role definitions.

Example:
  gmd context list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("AGENTS.md documents:")
		valid, err := agentsmd.ValidNames()
		if err != nil {
			return err
		}
		for _, name := range valid {
			fmt.Printf("  agentsmd/%s\n", name)
		}

		fmt.Println("\nSkills:")
		names, err := skills.ListSkillNames()
		if err != nil {
			return err
		}
		for _, name := range names {
			fmt.Printf("  skills/%s\n", name)
		}

		cfg, err := getConfig()
		if err != nil {
			return err
		}
		projectRoot := cfg.ProjectRoot

		if contextGlobal {
			projectRoot = ""
		}

		fmt.Println("\nAgent role definitions:")
		agentNames, err := agents.ListAgents(contextGlobal, projectRoot)
		if err != nil {
			return err
		}
		if len(agentNames) == 0 {
			fmt.Println("  (none)")
		} else {
			for _, name := range agentNames {
				fmt.Printf("  agents/%s\n", name)
			}
		}

		return nil
	},
}
