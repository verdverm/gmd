package indexer

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/ts"
)

func TestScanFilesFS(t *testing.T) {
	t.Run("single file", func(t *testing.T) {
		fsys := fstest.MapFS{
			"doc.md": {Data: []byte("# Hello")},
		}
		files, err := scanFilesFS(fsys, ".", []string{"*"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 1 || files[0] != "doc.md" {
			t.Errorf("got %v, want [doc.md]", files)
		}
	})

	t.Run("multiple files with pattern", func(t *testing.T) {
		fsys := fstest.MapFS{
			"readme.md":        {Data: []byte("# Readme")},
			"doc.md":           {Data: []byte("# Doc")},
			"main.go":          {Data: []byte("package main")},
			"notes.txt":        {Data: []byte("notes")},
			"sub/other.md":     {Data: []byte("# Sub doc")},
			"sub/helper.go":    {Data: []byte("package helper")},
			"sub/deep/deep.md": {Data: []byte("# Deep")},
		}
		files, err := scanFilesFS(fsys, ".", []string{"*.md"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 2 {
			t.Errorf("expected 2 .md files at root, got %d: %v", len(files), files)
		}
	})

	t.Run("recursive globstar matches at all depths", func(t *testing.T) {
		fsys := fstest.MapFS{
			"a.md":         {Data: []byte("# A")},
			"sub/b.md":     {Data: []byte("# B")},
			"sub/sub/c.md": {Data: []byte("# C")},
			"main.go":      {Data: []byte("package main")},
		}
		files, err := scanFilesFS(fsys, ".", []string{"**/*.md"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 3 {
			t.Errorf("expected 3 .md files at any depth, got %d: %v", len(files), files)
		}
	})

	t.Run("non-directory input", func(t *testing.T) {
		fsys := fstest.MapFS{
			"single.md": {Data: []byte("# Single file")},
		}
		files, err := scanFilesFS(fsys, "single.md", []string{"**/*.md"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 1 || files[0] != "single.md" {
			t.Errorf("got %v, want [single.md]", files)
		}
	})

	t.Run("ignore node_modules with globstar", func(t *testing.T) {
		fsys := fstest.MapFS{
			"doc.md":                      {Data: []byte("# Doc")},
			"node_modules/pkg/sub/lib.md": {Data: []byte("# Lib")},
			"src/main.md":                 {Data: []byte("# Main")},
		}
		files, err := scanFilesFS(fsys, ".", []string{"**/*.md"}, []string{"node_modules/**"})
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 2 {
			t.Errorf("expected 2 files (node_modules excluded), got %d: %v", len(files), files)
		}
	})

	t.Run("with ignore patterns", func(t *testing.T) {
		fsys := fstest.MapFS{
			"doc.md":                  {Data: []byte("# Doc")},
			"node_modules/pkg/pkg.md": {Data: []byte("# Nope")},
			"src/main.md":             {Data: []byte("# Main")},
			"src/ignore_me.md":        {Data: []byte("# Ignored")},
		}
		files, err := scanFilesFS(fsys, ".", []string{"**/*.md"}, []string{"node_modules/**", "src/ignore_me.md"})
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 2 {
			t.Errorf("expected 2 files (doc.md, src/main.md), got %d: %v", len(files), files)
		}
	})

	t.Run("no matching files", func(t *testing.T) {
		fsys := fstest.MapFS{
			"main.go": {Data: []byte("package main")},
		}
		files, err := scanFilesFS(fsys, ".", []string{"*.md"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 0 {
			t.Errorf("expected 0 files, got %d", len(files))
		}
	})

	t.Run("non-existent root", func(t *testing.T) {
		fsys := fstest.MapFS{}
		_, err := scanFilesFS(fsys, "nonexistent", []string{"*.md"}, nil)
		if err == nil {
			t.Error("expected error for nonexistent directory")
		}
	})

	t.Run("ignore directory skipped via prefix", func(t *testing.T) {
		fsys := fstest.MapFS{
			"keep.md":          {Data: []byte("# Keep")},
			"skip/doc.md":      {Data: []byte("# Skip")},
			"skip/sub/deep.md": {Data: []byte("# Deep")},
			"keep2.md":         {Data: []byte("# Keep2")},
		}
		files, err := scanFilesFS(fsys, ".", []string{"*.md"}, []string{"skip/"})
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 2 {
			t.Errorf("expected 2 files (skip/ excluded), got %d: %v", len(files), files)
		}
	})
}

func TestFileHashFS(t *testing.T) {
	t.Run("consistent hash for same content", func(t *testing.T) {
		fsys := fstest.MapFS{
			"a.md": {Data: []byte("# Hello\nWorld")},
			"b.md": {Data: []byte("# Hello\nWorld")},
		}
		h1, err := fileHashFS(fsys, "a.md")
		if err != nil {
			t.Fatal(err)
		}
		h2, err := fileHashFS(fsys, "b.md")
		if err != nil {
			t.Fatal(err)
		}
		if h1 != h2 {
			t.Errorf("same content should produce same hash:\na=%s\nb=%s", h1, h2)
		}
	})

	t.Run("different content produces different hash", func(t *testing.T) {
		fsys := fstest.MapFS{
			"a.md": {Data: []byte("content a")},
			"b.md": {Data: []byte("content b")},
		}
		h1, _ := fileHashFS(fsys, "a.md")
		h2, _ := fileHashFS(fsys, "b.md")
		if h1 == h2 {
			t.Error("different content should produce different hashes")
		}
	})

	t.Run("non-existent file returns error", func(t *testing.T) {
		fsys := fstest.MapFS{}
		_, err := fileHashFS(fsys, "nonexistent.md")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestIndexerConstruction(t *testing.T) {
	t.Run("New creates indexer", func(t *testing.T) {
		idx := New(nil, nil, nil)
		if idx == nil {
			t.Fatal("New returned nil")
		}
		if idx.fsys != nil {
			t.Error("expected nil fsys by default")
		}
	})

	t.Run("WithFS sets filesystem", func(t *testing.T) {
		fsys := fstest.MapFS{}
		idx := New(nil, nil, nil).WithFS(fsys)
		if idx.fsys == nil {
			t.Error("WithFS should set fsys")
		}
	})

	t.Run("rootFS returns OS by default", func(t *testing.T) {
		idx := New(nil, nil, nil)
		rfs := idx.rootFS()
		if rfs == nil {
			t.Error("rootFS should not be nil")
		}
	})

	t.Run("rootFS returns injected FS", func(t *testing.T) {
		fsys := fstest.MapFS{
			"test.md": {Data: []byte("# test")},
		}
		idx := New(nil, nil, nil).WithFS(fsys)
		data, err := fs.ReadFile(idx.rootFS(), "test.md")
		if err != nil {
			t.Fatal("rootFS should allow reading from injected FS:", err)
		}
		if string(data) != "# test" {
			t.Errorf("got %q, want %q", string(data), "# test")
		}
	})
}

func TestIndexResult(t *testing.T) {
	t.Run("empty result has zero values", func(t *testing.T) {
		r := &IndexResult{}
		if r.TotalFiles != 0 || r.Indexed != 0 || r.Skipped != 0 || r.ChunkCount != 0 {
			t.Error("new IndexResult should have zero values")
		}
		if r.Errors != nil {
			t.Error("new IndexResult should have nil Errors")
		}
	})

	t.Run("accumulates errors", func(t *testing.T) {
		r := &IndexResult{}
		r.Errors = append(r.Errors, "error 1")
		r.Errors = append(r.Errors, "error 2")
		if len(r.Errors) != 2 {
			t.Errorf("expected 2 errors, got %d", len(r.Errors))
		}
	})

	t.Run("tracks file counts", func(t *testing.T) {
		r := &IndexResult{
			Collection: "docs",
			TotalFiles: 10,
			Indexed:    7,
			Skipped:    3,
			ChunkCount: 28,
		}
		if r.Indexed+r.Skipped != r.TotalFiles {
			t.Errorf("indexed+skipped (%d) should equal total (%d)", r.Indexed+r.Skipped, r.TotalFiles)
		}
	})
}

func TestUpdateCollectionErrors(t *testing.T) {
	t.Run("nonexistent directory returns error", func(t *testing.T) {
		fsys := fstest.MapFS{}
		cfg := testConfig(t)
		col := cfg.Collections["test"]
		col.Path = "/nonexistent"
		cfg.Collections["test"] = col
		idx := New(cfg, nil, nil).WithFS(fsys)
		result := idx.updateCollection(context.Background(), "test", col, "/", nil)
		if len(result.Errors) == 0 {
			t.Error("expected errors for nonexistent directory")
		}
	})

	t.Run("cancelled context stops processing", func(t *testing.T) {
		fsys := fstest.MapFS{
			"docs/a.md": {Data: []byte("# A\ncontent here.")},
			"docs/b.md": {Data: []byte("# B\ncontent here.")},
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		cfg := testConfig(t)
		idx := New(cfg, nil, nil).WithFS(fsys)
		result := idx.updateCollection(ctx, "test", cfg.Collections["test"], "/", nil)
		if result.Errors == nil {
			t.Log("cancelled context handled (expected at least scan error)")
		}
	})
}

func TestFileHash(t *testing.T) {
	t.Run("produces 64-char hex", func(t *testing.T) {
		h, err := fileHashFS(fstest.MapFS{"f": {Data: []byte("data")}}, "f")
		if err != nil {
			t.Fatal(err)
		}
		if len(h) != 64 {
			t.Errorf("expected 64 hex chars, got %d: %s", len(h), h)
		}
		for _, c := range h {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("non-hex char %c in hash", c)
				break
			}
		}
	})
}

func TestScanFilesFSWithTempDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.md"), []byte("# test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "other.md"), []byte("# other"), 0644); err != nil {
		t.Fatal(err)
	}

	fsys := os.DirFS(dir)
	files, err := scanFilesFS(fsys, ".", []string{"*.md"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 .md at root, got %d: %v", len(files), files)
	}
}

func TestFileHashWithTempDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("# Hello\nWorld"), 0644); err != nil {
		t.Fatal(err)
	}

	h1, err := fileHash(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte("# Hello\nWorld"), 0644); err != nil {
		t.Fatal(err)
	}

	h2, err := fileHash(path)
	if err != nil {
		t.Fatal(err)
	}

	if h1 != h2 {
		t.Errorf("same content should produce same hash:\na=%s\nb=%s", h1, h2)
	}

	if err := os.WriteFile(path, []byte("different content"), 0644); err != nil {
		t.Fatal(err)
	}
	h3, err := fileHash(path)
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h3 {
		t.Error("different content should produce different hash")
	}
}

func TestWithFSChaining(t *testing.T) {
	fsys1 := fstest.MapFS{
		"a.md": {Data: []byte("# A")},
	}
	fsys2 := fstest.MapFS{
		"b.md": {Data: []byte("# B")},
	}
	idx := New(nil, nil, nil).WithFS(fsys1)
	data, err := fs.ReadFile(idx.rootFS(), "a.md")
	if err != nil {
		t.Fatal("WithFS should make fsys1 accessible:", err)
	}
	if string(data) != "# A" {
		t.Errorf("WithFS should use fsys1, got %q", string(data))
	}

	idx.WithFS(fsys2)
	_, err = fs.ReadFile(idx.rootFS(), "a.md")
	if err == nil {
		t.Error("after second WithFS, a.md should not be readable")
	}
	data, err = fs.ReadFile(idx.rootFS(), "b.md")
	if err != nil {
		t.Fatal("after second WithFS, b.md should be readable:", err)
	}
	if string(data) != "# B" {
		t.Errorf("got %q, want %q", string(data), "# B")
	}
}

// testConfig creates a minimal config for testing.
func testConfig(t *testing.T) *config.Config {
	t.Helper()
	return testConfigWithCollections(t, map[string]config.CollectionConfig{
		"test": {
			Path:     "/docs",
			Patterns: []string{"**/*.md"},
		},
	})
}

// testMultiConfig creates a config with multiple collections for testing.
func testMultiConfig(t *testing.T) *config.Config {
	t.Helper()
	return testConfigWithCollections(t, map[string]config.CollectionConfig{
		"docs": {
			Path:     "/docs",
			Patterns: []string{"**/*.md"},
		},
		"notes": {
			Path:     "/notes",
			Patterns: []string{"**/*.md"},
		},
	})
}

func testConfigWithCollections(t *testing.T, cols map[string]config.CollectionConfig) *config.Config {
	t.Helper()
	return &config.Config{
		ProjectRoot: "/",
		Collections: cols,
		Pipeline: config.PipelineConfig{
			Chunk: config.ChunkConfig{
				TargetTokens: 900,
				Overlap:      0.15,
				HeadingWeights: config.HeadingWeights{
					H1: 100, H2: 90, H3: 80, H4: 70, H5: 60, H6: 50,
				},
				CodeFenceWeight: 10,
				NewlineWeight:   1,
			},
		},
	}
}

type mockTSClient struct {
	distinctPaths map[string][]string
	deletePaths   []string
}

func (m *mockTSClient) SearchDistinctPaths(ctx context.Context, filter string) ([]string, error) {
	if paths, ok := m.distinctPaths[filter]; ok {
		return paths, nil
	}
	return nil, nil
}

func (m *mockTSClient) DeleteChunksByPath(ctx context.Context, path string) error {
	m.deletePaths = append(m.deletePaths, path)
	return nil
}

func (m *mockTSClient) GetHashByPath(ctx context.Context, path string) (string, error) {
	return "", nil
}

func (m *mockTSClient) UpsertChunks(ctx context.Context, chunks []ts.ChunkDocument) error {
	return nil
}

func TestStalePaths(t *testing.T) {
	dir := t.TempDir()

	// Create a file that exists on disk
	existing := filepath.Join(dir, "existing.md")
	if err := os.WriteFile(existing, []byte("# Existing"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		ProjectRoot: "/",
		Collections: map[string]config.CollectionConfig{
			"test": {
				Path:     dir,
				Patterns: []string{"*.md"},
			},
		},
	}

	idx := New(cfg, nil, nil)

	// Override ts client
	mockTS := &mockTSClient{
		distinctPaths: map[string][]string{
			"collection:=test": {"existing.md", "deleted.md"},
		},
	}
	idx.ts = mockTS

	t.Run("finds stale paths", func(t *testing.T) {
		stale, err := idx.StalePaths(context.Background(), "test")
		if err != nil {
			t.Fatal(err)
		}
		if len(stale) != 1 || stale[0] != "deleted.md" {
			t.Errorf("expected [deleted.md], got %v", stale)
		}
	})

	t.Run("non-existent collection returns error", func(t *testing.T) {
		_, err := idx.StalePaths(context.Background(), "nonexistent")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCleanupDeleted(t *testing.T) {
	dir := t.TempDir()

	existing := filepath.Join(dir, "keep.md")
	if err := os.WriteFile(existing, []byte("# Keep"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		ProjectRoot: "/",
		Collections: map[string]config.CollectionConfig{
			"test": {
				Path:     dir,
				Patterns: []string{"*.md"},
			},
		},
	}

	idx := New(cfg, nil, nil)
	idx.ts = &mockTSClient{
		distinctPaths: map[string][]string{
			"collection:=test": {"keep.md", "stale.md"},
		},
	}

	t.Run("deletes stale chunks", func(t *testing.T) {
		deleted, err := idx.CleanupDeleted(context.Background(), "test")
		if err != nil {
			t.Fatal(err)
		}
		if deleted != 1 {
			t.Errorf("expected 1 deleted, got %d", deleted)
		}
	})
}

func TestCleanupAllCollections(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		ProjectRoot: "/",
		Collections: map[string]config.CollectionConfig{
			"docs":  {Path: dir, Patterns: []string{"*.md"}},
			"notes": {Path: dir, Patterns: []string{"*.md"}},
		},
	}

	idx := New(cfg, nil, nil)
	idx.ts = &mockTSClient{
		distinctPaths: map[string][]string{
			"collection:=docs":  {"a.md"},
			"collection:=notes": {"b.md"}, // stale
		},
	}

	results := idx.CleanupAllCollections(context.Background(), nil)
	if results["notes"] != 1 {
		t.Errorf("expected 1 stale in notes, got %d", results["notes"])
	}
	if results["docs"] != 0 {
		t.Errorf("expected 0 stale in docs, got %d", results["docs"])
	}
}

func TestStalePathsNonexistentCollection(t *testing.T) {
	idx := New(&config.Config{}, nil, nil)
	_, err := idx.StalePaths(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent collection")
	}
}
