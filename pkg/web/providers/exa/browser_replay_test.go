package exa

import (
	"net/http"
	"testing"

	"github.com/verdverm/gmd/pkg/testutil"
	"github.com/verdverm/gmd/pkg/web"
)

func TestBrowserAdapter_Replay(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/Exa_Browser.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	adapter, err := NewBrowserAdapter(web.ProviderConfig{
		Name: "exa",
		Extra: map[string]any{
			"api_key": "test-key",
		},
		HTTPClient: &http.Client{Transport: tape.Transport()},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := adapter.GetContent(t.Context(), "https://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Content == "" {
		t.Error("expected non-empty content from example.com")
	}
}
