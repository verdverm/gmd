package cloudflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/verdverm/gmd/pkg/web"
)

func cloudflareOkServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization header, got %s", r.Header.Get("Authorization"))
		}
		body := map[string]any{
			"success": true,
			"errors":  []any{},
			"result":  "rendered markdown content",
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
}

func TestBrowserClient_New(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		c, err := NewBrowserClient(web.ProviderConfig{
			Extra: map[string]any{
				"api_key":    "test-key",
				"account_id": "acct-123",
			},
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if c == nil {
			t.Fatal("expected non-nil client")
		}
	})

	t.Run("missing api_key", func(t *testing.T) {
		_, err := NewBrowserClient(web.ProviderConfig{
			Extra: map[string]any{"account_id": "acct-123"},
		})
		if err != web.ErrAuthMissing {
			t.Errorf("expected ErrAuthMissing, got %v", err)
		}
	})

	t.Run("missing account_id", func(t *testing.T) {
		_, err := NewBrowserClient(web.ProviderConfig{
			Extra: map[string]any{"api_key": "test-key"},
		})
		if err == nil {
			t.Fatal("expected error for missing account_id")
		}
	})
}

func TestBrowserClient_GetContent(t *testing.T) {
	ts := cloudflareOkServer(t)
	defer ts.Close()

	c := &BrowserClient{
		apiKey:     "test-key",
		accountID:  "acct-123",
		baseURL:    ts.URL,
		httpClient: ts.Client(),
	}

	t.Run("default markdown", func(t *testing.T) {
		result, err := c.GetContent(t.Context(), "https://example.com", nil)
		if err != nil {
			t.Fatalf("GetContent: %v", err)
		}
		expectedContent := `"rendered markdown content"`
		if result.Content != expectedContent {
			t.Errorf("unexpected content: %s", result.Content)
		}
		if result.Cost == nil || result.Cost.Provider != "cloudflare" {
			t.Error("expected cloudflare cost summary")
		}
	})

	t.Run("text format", func(t *testing.T) {
		result, err := c.GetContent(t.Context(), "https://example.com",
			&web.GetContentOptions{Format: "text"})
		if err != nil {
			t.Fatalf("GetContent: %v", err)
		}
		expectedContent := `"rendered markdown content"`
		if result.Content != expectedContent {
			t.Errorf("unexpected content: %s", result.Content)
		}
	})
}

func TestBrowserClient_GetContentErrors(t *testing.T) {
	t.Run("rate limited", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer ts.Close()

		c := &BrowserClient{apiKey: "k", accountID: "a", baseURL: ts.URL, httpClient: ts.Client()}
		_, err := c.GetContent(t.Context(), "https://example.com", nil)
		if err != web.ErrRateLimited {
			t.Errorf("expected ErrRateLimited, got %v", err)
		}
	})

	t.Run("auth failed", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer ts.Close()

		c := &BrowserClient{apiKey: "k", accountID: "a", baseURL: ts.URL, httpClient: ts.Client()}
		_, err := c.GetContent(t.Context(), "https://example.com", nil)
		if err != web.ErrAuthFailed {
			t.Errorf("expected ErrAuthFailed, got %v", err)
		}
	})

	t.Run("api error response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"errors":  []map[string]any{{"code": 1001, "message": "bad request"}},
			})
		}))
		defer ts.Close()

		c := &BrowserClient{apiKey: "k", accountID: "a", baseURL: ts.URL, httpClient: ts.Client()}
		_, err := c.GetContent(t.Context(), "https://example.com", nil)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "1001") {
			t.Errorf("expected error to contain error code, got %v", err)
		}
	})
}

