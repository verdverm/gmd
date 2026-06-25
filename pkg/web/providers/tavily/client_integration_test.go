//go:build integration

package tavily

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/testutil"
	"github.com/verdverm/gmd/pkg/web"
)

func TestMain(m *testing.M) {
	config.LoadEnvFiles(config.FindProjectRoot("."), nil, nil)
	if _, err := config.Load("."); err != nil {
		fmt.Fprintf(os.Stderr, "tavily integration: config load failed (%v)\n", err)
	}
	os.Exit(m.Run())
}

func maybeNewTape(t *testing.T, filePath string) *testutil.Tape {
	t.Helper()
	if os.Getenv("GMD_NORECORD") == "1" {
		return nil
	}
	return testutil.NewTape(filePath, "https://api.tavily.com", nil, testutil.ModeRecord)
}

func requireTavilyKey(t *testing.T) string {
	t.Helper()
	apiKey := os.Getenv("TAVILY_API_KEY")
	if apiKey == "" {
		t.Fatalf("TAVILY_API_KEY not set — integration test requires credentials")
	}
	return apiKey
}

func TestIntegrationTavily_Search(t *testing.T) {
	apiKey := requireTavilyKey(t)

	tape := maybeNewTape(t, "testdata/Tavily_Search.json")
	var httpClient *http.Client
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
		httpClient = &http.Client{Transport: tape.Transport()}
	}

	c, err := NewSearchClient(web.ProviderConfig{
		Name: "tavily",
		Extra: map[string]any{
			"api_key": apiKey,
		},
		HTTPClient: httpClient,
	})
	if err != nil {
		t.Fatalf("NewSearchClient: %v", err)
	}

	results, err := c.Search(context.Background(), web.SearchOptions{
		Query:      "golang concurrency patterns",
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
		t.Logf("  %s — %s (score=%.2f)", r.Title, r.URL, r.Score)
	}
}
