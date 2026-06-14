package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/web"
	"github.com/verdverm/gmd/pkg/web/persist"
)

var (
	webCrawlDepth      int
	webCrawlMaxPages   int
	webCrawlSameDomain bool
	webCrawlInclude    string
	webCrawlExclude    string
	webCrawlJSON       bool
)

var webCrawlCmd = &cobra.Command{
	Use:   "crawl <url>",
	Short: "Crawl from a seed URL, discovering and fetching linked pages",
	Long: `Crawl a site starting from a seed URL. Discovers links and retrieves
content recursively, bounded by depth and page count. Requires a browser
provider that supports Crawl (Cloudflare or Local).

Tier 1 — Deterministic. No LLM involved.

Examples:
  gmd web crawl https://example.com/docs
  gmd web crawl https://blog.example.com --depth 2 --max-pages 50
  gmd web crawl https://example.com --browser-provider cloudflare`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}

		bp, err := resolveBrowserProvider(cfg.Web)
		if err != nil {
			return err
		}

		caps := bp.Capabilities()
		if !caps.Crawl {
			return fmt.Errorf("crawl not supported by the configured browser provider")
		}

		ctx := context.Background()

		crawlOpts := &web.CrawlOptions{
			MaxDepth:       webCrawlDepth,
			MaxPages:       webCrawlMaxPages,
			SameDomain:     webCrawlSameDomain,
			IncludePattern: webCrawlInclude,
			ExcludePattern: webCrawlExclude,
		}

		pages, err := bp.Crawl(ctx, args[0], crawlOpts)
		if err != nil {
			return fmt.Errorf("crawling: %w", err)
		}

		if !webNoPersist && cfg.Web.Persistence != nil && cfg.Web.Persistence.Enabled {
			persistDir := resolvePersistDir(cmd, cfg)
			caller := webCaller
			if caller == "" {
				caller = "human"
			}
			meta := persist.Metadata{
				Caller: caller,
				Flags: map[string]any{
					"depth":      float64(webCrawlDepth),
					"maxPages":   float64(webCrawlMaxPages),
					"sameDomain": webCrawlSameDomain,
					"include":    webCrawlInclude,
					"exclude":    webCrawlExclude,
					"json":       webCrawlJSON,
				},
			}
			if err := persist.Crawl(persistDir, args[0], pages, meta); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: persist failed: %v\n", err)
			}
		}

		if webCrawlJSON {
			data, _ := json.MarshalIndent(pages, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(pages) == 0 {
			fmt.Println("No pages found.")
			return nil
		}

		fmt.Printf("Crawled %d pages:\n\n", len(pages))
		for i, p := range pages {
			fmt.Printf("%d. %s\n", i+1, p.URL)
			if p.Title != "" {
				fmt.Printf("   Title: %s\n", p.Title)
			}
			fmt.Printf("   Depth: %d\n", p.Depth)
			if p.Error != "" {
				fmt.Printf("   Error: %s\n", p.Error)
			}
			if p.Content != "" {
				truncated := p.Content
				if len(truncated) > 500 {
					truncated = truncated[:500] + "..."
				}
				fmt.Printf("   Content: %s\n", truncated)
			}
			fmt.Println()
		}

		return nil
	},
}

func init() {
	webCrawlCmd.Flags().IntVar(&webCrawlDepth, "depth", 2, "Maximum crawl depth")
	webCrawlCmd.Flags().IntVar(&webCrawlMaxPages, "max-pages", 20, "Maximum pages to fetch")
	webCrawlCmd.Flags().BoolVar(&webCrawlSameDomain, "same-domain", true, "Stay within the starting domain")
	webCrawlCmd.Flags().StringVar(&webCrawlInclude, "include", "", "URL pattern to include")
	webCrawlCmd.Flags().StringVar(&webCrawlExclude, "exclude", "", "URL pattern to exclude")
	webCrawlCmd.Flags().BoolVar(&webCrawlJSON, "json", false, "Output raw JSON")

	webCmd.AddCommand(webCrawlCmd)
}
