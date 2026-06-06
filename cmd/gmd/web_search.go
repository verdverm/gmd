package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/web"
	"github.com/verdverm/gmd/pkg/web/fusion"
)

var (
	webSearchLimit             int
	webSearchType              string
	webSearchText              bool
	webSearchHighlights        bool
	webSearchMaxChars          int
	webSearchJSON              bool
	webSearchNoAutoprompt      bool
	webSearchDomains           []string
	webSearchExcludeDom        []string
	webSearchDateStart         string
	webSearchDateEnd           string
	webSearchCategory          string
	webSearchAdditionalQueries []string
	webSearchSystemPrompt      string
	webSearchNoModeration      bool
	webSearchDedup             string
	webSearchSynthesize        bool
	webSearchSynthesisPrompt   string
	webSearchNoSynthesize      bool
)

var webSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Multi-provider web search with optional AI synthesis",
	Long: `Search the web across multiple providers in parallel. Results are merged and
deduplicated. An optional LLM synthesis step produces a unified, cited answer.

Examples:
  gmd web search "transformer architecture"
  gmd web search "golang generics" --type deep --limit 5 --text
  gmd web search "startup funding" --category company --date-start 2026-01-01
  gmd web search "kubernetes" --domain kubernetes.io --highlights
  gmd web search "ai safety" --search-provider exa,tavily --synthesize
  gmd web search "climate change" --dedup llm --no-synthesize`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rt, err := getRuntime()
		if err != nil {
			return err
		}
		config := rt.Config()

		providers, err := resolveSearchProviders(config.Web)
		if err != nil {
			return err
		}

		ctx := context.Background()

		extra := map[string]any{
			"search_type":     webSearchType,
			"use_autoprompt":  !webSearchNoAutoprompt,
			"category":        webSearchCategory,
			"with_text":       webSearchText,
			"with_highlights": webSearchHighlights,
			"max_chars":       webSearchMaxChars,
		}

		if webSearchDateStart != "" {
			extra["start_published_date"] = webSearchDateStart
		}
		if webSearchDateEnd != "" {
			extra["end_published_date"] = webSearchDateEnd
		}
		if webSearchSystemPrompt != "" {
			extra["system_prompt"] = webSearchSystemPrompt
		}
		if !webSearchNoModeration {
			extra["moderation"] = true
		}
		if len(webSearchAdditionalQueries) > 0 {
			extra["additional_queries"] = webSearchAdditionalQueries
		}

		opts := web.SearchOptions{
			Query:          args[0],
			NumResults:     webSearchLimit,
			IncludeDomains: webSearchDomains,
			ExcludeDomains: webSearchExcludeDom,
			Extra:          extra,
		}

		searchCfg := config.Web.Search

		dedup := searchCfg.Dedup
		if cmd.Flags().Changed("dedup") {
			dedup = webSearchDedup
		}

		synthesize := searchCfg.Synthesize
		if cmd.Flags().Changed("synthesize") {
			synthesize = webSearchSynthesize
		}
		if cmd.Flags().Changed("no-synthesize") {
			synthesize = false
		}

		synthesisPrompt := searchCfg.SynthesisPrompt
		if cmd.Flags().Changed("synthesis-prompt") {
			synthesisPrompt = webSearchSynthesisPrompt
		}

		var llmClient *llm.Client
		if synthesize || dedup == "llm" {
			llmClient = llm.New(llmConfigFromConfig(config))
		}

		fcfg := fusion.Config{
			Dedup:           dedup,
			Synthesize:      synthesize,
			SynthesisPrompt: synthesisPrompt,
			LLMClient:       llmClient,
		}

		result, err := fusion.Run(ctx, args[0], providers, opts, fcfg)
		if err != nil {
			return fmt.Errorf("searching: %w", err)
		}

		if webSearchJSON {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			for _, c := range result.Costs {
				printCost(&c)
			}
			return nil
		}

		if result.Answer != "" {
			fmt.Println(result.Answer)
			fmt.Println()
			fmt.Println("---")
			fmt.Println()
		}

		for i, r := range result.Results {
			fmt.Printf("%d. %s\n", i+1, r.Title)
			fmt.Printf("   %s\n", r.URL)

			if provider, ok := r.Extra["_provider"].(string); ok {
				fmt.Printf("   Provider: %s\n", provider)
			}
			if author, ok := r.Extra["author"].(string); ok && author != "" {
				fmt.Printf("   Author: %s\n", author)
			}
			if date, ok := r.Extra["published_date"].(string); ok && date != "" {
				fmt.Printf("   Date: %s\n", date)
			}
			if r.Score > 0 {
				fmt.Printf("   Score: %.4f\n", r.Score)
			}

			if r.Content != "" {
				fmt.Printf("\n%s\n", r.Content)
			}
			if highlights, ok := r.Extra["highlights"].([]string); ok && len(highlights) > 0 {
				fmt.Println("   Highlights:")
				for _, h := range highlights {
					fmt.Printf("     - %s\n", h)
				}
			}
			if summary, ok := r.Extra["summary"].(string); ok && summary != "" {
				fmt.Printf("   Summary: %s\n", summary)
			}

			if i < len(result.Results)-1 {
				fmt.Println()
			}
		}

		for _, c := range result.Costs {
			printCost(&c)
		}
		return nil
	},
}

