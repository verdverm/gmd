package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/agent"
)

var agentSessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active sessions and orphaned workspaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}

		mgr := agent.NewSessionManager(cfg.ProjectRoot)
		sessions, err := mgr.ListSessions()
		if err != nil {
			return err
		}

		if len(sessions) == 0 {
			fmt.Println("No active sessions or workspaces.")
			return nil
		}

		fmt.Println("Sessions:")
		for _, s := range sessions {
			status := ""
			if s.Tmux {
				status += "tmux"
			}
			if s.Workspace {
				if status != "" {
					status += " + "
				}
				status += "workspace"
			}
			if s.Orphaned {
				status += " [orphaned]"
			}
			fmt.Printf("  %s: %s\n", s.Name, status)
		}

		return nil
	},
}

func init() {
	agentSessionCmd.AddCommand(agentSessionListCmd)
}
