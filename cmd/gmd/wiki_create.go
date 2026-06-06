package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/wiki"
)

var wikiCreateWikiDir string
var wikiCreateRawDir string
var wikiCreateFrom []string

var wikiCreateCmd = &cobra.Command{
	Use:   "create <name> [--path <path>] [--wiki-dir <dir>] [--raw-dir <dir>] [--skills] [--from <source>]",
	Short: "Create a new wiki with directory structure and config",
	Long: `Scaffolds a wiki directory with a standard layout (wiki/, raw/) and adds
a wiki entry to the project config.

The wiki uses wikilinks ([[Page Name]]) for cross-references. After creation,
add source material to raw/ and run 'gmd wiki ingest' to populate pages.

Example:
  gmd wiki create mywiki
  gmd wiki create mywiki --path ./docs
  gmd wiki create mywiki --wiki-dir pages --raw-dir inputs
  gmd wiki create mywiki --skills`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		cfg := r.Config()

		name := args[0]

		wikiPathStr := wikiPath
		if wikiPathStr == "" {
			wikiPathStr = cfg.ProjectRoot
		}
		if !filepath.IsAbs(wikiPathStr) {
			wikiPathStr = filepath.Join(cfg.ProjectRoot, wikiPathStr)
		}

		wd := wikiCreateWikiDir
		if wd == "" {
			wd = "wiki"
		}
		rd := wikiCreateRawDir
		if rd == "" {
			rd = "raw"
		}

		// Validation: name uniqueness
		if cfg.SourceExists(name) {
			return fmt.Errorf("source %q already exists", name)
		}

		// Validation: wikiDir != rawDir
		if wd == rd {
			return fmt.Errorf("wikiDir and rawDir must be different (both are %q)", wd)
		}

		// Validate --from sources
		var sourceRefs []string
		for _, src := range wikiCreateFrom {
			if !cfg.SourceExists(src) {
				return fmt.Errorf("source %q not found in collections or wikis", src)
			}
			sourceRefs = append(sourceRefs, src)
		}

		wcfg := &config.WikiConfig{
			SourceConfig: config.SourceConfig{
				Path:     wikiPathStr,
				Patterns: []string{wd + "/**/*.md", rd + "/**/*.md"},
				Ignore:   []string{wd + "/_index.md", wd + "/_log.md"},
			},
			WikiDir:    wd,
			RawDir:     rd,
			SourceRefs: sourceRefs,
		}

		if err := wiki.InitWiki(name, wikiPathStr, wcfg); err != nil {
			return fmt.Errorf("initializing wiki: %w", err)
		}

		if err := config.CreateWiki(cfg, name, wikiPathStr, wcfg.Patterns, wd, rd, sourceRefs); err != nil {
			return fmt.Errorf("updating config: %w", err)
		}

		// Add ignore patterns for meta files
		_ = config.AddIgnorePatterns(cfg, name, []string{wd + "/_index.md", wd + "/_log.md"}, false)

		fmt.Printf("Wiki %q initialized at %s\n", name, wikiPathStr)
		fmt.Printf("  Directory layout created under %s/\n", wikiPathStr)
		fmt.Printf("  Next: add sources to %s/, then run 'gmd wiki ingest %s <file>'\n", filepath.Join(wikiPathStr, rd), name)

		installSkills, _ := cmd.Flags().GetBool("skills")
		if installSkills {
			target := wikiTarget
			if target == "" {
				target = "all"
			}
			written, err := wiki.WriteSkills(target)
			if err != nil {
				fmt.Printf("Warning: %v\n", err)
			}
			for _, w := range written {
				fmt.Printf("  Skill written: %s\n", w)
			}
		}

		return nil
	},
}

func init() {
	wikiCreateCmd.Flags().Bool("skills", false, "Also write skill templates for agent discovery")
	wikiCreateCmd.Flags().StringVar(&wikiCreateWikiDir, "wiki-dir", "", "Wiki pages subdirectory (default: wiki)")
	wikiCreateCmd.Flags().StringVar(&wikiCreateRawDir, "raw-dir", "", "Raw source material subdirectory (default: raw)")
	wikiCreateCmd.Flags().StringSliceVar(&wikiCreateFrom, "from", nil, "Add source reference to collection or wiki")
	wikiCmd.AddCommand(wikiCreateCmd)
}
