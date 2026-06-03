package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/wiki"
)

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

func init() {
	wikiQueryCmd.Flags().Bool("save", false, "Save answer as new wiki page")
	wikiQueryCmd.Flags().Int("limit", 5, "Number of pages to search")
	wikiCmd.AddCommand(wikiQueryCmd)
}
