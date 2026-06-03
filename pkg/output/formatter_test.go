package output

import (
	"strings"
	"testing"

	"github.com/verdverm/gmd/pkg/search"
	"github.com/verdverm/gmd/pkg/ts"
)

func makeResult(path, title, content string, score float64) search.Result {
	return search.Result{
		Collection: "docs",
		Path:       path,
		Title:      title,
		Content:    content,
		ChunkSeq:   0,
		Score:      score,
	}
}

func TestMakeSnippet(t *testing.T) {
	tests := []struct {
		name    string
		content string
		maxLen  int
		want    string
	}{
		{
			name:    "empty content",
			content: "",
			maxLen:  200,
			want:    "",
		},
		{
			name:    "whitespace only",
			content: "  \n  \n  ",
			maxLen:  200,
			want:    "",
		},
		{
			name:    "short content",
			content: "Hello world",
			maxLen:  200,
			want:    "Hello world",
		},
		{
			name:    "exact length",
			content: "Hello world",
			maxLen:  11,
			want:    "Hello world",
		},
		{
			name:    "long content truncated",
			content: "This is a very long content that should be truncated at some point",
			maxLen:  20,
			want:    "This is a very long ...",
		},
		{
			name:    "multiline condensed",
			content: "line1\n\nline2\n\nline3",
			maxLen:  200,
			want:    "line1 line2 line3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeSnippet(tt.content, tt.maxLen)
			if got != tt.want {
				t.Errorf("makeSnippet() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatCLI(t *testing.T) {
	t.Run("no results", func(t *testing.T) {
		got := formatCLI(nil)
		if got != "No results found." {
			t.Errorf("got %q, want %q", got, "No results found.")
		}
	})

	t.Run("empty results", func(t *testing.T) {
		got := formatCLI([]search.Result{})
		if got != "No results found." {
			t.Errorf("got %q, want %q", got, "No results found.")
		}
	})

	t.Run("single result", func(t *testing.T) {
		results := []search.Result{
			makeResult("test.md", "Test Doc", "Some content here", 0.95),
		}
		got := formatCLI(results)
		if !strings.Contains(got, "test.md") {
			t.Errorf("output should contain path")
		}
		if !strings.Contains(got, "Test Doc") {
			t.Errorf("output should contain title")
		}
		if !strings.Contains(got, "0.9500") {
			t.Errorf("output should contain score, got: %s", got)
		}
	})

	t.Run("multiple results numbered", func(t *testing.T) {
		results := []search.Result{
			makeResult("a.md", "", "content a", 0.9),
			makeResult("b.md", "", "content b", 0.8),
		}
		got := formatCLI(results)
		if !strings.Contains(got, "1. a.md") {
			t.Errorf("expected '1. a.md', got: %s", got)
		}
		if !strings.Contains(got, "2. b.md") {
			t.Errorf("expected '2. b.md', got: %s", got)
		}
	})

	t.Run("result with no title omits title line", func(t *testing.T) {
		results := []search.Result{
			makeResult("path.md", "", "some content", 0.5),
		}
		got := formatCLI(results)
		if strings.Contains(got, "Title:") {
			t.Errorf("should not contain Title line when title is empty")
		}
	})

	t.Run("result with no collection omits collection line", func(t *testing.T) {
		r := makeResult("path.md", "", "content", 0.5)
		r.Collection = ""
		got := formatCLI([]search.Result{r})
		if strings.Contains(got, "Collection:") {
			t.Errorf("should not contain Collection line when collection is empty")
		}
	})
}

func TestFormatJSON(t *testing.T) {
	t.Run("valid JSON output", func(t *testing.T) {
		results := []search.Result{
			makeResult("test.md", "Test", "content", 0.95),
		}
		got, err := formatJSON(results)
		if err != nil {
			t.Fatalf("formatJSON error: %v", err)
		}
		if !strings.Contains(got, "test.md") {
			t.Errorf("JSON should contain path")
		}
		if !strings.Contains(got, "0.95") {
			t.Errorf("JSON should contain score")
		}
	})
}

func TestFormatResults(t *testing.T) {
	results := []search.Result{
		makeResult("test.md", "Test", "content", 0.95),
	}

	t.Run("json format", func(t *testing.T) {
		got, err := FormatResults(results, "json")
		if err != nil {
			t.Fatalf("FormatResults error: %v", err)
		}
		if !strings.HasPrefix(got, "[") {
			t.Errorf("JSON should start with [")
		}
	})

	t.Run("cli format", func(t *testing.T) {
		got, err := FormatResults(results, "cli")
		if err != nil {
			t.Fatalf("FormatResults error: %v", err)
		}
		if !strings.Contains(got, "test.md") {
			t.Errorf("CLI output should contain path")
		}
	})

	t.Run("text format same as cli", func(t *testing.T) {
		got, err := FormatResults(results, "text")
		if err != nil {
			t.Fatalf("FormatResults error: %v", err)
		}
		if !strings.Contains(got, "test.md") {
			t.Errorf("text output should contain path")
		}
	})

	t.Run("unknown format defaults to cli", func(t *testing.T) {
		got, err := FormatResults(results, "unknown")
		if err != nil {
			t.Fatalf("FormatResults error: %v", err)
		}
		if !strings.Contains(got, "test.md") {
			t.Errorf("unknown format should fallback to CLI")
		}
	})

	t.Run("empty results cli format", func(t *testing.T) {
		got, err := FormatResults(nil, "cli")
		if err != nil {
			t.Fatalf("FormatResults error: %v", err)
		}
		if got != "No results found." {
			t.Errorf("got %q, want %q", got, "No results found.")
		}
	})
}

func TestSnippetTruncation(t *testing.T) {
	longText := strings.Repeat("word ", 100)
	snippet := makeSnippet(longText, 20)
	if len(snippet) > 20+3 {
		t.Errorf("snippet length %d exceeds maxLen+3", len(snippet))
	}
	if !strings.HasSuffix(snippet, "...") {
		t.Errorf("truncated snippet should end with ...")
	}
}

func TestSnippetNoTruncation(t *testing.T) {
	text := "short text"
	snippet := makeSnippet(text, 200)
	if snippet != text {
		t.Errorf("short content should not be truncated, got %q", snippet)
	}
}

func makeTSResult(collection, path string) ts.HybridSearchResult {
	return ts.HybridSearchResult{
		Collection: collection,
		Path:       path,
		Title:      path,
		Content:    "content",
		Score:      1.0,
	}
}

func TestFormatLS(t *testing.T) {
	t.Run("no results", func(t *testing.T) {
		got := FormatLS(nil)
		if got != "No indexed documents." {
			t.Errorf("got %q, want %q", got, "No indexed documents.")
		}
	})

	t.Run("empty results", func(t *testing.T) {
		got := FormatLS([]ts.HybridSearchResult{})
		if got != "No indexed documents." {
			t.Errorf("got %q, want %q", got, "No indexed documents.")
		}
	})

	t.Run("single collection sorts paths", func(t *testing.T) {
		results := []ts.HybridSearchResult{
			makeTSResult("docs", "z.md"),
			makeTSResult("docs", "a.md"),
			makeTSResult("docs", "m.md"),
		}
		got := FormatLS(results)
		if !strings.Contains(got, "docs:\n") {
			t.Errorf("should contain collection header")
		}
		if !strings.Contains(got, "  a.md\n") {
			t.Errorf("paths should be sorted, missing a.md")
		}
		if !strings.Contains(got, "  m.md\n") {
			t.Errorf("paths should be sorted, missing m.md")
		}
		if !strings.Contains(got, "  z.md\n") {
			t.Errorf("paths should be sorted, missing z.md")
		}
	})

	t.Run("multiple collections sorted alphabetically", func(t *testing.T) {
		results := []ts.HybridSearchResult{
			makeTSResult("zzz", "f.md"),
			makeTSResult("aaa", "b.md"),
			makeTSResult("mmm", "c.md"),
		}
		got := FormatLS(results)
		if !strings.HasPrefix(got, "aaa:\n") {
			t.Errorf("collections should be sorted, first should be aaa, got: %s", got)
		}
	})

	t.Run("count matches result length", func(t *testing.T) {
		results := []ts.HybridSearchResult{
			makeTSResult("docs", "a.md"),
			makeTSResult("docs", "b.md"),
		}
		got := FormatLS(results)
		if !strings.Contains(got, "2 document(s) indexed") {
			t.Errorf("should show correct count")
		}
	})
}
