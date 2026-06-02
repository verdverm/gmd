package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/wiki"
)

var (
	wikiName   string
	wikiPath   string
	wikiTarget string
)

var wikiCmd = &cobra.Command{
	Use:   "wiki",
	Short: "Manage a Karpathy-style LLM Wiki",
	Long: `Wiki commands for creating and managing a compounding knowledge base.

A Karpathy-style LLM wiki: ingest source material, let an AI agent write
structured wiki pages with wikilinks, then query the growing knowledge base
with citations, contradiction detection, and gap analysis.

Workflow:
  1. gmd wiki init --name mywiki        # scaffold + config
  2. gmd wiki ingest paper.pdf          # agent reads → writes wiki pages
  3. gmd wiki query "key findings"      # search + LLM synthesis
  4. gmd wiki lint                       # check health
  5. gmd wiki graph --format mermaid    # visualize wikilinks`,
}

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

var wikiIngestCmd = &cobra.Command{
	Use:   "ingest <source> [--name <name>] [--batch]",
	Short: "Ingest a source into the wiki using the built-in agent",
	Long: `Feeds a source file (PDF, text, markdown, docx) to the wiki agent which
reads, analyzes, and writes structured wiki pages with wikilinks.

The agent automatically creates or updates pages, flags contradictions,
and appends to the wiki log.

Example:
  gmd wiki ingest paper.pdf --name mywiki`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		cfg := r.Config()

		if wikiName == "" {
			return fmt.Errorf("wiki name required (--name)")
		}

		col, ok := cfg.Collections[wikiName]
		if !ok {
			return fmt.Errorf("wiki collection %q not found", wikiName)
		}

		w, err := wiki.NewWiki(wikiName, col.Path, col)
		if err != nil {
			return err
		}

		tsClient := r.TSClient()
		llmClient := llm.New(llmConfigFromConfig(cfg))

		agent := wiki.NewAgent(w, cfg, tsClient, llmClient)

		sourcePath := args[0]
		batchMode, _ := cmd.Flags().GetBool("batch")

		ctx := context.Background()
		report, err := agent.Ingest(ctx, sourcePath, wiki.IngestOpts{Batch: batchMode})
		if err != nil {
			return fmt.Errorf("ingesting: %w", err)
		}

		fmt.Printf("Ingested %s\n", sourcePath)
		if len(report.CreatedPages) > 0 {
			fmt.Printf("  Created %d pages:\n", len(report.CreatedPages))
			for _, p := range report.CreatedPages {
				fmt.Printf("    + %s\n", p)
			}
		}
		if len(report.UpdatedPages) > 0 {
			fmt.Printf("  Updated %d pages:\n", len(report.UpdatedPages))
			for _, p := range report.UpdatedPages {
				fmt.Printf("    ~ %s\n", p)
			}
		}
		if len(report.Contradictions) > 0 {
			fmt.Printf("  Contradictions flagged:\n")
			for _, c := range report.Contradictions {
				fmt.Printf("    ! %s\n", c)
			}
		}
		if len(report.Errors) > 0 {
			fmt.Printf("  Errors:\n")
			for _, e := range report.Errors {
				fmt.Printf("    x %s\n", e)
			}
		}

		return nil
	},
}

var wikiQueryCmd = &cobra.Command{
	Use:   "query <question> [--name <name>] [--save] [--limit N]",
	Short: "Query the wiki using the built-in agent",
	Long: `Searches the wiki and synthesizes an answer with citations using the LLM.
Results are grounded in wiki content with source references.

Use --save to persist the answer as a new wiki page. Use --limit to
control how many pages are searched.

Example:
  gmd wiki query "What are the key findings?" --name mywiki --save`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		cfg := r.Config()

		if wikiName == "" {
			return fmt.Errorf("wiki name required (--name)")
		}

		col, ok := cfg.Collections[wikiName]
		if !ok {
			return fmt.Errorf("wiki collection %q not found", wikiName)
		}

		w, err := wiki.NewWiki(wikiName, col.Path, col)
		if err != nil {
			return err
		}

		tsClient := r.TSClient()
		llmClient := llm.New(llmConfigFromConfig(cfg))

		agent := wiki.NewAgent(w, cfg, tsClient, llmClient)

		limit, _ := cmd.Flags().GetInt("limit")
		save, _ := cmd.Flags().GetBool("save")

		ctx := context.Background()
		result, err := agent.Query(ctx, args[0], wiki.QueryOpts{
			Save:  save,
			Limit: limit,
		})
		if err != nil {
			return fmt.Errorf("querying: %w", err)
		}

		fmt.Println(result.Answer)
		fmt.Println()
		fmt.Println("Sources:")
		for _, s := range result.Sources {
			fmt.Printf("  - %s\n", s)
		}

		return nil
	},
}

