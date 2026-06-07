package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/agent"
)

var agentSessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage agent sessions and workspaces",
	Long: `Manage active tmux sessions and git worktree workspaces.

Commands:
  list      List active sessions and orphaned workspaces
  kill      Kill a tmux session and remove its workspace
  merge     Merge a workspace into the current branch`,
}

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

var agentSessionKillCmd = &cobra.Command{
	Use:   "kill <name>",
	Short: "Kill a session and remove its workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}

		mgr := agent.NewSessionManager(cfg.ProjectRoot)
		return mgr.KillSession(args[0])
	},
}

var agentSessionMergeCmd = &cobra.Command{
	Use:   "merge <name>",
	Short: "Merge a workspace into the current branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}

		squash, _ := cmd.Flags().GetBool("squash")
		mgr := agent.NewSessionManager(cfg.ProjectRoot)
		return mgr.MergeSession(args[0], squash)
	},
}

func init() {
	agentSessionMergeCmd.Flags().Bool("squash", false, "Squash all workspace commits into a single change")

	agentSessionCmd.AddCommand(agentSessionListCmd)
	agentSessionCmd.AddCommand(agentSessionKillCmd)
	agentSessionCmd.AddCommand(agentSessionMergeCmd)
}
