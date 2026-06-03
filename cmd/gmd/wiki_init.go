package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/wiki"
)

var wikiInitCmd = &cobra.Command{
	Use:   "init [--name <name>] [--path <path>] [--skills]",
	Short: "Create a new wiki with directory structure and config",
	Long: `Scaffolds a wiki directory with a standard layout (raw/, wiki/) and adds
a collection entry to the project config.

The wiki uses wikilinks ([[Page Name]]) for cross-references. After init,
add source material to raw/ and run 'gmd wiki ingest' to populate pages.

Example:
  gmd wiki init --name mywiki --path ./docs`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		cfg := r.Config()

		if wikiPath == "" {
			wikiPath = cfg.ProjectRoot
		}
		if !filepath.IsAbs(wikiPath) {
			wikiPath = filepath.Join(cfg.ProjectRoot, wikiPath)
		}

		if wikiName == "" {
			wikiName = filepath.Base(wikiPath)
		}

		col := config.CollectionConfig{
			Path:             wikiPath,
			Patterns:         []string{"wiki/**/*.md"},
			IncludeByDefault: true,
			Ignore:           []string{"wiki/_index.md", "wiki/_log.md"},
			Wiki: &config.WikiConfig{
				Enabled:    true,
				IndexFile:  "_index.md",
				LogFile:    "_log.md",
				GraphLinks: true,
			},
		}

		if err := wiki.InitWiki(wikiName, wikiPath, col); err != nil {
			return fmt.Errorf("initializing wiki: %w", err)
		}

		if cfg.ProjectRoot != "" {
			configPath := config.ProjectConfigPath(cfg.ProjectRoot)
			if _, err := os.Stat(configPath); err == nil {
				if err := config.AddCollection(cfg, wikiName, wikiPath, []string{"wiki/**/*.md"}); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not update config file: %v\n", err)
					fmt.Fprintf(os.Stderr, "Add this collection manually or run: gmd collection add %s --path %s --pattern 'wiki/**/*.md'\n",
						wikiName, wikiPath)
				} else {
					if err := config.AddIgnorePattern(cfg, wikiName, "wiki/_index.md"); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
					}
					if err := config.AddIgnorePattern(cfg, wikiName, "wiki/_log.md"); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
					}
					fmt.Printf("Updated %s with wiki collection %q\n", configPath, wikiName)
				}
			}
		}

		fmt.Printf("Wiki %q initialized at %s\n", wikiName, wikiPath)
		fmt.Printf("  Directory layout created under %s/\n", wikiPath)
		fmt.Printf("  Next: add sources to raw/, then run 'gmd wiki ingest <file>'\n")

		installSkills, _ := cmd.Flags().GetBool("skills")
		if installSkills {
			target := wikiTarget
			if target == "" {
				target = "all"
			}
			written, err := wiki.WriteSkills(target)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
			for _, w := range written {
				fmt.Printf("  Skill written: %s\n", w)
			}
		}

		return nil
	},
}

func init() {
	wikiInitCmd.Flags().Bool("skills", false, "Also write skill templates for agent discovery")
	wikiCmd.AddCommand(wikiInitCmd)
}
