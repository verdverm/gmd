package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	webCrawlDepth      int
	webCrawlMaxPages   int
	webCrawlSameDomain bool
	webCrawlInclude    string
	webCrawlExclude    string
	webCrawlProvider   string
)

var webCrawlCmd = &cobra.Command{
	Use:   "crawl <url>",
	Short: "Crawl from a seed URL, discovering and fetching linked pages",
	Long: `Crawl a site starting from a seed URL. Discovers links and retrieves
content recursively, bounded by depth and page count. Requires a browser
provider that supports Crawl (Local or Cloudflare).

Tier 1 — Deterministic. No LLM involved.

Examples:
  gmd web crawl https://example.com/docs
  gmd web crawl https://blog.example.com --depth 2 --max-pages 50`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("not yet implemented: gmd web crawl (Phase 2)")
	},
}

func init() {
	webCrawlCmd.Flags().IntVar(&webCrawlDepth, "depth", 2, "Maximum crawl depth")
	webCrawlCmd.Flags().IntVar(&webCrawlMaxPages, "max-pages", 20, "Maximum pages to fetch")
	webCrawlCmd.Flags().BoolVar(&webCrawlSameDomain, "same-domain", true, "Stay within the starting domain")
	webCrawlCmd.Flags().StringVar(&webCrawlInclude, "include", "", "URL pattern to include")
	webCrawlCmd.Flags().StringVar(&webCrawlExclude, "exclude", "", "URL pattern to exclude")
	webCrawlCmd.Flags().StringVar(&webCrawlProvider, "browser-provider", "", "Browser provider override")
}
