package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/web"
)

var (
	webFetchFormat     string
	webFetchHighlights bool
	webFetchSummary    string
	webFetchMaxChars   int
	webFetchOutput     string
	webFetchOutdir     string
	webFetchJSON       bool
	webFetchMaxAge     int
)

var webFetchCmd = &cobra.Command{
	Use:   "fetch <url> [url2 ...]",
	Short: "Fetch clean content from URLs",
	Long: `Fetch clean, readable content from one or more URLs.

Examples:
  gmd web fetch https://example.com/article
  gmd web fetch https://a.com https://b.com --max-chars 2000
  gmd web fetch https://example.com --output file -o ./fetched/
  gmd web fetch https://example.com --max-age 48`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}

		bp, err := resolveBrowserProvider(cfg.Web)
		if err != nil {
			return err
		}

		ctx := context.Background()

		for _, urlStr := range args {
			contentOpts := &web.GetContentOptions{
				Format:   webFetchFormat,
				MaxChars: webFetchMaxChars,
			}
			if webFetchMaxAge > 0 {
				contentOpts.MaxAge = 0
				contentOpts.Extra = map[string]any{"max_age_hours": webFetchMaxAge}
			}

			result, err := bp.GetContent(ctx, urlStr, contentOpts)
			if err != nil {
				return fmt.Errorf("fetching %s: %w", urlStr, err)
			}

			if webFetchJSON {
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				if result.Cost != nil {
					printCost(result.Cost)
				}
				continue
			}

			if result.Content == "" {
				fmt.Fprintf(os.Stderr, "Warning: no content returned for %s\n", urlStr)
				continue
			}

			title := urlStr
			if t, ok := result.Extra["title"].(string); ok && t != "" {
				title = t
			}

			switch webFetchOutput {
			case "file":
				if err := os.MkdirAll(webFetchOutdir, 0755); err != nil {
					return fmt.Errorf("creating output directory: %w", err)
				}
				filename := slugify(title)
				if filename == "" {
					filename = "page"
				}
				ext := ".md"
				if webFetchFormat == "text" {
					ext = ".txt"
				}
				outPath := filepath.Join(webFetchOutdir, filename+ext)
				if err := os.WriteFile(outPath, []byte(result.Content), 0644); err != nil { //nolint:gosec // downloaded public content
					return fmt.Errorf("writing %s: %w", outPath, err)
				}
				fmt.Printf("Wrote: %s\n", outPath)
			default:
				if len(args) > 1 {
					fmt.Printf("=== %s ===\n", title)
				}
				fmt.Println(result.Content)
			}

			if result.Cost != nil {
				printCost(result.Cost)
			}
		}

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
	webFetchCmd.Flags().BoolVar(&webFetchJSON, "json", false, "Output raw JSON")
	webFetchCmd.Flags().IntVar(&webFetchMaxAge, "max-age", 0, "Max age in hours for cached content")
}
