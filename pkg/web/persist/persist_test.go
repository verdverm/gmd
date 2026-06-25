package persist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/verdverm/gmd/pkg/web"
	"github.com/verdverm/gmd/pkg/web/fusion"
)

func TestPersist_Sluggify(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "Hello World", "hello-world"},
		{"underscores", "hello_world", "hello-world"},
		{"special chars", "Hello! @World#", "hello-world"},
		{"numbers", "test 123", "test-123"},
		{"hyphens preserved", "already-hyphenated", "already-hyphenated"},
		{"trim hyphens", "-leading-trailing-", "leading-trailing"},
		{"truncation", strings.Repeat("a", 200), strings.Repeat("a", 100)},
		{"empty", "", ""},
		{"only special", "!!!", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := web.Sluggify(tt.input)
			if got != tt.expected {
				t.Errorf("Sluggify(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestPersist_URLSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"simple url", "https://example.com/article", "example-com-article"},
		{"host only", "https://example.com", "example-com"},
		{"long path", "https://example.com/docs/getting-started/intro", "example-com-intro"},
		{"no scheme", "example.com/page", "example-com-page"},
		{"trailing slash", "https://example.com/page/", "example-com-page"},
		{"invalid url", "not a url at all", "not-a-url-at-all"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := urlSlug(tt.input)
			if !strings.Contains(got, tt.contains) {
				t.Errorf("urlSlug(%q) = %q, expected to contain %q", tt.input, got, tt.contains)
			}
		})
	}
}

func TestPersist_Timestamp(t *testing.T) {
	ts := timestamp()
	if ts == "" {
		t.Error("timestamp() returned empty string")
	}
	if strings.Contains(ts, ":") {
		t.Error("timestamp should not contain colons")
	}
	if strings.Contains(ts, "+") {
		t.Error("timestamp should not contain + signs")
	}
}

func TestPersist_TimestampDir(t *testing.T) {
	ts := "2026-06-10T15_30_00_123456789Z"
	slug := "test-slug"
	got := timestampDir(ts, slug)
	if !strings.Contains(got, "test-slug") {
		t.Errorf("timestampDir should contain slug: %s", got)
	}
}

func TestPersist_Fetch(t *testing.T) {
	dir := t.TempDir()
	result := &web.GetContentResult{
		Content: "# Hello\n\nThis is test content.",
		Extra: map[string]any{
			"title": "Test Page",
		},
	}
	meta := Metadata{
		Caller: "human",
	}

	err := Fetch(dir, "https://example.com/article", result, meta)
	if err != nil {
		t.Fatalf("PersistFetchResult failed: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no directories created")
	}

	fetchDir := filepath.Join(dir, "fetch")
	subs, err := os.ReadDir(fetchDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) == 0 {
		t.Fatal("no result directory created")
	}

	resultDir := filepath.Join(fetchDir, subs[0].Name())

	if _, err := os.Stat(filepath.Join(resultDir, "result.json")); os.IsNotExist(err) {
		t.Error("result.json not found")
	}
	if _, err := os.Stat(filepath.Join(resultDir, "metadata.json")); os.IsNotExist(err) {
		t.Error("metadata.json not found")
	}
	if _, err := os.Stat(filepath.Join(resultDir, "content.md")); os.IsNotExist(err) {
		t.Error("content.md not found")
	}
}

func TestPersist_FetchResultNil(t *testing.T) {
	dir := t.TempDir()
	meta := Metadata{Caller: "human"}

	err := Fetch(dir, "https://example.com/error", nil, meta)
	if err != nil {
		t.Fatalf("PersistFetchResult with nil result failed: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no directories created for nil result")
	}

	fetchDir := filepath.Join(dir, "fetch")
	subs, _ := os.ReadDir(fetchDir)
	resultDir := filepath.Join(fetchDir, subs[0].Name())

	if _, err := os.Stat(filepath.Join(resultDir, "metadata.json")); os.IsNotExist(err) {
		t.Error("metadata.json should exist even for nil results")
	}
	if _, err := os.Stat(filepath.Join(resultDir, "result.json")); !os.IsNotExist(err) {
		t.Error("result.json should not exist for nil result")
	}
}

func TestPersist_Crawl(t *testing.T) {
	dir := t.TempDir()
	pages := []web.Page{
		{URL: "https://example.com", Title: "Home", Content: "Home page content", Depth: 0},
		{URL: "https://example.com/about", Title: "About", Content: "About page content", Depth: 1},
	}
	meta := Metadata{Caller: "human"}

	err := Crawl(dir, "https://example.com", pages, meta)
	if err != nil {
		t.Fatalf("PersistCrawlResult failed: %v", err)
	}

	crawlDir := filepath.Join(dir, "crawl")
	subs, _ := os.ReadDir(crawlDir)
	resultDir := filepath.Join(crawlDir, subs[0].Name())

	if _, err := os.Stat(filepath.Join(resultDir, "result.json")); os.IsNotExist(err) {
		t.Error("result.json not found")
	}

	pagesDir := filepath.Join(resultDir, "pages")
	pageFiles, _ := os.ReadDir(pagesDir)
	if len(pageFiles) != 2 {
		t.Errorf("expected 2 page files, got %d", len(pageFiles))
	}
}

