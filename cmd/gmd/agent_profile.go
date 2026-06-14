package main

import (
	"github.com/spf13/cobra"
)

var agentProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage agent profiles",
}

func init() {
	agentProfileCmd.AddCommand(agentProfileListCmd)
	agentProfileCmd.AddCommand(agentProfileShowCmd)
	agentCmd.AddCommand(agentProfileCmd)
}
