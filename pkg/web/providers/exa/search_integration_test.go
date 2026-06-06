//go:build integration

package exa

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
		fmt.Fprintf(os.Stderr, "exa integration: config load failed (%v)\n", err)
	}
	os.Exit(m.Run())
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		if os.Getenv("GMD_WEB_INTEGRATION_FAIL") == "1" {
			t.Fatalf("%s not set — integration test requires credentials", key)
		}
		t.Skipf("%s not set — skipping integration test", key)
	}
	return v
}

func TestSearchAdapter_Integration(t *testing.T) {
	apiKey := requireEnv(t, "EXA_API_KEY")

	adapter, err := NewSearchAdapter(web.ProviderConfig{
		Name: "exa",
		Extra: map[string]any{
			"api_key": apiKey,
		},
	})
	if err != nil {
		t.Fatalf("NewSearchAdapter: %v", err)
	}

	results, err := adapter.Search(context.Background(), web.SearchOptions{
		Query:      "golang error handling",
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
		t.Logf("  %s — %s", r.Title, r.URL)
	}
}
