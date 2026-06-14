package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/agent"
)

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured agent harnesses and profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}

		harnesses := agent.ListHarnesses(cfg)
		profiles := agent.ListProfiles(cfg)

		if len(harnesses) == 0 && len(profiles) == 0 {
			fmt.Println("No agent harnesses or profiles configured.")
			fmt.Println("Add an 'agent:' section to your gmd config.")
			return nil
		}

		if len(harnesses) > 0 {
			fmt.Println("Harnesses:")
			for _, name := range harnesses {
				hc := cfg.Agent.Harnesses[name]
				marker := ""
				if cfg.Agent.DefaultHarness == name {
					marker = " (default)"
				}
				fmt.Printf("  %s%s: bin=%s\n", name, marker, hc.Bin)
			}
		}

		if len(profiles) > 0 {
			fmt.Println("\nProfiles:")
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
		}

		return nil
	},
}

func init() {
	agentCmd.AddCommand(agentListCmd)
}
