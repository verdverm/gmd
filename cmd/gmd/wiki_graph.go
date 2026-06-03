package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/wiki"
)

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

func init() {
	wikiGraphCmd.Flags().String("format", "dot", "Output format: dot, mermaid, or json")
	wikiCmd.AddCommand(wikiGraphCmd)
}
