package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/web"
	"github.com/verdverm/gmd/pkg/web/exa"
)

var (
	webAgentLimit  int
	webAgentSteps  int
	webAgentDepth  string
	webAgentText   bool
	webAgentOutput string
	webAgentJSON   bool
	webAgentSave   bool
	webAgentWiki   string
)

var webAgentCmd = &cobra.Command{
	Use:   "agent <query>",
	Short: "Search the web with LLM-orchestrated multi-step agent",
	Long: `Search agent with LLM orchestration. Multi-step: search → analyze results
→ optionally search deeper → synthesize. The LLM acts as the "brain" deciding
what to search for next, while EXA does the actual retrieval.

Examples:
  gmd web agent "what are the latest developments in Go 1.24?"
  gmd web agent "compare Nuxt 4 vs Next.js 16" --depth deep --save
  gmd web agent "rust async runtime performance" --steps 5 --text`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getRuntime()
		if err != nil {
			return err
		}

		config := cfg.Config()

		if config.Web.EXA.APIKey == "" {
			return fmt.Errorf("EXA_API_KEY environment variable is not set")
		}

		exaClient := exa.New(config.Web.EXA.APIKey)

		llmClient := llm.New(llmConfigFromConfig(config))

		maxSteps := webAgentSteps
		switch webAgentDepth {
		case "shallow":
			maxSteps = 1
		case "deep":
			maxSteps = 5
		}

		agent := web.NewAgent(exaClient, llmClient, web.AgentConfig{
			MaxSteps:       maxSteps,
			ResultsPerStep: webAgentLimit,
			FetchText:      webAgentText,
		})

		ctx := context.Background()
		result, err := agent.Run(ctx, args[0])
		if err != nil {
			return fmt.Errorf("agent: %w", err)
		}

		if webAgentJSON {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Println(result.Answer)
		fmt.Println()
		fmt.Println("Sources:")
		for i, s := range result.Sources {
			fmt.Printf("  %d. [%s](%s)\n", i+1, s.Title, s.URL)
		}

		return nil
	},
}

func init() {
	webAgentCmd.Flags().IntVarP(&webAgentLimit, "limit", "n", 5, "Max results per search step")
	webAgentCmd.Flags().IntVar(&webAgentSteps, "steps", 3, "Max search steps")
	webAgentCmd.Flags().StringVar(&webAgentDepth, "depth", "medium", "Research depth: shallow, medium, deep")
	webAgentCmd.Flags().BoolVar(&webAgentText, "text", false, "Fetch full text for results (not just highlights)")
	webAgentCmd.Flags().StringVar(&webAgentOutput, "output", "markdown", "Output format: json or markdown")
	webAgentCmd.Flags().BoolVarP(&webAgentJSON, "json", "", false, "Short for --output json")
	webAgentCmd.Flags().BoolVarP(&webAgentSave, "save", "s", false, "Save result to wiki/synthesis/ (requires wiki)")
	webAgentCmd.Flags().StringVar(&webAgentWiki, "wiki", "", "Wiki name to save to (default: first wiki found)")
}
