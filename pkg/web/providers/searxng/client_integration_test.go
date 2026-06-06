//go:build integration

package searxng

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/web"
)

var testCfg *config.Config

func TestMain(m *testing.M) {
	config.LoadEnvFiles(config.FindProjectRoot("."), nil, nil)
	cfg, err := config.Load(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "searxng integration: config load failed (%v)\n", err)
	} else {
		testCfg = cfg
	}
	os.Exit(m.Run())
}

func requireSearXNGURL(t *testing.T) string {
	t.Helper()
	baseURL := os.Getenv("SEARXNG_BASE_URL")
	if baseURL == "" && testCfg != nil {
		baseURL = testCfg.Web.SearXNG.BaseURL
	}
	if baseURL == "" {
		if os.Getenv("GMD_WEB_INTEGRATION_FAIL") == "1" {
			t.Fatalf("SEARXNG_BASE_URL not set — set in env var, env file, or config.cue")
		}
		t.Skip("SEARXNG_BASE_URL not set — skipping integration test")
	}
	return baseURL
}

func TestSearchClient_Integration(t *testing.T) {
	baseURL := requireSearXNGURL(t)

	c, err := NewSearchClient(web.ProviderConfig{
		Name: "searxng",
		Extra: map[string]any{
			"base_url": baseURL,
		},
	})
	if err != nil {
		t.Fatalf("NewSearchClient: %v", err)
	}

	results, err := c.Search(context.Background(), web.SearchOptions{
		Query:      "open source search",
		NumResults: 3,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least 1 search result")
	}

	t.Logf("Got %d results, cost=%+v", len(results), results[0].Cost)
	for _, r := range results {
		if r.Title == "" {
			t.Error("result has empty title")
		}
		if r.URL == "" {
			t.Error("result has empty URL")
		}
		t.Logf("  %s — %s (engine=%v)", r.Title, r.URL, r.Extra["engine"])
	}
}
