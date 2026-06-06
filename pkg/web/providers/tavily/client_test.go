package tavily

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/verdverm/gmd/pkg/web"
)

func tavilyOkServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		body := map[string]any{
			"results": []map[string]any{
				{
					"url":     "https://example.com",
					"title":   "Test Page",
					"content": "content text",
					"score":   0.91,
				},
			},
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
}

func TestNewSearchClient(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		c, err := NewSearchClient(web.ProviderConfig{
			Extra: map[string]any{"api_key": "test-key"},
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if c == nil {
			t.Fatal("expected non-nil client")
		}
	})

	t.Run("missing api_key", func(t *testing.T) {
		_, err := NewSearchClient(web.ProviderConfig{})
		if err != web.ErrAuthMissing {
			t.Errorf("expected ErrAuthMissing, got %v", err)
		}
	})

	t.Run("custom base_url", func(t *testing.T) {
		c, err := NewSearchClient(web.ProviderConfig{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": "https://custom.example.com",
			},
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if c.baseURL != "https://custom.example.com" {
			t.Errorf("expected base_url to be set, got %s", c.baseURL)
		}
	})
}

func TestSearchClient_Search(t *testing.T) {
	ts := tavilyOkServer(t)
	defer ts.Close()

	c, _ := NewSearchClient(web.ProviderConfig{
		Extra: map[string]any{"api_key": "test-key", "base_url": ts.URL},
	})

	results, err := c.Search(t.Context(), web.SearchOptions{
		Query:      "test query",
		NumResults: 5,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Title != "Test Page" {
		t.Errorf("expected Title=Test Page, got %s", r.Title)
	}
	if r.URL != "https://example.com" {
		t.Errorf("expected URL, got %s", r.URL)
	}
	if r.Score != 0.91 {
		t.Errorf("expected Score=0.91, got %f", r.Score)
	}
	if r.Cost == nil || r.Cost.Provider != "tavily" {
		t.Error("expected tavily cost summary")
	}
}

func TestSearchClient_ExtraOptions(t *testing.T) {
	var captured map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{}})
	}))
	defer ts.Close()

	c, _ := NewSearchClient(web.ProviderConfig{
		Extra: map[string]any{"api_key": "k", "base_url": ts.URL},
	})

	opts := web.SearchOptions{
		Query:          "test",
		NumResults:     7,
		IncludeDomains: []string{"in.example.com"},
		ExcludeDomains: []string{"ex.example.com"},
		Extra: map[string]any{
			"search_depth":        "advanced",
			"include_answer":      true,
			"include_raw_content": true,
		},
	}

	_, err := c.Search(t.Context(), opts)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if v, _ := captured["search_depth"].(string); v != "advanced" {
		t.Errorf("expected search_depth=advanced, got %s", v)
	}
	if v, _ := captured["include_answer"].(bool); !v {
		t.Error("expected include_answer=true")
	}
	if v, _ := captured["include_raw_content"].(bool); !v {
		t.Error("expected include_raw_content=true")
	}
	if v, _ := captured["max_results"].(float64); v != 7 {
		t.Errorf("expected max_results=7, got %v", v)
	}
}

func TestSearchClient_ErrorPaths(t *testing.T) {
	t.Run("rate limited", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer ts.Close()

		c, _ := NewSearchClient(web.ProviderConfig{
			Extra: map[string]any{"api_key": "k", "base_url": ts.URL},
		})
		_, err := c.Search(t.Context(), web.SearchOptions{Query: "q"})
		if !errorsIs(err, web.ErrRateLimited) {
			t.Errorf("expected ErrRateLimited, got %v", err)
		}
	})

	t.Run("auth failed", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer ts.Close()

		c, _ := NewSearchClient(web.ProviderConfig{
			Extra: map[string]any{"api_key": "k", "base_url": ts.URL},
		})
		_, err := c.Search(t.Context(), web.SearchOptions{Query: "q"})
		if !errorsIs(err, web.ErrAuthFailed) {
			t.Errorf("expected ErrAuthFailed, got %v", err)
		}
	})

	t.Run("bad JSON response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not json"))
		}))
		defer ts.Close()

		c, _ := NewSearchClient(web.ProviderConfig{
			Extra: map[string]any{"api_key": "k", "base_url": ts.URL},
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

func errorsIs(err error, target error) bool {
	for {
		if err == target {
			return true
		}
		if u, ok := err.(interface{ Unwrap() error }); ok {
			err = u.Unwrap()
		} else {
			return false
		}
	}
}
