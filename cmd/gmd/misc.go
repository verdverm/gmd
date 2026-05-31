package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List indexed documents",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("ls (not yet implemented, Phase 4)")
		return nil
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run diagnostics on GMD configuration and index",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("doctor (not yet implemented, Phase 4)")
		return nil
	},
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove stale chunks for deleted or changed files",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("cleanup (not yet implemented, Phase 4)")
		return nil
	},
}
