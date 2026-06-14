package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/wiki"
)

var wikiLintCmd = &cobra.Command{
	Use:   "lint <name>",
	Short: "Run wiki health checks (structure + content + OKF conformance)",
	Long: `Scans the wiki for orphan pages (no inbound links), broken links,
stale index entries, potential contradictions, knowledge gaps, and
Open Knowledge Format (OKF) v0.1 conformance.

OKF checks:
  - Every .md file has YAML frontmatter with a non-empty type field
  - Reserved files (index.md, log.md) follow structure
  - Bundle-root index.md declares okf_version
  - Non-root index.md files have no frontmatter

Example:
  gmd wiki lint mywiki`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		cfg := r.Config()

		name := args[0]

		wc, ok := cfg.Wikis[name]
		if !ok {
			return fmt.Errorf("wiki %q not found", name)
		}

		w, err := wiki.NewWiki(name, wc.Path, &wc)
		if err != nil {
			return err
		}

		tsClient := r.TSClient()
		llmClient, err := llmConfigFromConfig(cfg)
		if err != nil {
			return fmt.Errorf("resolving LLM config: %w", err)
		}

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
			fmt.Printf("Broken links: %d\n", len(result.BrokenLinks))
			for _, b := range result.BrokenLinks {
				fmt.Printf("  - %s from %s\n", b.LinkTarget, b.FromPage)
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

		okfReport, okfErr := wiki.ValidateOKF(w)
		if okfErr != nil {
			fmt.Printf("OKF validation error: %v\n", okfErr)
		}
		fmt.Printf("\nOKF conformance: %d pages checked, %d violations\n", okfReport.PassCount, okfReport.ErrorCount)
		for _, v := range okfReport.Violations {
			mark := "ERROR"
			if !v.IsError {
				mark = "WARN"
			}
			fmt.Printf("  %s: %s: %s\n", mark, v.Page, v.Message)
		}

		if okfReport.HasErrors() {
			return fmt.Errorf("OKF validation failed with %d error(s)", okfReport.ErrorCount)
		}

		if len(result.Orphans) == 0 && len(result.BrokenLinks) == 0 && len(result.StaleEntries) == 0 && len(result.Contradictions) == 0 {
			fmt.Println("\nWiki looks healthy!")
		}

		return nil
	},
}

func init() {
	wikiCmd.AddCommand(wikiLintCmd)
}