func TestPersist_Search(t *testing.T) {
	dir := t.TempDir()
	result := &fusion.Result{
		Answer: "Synthesized answer",
		Results: []web.SearchResult{
			{Title: "Result 1", URL: "https://a.com/1", Content: "Content 1", Score: 0.95},
			{Title: "Result 2", URL: "https://b.com/2", Content: "Content 2", Score: 0.85},
		},
	}
	rawResults := map[string][]web.SearchResult{
		"exa": {
			{Title: "R1", URL: "https://a.com/1", Content: "C1", Score: 0.95, Extra: map[string]any{"_provider": "exa"}},
		},
		"tavily": {
			{Title: "R2", URL: "https://b.com/2", Content: "C2", Score: 0.85, Extra: map[string]any{"_provider": "tavily"}},
		},
	}
	meta := Metadata{
		Caller:    "human",
		Providers: []string{"exa", "tavily"},
	}

	err := Search(dir, "test query", result, rawResults, meta)
	if err != nil {
		t.Fatalf("PersistSearchResult failed: %v", err)
	}

	searchDir := filepath.Join(dir, "search")
	subs, _ := os.ReadDir(searchDir)
	resultDir := filepath.Join(searchDir, subs[0].Name())

	if _, err := os.Stat(filepath.Join(resultDir, "result.json")); os.IsNotExist(err) {
		t.Error("result.json not found")
	}
	if _, err := os.Stat(filepath.Join(resultDir, "query.txt")); os.IsNotExist(err) {
		t.Error("query.txt not found")
	}
	if _, err := os.Stat(filepath.Join(resultDir, "answer.md")); os.IsNotExist(err) {
		t.Error("answer.md not found")
	}
	if _, err := os.Stat(filepath.Join(resultDir, "metadata.json")); os.IsNotExist(err) {
		t.Error("metadata.json not found")
	}

	resultsDir := filepath.Join(resultDir, "results")
	resFiles, _ := os.ReadDir(resultsDir)
	if len(resFiles) != 2 {
		t.Errorf("expected 2 result files, got %d", len(resFiles))
	}

	rawDir := filepath.Join(resultDir, "raw")
	rawFiles, _ := os.ReadDir(rawDir)
	if len(rawFiles) != 2 {
		t.Errorf("expected 2 raw files, got %d", len(rawFiles))
	}

	for _, f := range rawFiles {
		filePath := filepath.Join(rawDir, f.Name())
		data, _ := os.ReadFile(filePath)
		var results []web.SearchResult
		if err := json.Unmarshal(data, &results); err != nil {
			t.Errorf("failed to unmarshal raw file %s: %v", f.Name(), err)
		}
		for _, r := range results {
			if _, ok := r.Extra["_provider"]; ok {
				t.Errorf("_provider key should be stripped from raw file %s", f.Name())
			}
		}
	}
}

func TestPersist_SearchResultEmptyResults(t *testing.T) {
	dir := t.TempDir()
	result := &fusion.Result{}
	meta := Metadata{Caller: "human"}

	err := Search(dir, "empty query", result, nil, meta)
	if err != nil {
		t.Fatalf("PersistSearchResult failed: %v", err)
	}
}

func TestPersist_Agent(t *testing.T) {
	dir := t.TempDir()
	result := &web.AgentResult{
		Answer: "Agent answer",
		Sources: []web.AgentSource{
			{Title: "Source 1", URL: "https://a.com", Text: "Text 1"},
			{Title: "Source 2", URL: "https://b.com", Text: "Text 2"},
		},
	}
	steps := []json.RawMessage{
		json.RawMessage(`{"results":[{"title":"R1","url":"https://a.com"}]}`),
		json.RawMessage(`{"results":[{"title":"R2","url":"https://b.com"}]}`),
	}
	meta := Metadata{Caller: "gmd-agent"}

	err := Agent(dir, "agent query", result, steps, meta)
	if err != nil {
		t.Fatalf("PersistAgentResult failed: %v", err)
	}

	agentDir := filepath.Join(dir, "agent")
	subs, _ := os.ReadDir(agentDir)
	resultDir := filepath.Join(agentDir, subs[0].Name())

	if _, err := os.Stat(filepath.Join(resultDir, "result.json")); os.IsNotExist(err) {
		t.Error("result.json not found")
	}
	if _, err := os.Stat(filepath.Join(resultDir, "query.txt")); os.IsNotExist(err) {
		t.Error("query.txt not found")
	}
	if _, err := os.Stat(filepath.Join(resultDir, "answer.md")); os.IsNotExist(err) {
		t.Error("answer.md not found")
	}
	if _, err := os.Stat(filepath.Join(resultDir, "metadata.json")); os.IsNotExist(err) {
		t.Error("metadata.json not found")
	}

	stepsDir := filepath.Join(resultDir, "steps")
	stepFiles, _ := os.ReadDir(stepsDir)
	if len(stepFiles) != 2 {
		t.Errorf("expected 2 step files, got %d", len(stepFiles))
	}

	sourcesDir := filepath.Join(resultDir, "sources")
	sourceFiles, _ := os.ReadDir(sourcesDir)
	if len(sourceFiles) != 2 {
		t.Errorf("expected 2 source files, got %d", len(sourceFiles))
	}
}

