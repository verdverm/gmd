package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/exa"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Web search and content retrieval via EXA",
	Long: `Web commands for searching and fetching web content using the EXA neural search API.

EXA indexes and embeds the entire web, enabling semantic search (not just
keyword matching) and clean content extraction from any URL.

Workflows:
  1. Search:   gmd web search "your query"
  2. Fetch:    gmd web fetch https://example.com
  3. Agent:    gmd web agent "your research question" --steps 5

Requires EXA_API_KEY environment variable to be set.`,
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
	webCmd.AddCommand(webAgentCmd)
	rootCmd.AddCommand(webCmd)
}
