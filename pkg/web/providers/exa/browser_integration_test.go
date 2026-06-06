//go:build integration

package exa

import (
	"context"
	"testing"

	"github.com/verdverm/gmd/pkg/web"
)

func TestBrowserAdapter_Integration(t *testing.T) {
	apiKey := requireEnv(t, "EXA_API_KEY")

	adapter, err := NewBrowserAdapter(web.ProviderConfig{
		Name: "exa",
		Extra: map[string]any{
			"api_key": apiKey,
		},
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
