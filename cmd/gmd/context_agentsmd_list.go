package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/context/agentsmd"
)

var contextAgentsmdListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available AGENTS.md detail levels",
	Long:  "Shows the available detail levels for AGENTS.md reference content.",
	RunE: func(cmd *cobra.Command, args []string) error {
		valid, err := agentsmd.ValidNames()
		if err != nil {
			return err
		}
		fmt.Println("Available detail levels:")
		for _, name := range valid {
			fmt.Printf("  %s\n", name)
		}
		return nil
	},
}
