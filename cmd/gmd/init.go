package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize GMD configuration in the current directory",
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
		defaultConfig := `package gmd

Config: {
	llm: {
		embedding_base_url:  "http://localhost:8001/v1"
		expansion_base_url:  "http://localhost:8002/v1"
		rerank_base_url:     "http://localhost:8003/v1"
		api_key:             ""
		embedding_model:     "google/embeddinggemma-300m"
		expansion_model:     "Qwen/Qwen3-1.7B"
		rerank_model:        "Qwen/Qwen3-Reranker-0.6B"
	}
	typesense: {
		host:    "http://localhost:8108"
		api_key: "xyz"
	}
	collections: docs: {
		path:    "."
		pattern: "**/*.md"
		context: "Project documentation"
	}
}
`
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}
		fmt.Printf("Created GMD config at %s\n", configPath)
		return nil
	},
}
