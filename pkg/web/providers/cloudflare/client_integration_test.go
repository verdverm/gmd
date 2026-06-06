//go:build integration

package cloudflare

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/web"
)

func TestMain(m *testing.M) {
	config.LoadEnvFiles(config.FindProjectRoot("."), nil, nil)
	if _, err := config.Load("."); err != nil {
		fmt.Fprintf(os.Stderr, "cloudflare integration: config load failed (%v)\n", err)
	}
	os.Exit(m.Run())
}

func requireEnvCF(t *testing.T) (string, string) {
	t.Helper()
	apiKey := os.Getenv("CLOUDFLARE_API_KEY")
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	if apiKey == "" || accountID == "" {
		if os.Getenv("GMD_WEB_INTEGRATION_FAIL") == "1" {
			t.Fatalf("CLOUDFLARE_API_KEY and CLOUDFLARE_ACCOUNT_ID required for integration test")
		}
		t.Skip("CLOUDFLARE_API_KEY and CLOUDFLARE_ACCOUNT_ID not set — skipping integration test")
	}
	return apiKey, accountID
}

func TestBrowserClient_Integration_GetContent(t *testing.T) {
	apiKey, accountID := requireEnvCF(t)

	c, err := NewBrowserClient(web.ProviderConfig{
		Name: "cloudflare",
		Extra: map[string]any{
			"api_key":    apiKey,
			"account_id": accountID,
		},
	})
	if err != nil {
		t.Fatalf("NewBrowserClient: %v", err)
	}

	result, err := c.GetContent(context.Background(), "https://example.com", nil)
	if err != nil {
		t.Fatalf("GetContent: %v", err)
	}

	if result.Content == "" {
		t.Error("expected non-empty content from example.com")
	}
	t.Logf("Content length: %d, cost=%+v", len(result.Content), result.Cost)
}

func TestBrowserClient_Integration_Crawl(t *testing.T) {
	apiKey, accountID := requireEnvCF(t)

	c, err := NewBrowserClient(web.ProviderConfig{
		Name: "cloudflare",
		Extra: map[string]any{
			"api_key":    apiKey,
			"account_id": accountID,
		},
	})
	if err != nil {
		t.Fatalf("NewBrowserClient: %v", err)
	}

	pages, err := c.Crawl(context.Background(), "https://example.com", &web.CrawlOptions{
		MaxDepth:   1,
		MaxPages:   3,
		SameDomain: true,
	})
	if err != nil {
		t.Fatalf("Crawl: %v", err)
	}

	if len(pages) == 0 {
		t.Fatal("expected at least 1 page from crawl")
	}
	if pages[0].Error != "" {
		t.Errorf("first page has error: %s", pages[0].Error)
	}
	t.Logf("Crawled %d pages", len(pages))
	for i, p := range pages {
		t.Logf("  [%d] depth=%d url=%s links=%d error=%s", i, p.Depth, p.URL, len(p.Links), p.Error)
	}
}

func TestBrowserClient_Integration_Capabilities(t *testing.T) {
	c, _ := NewBrowserClient(web.ProviderConfig{
		Extra: map[string]any{
			"api_key":    "test",
			"account_id": "test",
		},
	})

	caps := c.Capabilities()
	if !caps.GetContent {
		t.Error("expected GetContent=true")
	}
	if !caps.Crawl {
		t.Error("expected Crawl=true")
	}
	if caps.Scrape {
		t.Error("expected Scrape=false")
	}
}
