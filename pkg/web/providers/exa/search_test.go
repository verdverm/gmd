package exa

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/verdverm/gmd/pkg/web"
	exaclient "github.com/verdverm/gmd/pkg/web/exa"
)

func TestSearchAdapter_Search(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var req exaclient.SearchRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Query == "" {
			t.Error("expected non-empty query")
		}

		resp := exaclient.SearchResponse{
			Results: []exaclient.SearchResult{
				{
					Title: "Test Result",
					URL:   "https://example.com",
					Text:  "Some content",
					Score: ptr(0.95),
				},
			},
			CostDollars: &exaclient.CostDollars{Total: 0.0015},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	adapter, err := NewSearchAdapter(web.ProviderConfig{
		Name: "exa",
		Extra: map[string]any{
			"api_key":  "test-key",
			"base_url": ts.URL,
		},
	})
	if err != nil {
		t.Fatalf("NewSearchAdapter: %v", err)
	}

	results, err := adapter.Search(t.Context(), web.SearchOptions{
		Query:          "test query",
		NumResults:     5,
		IncludeDomains: []string{"example.com"},
		Extra: map[string]any{
			"search_type": "auto",
		},
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Title != "Test Result" {
		t.Errorf("expected Title=Test Result, got %s", r.Title)
	}
	if r.URL != "https://example.com" {
		t.Errorf("expected URL=https://example.com, got %s", r.URL)
	}
	if r.Score != 0.95 {
		t.Errorf("expected Score=0.95, got %f", r.Score)
	}
	if r.Cost == nil || r.Cost.Provider != "exa" {
		t.Error("expected exa cost summary")
	}
}

func TestSearchAdapter_SearchEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(exaclient.SearchResponse{})
	}))
	defer ts.Close()

	adapter, _ := NewSearchAdapter(web.ProviderConfig{
		Extra: map[string]any{"api_key": "k", "base_url": ts.URL},
	})

	results, err := adapter.Search(t.Context(), web.SearchOptions{Query: "q"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchAdapter_ExtraMapping(t *testing.T) {
	var captured exaclient.SearchRequest

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		_ = json.NewEncoder(w).Encode(exaclient.SearchResponse{})
	}))
	defer ts.Close()

	adapter, _ := NewSearchAdapter(web.ProviderConfig{
		Extra: map[string]any{"api_key": "k", "base_url": ts.URL},
	})

	opts := web.SearchOptions{
		Query:          "test",
		NumResults:     10,
		IncludeDomains: []string{"in.example.com"},
		ExcludeDomains: []string{"ex.example.com"},
		Extra: map[string]any{
			"search_type":          "deep",
			"use_autoprompt":       true,
			"category":             "news",
			"start_published_date": "2026-01-01",
			"end_published_date":   "2026-06-01",
			"with_text":            true,
			"max_chars":            2000,
		},
	}

	_, err := adapter.Search(t.Context(), opts)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if captured.Type != "deep" {
		t.Errorf("expected Type=deep, got %s", captured.Type)
	}
	if captured.NumResults != 10 {
		t.Errorf("expected NumResults=10, got %d", captured.NumResults)
	}
	if len(captured.IncludeDomains) != 1 || captured.IncludeDomains[0] != "in.example.com" {
		t.Errorf("unexpected IncludeDomains: %v", captured.IncludeDomains)
	}
	if len(captured.ExcludeDomains) != 1 || captured.ExcludeDomains[0] != "ex.example.com" {
		t.Errorf("unexpected ExcludeDomains: %v", captured.ExcludeDomains)
	}
	if captured.Category != "news" {
		t.Errorf("expected Category=news, got %s", captured.Category)
	}
	if captured.UseAutoprompt == nil || !*captured.UseAutoprompt {
		t.Error("expected UseAutoprompt=true")
	}
	if captured.Contents == nil || captured.Contents.Text == nil || captured.Contents.Text.MaxCharacters != 2000 {
		t.Error("expected Contents with text at 2000 chars")
	}
	if captured.StartPublishedDate == nil {
		t.Error("expected StartPublishedDate")
	}
	if captured.EndPublishedDate == nil {
		t.Error("expected EndPublishedDate")
	}
}

func TestSearchAdapter_ErrorPropagation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	adapter, _ := NewSearchAdapter(web.ProviderConfig{
		Extra: map[string]any{"api_key": "k", "base_url": ts.URL},
	})

	_, err := adapter.Search(t.Context(), web.SearchOptions{Query: "q"})
	if err == nil {
		t.Fatal("expected error from server")
	}
}

func TestSearchAdapter_HighlightsMode(t *testing.T) {
	var captured exaclient.SearchRequest

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		_ = json.NewEncoder(w).Encode(exaclient.SearchResponse{})
	}))
	defer ts.Close()

	adapter, _ := NewSearchAdapter(web.ProviderConfig{
		Extra: map[string]any{"api_key": "k", "base_url": ts.URL},
	})

	_, _ = adapter.Search(t.Context(), web.SearchOptions{
		Query: "test",
		Extra: map[string]any{
			"with_highlights": true,
		},
	})

	if captured.Contents == nil || captured.Contents.Highlights == nil {
		t.Error("expected Highlights in Contents when with_highlights=true")
	}
}

func TestExa_CostSummary(t *testing.T) {
	t.Run("nil cost", func(t *testing.T) {
		if c := exaCostSummary(nil); c != nil {
			t.Error("expected nil for nil input")
		}
	})

	t.Run("non-nil cost", func(t *testing.T) {
		c := exaCostSummary(&exaclient.CostDollars{Total: 0.005})
		if c == nil {
			t.Fatal("expected non-nil")
		}
		if c.Provider != "exa" || c.Cost != 0.005 || c.Unit != "query" || c.Currency != "USD" {
			t.Errorf("unexpected cost summary: %+v", c)
		}
	})
}

func TestExa_ParseDateExtra(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		d, err := parseDateExtra("2026-01-15")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d.Year() != 2026 || d.Month() != time.January || d.Day() != 15 {
			t.Errorf("unexpected date: %v", d)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := parseDateExtra("not-a-date")
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestSearchAdapter_Interface(t *testing.T) {
	var _ web.SearchProvider = (*SearchAdapter)(nil)
}

func ptr(v float64) *float64 { return &v }
