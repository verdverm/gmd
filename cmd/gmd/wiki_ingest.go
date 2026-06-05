package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/wiki"
)

var wikiIngestCmd = &cobra.Command{
	Use:   "ingest <name> <source> [--batch]",
	Short: "Ingest a source into the wiki using the built-in agent",
	Long: `Feeds a source file (PDF, text, markdown, docx) to the wiki agent which
reads, analyzes, and writes structured wiki pages with wikilinks.

The agent automatically creates or updates pages, flags contradictions,
and appends to the wiki log.

Example:
  gmd wiki ingest mywiki paper.pdf`,
	Args: cobra.MinimumNArgs(2),
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
		llmClient := llm.New(llmConfigFromConfig(cfg))

		agent := wiki.NewAgent(w, cfg, tsClient, llmClient)

		sourcePath := args[1]
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
