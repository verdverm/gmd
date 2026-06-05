package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/web/exa"
)

var (
	webSearchLimit             int
	webSearchType              string
	webSearchText              bool
	webSearchHighlights        bool
	webSearchSummary           string
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
)

var webSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Traditional web search via EXA",
	Long: `Traditional web search using the EXA search API.

EXA indexes and embeds the entire web. Searches are semantic and rank results
by relevance rather than keyword matching.

Examples:
  gmd web search "transformer architecture"
  gmd web search "golang generics" --type deep --limit 5 --text
  gmd web search "startup funding" --category company --date-start 2026-01-01
  gmd web search "kubernetes" --domain kubernetes.io --highlights`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getRuntime()
		if err != nil {
			return err
		}

		if cfg.Config().Web.EXA.APIKey == "" {
			return fmt.Errorf("EXA_API_KEY environment variable is not set")
		}

		client := exa.New(cfg.Config().Web.EXA.APIKey)
		ctx := context.Background()

		req := exa.SearchRequest{
			Query:      args[0],
			Type:       webSearchType,
			NumResults: webSearchLimit,
			Category:   webSearchCategory,
		}

		if len(webSearchDomains) > 0 {
			req.IncludeDomains = webSearchDomains
		}
		if len(webSearchExcludeDom) > 0 {
			req.ExcludeDomains = webSearchExcludeDom
		}

		if webSearchDateStart != "" {
			t, err := time.Parse("2006-01-02", webSearchDateStart)
			if err != nil {
				return fmt.Errorf("invalid date-start: %w", err)
			}
			req.StartPublishedDate = &t
		}
		if webSearchDateEnd != "" {
			t, err := time.Parse("2006-01-02", webSearchDateEnd)
			if err != nil {
				return fmt.Errorf("invalid date-end: %w", err)
			}
			req.EndPublishedDate = &t
		}

		if !webSearchNoAutoprompt {
			req.UseAutoprompt = boolPtr(true)
		}

		if len(webSearchAdditionalQueries) > 0 {
			req.AdditionalQueries = webSearchAdditionalQueries
		}
		if webSearchSystemPrompt != "" {
			req.SystemPrompt = webSearchSystemPrompt
		}
		if !webSearchNoModeration {
			req.Moderation = boolPtr(true)
		}

		if webSearchText {
			req.Contents = &exa.ContentsOptions{
				Text: &exa.ContentsText{
					MaxCharacters: webSearchMaxChars,
				},
			}
		} else if webSearchHighlights {
			req.Contents = &exa.ContentsOptions{
				Highlights: &exa.HighlightOpts{},
			}
		} else if webSearchSummary != "" {
			req.Contents = &exa.ContentsOptions{
				Summary: &exa.SummaryOpts{
					Query: webSearchSummary,
				},
			}
		} else {
			req.Contents = &exa.ContentsOptions{
				Highlights: &exa.HighlightOpts{},
			}
		}

		resp, err := client.Search(ctx, req)
		if err != nil {
			return fmt.Errorf("searching: %w", err)
		}

		if webSearchJSON {
			data, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(data))
			printCost(resp.CostDollars)
			return nil
		}

		if resp.AutopromptString != "" {
			fmt.Printf("Autoprompt: %s\n\n", resp.AutopromptString)
		}

		for i, r := range resp.Results {
			fmt.Printf("%d. %s\n", i+1, r.Title)
			fmt.Printf("   %s\n", r.URL)
			if r.Author != "" {
				fmt.Printf("   Author: %s\n", r.Author)
			}
			if r.PublishedDate != nil {
				fmt.Printf("   Date: %s\n", r.PublishedDate.Format("2006-01-02"))
			}
			if r.Score != nil {
				fmt.Printf("   Score: %.4f\n", *r.Score)
			}

			if r.Text != "" {
				fmt.Printf("\n%s\n", r.Text)
			}
			if len(r.Highlights) > 0 {
				fmt.Println("   Highlights:")
				for _, h := range r.Highlights {
					fmt.Printf("     - %s\n", h)
				}
			}
			if r.Summary != "" {
				fmt.Printf("   Summary: %s\n", r.Summary)
			}

			if i < len(resp.Results)-1 {
				fmt.Println()
			}
		}

		printCost(resp.CostDollars)
		return nil
	},
}

func init() {
	webSearchCmd.Flags().IntVarP(&webSearchLimit, "limit", "n", 10, "Max results")
	webSearchCmd.Flags().StringVar(&webSearchType, "type", "auto", "Search type: auto, fast, instant, deep-lite, deep, deep-reasoning")
	webSearchCmd.Flags().BoolVar(&webSearchText, "text", false, "Return full text content")
	webSearchCmd.Flags().BoolVar(&webSearchHighlights, "highlights", false, "Return highlights only")
	webSearchCmd.Flags().StringVar(&webSearchSummary, "summary", "", "LLM summary targeting query")
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

	webSearchCmd.Flags().MarkHidden("summary")
}
