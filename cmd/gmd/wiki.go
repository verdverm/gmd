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

Getting started:
  gmd wiki init         Create a new wiki with directory structure and config
  gmd wiki ingest ...   Ingest a source into the wiki using the built-in agent
  gmd wiki query "..."  Query the wiki using the built-in agent
  gmd wiki skills list  List available agent skill templates
  gmd wiki skills write Install skill templates for agent discovery`,
}

var wikiInitCmd = &cobra.Command{
	Use:   "init [--name <name>] [--path <path>] [--skills]",
	Short: "Create a new wiki with directory structure and config",
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
			Pattern:          "wiki/**/*.md",
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
				if err := config.AddCollection(cfg, wikiName, wikiPath, "wiki/**/*.md"); err != nil {
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
	Args:  cobra.MinimumNArgs(1),
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
		llmClient := llm.New(llm.Config{
			APIKey:         cfg.LLM.APIKey,
			EmbeddingModel: cfg.LLM.EmbeddingModel,
			ExpansionModel: cfg.LLM.ExpansionModel,
			RerankModel:    cfg.LLM.RerankModel,
			EmbedURL:       cfg.LLM.EmbeddingBaseURL,
			ExpandURL:      cfg.LLM.ExpansionBaseURL,
			RerankURL:      cfg.LLM.RerankBaseURL,
		})

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
	Args:  cobra.MinimumNArgs(1),
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
		llmClient := llm.New(llm.Config{
			APIKey:         cfg.LLM.APIKey,
			EmbeddingModel: cfg.LLM.EmbeddingModel,
			ExpansionModel: cfg.LLM.ExpansionModel,
			RerankModel:    cfg.LLM.RerankModel,
			EmbedURL:       cfg.LLM.EmbeddingBaseURL,
			ExpandURL:      cfg.LLM.ExpansionBaseURL,
			RerankURL:      cfg.LLM.RerankBaseURL,
		})

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
		llmClient := llm.New(llm.Config{
			APIKey:         cfg.LLM.APIKey,
			EmbeddingModel: cfg.LLM.EmbeddingModel,
			ExpansionModel: cfg.LLM.ExpansionModel,
			RerankModel:    cfg.LLM.RerankModel,
			EmbedURL:       cfg.LLM.EmbeddingBaseURL,
			ExpandURL:      cfg.LLM.ExpansionBaseURL,
			RerankURL:      cfg.LLM.RerankBaseURL,
		})

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
		llmClient := llm.New(llm.Config{
			APIKey:         cfg.LLM.APIKey,
			EmbeddingModel: cfg.LLM.EmbeddingModel,
			ExpansionModel: cfg.LLM.ExpansionModel,
			RerankModel:    cfg.LLM.RerankModel,
			EmbedURL:       cfg.LLM.EmbeddingBaseURL,
			ExpandURL:      cfg.LLM.ExpansionBaseURL,
			RerankURL:      cfg.LLM.RerankBaseURL,
		})

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
}

var wikiSkillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available skill templates",
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
	Args:  cobra.ExactArgs(1),
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
		llmClient := llm.New(llm.Config{
			APIKey:         cfg.LLM.APIKey,
			EmbeddingModel: cfg.LLM.EmbeddingModel,
			ExpansionModel: cfg.LLM.ExpansionModel,
			RerankModel:    cfg.LLM.RerankModel,
			EmbedURL:       cfg.LLM.EmbeddingBaseURL,
			ExpandURL:      cfg.LLM.ExpansionBaseURL,
			RerankURL:      cfg.LLM.RerankBaseURL,
		})

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
