package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/wiki"
)

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

func init() {
	wikiCmd.AddCommand(wikiLintCmd)
}
