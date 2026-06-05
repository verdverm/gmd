package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/web/exa"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Web search, fetch, crawl, agent, and research",
	Long: `Web commands for searching and retrieving web content via multiple providers.

Three-tier command spectrum (each builds on the prior):
  Tier 1 — Deterministic: search, fetch, crawl (no LLM, direct provider calls)
  Tier 2 — Agent:         agent (conversational, multi-step, LLM-orchestrated)
  Tier 3 — Research:      research (structured deep pipeline, formal reports)

Workflows:
  1. Search:   gmd web search "your query"
  2. Fetch:    gmd web fetch https://example.com
  3. Crawl:    gmd web crawl https://example.com/docs --depth 2
  4. Agent:    gmd web agent "your research question" --steps 5
  5. Research: gmd web research "comprehensive topic analysis" --depth deep

Currently backed by EXA. Multi-provider support (Cloudflare, Tavily, SearXNG,
Local) is in progress — see .design/web-providers.md.`,
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		} else if r == ' ' || r == '_' {
			result.WriteRune('-')
		}
	}
	out := result.String()
	if len(out) > 60 {
		out = out[:60]
	}
	return strings.Trim(out, "-")
}

func printCost(cost *exa.CostDollars) {
	if cost == nil {
		fmt.Fprintf(os.Stderr, "\nCost: unavailable\n")
		return
	}
	fmt.Fprintf(os.Stderr, "\nCost: $%.6f\n", cost.Total)
}

func boolPtr(b bool) *bool {
	return &b
}

func init() {
	webCmd.AddCommand(webFetchCmd)
	webCmd.AddCommand(webSearchCmd)
	webCmd.AddCommand(webCrawlCmd)
	webCmd.AddCommand(webAgentCmd)
	webCmd.AddCommand(webResearchCmd)
	rootCmd.AddCommand(webCmd)
}
