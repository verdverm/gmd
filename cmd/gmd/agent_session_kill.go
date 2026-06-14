package main

import (
	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/agent"
)

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

func init() {
	agentSessionCmd.AddCommand(agentSessionKillCmd)
}
