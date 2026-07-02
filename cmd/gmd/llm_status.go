package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var llmStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Health check all LLM providers",
	Long:  `Checks connectivity and model availability for all configured providers and roles.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}

		registry, err := newRegistry(cfg)
		if err != nil {
			fmt.Printf("LLM config: %v\n", err)
			return nil
		}

		fmt.Println("Providers & Roles:")
		statuses := registry.CheckProviders(context.Background())
		for _, s := range statuses {
			if !s.OK {
				fmt.Printf("  FAIL   %-15s %-20s (%s)\n", s.Label, s.Provider, s.Err)
				continue
			}
			fmt.Printf("  OK     %-15s %-20s model=%s\n", s.Label, s.Provider, s.Model)
		}

		return nil
	},
}

func init() {
	llmCmd.AddCommand(llmStatusCmd)
}
