//go:build integration

package exa

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
		fmt.Fprintf(os.Stderr, "exa integration: config load failed (%v)\n", err)
	}
	os.Exit(m.Run())
}

func maybeNewTape(t *testing.T, filePath string) *testutil.Tape {
	t.Helper()
	if os.Getenv("GMD_NORECORD") == "1" {
		return nil
	}
	return testutil.NewTape(filePath, "https://api.exa.ai", nil, testutil.ModeRecord)
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Fatalf("%s not set — integration test requires credentials", key)
	}
	return v
}

func TestIntegrationExa_Search(t *testing.T) {
	apiKey := requireEnv(t, "EXA_API_KEY")

	tape := maybeNewTape(t, "testdata/Exa_Search.json")
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

	adapter, err := NewSearchAdapter(web.ProviderConfig{
		Name: "exa",
		Extra: map[string]any{
			"api_key": apiKey,
		},
		HTTPClient: httpClient,
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
