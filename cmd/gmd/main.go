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
	Short: "Markdown search engine — index, search, and retrieve local docs",
	Long: `GMD indexes local markdown files and provides full-text, vector, and hybrid
search backed by Typesense with LLM-powered query expansion and reranking.

Getting started:
  gmd init        create .gmd/config.cue
  gmd update      index all collections
  gmd query ...   full hybrid search pipeline
  gmd agentsmd    output AGENTS.md content for AI coding assistants`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip env loading for commands that don't need runtime
		// (help/completion are handled by cobra natively)
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting cwd: %w", err)
		}
		projectRoot := config.FindProjectRoot(cwd)
		return config.LoadEnvFiles(projectRoot, envFlag, secretFlag)
	},
}

var globalRuntime *runtime.Runtime
var globalConfig *config.Config

var envFlag, secretFlag []string
var verboseFlag bool

func getConfig() (*config.Config, error) {
	if globalConfig != nil {
		return globalConfig, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting cwd: %w", err)
	}
	cfg, err := config.Load(cwd)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	globalConfig = cfg
	return cfg, nil
}

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
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Show file contents in env output (excludes secrets)")
	rootCmd.PersistentFlags().StringArrayVar(&envFlag, "env", nil, "Extra env VAR=VAL (processed before --secret, can be repeated)")
	rootCmd.PersistentFlags().StringArrayVar(&secretFlag, "secret", nil, "Extra secret VAR=VAL (processed after --env, highest precedence, can be repeated)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(embedCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(vsearchCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(collectionCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(cleanupCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(agentsmdCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
