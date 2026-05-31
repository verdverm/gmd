package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddCollection(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "project")
	gmdDir := filepath.Join(root, ".gmd")
	if err := os.MkdirAll(gmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(gmdDir, "config.cue")
	initialConfig := `package gmd

Config: {
	collections: docs: {
		path:    "."
		pattern: "**/*.md"
	}
}
`
	if err := os.WriteFile(cfgPath, []byte(initialConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		ProjectRoot: root,
		Collections: map[string]CollectionConfig{
			"docs": {Path: ".", Pattern: "**/*.md"},
		},
	}

	t.Run("add new collection", func(t *testing.T) {
		err := AddCollection(cfg, "notes", "/path/to/notes", "*.md")
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := cfg.Collections["notes"]; !ok {
			t.Fatal("notes collection should exist in memory")
		}
		if cfg.Collections["notes"].Path != "/path/to/notes" {
			t.Errorf("path = %q, want %q", cfg.Collections["notes"].Path, "/path/to/notes")
		}

		data, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		if !contains(content, "notes") {
			t.Errorf("config should contain 'notes', got:\n%s", content)
		}
		if !contains(content, "/path/to/notes") {
			t.Errorf("config should contain '/path/to/notes', got:\n%s", content)
		}
	})

	t.Run("add duplicate collection returns error", func(t *testing.T) {
		err := AddCollection(cfg, "docs", ".", "*.md")
		if err == nil {
			t.Fatal("expected error for duplicate collection")
		}
	})

	t.Run("add to non-existent config file", func(t *testing.T) {
		cfg2 := &Config{ProjectRoot: "/nonexistent"}
		err := AddCollection(cfg2, "test", ".", "*.md")
		if err == nil {
			t.Fatal("expected error for nonexistent config")
		}
	})
}

func TestRemoveCollection(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "project")
	gmdDir := filepath.Join(root, ".gmd")
	if err := os.MkdirAll(gmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(gmdDir, "config.cue")
	initialConfig := `package gmd

Config: {
	collections: docs: {
		path:    "."
		pattern: "*.md"
	}
}
`
	if err := os.WriteFile(cfgPath, []byte(initialConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		ProjectRoot: root,
		Collections: map[string]CollectionConfig{
			"docs": {Path: ".", Pattern: "*.md"},
		},
	}

	t.Run("remove existing collection", func(t *testing.T) {
		err := RemoveCollection(cfg, "docs")
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := cfg.Collections["docs"]; ok {
			t.Fatal("docs collection should be removed from memory")
		}

		data, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		if contains(string(data), "docs") {
			t.Errorf("config should not contain 'docs', got:\n%s", data)
		}
	})

	t.Run("remove non-existent collection", func(t *testing.T) {
		cfg2 := &Config{
			ProjectRoot: root,
			Collections: map[string]CollectionConfig{},
		}
		err := RemoveCollection(cfg2, "nonexistent")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestRenameCollection(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "project")
	gmdDir := filepath.Join(root, ".gmd")
	if err := os.MkdirAll(gmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(gmdDir, "config.cue")
	initialConfig := `package gmd

Config: {
	collections: docs: {
		path:    "."
		pattern: "*.md"
	}
}
`
	if err := os.WriteFile(cfgPath, []byte(initialConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		ProjectRoot: root,
		Collections: map[string]CollectionConfig{
			"docs": {Path: ".", Pattern: "*.md"},
		},
	}

	err := RenameCollection(cfg, "docs", "documents")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Collections["docs"]; ok {
		t.Fatal("old name should be removed")
	}
	if _, ok := cfg.Collections["documents"]; !ok {
		t.Fatal("new name should exist")
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if contains(content, "docs") {
		t.Errorf("old name should not be in config, got:\n%s", content)
	}
	if !contains(content, "documents") {
		t.Errorf("new name should be in config, got:\n%s", content)
	}
}

func TestSetCollectionPattern(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "project")
	gmdDir := filepath.Join(root, ".gmd")
	if err := os.MkdirAll(gmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(gmdDir, "config.cue")
	initialConfig := `package gmd

Config: {
	collections: docs: {
		path:    "."
		pattern: "*.md"
	}
}
`
	if err := os.WriteFile(cfgPath, []byte(initialConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		ProjectRoot: root,
		Collections: map[string]CollectionConfig{
			"docs": {Path: ".", Pattern: "*.md"},
		},
	}

	err := SetCollectionPattern(cfg, "docs", "**/*.txt")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Collections["docs"].Pattern != "**/*.txt" {
		t.Errorf("pattern = %q, want %q", cfg.Collections["docs"].Pattern, "**/*.txt")
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(data), "**/*.txt") {
		t.Errorf("config should contain new pattern, got:\n%s", data)
	}
}

func TestIgnorePattern(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "project")
	gmdDir := filepath.Join(root, ".gmd")
	if err := os.MkdirAll(gmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(gmdDir, "config.cue")
	initialConfig := `package gmd

Config: {
	collections: docs: {
		path:    "."
		pattern: "*.md"
	}
}
`
	if err := os.WriteFile(cfgPath, []byte(initialConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		ProjectRoot: root,
		Collections: map[string]CollectionConfig{
			"docs": {Path: ".", Pattern: "*.md"},
		},
	}

	t.Run("add ignore pattern", func(t *testing.T) {
		err := AddIgnorePattern(cfg, "docs", "node_modules/**")
		if err != nil {
			t.Fatal(err)
		}
		if len(cfg.Collections["docs"].Ignore) != 1 {
			t.Fatalf("expected 1 ignore, got %d", len(cfg.Collections["docs"].Ignore))
		}
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(string(data), "node_modules/**") {
			t.Errorf("config should contain ignore pattern, got:\n%s", data)
		}
	})

	t.Run("add duplicate ignore pattern is no-op", func(t *testing.T) {
		err := AddIgnorePattern(cfg, "docs", "node_modules/**")
		if err != nil {
			t.Fatal(err)
		}
		if len(cfg.Collections["docs"].Ignore) != 1 {
			t.Errorf("duplicate should not add, got %d", len(cfg.Collections["docs"].Ignore))
		}
	})

	t.Run("remove ignore pattern", func(t *testing.T) {
		err := RemoveIgnorePattern(cfg, "docs", "node_modules/**")
		if err != nil {
			t.Fatal(err)
		}
		if len(cfg.Collections["docs"].Ignore) != 0 {
			t.Errorf("expected 0 ignore after remove, got %d", len(cfg.Collections["docs"].Ignore))
		}
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		if contains(string(data), "node_modules") {
			t.Errorf("config should not contain removed ignore, got:\n%s", data)
		}
	})
}

func TestContextDoc(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "project")
	gmdDir := filepath.Join(root, ".gmd")
	if err := os.MkdirAll(gmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(gmdDir, "config.cue")
	initialConfig := `package gmd

Config: {
	collections: docs: {
		path:    "."
		pattern: "*.md"
	}
}
`
	if err := os.WriteFile(cfgPath, []byte(initialConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		ProjectRoot: root,
		Collections: map[string]CollectionConfig{
			"docs": {Path: ".", Pattern: "*.md"},
		},
	}

	t.Run("add context doc", func(t *testing.T) {
		err := AddContextDoc(cfg, "docs", "README.md")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Collections["docs"].Context != "README.md" {
			t.Errorf("context = %q, want %q", cfg.Collections["docs"].Context, "README.md")
		}
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(string(data), "README.md") {
			t.Errorf("config should contain context, got:\n%s", data)
		}
	})

	t.Run("list context docs", func(t *testing.T) {
		ctxs := ListContextDocs(cfg)
		if len(ctxs) != 1 {
			t.Fatalf("expected 1 context doc, got %d", len(ctxs))
		}
		if ctxs["docs"] != "README.md" {
			t.Errorf("context path = %q, want %q", ctxs["docs"], "README.md")
		}
	})

	t.Run("remove context doc", func(t *testing.T) {
		err := RemoveContextDoc(cfg, "docs")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Collections["docs"].Context != "" {
			t.Errorf("context should be empty, got %q", cfg.Collections["docs"].Context)
		}
	})

	t.Run("list context docs after remove", func(t *testing.T) {
		ctxs := ListContextDocs(cfg)
		if len(ctxs) != 0 {
			t.Errorf("expected 0 after remove, got %d", len(ctxs))
		}
	})
}

func TestProjectConfigPath(t *testing.T) {
	got := ProjectConfigPath("/my/project")
	want := "/my/project/.gmd/config.cue"
	if got != want {
		t.Errorf("ProjectConfigPath() = %q, want %q", got, want)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
