package wiki

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/verdverm/gmd/pkg/config"
)

// ---------------------------------------------------------------------------
// frontmatter.go
// ---------------------------------------------------------------------------

func TestParseFrontmatter(t *testing.T) {
	t.Run("no frontmatter", func(t *testing.T) {
		input := "# Just content\n\nhello"
		fm, remaining, err := ParseFrontmatter(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fm != nil {
			t.Errorf("expected nil frontmatter, got %v", fm)
		}
		if remaining != input {
			t.Errorf("expected remaining == input")
		}
	})

	t.Run("valid frontmatter with string", func(t *testing.T) {
		input := "---\ntitle: My Doc\n---\n# Content"
		fm, remaining, err := ParseFrontmatter(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fm == nil {
			t.Fatal("expected frontmatter, got nil")
		}
		if fm["title"] != "My Doc" {
			t.Errorf("title = %v, want %v", fm["title"], "My Doc")
		}
		if remaining != "# Content" {
			t.Errorf("remaining = %q, want %q", remaining, "# Content")
		}
	})

	t.Run("valid frontmatter with array", func(t *testing.T) {
		input := "---\ntags: [a, b, c]\n---\nbody"
		fm, remaining, err := ParseFrontmatter(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fm == nil {
			t.Fatal("expected frontmatter, got nil")
		}
		tags, ok := fm["tags"].([]interface{})
		if !ok {
			t.Fatalf("tags is %T, want []interface{}", fm["tags"])
		}
		if len(tags) != 3 || tags[0] != "a" || tags[1] != "b" || tags[2] != "c" {
			t.Errorf("tags = %v, want [a b c]", tags)
		}
		if remaining != "body" {
			t.Errorf("remaining = %q, want %q", remaining, "body")
		}
	})

	t.Run("invalid yaml frontmatter", func(t *testing.T) {
		input := "---\n: invalid yaml\n---\n# Content"
		_, _, err := ParseFrontmatter(input)
		if err == nil {
			t.Fatal("expected error for invalid YAML")
		}
	})

	t.Run("dashes in content after frontmatter", func(t *testing.T) {
		input := "---\nkey: val\n---\nMore --- dashes"
		fm, remaining, err := ParseFrontmatter(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fm["key"] != "val" {
			t.Errorf("key = %v, want val", fm["key"])
		}
		if remaining != "More --- dashes" {
			t.Errorf("remaining = %q, want %q", remaining, "More --- dashes")
		}
	})

	t.Run("empty frontmatter (blank line between)", func(t *testing.T) {
		input := "---\n\n---\ncontent"
		fm, remaining, err := ParseFrontmatter(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if remaining != "content" {
			t.Errorf("remaining = %q, want %q", remaining, "content")
		}
		_ = fm
	})

	t.Run("multiline frontmatter", func(t *testing.T) {
		input := "---\ntitle: Doc\nstatus: draft\ntags: [foo]\n---\nbody"
		fm, _, err := ParseFrontmatter(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fm["title"] != "Doc" {
			t.Errorf("title = %v, want Doc", fm["title"])
		}
		if fm["status"] != "draft" {
			t.Errorf("status = %v, want draft", fm["status"])
		}
	})

	t.Run("no closing dashes", func(t *testing.T) {
		input := "---\ntitle: Doc\n# Content"
		fm, remaining, err := ParseFrontmatter(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fm != nil {
			t.Errorf("expected nil frontmatter for no closing ---")
		}
		if remaining != input {
			t.Errorf("remaining should match input when no frontmatter matched")
		}
	})
}

func TestStripFrontmatter(t *testing.T) {
	t.Run("no frontmatter", func(t *testing.T) {
		input := "# Just content\n\nhello"
		got := StripFrontmatter(input)
		if got != input {
			t.Errorf("got %q, want %q", got, input)
		}
	})

	t.Run("strips frontmatter", func(t *testing.T) {
		input := "---\ntitle: My Doc\n---\n# Content"
		got := StripFrontmatter(input)
		if got != "# Content" {
			t.Errorf("got %q, want %q", got, "# Content")
		}
	})

	t.Run("empty body after frontmatter", func(t *testing.T) {
		input := "---\ntitle: x\n---\n"
		got := StripFrontmatter(input)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

func TestValidateFrontmatter(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		err := ValidateFrontmatter(map[string]interface{}{"x": 1}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty fields", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{Fields: map[string]config.FrontmatterField{}}
		err := ValidateFrontmatter(map[string]interface{}{"x": 1}, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid string field", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"title": {Type: "string"},
			},
		}
		err := ValidateFrontmatter(map[string]interface{}{"title": "My Doc"}, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid string field", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"title": {Type: "string"},
			},
		}
		err := ValidateFrontmatter(map[string]interface{}{"title": 42}, cfg)
		if err == nil {
			t.Fatal("expected error for non-string value")
		}
	})

	t.Run("valid string array field", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"tags": {Type: "string[]"},
			},
		}
		err := ValidateFrontmatter(map[string]interface{}{"tags": []interface{}{"a", "b"}}, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid Go string array", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"tags": {Type: "string[]"},
			},
		}
		err := ValidateFrontmatter(map[string]interface{}{"tags": []string{"a", "b"}}, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid string array element", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"tags": {Type: "string[]"},
			},
		}
		err := ValidateFrontmatter(map[string]interface{}{"tags": []interface{}{"a", 42}}, cfg)
		if err == nil {
			t.Fatal("expected error for non-string array element")
		}
	})

	t.Run("invalid string array type", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"tags": {Type: "string[]"},
			},
		}
		err := ValidateFrontmatter(map[string]interface{}{"tags": 42}, cfg)
		if err == nil {
			t.Fatal("expected error for non-array value")
		}
	})

	t.Run("valid int32 field", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"level": {Type: "int32"},
			},
		}
		err := ValidateFrontmatter(map[string]interface{}{"level": 3}, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid int32 field", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"level": {Type: "int32"},
			},
		}
		err := ValidateFrontmatter(map[string]interface{}{"level": "high"}, cfg)
		if err == nil {
			t.Fatal("expected error for non-int value")
		}
	})

	t.Run("valid float field", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"score": {Type: "float"},
			},
		}
		err := ValidateFrontmatter(map[string]interface{}{"score": 3.5}, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid float field", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"score": {Type: "float"},
			},
		}
		err := ValidateFrontmatter(map[string]interface{}{"score": "high"}, cfg)
		if err == nil {
			t.Fatal("expected error for non-float value")
		}
	})

	t.Run("valid bool field", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"published": {Type: "bool"},
			},
		}
		err := ValidateFrontmatter(map[string]interface{}{"published": true}, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid bool field", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"published": {Type: "bool"},
			},
		}
		err := ValidateFrontmatter(map[string]interface{}{"published": "yes"}, cfg)
		if err == nil {
			t.Fatal("expected error for non-bool value")
		}
	})

	t.Run("unknown field type", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"custom": {Type: "unknown"},
			},
		}
		err := ValidateFrontmatter(map[string]interface{}{"custom": "val"}, cfg)
		if err == nil {
			t.Fatal("expected error for unknown type")
		}
	})

	t.Run("missing field does not error", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"title": {Type: "string"},
			},
		}
		err := ValidateFrontmatter(map[string]interface{}{"other": "val"}, cfg)
		if err != nil {
			t.Fatalf("unexpected error for missing field: %v", err)
		}
	})
}

