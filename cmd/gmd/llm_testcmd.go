package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
)

var llmTestCmd = &cobra.Command{
	Use:   "test <provider>",
	Short: "Quick chat test against a provider",
	Long: `Sends a simple "hello" message to test chat on a provider.

Example:
  gmd llm test vllm8000`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}
		providerName := args[0]
		if _, ok := cfg.LLM.Providers[providerName]; !ok {
			return fmt.Errorf("provider %q not found", providerName)
		}

		registry, err := newRegistry(cfg)
		if err != nil {
			return fmt.Errorf("building registry: %w", err)
		}

		model := findModelForProvider(cfg, providerName)
		if model == "" {
			// Use any role's model from this provider
			for _, role := range registry.Roles() {
				m := registry.Model(role)
				if m != nil {
					model = m.Name()
					break
				}
			}
		}
		if model == "" {
			return fmt.Errorf("no model found for provider %q", providerName)
		}

		// Find a ChatModel for this provider
		var chatModel llm.ChatModel
		for _, role := range registry.Roles() {
			m := registry.Model(role)
			if m != nil && m.Name() == model {
				chatModel = m
				break
			}
		}
		if chatModel == nil {
			// Fall back to any available model
			for _, role := range registry.Roles() {
				if m := registry.Model(role); m != nil {
					chatModel = m
					break
				}
			}
		}
		if chatModel == nil {
			return fmt.Errorf("no chat model available for provider %q", providerName)
		}

		fmt.Printf("Testing %s with model %s...\n", providerName, model)

		resp, err := chatModel.Chat(context.Background(), "", "Say hello and identify yourself briefly.")
		if err != nil {
			return fmt.Errorf("chat: %w", err)
		}

		fmt.Println(resp)
		return nil
	},
}

func init() {
	llmCmd.AddCommand(llmTestCmd)
}

func findModelForProvider(cfg *config.Config, providerName string) string {
	for _, profile := range cfg.LLM.Profiles {
		for _, rc := range []*config.LLMRoleConfig{
			profile.GeneralSmall, profile.GeneralMid, profile.GeneralBig,
		} {
			if rc != nil && rc.Provider == providerName && rc.Model != "" {
				return rc.Model
			}
		}
	}
	return ""
}
