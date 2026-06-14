package main

import (
	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/agent"
)

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
	agentSessionCmd.AddCommand(agentSessionMergeCmd)
}
