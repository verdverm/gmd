package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/agent"
)

var agentProfileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured agent profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}

		profiles := agent.ListProfiles(cfg)

		if len(profiles) == 0 {
			fmt.Println("No agent profiles configured.")
			return nil
		}

		fmt.Println("Profiles:")
		for _, name := range profiles {
			p := cfg.Agent.Profiles[name]
			fmt.Printf("  %s: harness=%s", name, p.Harness)
			if p.Message != "" {
				fmt.Printf(", message=%q", p.Message)
			}
			if p.Tmux {
				fmt.Printf(", tmux")
			}
			if p.Workspace {
				fmt.Printf(", workspace")
			}
			if p.Async {
				fmt.Printf(", async")
			}
			fmt.Println()
		}

		return nil
	},
}
