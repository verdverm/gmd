package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/context/agents"
	"github.com/verdverm/gmd/pkg/context/agentsmd"
	"github.com/verdverm/gmd/pkg/context/skills"
)

var contextStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show installed skills and available context items",
	Long: `Cross-category status overview: available AGENTS.md detail levels,
skill installation status per harness, and available
agent role definitions.

Use --global to target global (home directory) scope.

Examples:
  gmd context status
  gmd context status --global`,
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

		scopeLabel := "project-local"
		if isGlobal {
			scopeLabel = "global"
		}
		fmt.Printf("Scope: %s (%s)\n\n", scopeLabel, baseDir)

		fmt.Println("AGENTS.md documents:")
		valid, err := agentsmd.ValidNames()
		if err != nil {
			return err
		}
		fmt.Printf("  available: %s\n\n", strings.Join(valid, ", "))

		fmt.Println("Skills:")
		names, err := skills.ListSkillNames()
		if err != nil {
			return err
		}
		fmt.Printf("  %d available:\n", len(names))
		for _, n := range names {
			fmt.Printf("    %s\n", n)
			for _, h := range skills.HarnessNames() {
				installed, err := skills.SkillInstalled(baseDir, isGlobal, h, n)
				if err != nil {
					return err
				}
				sp, err := skills.SkillPath(baseDir, isGlobal, h, n)
				if err != nil {
					return err
				}
				mark := "not installed"
				if installed {
					mark = "installed"
				}
				fmt.Printf("      %s:  %s  (%s)\n", h, mark, sp)
			}
		}

		fmt.Println("\nHarness discovery:")
		for _, h := range skills.HarnessNames() {
			detected, err := skills.CheckHarnessInstalled(baseDir, isGlobal, h)
			if err != nil {
				return err
			}
			harnessMark := "absent"
			if detected {
				harnessMark = "detected"
			}
			fmt.Printf("  %s:  %s\n", h, harnessMark)
		}

		fmt.Println("\nAgent role definitions:")
		agentNames, err := agents.ListAgents(isGlobal, baseDir)
		if err != nil {
			return err
		}
		if len(agentNames) == 0 {
			fmt.Println("  (none)")
		} else {
			for _, name := range agentNames {
				fmt.Printf("  %s\n", name)
			}
		}

		return nil
	},
}

func init() {
	contextCmd.AddCommand(contextStatusCmd)
}
