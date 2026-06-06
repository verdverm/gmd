package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
		printConfigSources(cfg)

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

func printConfigSources(cfg *config.Config) {
	fmt.Fprintln(os.Stderr, "\n--- config sources ---")

	// Global config dir
	globalDir, err := config.GlobalConfigDir()
	if err == nil {
		globalFile := filepath.Join(globalDir, "config.cue")
		tag := "(found)"
		if _, e := os.Stat(globalFile); e != nil {
			tag = "(not found)"
		}
		fmt.Fprintf(os.Stderr, "  global CUE:      %s %s\n", globalFile, tag)
		dumpFileContents(globalFile)

		globalEnv := filepath.Join(globalDir, "default.env")
		tag = "(found)"
		if _, e := os.Stat(globalEnv); e != nil {
			tag = "(none)"
		}
		fmt.Fprintf(os.Stderr, "  global env:      %s %s\n", globalEnv, tag)
		dumpFileContents(globalEnv)

		globalSec := filepath.Join(globalDir, "secret.env")
		tag = "(found)"
		if _, e := os.Stat(globalSec); e != nil {
			tag = "(none)"
		}
		fmt.Fprintf(os.Stderr, "  global secret:   %s %s\n", globalSec, tag)

	} else {
		fmt.Fprintf(os.Stderr, "  global dir:      error: %v\n", err)
	}

	// Project config
	if cfg.ProjectRoot != "" {
		projFile := filepath.Join(cfg.ProjectRoot, ".gmd", "config.cue")
		tag := "(found)"
		if _, e := os.Stat(projFile); e != nil {
			tag = "(not found)"
		}
		fmt.Fprintf(os.Stderr, "  project CUE:     %s %s\n", projFile, tag)
		dumpFileContents(projFile)

		projEnv := filepath.Join(cfg.ProjectRoot, ".gmd", "default.env")
		tag = "(found)"
		if _, e := os.Stat(projEnv); e != nil {
			tag = "(none)"
		}
		fmt.Fprintf(os.Stderr, "  project env:     %s %s\n", projEnv, tag)
		dumpFileContents(projEnv)

		projSec := filepath.Join(cfg.ProjectRoot, ".gmd", "secret.env")
		tag = "(found)"
		if _, e := os.Stat(projSec); e != nil {
			tag = "(none)"
		}
		fmt.Fprintf(os.Stderr, "  project secret:  %s %s\n", projSec, tag)

		fmt.Fprintf(os.Stderr, "  project root:    %s\n", cfg.ProjectRoot)
	} else {
		fmt.Fprintln(os.Stderr, "  project CUE:     (no project root detected)")
	}

	// --env and --secret CLI flags
	fmt.Fprintf(os.Stderr, "  --env flags:     ")
	if verboseFlag && len(envFlag) > 0 {
		fmt.Fprintln(os.Stderr)
		for _, e := range envFlag {
			fmt.Fprintf(os.Stderr, "    %s\n", e)
		}
	} else if len(envFlag) > 0 {
		fmt.Fprintf(os.Stderr, "%s\n", strings.Join(envFlag, ", "))
	} else {
		fmt.Fprintln(os.Stderr, "(none)")
	}
	fmt.Fprintf(os.Stderr, "  --secret flags:  ")
	if verboseFlag && len(secretFlag) > 0 {
		fmt.Fprintln(os.Stderr)
		for _, s := range secretFlag {
			if idx := strings.IndexByte(s, '='); idx > 0 {
				fmt.Fprintf(os.Stderr, "    %s*****\n", s[:idx+1])
			} else {
				fmt.Fprintf(os.Stderr, "    %s\n", s)
			}
		}
	} else if len(secretFlag) > 0 {
		masked := make([]string, len(secretFlag))
		for i, s := range secretFlag {
			if idx := strings.IndexByte(s, '='); idx > 0 {
				masked[i] = s[:idx+1] + "*****"
			} else {
				masked[i] = s
			}
		}
		fmt.Fprintf(os.Stderr, "%s\n", strings.Join(masked, ", "))
	} else {
		fmt.Fprintln(os.Stderr, "(none)")
	}

	fmt.Fprintln(os.Stderr, "\n--- known env vars ---")
	known := []struct {
		name string
		val  string
	}{
		{"OPENAI_API_KEY", os.Getenv("OPENAI_API_KEY")},
		{"ANTHROPIC_API_KEY", os.Getenv("ANTHROPIC_API_KEY")},
		{"OPENCODE_API_KEY", os.Getenv("OPENCODE_API_KEY")},
		{"GMD_LLM_API_KEY", os.Getenv("GMD_LLM_API_KEY")},
		{"GMD_GLOBAL_CONFIG_DIR", os.Getenv("GMD_GLOBAL_CONFIG_DIR")},
		{"GMD_TYPESENSE_API_KEY", os.Getenv("GMD_TYPESENSE_API_KEY")},
		{"EXA_API_KEY", os.Getenv("EXA_API_KEY")},
		{"TAVILY_API_KEY", os.Getenv("TAVILY_API_KEY")},
		{"CLOUDFLARE_API_KEY", os.Getenv("CLOUDFLARE_API_KEY")},
		{"CLOUDFLARE_ACCOUNT_ID", os.Getenv("CLOUDFLARE_ACCOUNT_ID")},
		{"SEARXNG_BASE_URL", os.Getenv("SEARXNG_BASE_URL")},
	}
	secretSuffixes := []string{"_API_KEY", "_ACCOUNT_ID", "_api_key", "_account_id"}
	isSecret := func(name string) bool {
		for _, s := range secretSuffixes {
			if len(name) >= len(s) && name[len(name)-len(s):] == s {
				return true
			}
		}
		return false
	}
	for _, kv := range known {
		if kv.val == "" {
			continue
		}
		if isSecret(kv.name) {
			fmt.Fprintf(os.Stderr, "  %-30s *****\n", kv.name)
		} else {
			fmt.Fprintf(os.Stderr, "  %-30s %s\n", kv.name, kv.val)
		}
	}
}

func dumpFileContents(path string) {
	if !verboseFlag {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fmt.Fprintf(os.Stderr, "    %s\n", line)
	}
}