func TestFrontmatterToFilter(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		got := FrontmatterToFilter(map[string]interface{}{"x": 1}, nil)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("empty fields", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{Fields: map[string]config.FrontmatterField{}}
		got := FrontmatterToFilter(map[string]interface{}{"x": 1}, cfg)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("string field", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"type": {Type: "string"},
			},
		}
		got := FrontmatterToFilter(map[string]interface{}{"type": "entity"}, cfg)
		if got != "type:=entity" {
			t.Errorf("got %q, want %q", got, "type:=entity")
		}
	})

	t.Run("string array field with []interface{}", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"tags": {Type: "string[]"},
			},
		}
		got := FrontmatterToFilter(map[string]interface{}{"tags": []interface{}{"a", "b"}}, cfg)
		if got != "tags:=[a,b]" {
			t.Errorf("got %q, want %q", got, "tags:=[a,b]")
		}
	})

	t.Run("string array field with []string", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"tags": {Type: "string[]"},
			},
		}
		got := FrontmatterToFilter(map[string]interface{}{"tags": []string{"x", "y"}}, cfg)
		if got != "tags:=[x,y]" {
			t.Errorf("got %q, want %q", got, "tags:=[x,y]")
		}
	})

	t.Run("empty string array is skipped", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"tags": {Type: "string[]"},
			},
		}
		got := FrontmatterToFilter(map[string]interface{}{"tags": []interface{}{}}, cfg)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("int32 field", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"level": {Type: "int32"},
			},
		}
		got := FrontmatterToFilter(map[string]interface{}{"level": 3}, cfg)
		if got != "level:=3" {
			t.Errorf("got %q, want %q", got, "level:=3")
		}
	})

	t.Run("float field", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"score": {Type: "float"},
			},
		}
		got := FrontmatterToFilter(map[string]interface{}{"score": 3.5}, cfg)
		if got != "score:=3.5" {
			t.Errorf("got %q, want %q", got, "score:=3.5")
		}
	})

	t.Run("bool field", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"published": {Type: "bool"},
			},
		}
		got := FrontmatterToFilter(map[string]interface{}{"published": true}, cfg)
		if got != "published:=true" {
			t.Errorf("got %q, want %q", got, "published:=true")
		}
	})

	t.Run("multiple fields combined with &&", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"type":      {Type: "string"},
				"published": {Type: "bool"},
			},
		}
		got := FrontmatterToFilter(map[string]interface{}{"type": "entity", "published": true}, cfg)
		if !strings.Contains(got, "type:=entity") {
			t.Errorf("expected type:=entity in %q", got)
		}
		if !strings.Contains(got, "published:=true") {
			t.Errorf("expected published:=true in %q", got)
		}
		if !strings.Contains(got, " && ") {
			t.Errorf("expected && separator in %q", got)
		}
	})

	t.Run("missing field skipped", func(t *testing.T) {
		cfg := &config.FrontmatterConfig{
			Fields: map[string]config.FrontmatterField{
				"type": {Type: "string"},
			},
		}
		got := FrontmatterToFilter(map[string]interface{}{}, cfg)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

// ---------------------------------------------------------------------------
// graph.go
// ---------------------------------------------------------------------------

func TestPageName(t *testing.T) {
	tests := []struct {
		wikiDir string
		path    string
		want    string
	}{
		{"wiki", "wiki/entities/foo.md", "entities/foo"},
		{"wiki", "wiki/concepts/bar.md", "concepts/bar"},
		{"/base/wiki", "/base/wiki/entities/x.md", "entities/x"},
		{"wiki", "wiki/sources/2024-01-01-title.md", "sources/2024-01-01-title"},
	}
	for _, tt := range tests {
		got := pageName(tt.wikiDir, tt.path)
		if got != tt.want {
			t.Errorf("pageName(%q, %q) = %q, want %q", tt.wikiDir, tt.path, got, tt.want)
		}
	}
}

func TestSanitizeNode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"kebab-case-name", "kebab_case_name"},
		{"path/to/node", "path_to_node"},
		{"mix-of-both/and-dashes", "mix_of_both_and_dashes"},
		{"already_underscore", "already_underscore"},
		{"empty", "empty"},
	}
	for _, tt := range tests {
		got := sanitizeNode(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeNode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatGraphDot(t *testing.T) {
	g := &Graph{
		Nodes: []string{"a", "b"},
		Edges: []GraphEdge{
			{From: "a", To: "b"},
		},
	}
	got := FormatGraph(g, "dot")
	if !strings.HasPrefix(got, "digraph wiki {") {
		t.Errorf("expected digraph prefix, got %q", got[:30])
	}
	if !strings.Contains(got, `"a" -> "b"`) {
		t.Errorf("expected edge, got %q", got)
	}
	if !strings.HasSuffix(strings.TrimSpace(got), "}") {
		t.Errorf("expected closing brace")
	}
}

func TestFormatGraphMermaid(t *testing.T) {
	g := &Graph{
		Nodes: []string{"entity-foo", "concept-bar"},
		Edges: []GraphEdge{
			{From: "entity-foo", To: "concept-bar"},
		},
	}
	got := FormatGraph(g, "mermaid")
	if !strings.HasPrefix(got, "graph TD") {
		t.Errorf("expected mermaid prefix, got %q", got[:20])
	}
	if !strings.Contains(got, "entity_foo --> concept_bar") {
		t.Errorf("expected sanitized edge, got %q", got)
	}
}

func TestFormatGraphJSON(t *testing.T) {
	g := &Graph{
		Nodes: []string{"a", "b"},
		Edges: []GraphEdge{
			{From: "a", To: "b"},
		},
	}
	got := FormatGraph(g, "json")
	if !strings.Contains(got, `"nodes"`) {
		t.Errorf("expected nodes key, got %q", got[:30])
	}
	if !strings.Contains(got, `"edges"`) {
		t.Errorf("expected edges key")
	}
	if !strings.Contains(got, `"from": "a"`) {
		t.Errorf("expected from field")
	}
	if !strings.Contains(got, `"to": "b"`) {
		t.Errorf("expected to field")
	}
}

func TestFormatGraphUnknownFormat(t *testing.T) {
	g := &Graph{
		Nodes: []string{"a"},
		Edges: []GraphEdge{
			{From: "a", To: "b"},
		},
	}
	got := FormatGraph(g, "unknown")
	if !strings.HasPrefix(got, "digraph wiki {") {
		t.Errorf("expected dot format as default, got %q", got[:30])
	}
}

func TestFormatGraphEmpty(t *testing.T) {
	g := &Graph{}
	got := FormatGraph(g, "dot")
	if !strings.Contains(got, "digraph wiki") {
		t.Errorf("expected digraph, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// agent.go — pure helper functions
// ---------------------------------------------------------------------------

func TestTruncate(t *testing.T) {
	t.Run("short string unchanged", func(t *testing.T) {
		got := truncate("hello", 10)
		if got != "hello" {
			t.Errorf("got %q, want %q", got, "hello")
		}
	})

	t.Run("exact length", func(t *testing.T) {
		got := truncate("hello", 5)
		if got != "hello" {
			t.Errorf("got %q, want %q", got, "hello")
		}
	})

	t.Run("long string truncated", func(t *testing.T) {
		got := truncate("hello world this is long", 10)
		if got != "hello worl..." {
			t.Errorf("got %q, want %q", got, "hello worl...")
		}
	})

	t.Run("empty string", func(t *testing.T) {
		got := truncate("", 10)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

func TestCleanJSON(t *testing.T) {
	t.Run("already clean", func(t *testing.T) {
		got := cleanJSON(`{"key": "val"}`)
		if string(got) != `{"key": "val"}` {
			t.Errorf("got %q, want %q", string(got), `{"key": "val"}`)
		}
	})

	t.Run("with json code fence", func(t *testing.T) {
		input := "```json\n{\"key\": \"val\"}\n```"
		got := cleanJSON(input)
		if string(got) != `{"key": "val"}` {
			t.Errorf("got %q, want %q", string(got), `{"key": "val"}`)
		}
	})

	t.Run("with generic code fence", func(t *testing.T) {
		input := "```\n{\"key\": \"val\"}\n```"
		got := cleanJSON(input)
		if string(got) != `{"key": "val"}` {
			t.Errorf("got %q, want %q", string(got), `{"key": "val"}`)
		}
	})

	t.Run("whitespace trimmed", func(t *testing.T) {
		got := cleanJSON("  {\"key\": \"val\"}  ")
		if string(got) != `{"key": "val"}` {
			t.Errorf("got %q, want %q", string(got), `{"key": "val"}`)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		got := cleanJSON("")
		if string(got) != "" {
			t.Errorf("got %q, want empty", string(got))
		}
	})
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello world", "hello-world"},
		{"Hello World", "hello-world"},
		{"simple", "simple"},
		{"special chars!@#$", "special-chars"},
		{"multiple   spaces", "multiple-spaces"},
		{"leading-and-trailing---", "leading-and-trailing"},
		{"a", "a"},
		{"", ""},
		{"already-kebab", "already-kebab"},
		{"UPPERCASE", "uppercase"},
		{"with-123-numbers", "with-123-numbers"},
	}
	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMarshalYAML(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		got, err := marshalYAML(map[string]interface{}{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("string values", func(t *testing.T) {
		got, err := marshalYAML(map[string]interface{}{"type": "entity", "status": "draft"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "type: entity") {
			t.Errorf("expected type: entity, got %q", got)
		}
		if !strings.Contains(got, "status: draft") {
			t.Errorf("expected status: draft, got %q", got)
		}
	})

	t.Run("array values", func(t *testing.T) {
		got, err := marshalYAML(map[string]interface{}{"tags": []interface{}{"a", "b"}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "tags: [a, b]") {
			t.Errorf("expected tags: [a, b], got %q", got)
		}
	})

	t.Run("bool values", func(t *testing.T) {
		got, err := marshalYAML(map[string]interface{}{"published": true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "published: true") {
			t.Errorf("expected published: true, got %q", got)
		}
	})

	t.Run("int values", func(t *testing.T) {
		got, err := marshalYAML(map[string]interface{}{"level": 3})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "level: 3") {
			t.Errorf("expected level: 3, got %q", got)
		}
	})

	t.Run("float values", func(t *testing.T) {
		got, err := marshalYAML(map[string]interface{}{"score": 3.5})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "score: 3.5") {
			t.Errorf("expected score: 3.5, got %q", got)
		}
	})
}

func TestExtractKeyTerms(t *testing.T) {
	t.Run("extracts word pairs", func(t *testing.T) {
		content := "This is a long document about machine learning and artificial intelligence"
		terms := extractKeyTerms(content, 5)
		if len(terms) == 0 {
			t.Fatal("expected at least one term")
		}
		if terms[0] != "long document" {
			t.Errorf("terms[0] = %q, want %q", terms[0], "long document")
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		content := "machine learning deep learning neural networks reinforcement learning"
		terms := extractKeyTerms(content, 2)
		if len(terms) > 2 {
			t.Errorf("expected at most 2 terms, got %d", len(terms))
		}
	})

	t.Run("skips headings and short lines", func(t *testing.T) {
		content := "# Heading\nmachine learning\nshort"
		terms := extractKeyTerms(content, 5)
		if len(terms) != 1 {
			t.Errorf("expected 1 term from non-heading line, got %d: %v", len(terms), terms)
		}
	})

	t.Run("deduplicates", func(t *testing.T) {
		content := "machine learning machine learning deep learning"
		terms := extractKeyTerms(content, 5)
		count := 0
		for _, t := range terms {
			if t == "machine learning" {
				count++
			}
		}
		if count > 1 {
			t.Errorf("expected no duplicates, found %d", count)
		}
	})

	t.Run("empty content", func(t *testing.T) {
		terms := extractKeyTerms("", 5)
		if len(terms) != 0 {
			t.Errorf("expected empty, got %v", terms)
		}
	})

	t.Run("requires words > 3 chars", func(t *testing.T) {
		content := "ab cd ef gh ij kl mn op"
		terms := extractKeyTerms(content, 5)
		if len(terms) != 0 {
			t.Errorf("expected no terms from short words, got %v", terms)
		}
	})
}

// ---------------------------------------------------------------------------
// wiki.go
// ---------------------------------------------------------------------------

func TestNewWiki(t *testing.T) {
	t.Run("with WikiConfig", func(t *testing.T) {
		wc := &config.WikiConfig{
			SourceConfig: config.SourceConfig{
				Path: "/tmp/test-wiki",
			},
			IndexFile:  "index.md",
			LogFile:    "log.md",
			GraphLinks: true,
			WikiDir:    "wiki",
			RawDir:     "raw",
		}
		w, err := NewWiki("test", "/tmp/test-wiki", wc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.Name != "test" {
			t.Errorf("Name = %q, want %q", w.Name, "test")
		}
		if w.Path != "/tmp/test-wiki" {
			t.Errorf("Path = %q, want %q", w.Path, "/tmp/test-wiki")
		}
		if w.WikiPath != "/tmp/test-wiki/wiki" {
			t.Errorf("WikiPath = %q, want %q", w.WikiPath, "/tmp/test-wiki/wiki")
		}
		if w.RawPath != "/tmp/test-wiki/raw" {
			t.Errorf("RawPath = %q, want %q", w.RawPath, "/tmp/test-wiki/raw")
		}
		if w.Config.IndexFile != "index.md" {
			t.Errorf("IndexFile = %q", w.Config.IndexFile)
		}
		if w.WikiConfig.IndexFile != "index.md" {
			t.Errorf("IndexFile = %q", w.WikiConfig.IndexFile)
		}
	})

	t.Run("with custom wikiDir/rawDir", func(t *testing.T) {
		wc := &config.WikiConfig{
			SourceConfig: config.SourceConfig{
				Path: "/tmp/test-wiki",
			},
			WikiDir: "pages",
			RawDir:  "inputs",
		}
		w, err := NewWiki("test", "/tmp/test-wiki", wc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.WikiPath != "/tmp/test-wiki/pages" {
			t.Errorf("WikiPath = %q, want %q", w.WikiPath, "/tmp/test-wiki/pages")
		}
		if w.RawPath != "/tmp/test-wiki/inputs" {
			t.Errorf("RawPath = %q, want %q", w.RawPath, "/tmp/test-wiki/inputs")
		}
	})
}

func TestWikiIndexFilePath(t *testing.T) {
	wc := &config.WikiConfig{SourceConfig: config.SourceConfig{Path: "/tmp/test-wiki"}, WikiDir: "wiki", RawDir: "raw", IndexFile: "index.md", LogFile: "log.md"}
	w, _ := NewWiki("test", "/tmp/test-wiki", wc)
	got := w.IndexFilePath()
	want := "/tmp/test-wiki/wiki/index.md"
	if got != want {
		t.Errorf("IndexFilePath = %q, want %q", got, want)
	}
}

func TestWikiLogFilePath(t *testing.T) {
	wc := &config.WikiConfig{SourceConfig: config.SourceConfig{Path: "/tmp/test-wiki"}, WikiDir: "wiki", RawDir: "raw", IndexFile: "index.md", LogFile: "log.md"}
	w, _ := NewWiki("test", "/tmp/test-wiki", wc)
	got := w.LogFilePath()
	want := "/tmp/test-wiki/wiki/log.md"
	if got != want {
		t.Errorf("LogFilePath = %q, want %q", got, want)
	}
}

func TestWikiInit(t *testing.T) {
	tmpDir := t.TempDir()
	wc := &config.WikiConfig{SourceConfig: config.SourceConfig{Path: tmpDir}, WikiDir: "wiki", RawDir: "raw", IndexFile: "index.md", LogFile: "log.md", OkfVersion: "0.1"}
	w, err := NewWiki("test", tmpDir, wc)
	if err != nil {
		t.Fatalf("NewWiki error: %v", err)
	}

	err = w.Init()
	if err != nil {
		t.Fatalf("Init error: %v", err)
	}

	expectedDirs := []string{
		"raw",
		"wiki",
	}
	for _, dir := range expectedDirs {
		p := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("expected directory %s to exist", p)
		}
	}

	indexPath := filepath.Join(tmpDir, "wiki", "index.md")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Errorf("expected index file %s to exist", indexPath)
	}

	logPath := filepath.Join(tmpDir, "wiki", "log.md")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("expected log file %s to exist", logPath)
	}

	schemaPath := filepath.Join(tmpDir, "WIKI_SCHEMA.md")
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		t.Errorf("expected schema file %s to exist", schemaPath)
	}
}

func TestInitWiki(t *testing.T) {
	tmpDir := t.TempDir()
	wc := &config.WikiConfig{SourceConfig: config.SourceConfig{Path: tmpDir}, WikiDir: "wiki", RawDir: "raw", IndexFile: "index.md", LogFile: "log.md", OkfVersion: "0.1"}

	err := InitWiki("test", tmpDir, wc)
	if err != nil {
		t.Fatalf("InitWiki error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "raw")); os.IsNotExist(err) {
		t.Error("expected raw directory to exist")
	}
}

// ---------------------------------------------------------------------------
// agent.go — NewAgent
// ---------------------------------------------------------------------------

func TestNewAgent(t *testing.T) {
	wc := &config.WikiConfig{SourceConfig: config.SourceConfig{Path: "/tmp/test-wiki"}, WikiDir: "wiki", RawDir: "raw", IndexFile: "index.md", LogFile: "log.md"}
	w, _ := NewWiki("test", "/tmp/test-wiki", wc)

	a := NewAgent(w, &config.Config{}, nil, nil)
	if a == nil {
		t.Fatal("NewAgent returned nil")
	}
	if a.wiki != w {
		t.Error("wiki not set")
	}
	if a.schema == "" {
		t.Error("schema should not be empty")
	}
	if a.indexCache == nil {
		t.Error("indexCache should be initialized")
	}
}

// ---------------------------------------------------------------------------
// agent_prompts.go
// ---------------------------------------------------------------------------

func TestSchemaPrompt(t *testing.T) {
	content := SchemaPrompt()
	if content == "" {
		t.Fatal("SchemaPrompt() returned empty string")
	}
	if !strings.Contains(content, "Wiki Schema") {
		t.Errorf("expected Wiki Schema heading, got %q", content[:50])
	}
	if !strings.Contains(content, "Directory Structure") {
		t.Errorf("expected Directory Structure section")
	}
}

func TestIngestSystemPrompt(t *testing.T) {
	prompt := IngestSystemPrompt("## Existing Pages\n- [[foo]]")
	if prompt == "" {
		t.Fatal("IngestSystemPrompt() returned empty")
	}
	if !strings.Contains(prompt, "disciplined wiki maintainer") {
		t.Errorf("expected opening instruction, got %q", prompt[:50])
	}
	if !strings.Contains(prompt, "## Existing Wiki Pages") {
		t.Errorf("expected Existing Wiki Pages section")
	}
	if !strings.Contains(prompt, "## Existing Pages") {
		t.Errorf("expected existing pages content to be included")
	}
	if !strings.Contains(prompt, "Wiki Schema") {
		t.Errorf("expected schema to be included")
	}
}

func TestQuerySystemPrompt(t *testing.T) {
	prompt := QuerySystemPrompt("## Relevant Pages\n- [foo](/wiki/foo.md)")
	if prompt == "" {
		t.Fatal("QuerySystemPrompt() returned empty")
	}
	if !strings.Contains(prompt, "research assistant") {
		t.Errorf("expected assistant description, got %q", prompt[:50])
	}
	if !strings.Contains(prompt, "## Relevant Wiki Pages") {
		t.Errorf("expected Relevant Wiki Pages section")
	}
	if !strings.Contains(prompt, "## Relevant Pages") {
		t.Errorf("expected relevant pages content to be included")
	}
	if !strings.Contains(prompt, "standard markdown links") {
		t.Errorf("expected citation instruction")
	}
}
