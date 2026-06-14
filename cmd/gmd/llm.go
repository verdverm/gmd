package main

import (
	"github.com/spf13/cobra"
)

var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "Manage LLM providers and profiles",
	Long:  `View provider status, list profiles, and test provider connections.`,
}

func init() {
	rootCmd.AddCommand(llmCmd)
}
