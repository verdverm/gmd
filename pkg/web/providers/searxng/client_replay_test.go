package searxng

import (
	"net/http"
	"testing"

	"github.com/verdverm/gmd/pkg/testutil"
	"github.com/verdverm/gmd/pkg/web"
)

func TestSearchClient_Replay(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/001_search.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	c, err := NewSearchClient(web.ProviderConfig{
		Name: "searxng",
		Extra: map[string]any{
			"base_url": "http://localhost:8080",
		},
		HTTPClient: &http.Client{Transport: tape.Transport()},
	})
	if err != nil {
		t.Fatal(err)
	}

	results, err := c.Search(t.Context(), web.SearchOptions{
		Query:      "open source search",
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

	_, err = c.Search(t.Context(), web.SearchOptions{Query: "another"})
	if err == nil {
		t.Fatal("expected error on exhausted tape")
	}
}
