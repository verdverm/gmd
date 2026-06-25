package exa

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/verdverm/gmd/pkg/web"
)

func TestBrowserAdapter_GetContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req exaclient.ContentsRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if len(req.URLs) != 1 || req.URLs[0] != "https://example.com/article" {
			t.Errorf("unexpected URLs: %v", req.URLs)
		}

		resp := exaclient.ContentsResponse{
			Results: []exaclient.SearchResult{
				{URL: "https://example.com/article", Text: "article content", Summary: "summary text"},
			},
			CostDollars: &exaclient.CostDollars{Total: 0.001},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	adapter, _ := NewBrowserAdapter(web.ProviderConfig{
		Extra: map[string]any{"api_key": "k", "base_url": ts.URL},
	})

	result, err := adapter.GetContent(t.Context(), "https://example.com/article", nil)
	if err != nil {
		t.Fatalf("GetContent: %v", err)
	}
	if result.Content != "article content" {
		t.Errorf("expected 'article content', got %q", result.Content)
	}
	if result.Cost == nil || result.Cost.Provider != "exa" {
		t.Error("expected exa cost summary")
	}
	if s, ok := result.Extra["summary"].(string); !ok || s != "summary text" {
		t.Errorf("expected summary in extra, got %v", result.Extra)
	}
}

func TestBrowserAdapter_GetContentWithOptions(t *testing.T) {
	var captured exaclient.ContentsRequest

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		_ = json.NewEncoder(w).Encode(exaclient.ContentsResponse{Results: []exaclient.SearchResult{{}}})
	}))
	defer ts.Close()

	adapter, _ := NewBrowserAdapter(web.ProviderConfig{
		Extra: map[string]any{"api_key": "k", "base_url": ts.URL},
	})

	opts := &web.GetContentOptions{
		MaxChars: 3000,
		MaxAge:   24 * time.Hour,
	}

	_, err := adapter.GetContent(t.Context(), "https://example.com", opts)
	if err != nil {
		t.Fatalf("GetContent: %v", err)
	}

	if captured.Text == nil {
		t.Fatal("expected Text in request")
	}
	if captured.Text.MaxCharacters != 3000 {
		t.Errorf("expected MaxCharacters=3000, got %d", captured.Text.MaxCharacters)
	}
	if captured.MaxAgeHours == nil || *captured.MaxAgeHours != 24 {
		t.Errorf("expected MaxAgeHours=24, got %v", captured.MaxAgeHours)
	}
}

func TestBrowserAdapter_GetContentEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(exaclient.ContentsResponse{})
	}))
	defer ts.Close()

	adapter, _ := NewBrowserAdapter(web.ProviderConfig{
		Extra: map[string]any{"api_key": "k", "base_url": ts.URL},
	})

	result, err := adapter.GetContent(t.Context(), "https://example.com", nil)
	if err != nil {
		t.Fatalf("GetContent: %v", err)
	}
	if result.Content != "" {
		t.Errorf("expected empty content, got %q", result.Content)
	}
}

func TestBrowserAdapter_CrawlUnsupported(t *testing.T) {
	adapter, _ := NewBrowserAdapter(web.ProviderConfig{
		Extra: map[string]any{"api_key": "k", "base_url": "http://localhost"},
	})

	pages, err := adapter.Crawl(t.Context(), "https://example.com", nil)
	if err != web.ErrNotSupported {
		t.Errorf("expected ErrNotSupported, got %v", err)
	}
	if pages != nil {
		t.Error("expected nil pages")
	}
}

func TestBrowserAdapter_ScrapeUnsupported(t *testing.T) {
	adapter, _ := NewBrowserAdapter(web.ProviderConfig{Extra: map[string]any{"api_key": "k"}})

	elements, err := adapter.Scrape(t.Context(), "https://example.com", "div")
	if err != web.ErrNotSupported {
		t.Errorf("expected ErrNotSupported, got %v", err)
	}
	if elements != nil {
		t.Error("expected nil elements")
	}
}

func TestBrowserAdapter_Capabilities(t *testing.T) {
	adapter, _ := NewBrowserAdapter(web.ProviderConfig{Extra: map[string]any{"api_key": "k"}})

	caps := adapter.Capabilities()
	if !caps.GetContent {
		t.Error("expected GetContent=true")
	}
	if caps.Crawl {
		t.Error("expected Crawl=false")
	}
	if caps.Scrape {
		t.Error("expected Scrape=false")
	}
	if caps.SelfHost {
		t.Error("expected SelfHost=false")
	}
}

func TestBrowserAdapter_ErrorPropagation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	adapter, _ := NewBrowserAdapter(web.ProviderConfig{
		Extra: map[string]any{"api_key": "k", "base_url": ts.URL},
	})

	_, err := adapter.GetContent(t.Context(), "https://example.com", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
