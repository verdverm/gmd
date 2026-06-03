package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/wiki"
)

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

func init() {
	wikiIngestCmd.Flags().Bool("batch", false, "Batch mode for multi-source ingest")
	wikiCmd.AddCommand(wikiIngestCmd)
}
