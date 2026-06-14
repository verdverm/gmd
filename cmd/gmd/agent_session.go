package main

import (
	"github.com/spf13/cobra"
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

func init() {
	agentCmd.AddCommand(agentSessionCmd)
}
