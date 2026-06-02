package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/agents"
)

func detectProjectName(dir string) string {
	if out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").Output(); err == nil {
		url := strings.TrimSpace(string(out))
		url = strings.TrimSuffix(url, ".git")
		if idx := strings.LastIndexByte(url, '/'); idx >= 0 {
			return url[idx+1:]
		}
	}
	return filepath.Base(dir)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create .gmd/config.cue in the current directory",
	Long: `Creates a .gmd/ directory with a default config.cue file in the current
directory, making it a GMD project root.

The generated config includes sensible defaults for LLM endpoints and
Typesense. Edit .gmd/config.cue to customize collections and settings,
then run 'gmd update' to index your files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		gmdDir := filepath.Join(dir, ".gmd")
		if err := os.MkdirAll(gmdDir, 0755); err != nil {
			return fmt.Errorf("creating .gmd directory: %w", err)
		}
		configPath := filepath.Join(gmdDir, "config.cue")
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("config already exists at %s", configPath)
		}
		project := detectProjectName(dir)
		defaultConfig := fmt.Sprintf(`package gmd

Config: {
	project:  %q
		llm: {
		embedding_base_url:  "http://localhost:8001/v1"
		expansion_base_url:  "http://localhost:8002/v1"
		rerank_base_url:     "http://localhost:8003/v1"
		embedding_model:     "google/embeddinggemma-300m"
		expansion_model:     "Qwen/Qwen3-1.7B"
		rerank_model:        "Qwen/Qwen3-Reranker-0.6B"
		summarizing_base_url:   "http://localhost:8000/v1"
		general_big_base_url:   "http://localhost:8000/v1"
		general_mid_base_url:   "http://localhost:8000/v1"
		general_small_base_url: "http://localhost:8000/v1"
	}
	typesense: {
		host:    "http://localhost:8108"
	}
	collections: docs: {
		path:    "."
		pattern: "**/*.md"
		context: "Project documentation"
	}
}
`, project)
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}
		fmt.Printf("Created GMD config at %s\n", configPath)
		fmt.Println()
		fmt.Println("Tip: run 'gmd agents' to get AGENTS.md content for your AI coding assistant.")
		fmt.Println()
		fmt.Println(agents.MustGetContent(agents.Oneline))
		fmt.Println()
		fmt.Println(agents.MustGetContent(agents.Summary))
		return nil
	},
}
