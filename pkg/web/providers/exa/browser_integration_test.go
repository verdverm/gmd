//go:build integration

package exa

import (
	"context"
	"net/http"
	"testing"

	"github.com/verdverm/gmd/pkg/web"
)

func TestBrowserAdapter_Integration(t *testing.T) {
	apiKey := requireEnv(t, "EXA_API_KEY")

	tape := maybeNewTape(t, "testdata/002_browser.json")
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

	adapter, err := NewBrowserAdapter(web.ProviderConfig{
		Name: "exa",
		Extra: map[string]any{
			"api_key": apiKey,
		},
		HTTPClient: httpClient,
	})
	if err != nil {
		t.Fatalf("NewBrowserAdapter: %v", err)
	}

	result, err := adapter.GetContent(context.Background(), "https://example.com", nil)
	if err != nil {
		t.Fatalf("GetContent: %v", err)
	}

	if result.Content == "" {
		t.Error("expected non-empty content from example.com")
	}

	t.Logf("Content length: %d, cost=%+v", len(result.Content), result.Cost)
}
