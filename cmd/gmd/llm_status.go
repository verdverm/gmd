package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/llm"
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

		if len(cfg.LLM.Providers) > 0 {
			fmt.Println("Providers:")
			for name, pc := range cfg.LLM.Providers {
				providerCfg := llm.ProviderConfig{
					Name:     pc.Name,
					BaseURL:  pc.BaseURL,
					Auth:     pc.Auth,
					AuthData: pc.AuthData,
				}
				client, err := llm.BuildClient(providerCfg)
				if err != nil {
					fmt.Printf("  FAIL   %-15s build error: %v\n", name, err)
					continue
				}
				s := llm.EndpointStatus{Label: name, URL: pc.BaseURL}
				page, err := client.Models.List(context.Background())
				if err != nil {
					s.Err = err.Error()
				} else {
					s.OK = true
					for _, m := range page.Data {
						s.Models = append(s.Models, m.ID)
					}
				}
				if s.OK {
					fmt.Printf("  OK     %-15s %-50s [%d models]\n", s.Label, s.URL, len(s.Models))
				} else {
					fmt.Printf("  FAIL   %-15s %-50s (%s)\n", s.Label, s.URL, s.Err)
				}
			}
			fmt.Println()
		}

		client, err := llmConfigFromConfig(cfg)
		if err != nil {
			fmt.Printf("LLM config: %v\n", err)
			return nil
		}
		fmt.Println("Roles:")
		statuses := client.CheckAll(context.Background())
		for _, s := range statuses {
			if !s.OK {
				fmt.Printf("  FAIL   %-15s %-50s (%s)\n", s.Label, s.URL, s.Err)
				continue
			}
			fmt.Printf("  OK     %-15s %-50s model=%s\n", s.Label, s.URL, s.Model)
		}
		return nil
	},
}

func init() {
	llmCmd.AddCommand(llmStatusCmd)
}
