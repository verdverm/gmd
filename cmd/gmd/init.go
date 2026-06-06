package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/agentsmd"
)

//go:embed embeds
var initEmbedsFS embed.FS

func initConfigContent() string {
	data, _ := initEmbedsFS.ReadFile("embeds/init_config.cue")
	return string(data)
}

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
		config := strings.ReplaceAll(initConfigContent(), "{{PROJECT}}", project)
		if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}
		fmt.Printf("Created GMD config at %s\n", configPath)
		fmt.Println()
		fmt.Println("Tip: run 'gmd agentsmd' to get AGENTS.md content for your AI coding assistant.")
		fmt.Println()
		oneline, err := agentsmd.GetContent("oneline")
		if err != nil {
			return err
		}
		fmt.Println(oneline)
		fmt.Println()
		summary, err := agentsmd.GetContent("summary")
		if err != nil {
			return err
		}
		fmt.Println(summary)
		return nil
	},
}
