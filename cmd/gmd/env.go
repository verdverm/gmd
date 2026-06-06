package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Print resolved config with secrets masked",
	Long: `Prints the fully resolved GMD configuration (global + project CUE + env vars)
with all secret values replaced by *****.

Use this to verify your configuration is being loaded correctly. The output
shows the effective config that GMD commands will use at runtime.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		b, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling config: %w", err)
		}

		fmt.Print(maskSecrets(string(b)))

		printHiddenSecrets(cfg)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}

var reSecretValue = regexp.MustCompile(`("(?:[^"]*_api_key|[^"]*_account_id|[^"]*_api_key_[^"]*)":\s*)"[^"]*"`)

func maskSecrets(s string) string {
	return reSecretValue.ReplaceAllString(s, `${1}"*****"`)
}

func printHiddenSecrets(cfg *config.Config) {
	var items []string

	add := func(label, value string) {
		if value != "" {
			items = append(items, fmt.Sprintf("  %s: *****", label))
		}
	}

	add("GMD_TYPESENSE_API_KEY", cfg.Typesense.APIKey)
	add("OPENAI_API_KEY", cfg.LLM.APIKey)
	add("EXA_API_KEY", cfg.Web.EXA.APIKey)
	add("TAVILY_API_KEY", cfg.Web.Tavily.APIKey)
	add("CLOUDFLARE_API_KEY", cfg.Web.Cloudflare.APIKey)
	add("CLOUDFLARE_ACCOUNT_ID", cfg.Web.Cloudflare.AccountID)

	if len(items) > 0 {
		fmt.Fprintln(os.Stderr, "\n--- secret env vars (json:\"-\") ---")
		for _, item := range items {
			fmt.Fprintln(os.Stderr, item)
		}
	}
}
