package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/web"
	"github.com/verdverm/gmd/pkg/web/builders"
)

var (
	webProviderGroup   string
	webSearchProvider  string
	webBrowserProvider string
	webNoPersist       bool
	webPersistDir      string
	webCaller          string
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

func printCost(cost *web.CostSummary) {
	if cost == nil {
		fmt.Fprintf(os.Stderr, "\nCost: unavailable\n")
		return
	}
	fmt.Fprintf(os.Stderr, "\nCost: $%.6f %s (%s)\n", cost.Cost, cost.Unit, cost.Provider)
}

func getRegistry() *web.ProviderRegistry {
	return builders.DefaultRegistry()
}

func resolveSearchProviders(webCfg config.WebConfig) ([]web.SearchProvider, error) {
	r := getRegistry()
	if webProviderGroup != "" {
		webCfg.Group = webProviderGroup
	}
	names := webCfg.ResolveSearchProviders(webSearchProvider)
	if len(names) == 0 {
		return nil, fmt.Errorf("no search providers configured")
	}
	seen := make(map[string]bool)
	providers := make([]web.SearchProvider, 0, len(names))
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		pc := makeProviderConfig(name, webCfg)
		got, err := r.Resolve("search", name, pc)
		if err != nil {
			return nil, err
		}
		sp, ok := got.(web.SearchProvider)
		if !ok {
			return nil, fmt.Errorf("provider %q does not implement SearchProvider", name)
		}
		providers = append(providers, sp)
	}
	return providers, nil
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

// resolvePersistDir returns the absolute persistence directory.
func resolvePersistDir(cmd *cobra.Command, cfg *config.Config) string {
	if cfg.Web.Persistence == nil {
		cfg.Web.Persistence = &config.WebPersistenceConfig{Enabled: true, Dir: ".gmd/web"}
	}
	dir := cfg.Web.Persistence.Dir
	if cmd.Flags().Changed("persist-dir") {
		dir = webPersistDir
	}
	if !filepath.IsAbs(dir) {
		if cfg.ProjectRoot != "" {
			dir = filepath.Join(cfg.ProjectRoot, dir)
		} else {
			cacheDir, err := os.UserCacheDir()
			if err != nil {
				cacheDir = filepath.Join(os.TempDir(), "gmd")
			}
			dir = filepath.Join(cacheDir, "gmd", "web")
		}
	}
	return dir
}

func init() {
	webCmd.PersistentFlags().StringVar(&webProviderGroup, "provider-group", "", "Provider group preset (overrides configured group)")
	webCmd.PersistentFlags().StringVar(&webSearchProvider, "search-provider", "", "Search provider override, comma-separated (exa, tavily, searxng)")
	webCmd.PersistentFlags().StringVar(&webBrowserProvider, "browser-provider", "", "Browser provider override (exa, cloudflare, local)")

	webCmd.PersistentFlags().BoolVar(&webNoPersist, "no-persist", false, "Skip persisting results to disk")
	webCmd.PersistentFlags().StringVar(&webPersistDir, "persist-dir", "", "Override persistence directory")
	webCmd.PersistentFlags().StringVar(&webCaller, "caller", "human", "Caller identifier for attribution")

	rootCmd.AddCommand(webCmd)
}
