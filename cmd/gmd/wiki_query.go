package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/wiki"
)

var wikiQueryCmd = &cobra.Command{
	Use:   "query <name> <question> [--save] [--limit N]",
	Short: "Query the wiki using the built-in agent",
	Long: `Searches the wiki and synthesizes an answer with citations using the LLM.
Results are grounded in wiki content with source references.

Use --save to persist the answer as a new wiki page. Use --limit to
control how many pages are searched.

Example:
  gmd wiki query mywiki "What are the key findings?" --save`,
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
		llmClient, err := llmConfigFromConfig(cfg)
		if err != nil {
			return fmt.Errorf("resolving LLM config: %w", err)
		}

		agent := wiki.NewAgent(w, cfg, tsClient, llmClient)

		limit, _ := cmd.Flags().GetInt("limit")
		save, _ := cmd.Flags().GetBool("save")

		ctx := context.Background()
		result, err := agent.Query(ctx, args[1], wiki.QueryOpts{
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
