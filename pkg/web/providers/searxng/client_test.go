package searxng

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/verdverm/gmd/pkg/web"
)

func searxngOkServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("expected Accept header, got %s", r.Header.Get("Accept"))
		}

		q := r.URL.Query().Get("q")
		if q == "" {
			t.Error("expected query parameter")
		}

		body := map[string]any{
			"results": []map[string]any{
				{
					"url":           "https://example.com",
					"title":         "SearXNG Result",
					"content":       "content here",
					"engine":        "google",
					"engines":       []string{"google"},
					"publishedDate": "2026-01-15",
					"category":      "general",
					"score":         0.85,
				},
			},
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
}

func TestNewSearchClient(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		c, err := NewSearchClient(web.ProviderConfig{
			Extra: map[string]any{"base_url": "https://searx.example.com"},
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if c == nil {
			t.Fatal("expected non-nil client")
		}
	})

	t.Run("missing base_url", func(t *testing.T) {
		_, err := NewSearchClient(web.ProviderConfig{})
		if err == nil {
			t.Fatal("expected error for missing base_url")
		}
	})
}

func TestSearchClient_Search(t *testing.T) {
	ts := searxngOkServer(t)
	defer ts.Close()

	c, _ := NewSearchClient(web.ProviderConfig{
		Extra: map[string]any{"base_url": ts.URL},
	})

	results, err := c.Search(t.Context(), web.SearchOptions{
		Query:      "test query",
		NumResults: 5,
		Extra: map[string]any{
			"categories": "general,news",
			"engines":    "google,ddg",
			"language":   "en",
		},
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Title != "SearXNG Result" {
		t.Errorf("expected Title, got %s", r.Title)
	}
	if r.URL != "https://example.com" {
		t.Errorf("expected URL, got %s", r.URL)
	}
	if r.Cost == nil || r.Cost.Provider != "searxng" {
		t.Error("expected searxng cost summary")
	}
	if engine, _ := r.Extra["engine"].(string); engine != "google" {
		t.Errorf("expected engine=google, got %s", engine)
	}
}

func TestSearchClient_EmptyResults(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{}})
	}))
	defer ts.Close()

	c, _ := NewSearchClient(web.ProviderConfig{
		Extra: map[string]any{"base_url": ts.URL},
	})

	results, err := c.Search(t.Context(), web.SearchOptions{Query: "q"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchClient_ErrorPaths(t *testing.T) {
	t.Run("server error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		c, _ := NewSearchClient(web.ProviderConfig{
			Extra: map[string]any{"base_url": ts.URL},
		})
		_, err := c.Search(t.Context(), web.SearchOptions{Query: "q"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("bad JSON response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not json"))
		}))
		defer ts.Close()

		c, _ := NewSearchClient(web.ProviderConfig{
			Extra: map[string]any{"base_url": ts.URL},
		})
		_, err := c.Search(t.Context(), web.SearchOptions{Query: "q"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestSearchClient_Interface(t *testing.T) {
	var _ web.SearchProvider = (*SearchClient)(nil)
}
