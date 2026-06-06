package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/web"
	"github.com/verdverm/gmd/pkg/web/builders"
)

var (
	webProviderGroup   string
	webSearchProvider  string
	webBrowserProvider string
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

Providers: exa, tavily, searxng (search); exa, cloudflare, local (browser).`,
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

func printCost(cost *web.CostSummary) {
	if cost == nil {
		fmt.Fprintf(os.Stderr, "\nCost: unavailable\n")
		return
	}
	fmt.Fprintf(os.Stderr, "\nCost: $%.6f %s (%s)\n", cost.Cost, cost.Unit, cost.Provider)
}

func boolPtr(b bool) *bool {
	return &b
}

func getRegistry() *web.ProviderRegistry {
	return builders.DefaultRegistry()
}

func resolveSearchProvider(webCfg config.WebConfig) (web.SearchProvider, error) {
	r := getRegistry()
	name := webCfg.ResolveProvider("search", webSearchProvider)
	pc := makeProviderConfig(name, webCfg)
	got, err := r.Resolve("search", name, pc)
	if err != nil {
		return nil, err
	}
	sp, ok := got.(web.SearchProvider)
	if !ok {
		return nil, fmt.Errorf("provider %q does not implement SearchProvider", name)
	}
	return sp, nil
}

func resolveBrowserProvider(webCfg config.WebConfig) (web.BrowserProvider, error) {
	r := getRegistry()
	name := webCfg.ResolveProvider("browser", webBrowserProvider)
	pc := makeProviderConfig(name, webCfg)
	got, err := r.Resolve("browser", name, pc)
	if err != nil {
		return nil, err
	}
	bp, ok := got.(web.BrowserProvider)
	if !ok {
		return nil, fmt.Errorf("provider %q does not implement BrowserProvider", name)
	}
	return bp, nil
}

func makeProviderConfig(name string, webCfg config.WebConfig) web.ProviderConfig {
	pc := web.ProviderConfig{
		Name:  name,
		Extra: make(map[string]any),
	}
	switch name {
	case "exa":
		pc.Extra["api_key"] = webCfg.EXA.APIKey
	case "tavily":
		pc.Extra["api_key"] = webCfg.Tavily.APIKey
	case "searxng":
		pc.Extra["base_url"] = webCfg.SearXNG.BaseURL
	case "cloudflare":
		pc.Extra["api_key"] = webCfg.Cloudflare.APIKey
		pc.Extra["account_id"] = webCfg.Cloudflare.AccountID
	}
	return pc
}

func init() {
	webCmd.PersistentFlags().StringVar(&webProviderGroup, "provider-group", "", "Provider group preset (overrides configured group)")
	webCmd.PersistentFlags().StringVar(&webSearchProvider, "search-provider", "", "Search provider override (exa, tavily, searxng)")
	webCmd.PersistentFlags().StringVar(&webBrowserProvider, "browser-provider", "", "Browser provider override (exa, cloudflare, local)")

	webCmd.AddCommand(webFetchCmd)
	webCmd.AddCommand(webSearchCmd)
	webCmd.AddCommand(webCrawlCmd)
	webCmd.AddCommand(webAgentCmd)
	webCmd.AddCommand(webResearchCmd)
	rootCmd.AddCommand(webCmd)
}
