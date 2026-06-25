package fusion

import (
	"context"
	"net/http"
	"testing"

	"github.com/verdverm/gmd/pkg/testutil"
	"github.com/verdverm/gmd/pkg/web"
	exaprovider "github.com/verdverm/gmd/pkg/web/providers/exa"
	searxngprovider "github.com/verdverm/gmd/pkg/web/providers/searxng"
	tavilyprovider "github.com/verdverm/gmd/pkg/web/providers/tavily"
)

func TestMultiSearch_Replay(t *testing.T) {
	exaTape, exaErr := testutil.NewReplayTape("testdata/Fusion_Exa.json")
	tavilyTape, tavErr := testutil.NewReplayTape("testdata/Fusion_Tavily.json")
	searxngTape, searxngErr := testutil.NewReplayTape("testdata/Fusion_Searxng.json")

	var providers []web.SearchProvider
	if exaErr == nil {
		exaTape.Start()
		defer func() { exaTape.Stop() }()
		adapter, err := exaprovider.NewSearchAdapter(web.ProviderConfig{
			Name: "exa", Extra: map[string]any{"api_key": "test-key"},
			HTTPClient: &http.Client{Transport: exaTape.Transport()},
		})
		if err == nil {
			providers = append(providers, adapter)
		}
	}
	if tavErr == nil {
		tavilyTape.Start()
		defer func() { tavilyTape.Stop() }()
		client, err := tavilyprovider.NewSearchClient(web.ProviderConfig{
			Name: "tavily", Extra: map[string]any{"api_key": "test-key"},
			HTTPClient: &http.Client{Transport: tavilyTape.Transport()},
		})
		if err == nil {
			providers = append(providers, client)
		}
	}
	if searxngErr == nil {
		searxngTape.Start()
		defer func() { searxngTape.Stop() }()
		client, err := searxngprovider.NewSearchClient(web.ProviderConfig{
			Name: "searxng", Extra: map[string]any{"base_url": "http://localhost:8080"},
			HTTPClient: &http.Client{Transport: searxngTape.Transport()},
		})
		if err == nil {
			providers = append(providers, client)
		}
	}

	if len(providers) == 0 {
		t.Fatal("no tape files available — run integration tests to generate tapes")
	}

	results, _, _, err := MultiSearch(context.Background(), "golang standard library features",
		providers, web.SearchOptions{Query: "golang standard library features", NumResults: 3})
	if err != nil {
		t.Fatalf("MultiSearch: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}

	providersSeen := map[string]int{}
	for _, r := range results {
		if r.Title == "" {
			t.Error("result has empty title")
		}
		if r.URL == "" {
			t.Error("result has empty URL")
		}
		if p, ok := r.Extra["_provider"].(string); ok {
			providersSeen[p]++
		} else {
			t.Error("result missing _provider tag")
		}
	}
	t.Logf("provider distribution: %v", providersSeen)
}