var wikiGraphCmd = &cobra.Command{
	Use:   "graph [--name <name>] [--format dot|mermaid|json]",
	Short: "Output the wiki link graph",
	Long: `Exports the wikilink graph in DOT, Mermaid, or JSON format.

Use this to visualize relationships between wiki pages or to feed the
graph into external tooling.

Examples:
  gmd wiki graph --name mywiki --format mermaid
  gmd wiki graph --name mywiki --format dot | dot -Tpng > graph.png`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		cfg := r.Config()

		if wikiName == "" {
			return fmt.Errorf("wiki name required (--name)")
		}

		col, ok := cfg.Collections[wikiName]
		if !ok {
			return fmt.Errorf("wiki collection %q not found", wikiName)
		}

		w, err := wiki.NewWiki(wikiName, col.Path, col)
		if err != nil {
			return err
		}

		tsClient := r.TSClient()
		llmClient := llm.New(llmConfigFromConfig(cfg))

		agent := wiki.NewAgent(w, cfg, tsClient, llmClient)

		format, _ := cmd.Flags().GetString("format")

		ctx := context.Background()
		g, err := agent.BuildGraph(ctx)
		if err != nil {
			return fmt.Errorf("building graph: %w", err)
		}

		fmt.Println(wiki.FormatGraph(g, format))
		return nil
	},
}

var wikiLintCmd = &cobra.Command{
	Use:   "lint [--name <name>]",
	Short: "Run wiki health checks (structure + content analysis)",
	Long: `Scans the wiki for orphan pages (no inbound links), broken wikilinks,
stale index entries, potential contradictions, and knowledge gaps.

Run periodically to keep the wiki healthy.

Example:
  gmd wiki lint --name mywiki`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		cfg := r.Config()

		if wikiName == "" {
			return fmt.Errorf("wiki name required (--name)")
		}

		col, ok := cfg.Collections[wikiName]
		if !ok {
			return fmt.Errorf("wiki collection %q not found", wikiName)
		}

		w, err := wiki.NewWiki(wikiName, col.Path, col)
		if err != nil {
			return err
		}

		tsClient := r.TSClient()
		llmClient := llm.New(llmConfigFromConfig(cfg))

		agent := wiki.NewAgent(w, cfg, tsClient, llmClient)

		ctx := context.Background()
		result, err := agent.Lint(ctx, wiki.LintOpts{})
		if err != nil {
			return fmt.Errorf("linting: %w", err)
		}

		if len(result.Orphans) > 0 {
			fmt.Printf("Orphan pages (no inbound links): %d\n", len(result.Orphans))
			for _, o := range result.Orphans {
				fmt.Printf("  - %s\n", o)
			}
		}
		if len(result.BrokenLinks) > 0 {
			fmt.Printf("Broken wikilinks: %d\n", len(result.BrokenLinks))
			for _, b := range result.BrokenLinks {
				fmt.Printf("  - [[%s]] from %s\n", b.LinkTarget, b.FromPage)
			}
		}
		if len(result.StaleEntries) > 0 {
			fmt.Printf("Stale index entries: %d\n", len(result.StaleEntries))
			for _, s := range result.StaleEntries {
				fmt.Printf("  - %s\n", s)
			}
		}
		if len(result.Contradictions) > 0 {
			fmt.Printf("Potential contradictions: %d\n", len(result.Contradictions))
			for _, c := range result.Contradictions {
				fmt.Printf("  - %s vs %s: %s\n", c.PageA, c.PageB, strings.Split(c.Resolution, "\n")[0])
			}
		}
		if result.Gaps != "" {
			fmt.Printf("\nGap analysis:\n%s\n", result.Gaps)
		}

		if len(result.Orphans) == 0 && len(result.BrokenLinks) == 0 && len(result.StaleEntries) == 0 && len(result.Contradictions) == 0 {
			fmt.Println("Wiki looks healthy!")
		}

		return nil
	},
}

var wikiSkillsCmd = &cobra.Command{
	Use:   "skills [list|show|write]",
	Short: "Manage wiki agent skill templates",
	Long: `Manage embedded agent skill templates for AI coding assistants.

Workflow:
  gmd wiki skills list                # see available templates
  gmd wiki skills show <name>         # view a template
  gmd wiki skills write --target all  # install to agent discovery paths`,
}

var wikiSkillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available skill templates",
	Long:  "Shows all embedded wiki agent skill templates with their name, target, and description.",
	RunE: func(cmd *cobra.Command, args []string) error {
		templates, err := wiki.ListSkillTemplates()
		if err != nil {
			return err
		}
		fmt.Println("Available skill templates:")
		for _, t := range templates {
			fmt.Printf("  %-20s %-12s %s\n", t.Name, t.Target, t.Description)
		}
		return nil
	},
}

var wikiSkillsShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show a skill template",
	Long: `Displays the full content of a named skill template.

Example:
  gmd wiki skills show research-agent`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tmpl, err := wiki.GetSkillTemplate(args[0])
		if err != nil {
			return err
		}
		fmt.Println(tmpl.Content)
		return nil
	},
}

var wikiSkillsWriteCmd = &cobra.Command{
	Use:   "write [--target claude|codex|opencode|all]",
	Short: "Write skill templates to agent discovery paths",
	Long: `Installs skill templates to the appropriate agent discovery directories
so that AI coding assistants discover and use them automatically.

Examples:
  gmd wiki skills write --target all
  gmd wiki skills write --target opencode`,
	RunE: func(cmd *cobra.Command, args []string) error {
		target := wikiTarget
		if target == "" {
			target = "all"
		}

		written, err := wiki.WriteSkills(target)
		if err != nil {
			return fmt.Errorf("writing skills: %w", err)
		}
		for _, w := range written {
			fmt.Printf("  Written: %s\n", w)
		}
		return nil
	},
}

var wikiDoctorCmd = &cobra.Command{
	Use:   "doctor [--name <name>] [--fix]",
	Short: "Run wiki diagnostics and auto-configure agents",
	Long: `Checks wiki configuration, file system structure, Typesense sync state,
and agent compatibility. Reports issues and suggests fixes.

Use --fix to automatically apply safe fixes (fixable issues only).

Example:
  gmd wiki doctor --name mywiki --fix`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		cfg := r.Config()

		if wikiName == "" {
			return fmt.Errorf("wiki name required (--name)")
		}

		col, ok := cfg.Collections[wikiName]
		if !ok {
			return fmt.Errorf("wiki collection %q not found", wikiName)
		}

		w, err := wiki.NewWiki(wikiName, col.Path, col)
		if err != nil {
			return err
		}

		tsClient := r.TSClient()
		llmClient := llm.New(llmConfigFromConfig(cfg))

		ctx := context.Background()
		result, err := wiki.Doctor(ctx, w, cfg, tsClient, llmClient)
		if err != nil {
			return err
		}

		fmt.Print(wiki.FormatDoctorResult(result))

		doFix, _ := cmd.Flags().GetBool("fix")
		if doFix {
			fixes, err := wiki.DoctorFix(w)
			if err != nil {
				return err
			}
			if len(fixes) > 0 {
				fmt.Printf("\n  Fixes applied:\n")
				for _, f := range fixes {
					fmt.Println(f)
				}
			}
		}

		return nil
	},
}

func init() {
	wikiCmd.PersistentFlags().StringVar(&wikiName, "name", "", "Wiki name (collection name)")
	wikiCmd.PersistentFlags().StringVar(&wikiPath, "path", "", "Wiki directory path")
	wikiCmd.PersistentFlags().StringVar(&wikiTarget, "target", "", "Target agent for skills")

	wikiInitCmd.Flags().Bool("skills", false, "Also write skill templates for agent discovery")

	wikiIngestCmd.Flags().Bool("batch", false, "Batch mode for multi-source ingest")

	wikiQueryCmd.Flags().Bool("save", false, "Save answer as new wiki page")
	wikiQueryCmd.Flags().Int("limit", 5, "Number of pages to search")

	wikiGraphCmd.Flags().String("format", "dot", "Output format: dot, mermaid, or json")

	wikiSkillsCmd.AddCommand(wikiSkillsListCmd)
	wikiSkillsCmd.AddCommand(wikiSkillsShowCmd)
	wikiSkillsCmd.AddCommand(wikiSkillsWriteCmd)

	wikiCmd.AddCommand(wikiInitCmd)
	wikiCmd.AddCommand(wikiIngestCmd)
	wikiCmd.AddCommand(wikiQueryCmd)
	wikiCmd.AddCommand(wikiGraphCmd)
	wikiCmd.AddCommand(wikiLintCmd)
	wikiCmd.AddCommand(wikiSkillsCmd)
	wikiCmd.AddCommand(wikiDoctorCmd)

	rootCmd.AddCommand(wikiCmd)
}