func TestBrowserClient_Crawl(t *testing.T) {
	var callCount int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		body := map[string]any{
			"success": true,
			"errors":  []any{},
			"result":  "[Home](https://example.com/page1) [About](https://example.com/page2)",
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer ts.Close()

	c := &BrowserClient{apiKey: "test-key", accountID: "acct-123", baseURL: ts.URL, httpClient: ts.Client()}

	t.Run("basic crawl", func(t *testing.T) {
		callCount = 0
		pages, err := c.Crawl(t.Context(), "https://example.com/", &web.CrawlOptions{
			MaxDepth:   1,
			MaxPages:   3,
			SameDomain: true,
		})
		if err != nil {
			t.Fatalf("Crawl: %v", err)
		}
		if len(pages) == 0 {
			t.Fatal("expected at least 1 page")
		}
		if pages[0].URL != "https://example.com/" {
			t.Errorf("expected first page to be start URL, got %s", pages[0].URL)
		}
		if pages[0].Depth != 0 {
			t.Errorf("expected depth 0 for seed, got %d", pages[0].Depth)
		}
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := c.Crawl(ctx, "https://example.com/", nil)
		if err == nil {
			t.Error("expected context cancelled error")
		}
	})
}

func TestBrowserClient_ScrapeUnsupported(t *testing.T) {
	c := &BrowserClient{apiKey: "k", accountID: "a", baseURL: "http://localhost", httpClient: http.DefaultClient}
	elements, err := c.Scrape(t.Context(), "https://example.com", "div")
	if err != web.ErrNotSupported {
		t.Errorf("expected ErrNotSupported, got %v", err)
	}
	if elements != nil {
		t.Error("expected nil elements")
	}
}

func TestBrowserClient_Capabilities(t *testing.T) {
	c := &BrowserClient{apiKey: "k", accountID: "a", baseURL: "http://localhost", httpClient: http.DefaultClient}
	caps := c.Capabilities()
	if !caps.GetContent {
		t.Error("expected GetContent=true")
	}
	if !caps.Crawl {
		t.Error("expected Crawl=true")
	}
	if caps.Scrape {
		t.Error("expected Scrape=false")
	}
}

func TestCloudflare_NormalizeURL(t *testing.T) {
	tests := []struct {
		raw, expected string
	}{
		{"https://example.com/", "https://example.com"},
		{"https://example.com/page/", "https://example.com/page"},
		{"https://example.com?a=1", "https://example.com"},
		{"https://example.com#frag", "https://example.com"},
		{"https://example.com/path?a=1#frag", "https://example.com/path"},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got := normalizeURL(tt.raw)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestCloudflare_IsSameDomain(t *testing.T) {
	tests := []struct {
		a, b string
		same bool
	}{
		{"https://example.com/a", "https://example.com/b", true},
		{"https://example.com", "https://other.com", false},
		{"https://sub.example.com", "https://example.com", false},
	}
	for _, tt := range tests {
		got := isSameDomain(tt.a, tt.b)
		if got != tt.same {
			t.Errorf("isSameDomain(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.same)
		}
	}
}

func TestCloudflare_ExtractLinks(t *testing.T) {
	content := `[Home](https://example.com/) [About](/about) [Contact](https://example.com/contact) [External](https://other.com)`
	links := extractLinks(content, "https://example.com/")

	if len(links) != 4 {
		t.Fatalf("expected 4 links, got %d: %v", len(links), links)
	}
	expected := []string{
		"https://example.com/",
		"https://example.com/about",
		"https://example.com/contact",
		"https://other.com",
	}
	for i, expectedURL := range expected {
		if i >= len(links) || links[i] != expectedURL {
			t.Errorf("links[%d]: expected %q, got %q", i, expectedURL, links[i])
		}
	}
}

func TestCloudflare_ExtractLinksSkipsAnchors(t *testing.T) {
	content := `[top](#top) [page](/page)`
	links := extractLinks(content, "https://example.com/")
	if len(links) != 1 || links[0] != "https://example.com/page" {
		t.Errorf("expected 1 link skipping anchor, got %v", links)
	}
}

func TestBrowserClient_Interface(t *testing.T) {
	var _ web.BrowserProvider = (*BrowserClient)(nil)
}
