package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/config"
	"github.com/verdverm/gmd/runtime"
)

var rootCmd = &cobra.Command{
	Use:   "gmd",
	Short: "GMD - Markdown search engine",
	Long: `GMD is a local search engine for markdown files.
It indexes markdown documents and provides full-text, vector, and hybrid search.
Powered by Typesense for search, no operational database required.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var globalRuntime *runtime.Runtime

func getRuntime() (*runtime.Runtime, error) {
	if globalRuntime != nil {
		return globalRuntime, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting cwd: %w", err)
	}
	cfg, err := config.Load(cwd)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	r, err := runtime.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("opening runtime: %w", err)
	}
	globalRuntime = r
	return r, nil
}

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

func init() {
	rootCmd.AddCommand(initCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
