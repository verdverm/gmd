package exa

import (
	"net/http"
	"testing"

	"github.com/verdverm/gmd/pkg/testutil"
	"github.com/verdverm/gmd/pkg/web"
)

func TestSearchAdapter_Replay(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/Exa_Search.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	adapter, err := NewSearchAdapter(web.ProviderConfig{
		Name: "exa",
		Extra: map[string]any{
			"api_key": "test-key",
		},
		HTTPClient: &http.Client{Transport: tape.Transport()},
	})
	if err != nil {
		t.Fatal(err)
	}

	results, err := adapter.Search(t.Context(), web.SearchOptions{
		Query:      "golang error handling",
		NumResults: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected non-empty results")
	}
	for _, r := range results {
		if r.Title == "" {
			t.Error("result has empty title")
		}
		if r.URL == "" {
			t.Error("result has empty URL")
		}
	}
}
