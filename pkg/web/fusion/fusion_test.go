package fusion

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/verdverm/gmd/pkg/web"
)

type mockSearchProvider struct {
	name    string
	results []web.SearchResult
	err     error
}

func (m *mockSearchProvider) Name() string {
	return m.name
}

func (m *mockSearchProvider) Search(_ context.Context, _ web.SearchOptions) ([]web.SearchResult, error) {
	results := make([]web.SearchResult, len(m.results))
	copy(results, m.results)
	return results, m.err
}

func TestMultiSearch_ParallelFanOut(t *testing.T) {
	providers := []web.SearchProvider{
		&mockSearchProvider{
			name: "provider-a",
			results: []web.SearchResult{
				{Title: "Result A1", URL: "https://a.com/1", Score: 0.9},
				{Title: "Result A2", URL: "https://a.com/2", Score: 0.7},
			},
		},
		&mockSearchProvider{
			name: "provider-b",
			results: []web.SearchResult{
				{Title: "Result B1", URL: "https://b.com/1", Score: 0.8},
			},
		},
	}

	results, err := MultiSearch(t.Context(), "test query", providers, web.SearchOptions{Query: "test query"})
	if err != nil {
		t.Fatalf("MultiSearch failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	provNames := make(map[string]int)
	for _, r := range results {
		p, ok := r.Extra["_provider"].(string)
		if !ok {
			t.Errorf("result missing _provider: %s", r.Title)
			continue
		}
		provNames[p]++
	}

	if provNames["provider-a"] != 2 {
		t.Errorf("expected 2 results from provider-a, got %d", provNames["provider-a"])
	}
	if provNames["provider-b"] != 1 {
		t.Errorf("expected 1 result from provider-b, got %d", provNames["provider-b"])
	}
}

func TestMultiSearch_PartialFailure(t *testing.T) {
	providers := []web.SearchProvider{
		&mockSearchProvider{
			name:    "good",
			results: []web.SearchResult{{Title: "Good", URL: "https://good.com", Score: 0.9}},
		},
		&mockSearchProvider{
			name: "bad",
			err:  errors.New("provider down"),
		},
	}

	results, err := MultiSearch(t.Context(), "test", providers, web.SearchOptions{Query: "test"})
	if err != nil {
		t.Fatalf("MultiSearch should tolerate partial failure: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestMultiSearch_AllFail(t *testing.T) {
	providers := []web.SearchProvider{
		&mockSearchProvider{name: "bad1", err: errors.New("down")},
		&mockSearchProvider{name: "bad2", err: errors.New("also down")},
	}

	_, err := MultiSearch(t.Context(), "test", providers, web.SearchOptions{Query: "test"})
	if err == nil {
		t.Fatal("expected error when all providers fail")
	}
}

func TestMultiSearch_NoProviders(t *testing.T) {
	_, err := MultiSearch(t.Context(), "test", nil, web.SearchOptions{Query: "test"})
	if err == nil {
		t.Fatal("expected error with no providers")
	}
}

func TestDedupHeuristic_URLDedup(t *testing.T) {
	results := []web.SearchResult{
		{Title: "First", URL: "https://example.com/page", Score: 0.5, Content: "old"},
		{Title: "Second (better)", URL: "https://example.com/page", Score: 0.9, Content: "better content"},
		{Title: "Unique", URL: "https://other.com", Score: 0.7, Content: "unique content"},
	}

	deduped := dedupHeuristic(results)

	if len(deduped) != 2 {
		t.Fatalf("expected 2 results after dedup, got %d", len(deduped))
	}

	if deduped[0].Score != 0.9 {
		t.Errorf("expected higher-scored duplicate to be kept, got score %.2f", deduped[0].Score)
	}
	if deduped[0].Content != "better content" {
		t.Errorf("expected better content to be kept, got %q", deduped[0].Content)
	}
}

func TestDedupHeuristic_KeepFirstOnTie(t *testing.T) {
	results := []web.SearchResult{
		{Title: "A", URL: "https://example.com/page", Score: 0.5, Content: "first"},
		{Title: "B", URL: "https://example.com/page", Score: 0.5, Content: "second"},
	}

	deduped := dedupHeuristic(results)

	if len(deduped) != 1 {
		t.Fatalf("expected 1 result, got %d", len(deduped))
	}
	if deduped[0].Content != "first" {
		t.Errorf("expected first result kept on score tie, got %q", deduped[0].Content)
	}
}

func TestDedupHeuristic_DifferentURLs(t *testing.T) {
	results := []web.SearchResult{
		{Title: "A", URL: "https://a.com", Score: 0.9},
		{Title: "B", URL: "https://b.com", Score: 0.8},
		{Title: "C", URL: "https://c.com", Score: 0.7},
	}

	deduped := dedupHeuristic(results)

	if len(deduped) != 3 {
		t.Fatalf("expected 3 unique results, got %d", len(deduped))
	}
}

func TestDedupHeuristic_EmptyURL(t *testing.T) {
	results := []web.SearchResult{
		{Title: "Same Title", URL: "", Score: 0.5},
		{Title: "Same Title", URL: "", Score: 0.9},
		{Title: "Other", URL: "", Score: 0.7},
	}

	deduped := dedupHeuristic(results)

	if len(deduped) != 2 {
		t.Fatalf("expected 2 results (one dedup by title), got %d", len(deduped))
	}
}

func TestDedup_DedupNone(t *testing.T) {
	results := []web.SearchResult{
		{Title: "A", URL: "https://example.com", Score: 0.5},
		{Title: "B", URL: "https://example.com", Score: 0.9},
	}

	deduped, err := Dedup(t.Context(), results, Config{Dedup: "none"})
	if err != nil {
		t.Fatalf("dedup none: %v", err)
	}

	if len(deduped) != 2 {
		t.Fatalf("expected no dedup (2 results), got %d", len(deduped))
	}
}

func TestDedup_InvalidMethod(t *testing.T) {
	results := []web.SearchResult{
		{Title: "A", URL: "https://example.com", Score: 0.5},
		{Title: "B", URL: "https://example.com", Score: 0.9},
	}

	deduped, err := Dedup(t.Context(), results, Config{Dedup: "unknown"})
	if err != nil {
		t.Fatalf("dedup unknown: %v", err)
	}

	if len(deduped) != 1 {
		t.Fatalf("expected heuristic dedup as fallback, got %d", len(deduped))
	}
}

func TestParseKeepIndices(t *testing.T) {
	tests := []struct {
		input  string
		maxIdx int
		want   []int
	}{
		{"[0, 2, 5]", 10, []int{0, 2, 5}},
		{"[0,2,5]", 10, []int{0, 2, 5}},
		{"```json\n[0, 2]\n```", 10, []int{0, 2}},
		{"```\n[1,3]\n```", 10, []int{1, 3}},
		{"not an array", 10, nil},
		{"[0, 999]", 5, []int{0}},
		{"[]", 10, nil},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q", tt.input), func(t *testing.T) {
			got := parseKeepIndices(tt.input, tt.maxIdx)
			if len(got) != len(tt.want) {
				t.Errorf("expected %v, got %v", tt.want, got)
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("at index %d: expected %d, got %d", i, tt.want[i], v)
				}
			}
		})
	}
}

func TestTruncateStr(t *testing.T) {
	if got := truncateStr("hello", 3); got != "hel..." {
		t.Errorf("expected 'hel...', got %q", got)
	}
	if got := truncateStr("hi", 5); got != "hi" {
		t.Errorf("expected 'hi', got %q", got)
	}
}

func TestProviderName(t *testing.T) {
	m := &mockSearchProvider{name: "test"}
	got := providerName(m)
	if got == "" {
		t.Error("expected non-empty provider name")
	}
}