func TestPersist_AgentResultEmptySteps(t *testing.T) {
	dir := t.TempDir()
	result := &web.AgentResult{
		Answer: "Quick answer",
	}
	meta := Metadata{Caller: "gmd-agent"}

	err := Agent(dir, "quick query", result, nil, meta)
	if err != nil {
		t.Fatalf("PersistAgentResult with empty steps failed: %v", err)
	}
}

func TestPersist_WithCaller(t *testing.T) {
	dir := t.TempDir()
	result := &web.GetContentResult{
		Content: "Test",
	}
	meta := Metadata{
		Caller: "my-agent",
	}

	err := Fetch(dir, "https://example.com", result, meta)
	if err != nil {
		t.Fatalf("PersistFetchResult failed: %v", err)
	}

	fetchDir := filepath.Join(dir, "fetch")
	subs, _ := os.ReadDir(fetchDir)
	resultDir := filepath.Join(fetchDir, subs[0].Name())

	data, err := os.ReadFile(filepath.Join(resultDir, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	var parsed Metadata
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Caller != "my-agent" {
		t.Errorf("caller = %q, want %q", parsed.Caller, "my-agent")
	}
}

func TestPersist_DirUnwritable(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "readonly")
	if err := os.Mkdir(dir, 0444); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(dir, 0755) }()

	result := &web.GetContentResult{Content: "Test"}
	meta := Metadata{Caller: "human"}

	err := Fetch(dir, "https://example.com", result, meta)
	if os.Getuid() == 0 {
		// root bypasses filesystem permission checks, so no error expected
		return
	}
	if err == nil {
		t.Error("expected error for unwritable directory")
	}
}

func TestPersist_TimestampCollision(t *testing.T) {
	dir := t.TempDir()
	result := &web.GetContentResult{Content: "First"}
	meta := Metadata{
		Timestamp: "2026-06-10T15_30_00_000000000Z",
		Caller:    "human",
	}

	err := Fetch(dir, "https://example.com", result, meta)
	if err != nil {
		t.Fatalf("first persist failed: %v", err)
	}

	result2 := &web.GetContentResult{Content: "Second"}
	meta2 := Metadata{
		Timestamp: "2026-06-10T15_30_00_000000000Z",
		Caller:    "human",
	}
	err = Fetch(dir, "https://example.com", result2, meta2)
	if err != nil {
		t.Fatalf("second persist with same timestamp failed: %v", err)
	}
}

func TestPersist_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	result := &web.GetContentResult{
		Content: "",
		Extra:   map[string]any{"title": "Empty Page"},
	}
	meta := Metadata{Caller: "human"}

	err := Fetch(dir, "https://example.com/empty", result, meta)
	if err != nil {
		t.Fatalf("PersistFetchResult with empty content failed: %v", err)
	}

	fetchDir := filepath.Join(dir, "fetch")
	subs, _ := os.ReadDir(fetchDir)
	resultDir := filepath.Join(fetchDir, subs[0].Name())

	if _, err := os.Stat(filepath.Join(resultDir, "result.json")); os.IsNotExist(err) {
		t.Error("result.json should exist even with empty content")
	}
	if _, err := os.Stat(filepath.Join(resultDir, "content.md")); !os.IsNotExist(err) {
		t.Error("content.md should not exist for empty content")
	}
}

func TestPersist_URLSlugEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"empty string", ""},
		{"only path", "/path/to/page"},
		{"no host colon", "//path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := urlSlug(tt.url)
			if got == "" {
				t.Errorf("urlSlug(%q) returned empty string", tt.url)
			}
		})
	}
}

func TestPersist_MetadataFlags(t *testing.T) {
	dir := t.TempDir()
	result := &web.GetContentResult{Content: "Test"}
	meta := Metadata{
		Caller: "human",
		Flags: map[string]any{
			"format":  "markdown",
			"maxAge":  float64(0),
			"json":    false,
			"outdir":  ".",
			"output":  "stdout",
			"summary": "",
		},
	}

	err := Fetch(dir, "https://example.com", result, meta)
	if err != nil {
		t.Fatalf("PersistFetchResult failed: %v", err)
	}

	fetchDir := filepath.Join(dir, "fetch")
	subs, _ := os.ReadDir(fetchDir)
	resultDir := filepath.Join(fetchDir, subs[0].Name())

	data, err := os.ReadFile(filepath.Join(resultDir, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	var parsed Metadata
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Flags == nil {
		t.Fatal("flags should not be nil")
	}
	if v, ok := parsed.Flags["format"]; !ok || v != "markdown" {
		t.Errorf("flags[format] = %v, want markdown", v)
	}
}

func TestPersist_URLSlugFallback(t *testing.T) {
	// urlSlug uses url.Parse which handles various edge cases
	result := urlSlug("://invalid")
	if result == "" {
		t.Error("urlSlug should produce a fallback slug for malformed URLs")
	}
}