func init() {
	webSearchCmd.Flags().IntVarP(&webSearchLimit, "limit", "n", 10, "Max results per provider")
	webSearchCmd.Flags().StringVar(&webSearchType, "type", "auto", "Search type: auto, fast, instant, deep-lite, deep, deep-reasoning")
	webSearchCmd.Flags().BoolVar(&webSearchText, "text", false, "Return full text content")
	webSearchCmd.Flags().BoolVar(&webSearchHighlights, "highlights", false, "Return highlights only")
	webSearchCmd.Flags().IntVar(&webSearchMaxChars, "max-chars", 5000, "Max characters when --text")
	webSearchCmd.Flags().BoolVar(&webSearchJSON, "json", false, "Output raw JSON")
	webSearchCmd.Flags().BoolVar(&webSearchNoAutoprompt, "no-autoprompt", false, "Disable EXA autoprompt")
	webSearchCmd.Flags().StringSliceVar(&webSearchDomains, "domain", nil, "Require results from domain (repeatable)")
	webSearchCmd.Flags().StringSliceVar(&webSearchExcludeDom, "exclude-domain", nil, "Exclude domain (repeatable)")
	webSearchCmd.Flags().StringVar(&webSearchDateStart, "date-start", "", "Filter by publish date start (ISO 8601)")
	webSearchCmd.Flags().StringVar(&webSearchDateEnd, "date-end", "", "Filter by publish date end (ISO 8601)")
	webSearchCmd.Flags().StringSliceVar(&webSearchAdditionalQueries, "additional-queries", nil, "Additional queries to expand search (repeatable)")
	webSearchCmd.Flags().StringVar(&webSearchSystemPrompt, "system-prompt", "", "System prompt for EXA's LLM summarization")
	webSearchCmd.Flags().BoolVar(&webSearchNoModeration, "no-moderation", false, "Disable content moderation")

	webSearchCmd.Flags().StringVar(&webSearchDedup, "dedup", "heuristic", "Dedup method: heuristic, llm, none")
	webSearchCmd.Flags().BoolVar(&webSearchSynthesize, "synthesize", true, "Synthesize results via LLM")
	webSearchCmd.Flags().StringVar(&webSearchSynthesisPrompt, "synthesis-prompt", "", "Path to custom synthesis system prompt")
	webSearchCmd.Flags().BoolVar(&webSearchNoSynthesize, "no-synthesize", false, "Disable synthesis (overrides --synthesize)")
}
