package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
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
		pc, ok := cfg.LLM.Providers[providerName]
		if !ok {
			return fmt.Errorf("provider %q not found", providerName)
		}

		providerCfg := llm.ProviderConfig{
			Name:     pc.Name,
			BaseURL:  pc.BaseURL,
			Auth:     pc.Auth,
			AuthData: pc.AuthData,
		}
		client, err := llm.BuildClient(providerCfg)
		if err != nil {
			return fmt.Errorf("building client: %w", err)
		}

		model := findModelForProvider(cfg, providerName)
		if model == "" {
			page, err := client.Models.List(context.Background())
			if err != nil {
				return fmt.Errorf("listing models: %w", err)
			}
			for _, m := range page.Data {
				lower := strings.ToLower(m.ID)
				if strings.Contains(lower, "chat") || strings.Contains(lower, "gpt") ||
					strings.Contains(lower, "claude") || strings.Contains(lower, "qwen") ||
					strings.Contains(lower, "instruct") {
					model = m.ID
					break
				}
			}
			if model == "" && len(page.Data) > 0 {
				model = page.Data[0].ID
			}
		}
		if model == "" {
			return fmt.Errorf("no model found for provider %q", providerName)
		}

		fmt.Printf("Testing %s with model %s...\n", providerName, model)

		resp, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
			Model: shared.ChatModel(model), //nolint:unconvert // required for string→ChatModel conversion
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("Say hello and identify yourself briefly."),
			},
		})
		if err != nil {
			return fmt.Errorf("chat: %w", err)
		}

		if len(resp.Choices) > 0 {
			fmt.Println(resp.Choices[0].Message.Content)
		} else {
			fmt.Println("No response received")
		}
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
