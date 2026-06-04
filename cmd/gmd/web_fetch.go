package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/web/exa"
)

var (
	webFetchFormat       string
	webFetchHighlights   bool
	webFetchSummary      string
	webFetchMaxChars     int
	webFetchOutput       string
	webFetchOutdir       string
	webFetchJSON         bool
	webFetchSubpageCount int
	webFetchSubpages     []string
	webFetchMaxAge       int
)

var webFetchCmd = &cobra.Command{
	Use:   "fetch <url> [url2 ...]",
	Short: "Fetch clean content from URLs via EXA",
	Long: `Fetch clean, readable content from one or more URLs using the EXA API.

The EXA API extracts clean markdown/text from web pages, stripping navigation,
ads, and boilerplate.

Examples:
  gmd web fetch https://example.com/article
  gmd web fetch https://a.com https://b.com --max-chars 2000
  gmd web fetch https://example.com --summary "key claims about"
  gmd web fetch https://example.com --output file -o ./fetched/
  gmd web fetch https://example.com --subpage-count 3
  gmd web fetch https://example.com --subpage-count 3 --subpage about --subpage careers --subpage press
  gmd web fetch https://example.com --max-age 48`,
	Args: cobra.MinimumNArgs(1),
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

		req := exa.ContentsRequest{
			URLs:          args,
			Subpages:      webFetchSubpageCount,
			SubpageTarget: webFetchSubpages,
			MaxAgeHours:   &webFetchMaxAge,
		}

		if webFetchHighlights {
			req.Highlights = &exa.HighlightOpts{}
		} else {
			req.Text = &exa.ContentsText{
				MaxCharacters: webFetchMaxChars,
			}
		}

		if webFetchSummary != "" {
			req.Summary = &exa.SummaryOpts{
				Query: webFetchSummary,
			}
		}

		resp, err := client.GetContents(ctx, req)
		if err != nil {
			return fmt.Errorf("fetching content: %w", err)
		}

		if webFetchJSON {
			data, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(data))
			printCost(resp.CostDollars)
			return nil
		}

		if len(resp.Results) == 0 {
			return fmt.Errorf("EXA returned no content for %d requested URL(s) — check accessibility, blocking, or try --max-age 0 for a live fetch", len(args))
		}

		switch webFetchOutput {
		case "file":
			if err := os.MkdirAll(webFetchOutdir, 0755); err != nil {
				return fmt.Errorf("creating output directory: %w", err)
			}
			for i, r := range resp.Results {
				filename := slugify(r.Title)
				if filename == "" {
					filename = fmt.Sprintf("page-%d", i+1)
				}
				ext := ".md"
				if webFetchFormat == "text" {
					ext = ".txt"
				}
				outPath := filepath.Join(webFetchOutdir, filename+ext)
				if err := os.WriteFile(outPath, []byte(r.Text), 0644); err != nil {
					return fmt.Errorf("writing %s: %w", outPath, err)
				}
				fmt.Printf("Wrote: %s\n", outPath)
				for j, sp := range r.Subpages {
					spFilename := filename + "--subpage-" + slugify(sp.Title)
					if spFilename == filename+"--subpage-" {
						spFilename = fmt.Sprintf("%s--subpage-%d", filename, j+1)
					}
					spPath := filepath.Join(webFetchOutdir, spFilename+ext)
					if err := os.WriteFile(spPath, []byte(sp.Text), 0644); err != nil {
						return fmt.Errorf("writing subpage %s: %w", spPath, err)
					}
					fmt.Printf("Wrote: %s\n", spPath)
				}
			}
		default:
			for i, r := range resp.Results {
				if len(resp.Results) > 1 {
					fmt.Printf("=== %s ===\n", r.Title)
				}
				if r.Text != "" {
					fmt.Println(r.Text)
				}
				if r.Summary != "" {
					fmt.Printf("\n--- Summary ---\n%s\n", r.Summary)
				}
				if len(r.Subpages) > 0 {
					fmt.Printf("\n--- Subpages (%d) ---\n", len(r.Subpages))
					for j, sp := range r.Subpages {
						fmt.Printf("\n  %d. %s\n", j+1, sp.URL)
						if sp.Title != "" {
							fmt.Printf("     %s\n", sp.Title)
						}
						if sp.Text != "" {
							for _, line := range strings.Split(sp.Text, "\n") {
								fmt.Printf("     %s\n", line)
							}
						}
					}
				}
				if i < len(resp.Results)-1 {
					fmt.Println()
				}
			}
		}

		printCost(resp.CostDollars)
		return nil
	},
}

func init() {
	webFetchCmd.Flags().StringVar(&webFetchFormat, "format", "markdown", "Output format: text or markdown")
	webFetchCmd.Flags().BoolVar(&webFetchHighlights, "highlights", false, "Return highlights only")
	webFetchCmd.Flags().StringVar(&webFetchSummary, "summary", "", "LLM-generated summary targeting query")
	webFetchCmd.Flags().IntVar(&webFetchMaxChars, "max-chars", 5000, "Max characters per page")
	webFetchCmd.Flags().StringVar(&webFetchOutput, "output", "stdout", "Write to stdout or file(s)")
	webFetchCmd.Flags().StringVarP(&webFetchOutdir, "outdir", "o", ".", "Output directory for --output file")
	webFetchCmd.Flags().BoolVar(&webFetchJSON, "json", false, "Output raw JSON from EXA API")
	webFetchCmd.Flags().IntVar(&webFetchSubpageCount, "subpage-count", 0, "Number of subpages to fetch per URL")
	webFetchCmd.Flags().StringSliceVar(&webFetchSubpages, "subpage", nil, "Keywords to prioritize when selecting subpages (repeatable)")
	webFetchCmd.Flags().IntVar(&webFetchMaxAge, "max-age", 0, "Max age in hours for cached content")
}
