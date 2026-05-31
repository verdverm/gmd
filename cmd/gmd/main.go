package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/runtime"
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

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(embedCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(vsearchCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(multiGetCmd)
	rootCmd.AddCommand(collectionCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(cleanupCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(mcpCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
