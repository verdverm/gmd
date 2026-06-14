package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var llmProvidersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List configured LLM providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}
		if len(cfg.LLM.Providers) == 0 {
			fmt.Println("No providers configured")
			return nil
		}
		for name, pc := range cfg.LLM.Providers {
			maskedKey := ""
			if pc.Auth != "" && pc.Auth != "none" {
				maskedKey = "***"
			}
			fmt.Printf("%-15s provider=%-10s base_url=%-40s auth=%-15s key=%s\n",
				name, pc.Provider, pc.BaseURL, pc.Auth, maskedKey)
		}
		return nil
	},
}

func init() {
	llmCmd.AddCommand(llmProvidersCmd)
}
