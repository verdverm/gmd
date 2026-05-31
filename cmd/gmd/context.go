package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context [add|list|rm]",
	Short: "Manage search context documents",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var contextAddCmd = &cobra.Command{
	Use:   "add <name> <path>",
	Short: "Add a context document",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("context add: %q %q (not yet implemented, Phase 4)\n", args[0], args[1])
		return nil
	},
}

var contextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List context documents",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("context list (not yet implemented, Phase 4)")
		return nil
	},
}

var contextRmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove a context document",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("context rm: %q (not yet implemented, Phase 4)\n", args[0])
		return nil
	},
}

func init() {
	contextCmd.AddCommand(contextAddCmd)
	contextCmd.AddCommand(contextListCmd)
	contextCmd.AddCommand(contextRmCmd)
}
