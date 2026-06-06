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

var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "Manage LLM providers and profiles",
	Long:  `View provider status, list profiles, and test provider connections.`,
}

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

var llmProvidersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List configured LLM providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}
		if len(cfg.LLM.Providers) == 0 {
			fmt.Println("No providers configured (using legacy flat config)")
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

var llmProfilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "List configured LLM profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}
		if len(cfg.LLM.Profiles) == 0 {
			fmt.Println("No profiles configured")
			return nil
		}
		active := cfg.LLM.Profile
		if active == "" {
			active = "default"
		}
		for name, profile := range cfg.LLM.Profiles {
			marker := " "
			if name == active {
				marker = "*"
			}
			fmt.Printf("%s %s\n", marker, name)
		printRole("embedding", profile.Embedding)
		printRole("expansion", profile.Expansion)
		printRole("rerank", profile.Rerank)
		printRole("summarizing", profile.Summarizing)
		printRole("general_big", profile.GeneralBig)
		printRole("general_mid", profile.GeneralMid)
		printRole("general_small", profile.GeneralSmall)
		}
		return nil
	},
}

func printRole(label string, rc *config.LLMRoleConfig) {
	if rc == nil || rc.Model == "" {
		return
	}
	fmt.Printf("  %-15s provider=%-15s model=%s\n", label, rc.Provider, rc.Model)
}

var llmProfileShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show role->provider mappings for a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}
		name := args[0]
		profile, ok := cfg.LLM.Profiles[name]
		if !ok {
			return fmt.Errorf("profile %q not found", name)
		}
		fmt.Printf("Profile: %s\n", name)
		printRole("embedding", profile.Embedding)
		printRole("expansion", profile.Expansion)
		printRole("rerank", profile.Rerank)
		printRole("summarizing", profile.Summarizing)
		printRole("general_big", profile.GeneralBig)
		printRole("general_mid", profile.GeneralMid)
		printRole("general_small", profile.GeneralSmall)
		return nil
	},
}

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

func init() {
	llmCmd.AddCommand(llmStatusCmd)
	llmCmd.AddCommand(llmProvidersCmd)
	llmCmd.AddCommand(llmProfilesCmd)
	llmCmd.AddCommand(llmProfileShowCmd)
	llmCmd.AddCommand(llmTestCmd)
	rootCmd.AddCommand(llmCmd)
}
