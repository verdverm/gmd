package chunking

import (
	"reflect"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.TargetTokens != 900 {
		t.Errorf("TargetTokens = %d, want 900", cfg.TargetTokens)
	}
	if cfg.Overlap != 0.15 {
		t.Errorf("Overlap = %f, want 0.15", cfg.Overlap)
	}
	if cfg.HeadingWeights.H1 != 100 {
		t.Errorf("H1 weight = %d, want 100", cfg.HeadingWeights.H1)
	}
	if cfg.HeadingWeights.H6 != 50 {
		t.Errorf("H6 weight = %d, want 50", cfg.HeadingWeights.H6)
	}
}

func TestTokenCount(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"abc", 1},
		{"abcdefgh", 2},
		{"hello world foo", 4},
		{"こんにちは", 2},
	}
	for _, tt := range tests {
		got := tokenCount(tt.input)
		if got != tt.want {
			t.Errorf("tokenCount(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestExtractHeadings(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []headingInfo
	}{
		{
			name:  "no headings",
			input: "plain text\nmore text",
			want:  nil,
		},
		{
			name:  "h1 heading",
			input: "# Title\ncontent",
			want:  []headingInfo{{level: 1, text: "Title", lineIdx: 0}},
		},
		{
			name:  "multiple heading levels",
			input: "# H1\n## H2\n### H3\ncontent",
			want: []headingInfo{
				{level: 1, text: "H1", lineIdx: 0},
				{level: 2, text: "H2", lineIdx: 1},
				{level: 3, text: "H3", lineIdx: 2},
			},
		},
		{
			name:  "heading with leading spaces",
			input: "  ## Indented",
			want:  []headingInfo{{level: 2, text: "Indented", lineIdx: 0}},
		},
		{
			name:  "not a heading (no space after #)",
			input: "#NotHeading",
			want:  nil,
		},
		{
			name:  "empty lines before heading",
			input: "\n\n# Title",
			want:  []headingInfo{{level: 1, text: "Title", lineIdx: 2}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHeadings(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d headings, want %d: %+v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i].level != tt.want[i].level || got[i].text != tt.want[i].text || got[i].lineIdx != tt.want[i].lineIdx {
					t.Errorf("heading[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSplitIntoSegments(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		headings []headingInfo
		wantLen  int
	}{
		{
			name:     "no headings",
			lines:    []string{"line1", "line2"},
			headings: nil,
			wantLen:  1,
		},
		{
			name:     "single heading",
			lines:    []string{"# H1", "content1", "content2"},
			headings: []headingInfo{{level: 1, text: "H1", lineIdx: 0}},
			wantLen:  1,
		},
		{
			name:     "multiple headings",
			lines:    []string{"# H1", "c1", "## H2", "c2"},
			headings: []headingInfo{{level: 1, text: "H1", lineIdx: 0}, {level: 2, text: "H2", lineIdx: 2}},
			wantLen:  2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segs := splitIntoSegments(tt.lines, tt.headings)
			if len(segs) != tt.wantLen {
				t.Errorf("got %d segments, want %d", len(segs), tt.wantLen)
			}
		})
	}
}

func TestChunkMarkdownSmallDoc(t *testing.T) {
	content := "Small document content"
	cfg := DefaultConfig()
	chunks := ChunkMarkdown(content, cfg)
	if len(chunks) != 1 {
		t.Fatalf("got %d chunks, want 1", len(chunks))
	}
	if chunks[0].Content != content {
		t.Errorf("chunk content = %q, want %q", chunks[0].Content, content)
	}
	if chunks[0].ChunkSeq != 0 {
		t.Errorf("chunk seq = %d, want 0", chunks[0].ChunkSeq)
	}
	if chunks[0].TotalChunks != 1 {
		t.Errorf("total chunks = %d, want 1", chunks[0].TotalChunks)
	}
}

func TestChunkMarkdownSplitByHeadings(t *testing.T) {
	content := strings.Repeat("intro text ", 100) + "\n# Section 1\n" + strings.Repeat("section1 content ", 100) + "\n## Section 2\n" + strings.Repeat("section2 content ", 50)
	cfg := DefaultConfig()
	cfg.TargetTokens = 50

	chunks := ChunkMarkdown(content, cfg)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	firstChunkContent := strings.TrimSpace(chunks[0].Content)
	if !strings.Contains(firstChunkContent, "# Section 1") && !strings.Contains(firstChunkContent, "intro text") {
		t.Errorf("first chunk should contain intro or section heading, got: %s", firstChunkContent)
	}

	for _, c := range chunks {
		if c.Title == "" && strings.HasPrefix(content, "#") {
		}
		if c.ChunkSeq < 0 || c.ChunkSeq >= c.TotalChunks {
			t.Errorf("invalid chunk seq %d / %d", c.ChunkSeq, c.TotalChunks)
		}
	}
}

func TestChunkMarkdownTitleFromH1(t *testing.T) {
	content := "# Document Title\n\nSome content here."
	cfg := DefaultConfig()
	chunks := ChunkMarkdown(content, cfg)

	if len(chunks) != 1 {
		t.Fatalf("got %d chunks, want 1", len(chunks))
	}
	if chunks[0].Title != "Document Title" {
		t.Errorf("title = %q, want %q", chunks[0].Title, "Document Title")
	}
}

func TestChunkMarkdownNoTitleWhenNoH1(t *testing.T) {
	content := "## Section\n\nSome content."
	cfg := DefaultConfig()
	chunks := ChunkMarkdown(content, cfg)

	if len(chunks) != 1 {
		t.Fatalf("got %d chunks, want 1", len(chunks))
	}
	if chunks[0].Title != "" {
		t.Errorf("title should be empty for no H1, got %q", chunks[0].Title)
	}
}

func TestChunkMarkdownOverlap(t *testing.T) {
	content := "# H1\n" + strings.Repeat("word ", 100) + "\n# H2\n" + strings.Repeat("word ", 100) + "\n# H3\n" + strings.Repeat("word ", 100)
	cfg := DefaultConfig()
	cfg.TargetTokens = 50

	chunks := ChunkMarkdown(content, cfg)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
}

func TestChunkMarkdownWithMixedElements(t *testing.T) {
	content := `# Introduction

This is the introduction paragraph.

## Section 1

` + strings.Repeat("Some content in section 1. ", 50) + `

## Section 2

` + "```javascript\nfunction hello() {\n  console.log(\"Hello\");\n}\n```" + `

` + strings.Repeat("More text after the code block. ", 50) + `

---

## Section 3

` + strings.Repeat("Final section content. ", 50)

	cfg := DefaultConfig()
	cfg.TargetTokens = 50

	chunks := ChunkMarkdown(content, cfg)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks for mixed content, got %d", len(chunks))
	}

	for _, c := range chunks {
		if strings.TrimSpace(c.Content) == "" {
			t.Errorf("found empty chunk at seq %d", c.ChunkSeq)
		}
	}
}

func TestChunkMarkdownLargeDoc(t *testing.T) {
	content := "# One\n" + strings.Repeat("word ", 200) + "\n# Two\n" + strings.Repeat("word ", 200) + "\n# Three\n" + strings.Repeat("word ", 200)
	cfg := DefaultConfig()
	cfg.TargetTokens = 50

	chunks := ChunkMarkdown(content, cfg)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks for large doc, got %d", len(chunks))
	}

	for i := 1; i < len(chunks); i++ {
		if chunks[i].ChunkSeq <= chunks[i-1].ChunkSeq {
			t.Errorf("chunk seq not monotonically increasing at %d", i)
		}
	}
}

func TestChunkMarkdownOverlapCarryFullPrev(t *testing.T) {
	content := "# H1\nhello\n# H2\n" + strings.Repeat("word ", 100)
	cfg := DefaultConfig()
	cfg.TargetTokens = 50
	cfg.Overlap = 0.9

	chunks := ChunkMarkdown(content, cfg)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	if !strings.Contains(chunks[1].Content, "hello") {
		t.Errorf("chunk 1 should carry full prev overlap, got: %q", chunks[1].Content[:50])
	}
}

func TestChunkMarkdownUTF8(t *testing.T) {
	content := strings.Repeat("こんにちは世界", 500)
	cfg := DefaultConfig()
	cfg.TargetTokens = 100

	chunks := ChunkMarkdown(content, cfg)

	if len(chunks) < 1 {
		t.Fatal("expected at least one chunk")
	}
	for _, c := range chunks {
		if c.Content == "" {
			t.Errorf("found empty chunk")
		}
	}
}

func TestExtractWikilinks(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "no wikilinks",
			in:   "plain text with no links",
			want: nil,
		},
		{
			name: "single wikilink",
			in:   "see [[TargetPage]] for details",
			want: []string{"TargetPage"},
		},
		{
			name: "multiple wikilinks",
			in:   "[[PageA]] and [[PageB]] are related",
			want: []string{"PageA", "PageB"},
		},
		{
			name: "deduplication",
			in:   "[[Same]] and [[Same]] again",
			want: []string{"Same"},
		},
		{
			name: "with display text after pipe",
			in:   "[[Target|display text]]",
			want: []string{"Target"},
		},
		{
			name: "with anchor fragment",
			in:   "[[Target#section]]",
			want: []string{"Target"},
		},
		{
			name: "whitespace trimmed",
			in:   "[[  Spaced  ]]",
			want: []string{"Spaced"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractWikilinks(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("links[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractFrontmatter(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		wantFM map[string]interface{}
		wantOK bool
	}{
		{
			name:   "no frontmatter",
			in:     "# Just content\n\nhello",
			wantFM: nil,
			wantOK: false,
		},
		{
			name: "valid frontmatter",
			in:   "---\ntitle: My Doc\ntags: [a, b]\n---\n# Content",
			wantFM: map[string]interface{}{
				"title": "My Doc",
				"tags":  []interface{}{"a", "b"},
			},
			wantOK: true,
		},

		{
			name:   "invalid yaml frontmatter",
			in:     "---\n: invalid yaml\n---\n# Content",
			wantFM: nil,
			wantOK: false,
		},
		{
			name:   "dashes in content",
			in:     "---\nkey: val\n---\nMore --- dashes",
			wantFM: map[string]interface{}{"key": "val"},
			wantOK: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, remaining := ExtractFrontmatter(tt.in)
			if tt.wantOK {
				if fm == nil {
					t.Fatal("expected frontmatter, got nil")
				}
				for k, v := range tt.wantFM {
					if !reflect.DeepEqual(fm[k], v) {
						t.Errorf("fm[%q] = %v (%T), want %v (%T)", k, fm[k], fm[k], v, v)
					}
				}
				if remaining == tt.in {
					t.Error("remaining should differ from input when frontmatter is extracted")
				}
			} else {
				if fm != nil {
					t.Errorf("expected nil frontmatter, got %v", fm)
				}
				if remaining != tt.in {
					t.Error("remaining should equal input when no frontmatter")
				}
			}
		})
	}
}

func TestChunkMarkdownWithMeta(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		wantTitle string
		wantLinks []string
		wantFM    bool
	}{
		{
			name:      "plain content, no metadata",
			in:        "# Doc\n\nhello world",
			wantTitle: "Doc",
			wantLinks: nil,
			wantFM:    false,
		},
		{
			name:      "with frontmatter and wikilinks",
			in:        "---\ntitle: Metadata\n---\n# Doc\n\nsee [[Other]] page",
			wantTitle: "Doc",
			wantLinks: []string{"Other"},
			wantFM:    true,
		},
		{
			name:      "multiple wikilinks",
			in:        "---\nkey: val\n---\n# Doc\n\n[[A]] and [[B]] and [[A]]",
			wantTitle: "Doc",
			wantLinks: []string{"A", "B"},
			wantFM:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks, links, fm := ChunkMarkdownWithMeta(tt.in, DefaultConfig())
			if len(chunks) == 0 {
				t.Fatal("expected at least one chunk")
			}
			if chunks[0].Title != tt.wantTitle {
				t.Errorf("title = %q, want %q", chunks[0].Title, tt.wantTitle)
			}
			if len(links) != len(tt.wantLinks) {
				t.Fatalf("links = %v, want %v", links, tt.wantLinks)
			}
			for i := range links {
				if links[i] != tt.wantLinks[i] {
					t.Errorf("links[%d] = %q, want %q", i, links[i], tt.wantLinks[i])
				}
			}
			if tt.wantFM && fm == nil {
				t.Error("expected frontmatter, got nil")
			}
			if !tt.wantFM && fm != nil {
				t.Errorf("expected nil frontmatter, got %v", fm)
			}
			for _, c := range chunks {
				if len(c.Links) != len(tt.wantLinks) {
					t.Errorf("chunk[%d] links = %v, want %v", c.ChunkSeq, c.Links, tt.wantLinks)
				}
				if tt.wantFM && c.Frontmatter == nil {
					t.Errorf("chunk[%d] frontmatter should not be nil", c.ChunkSeq)
				}
			}
		})
	}
}
