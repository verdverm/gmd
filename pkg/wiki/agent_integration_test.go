//go:build integration

package wiki

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/verdverm/gmd/pkg/config"
)

func newTestWikiAgent(t *testing.T) (*Wiki, *Agent) {
	t.Helper()
	tmpDir := t.TempDir()
	w, err := NewWiki("test-wiki", tmpDir, &config.WikiConfig{
		SourceConfig: config.SourceConfig{Path: tmpDir},
		WikiDir:      "wiki",
		RawDir:       "raw",
		IndexFile:    "_index.md",
		LogFile:      "_log.md",
		GraphLinks:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Init(); err != nil {
		t.Fatal(err)
	}
	agent := NewAgent(w, nil, nil, nil)
	return w, agent
}

func TestIntegrationReadWikiPage_Existing(t *testing.T) {
	_, agent := newTestWikiAgent(t)

	// readWikiPage uses a.wiki.Path, so files need to be at Path/wiki/...
	// which is WikiPath
	pagePath := filepath.Join(agent.wiki.WikiPath, "entities", "test.md")
	os.MkdirAll(filepath.Dir(pagePath), 0755)
	content := "---\ntype: entity\n---\n\n# Test Page\n\nBody text.\n"
	os.WriteFile(pagePath, []byte(content), 0644)

	// readWikiPage joins a.wiki.Path with the provided path
	// So we need to include "wiki/" prefix for it to resolve correctly
	result, err := agent.readWikiPage(filepath.Join("wiki", "entities", "test.md"))
	if err != nil {
		t.Fatalf("readWikiPage error: %v", err)
	}
	expected := "# Test Page\n\nBody text.\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestIntegrationReadWikiPage_NonExistent(t *testing.T) {
	_, agent := newTestWikiAgent(t)
	_, err := agent.readWikiPage("wiki/entities/nonexistent.md")
	if err == nil {
		t.Error("expected error for non-existent page")
	}
}

func TestIntegrationReadWikiPage_NoFrontmatter(t *testing.T) {
	_, agent := newTestWikiAgent(t)
	pagePath := filepath.Join(agent.wiki.WikiPath, "plain.md")
	os.WriteFile(pagePath, []byte("# Just content\n"), 0644)

	result, err := agent.readWikiPage("wiki/plain.md")
	if err != nil {
		t.Fatalf("readWikiPage error: %v", err)
	}
	if result != "# Just content\n" {
		t.Errorf("got %q, want %q", result, "# Just content\n")
	}
}

func TestIntegrationLoadIndexContext_Exists(t *testing.T) {
	_, agent := newTestWikiAgent(t)
	content := agent.loadIndexContext()
	if content == "" {
		t.Error("expected non-empty index content")
	}
	if !strings.Contains(content, "# Wiki Index") {
		t.Errorf("expected index header in content, got %q", content)
	}
}

func TestIntegrationLoadIndexContext_MissingFile(t *testing.T) {
	w, agent := newTestWikiAgent(t)
	os.Remove(w.IndexFilePath())

	content := agent.loadIndexContext()
	if content != "" {
		t.Errorf("expected empty content for missing index, got %q", content)
	}
}

func TestIntegrationReadSource_FromRaw(t *testing.T) {
	tmpDir := t.TempDir()
	rawDir := filepath.Join(tmpDir, "raw")
	os.MkdirAll(rawDir, 0755)
	os.WriteFile(filepath.Join(rawDir, "source.txt"), []byte("raw source content"), 0644)

	content, err := readSource("source.txt", rawDir)
	if err != nil {
		t.Fatalf("readSource error: %v", err)
	}
	if content != "raw source content" {
		t.Errorf("got %q, want %q", content, "raw source content")
	}
}

func TestIntegrationReadSource_AbsolutePath(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "external.txt")
	os.WriteFile(tmpFile, []byte("external content"), 0644)

	content, err := readSource(tmpFile, t.TempDir())
	if err != nil {
		t.Fatalf("readSource error: %v", err)
	}
	if content != "external content" {
		t.Errorf("got %q, want %q", content, "external content")
	}
}

func TestIntegrationReadSource_URLReturnsError(t *testing.T) {
	_, err := readSource("https://example.com/source", t.TempDir())
	if err == nil {
		t.Fatal("expected error for URL source")
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestIntegrationReadSource_NonExistent(t *testing.T) {
	_, err := readSource("nonexistent.txt", t.TempDir())
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestIntegrationUpdateWikiPage_CreateNew(t *testing.T) {
	_, agent := newTestWikiAgent(t)

	// updateWikiPage with non-existent page falls back to createWikiPage
	action := IngestAction{
		Name:    "New Page",
		Page:    "entities/new-page.md",
		Action:  "create",
		Content: "# New Page\n\nContent.",
		Frontmatter: map[string]interface{}{
			"type": "entity",
		},
	}
	if err := agent.updateWikiPage(action); err != nil {
		t.Fatalf("updateWikiPage (fallback create) error: %v", err)
	}

	// Verify the file was created
	fullPath := filepath.Join(agent.wiki.WikiPath, action.Page)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Fatal("expected file to be created")
	}
}

func TestIntegrationUpdateWikiPage_AppendContent(t *testing.T) {
	_, agent := newTestWikiAgent(t)

	// First create a page
	action := IngestAction{
		Name:    "Page",
		Page:    "entities/append-test.md",
		Action:  "create",
		Content: "# Original\n\nOriginal content.\n",
	}
	if err := agent.createWikiPage(action); err != nil {
		t.Fatalf("createWikiPage error: %v", err)
	}

	// Now append to it
	updateAction := IngestAction{
		Name:          "Page",
		Page:          "entities/append-test.md",
		Action:        "update",
		AppendContent: "Appended content.",
	}
	if err := agent.updateWikiPage(updateAction); err != nil {
		t.Fatalf("updateWikiPage append error: %v", err)
	}

	fullPath := filepath.Join(agent.wiki.WikiPath, action.Page)
	data, _ := os.ReadFile(fullPath)
	content := string(data)
	if !strings.Contains(content, "Appended content.") {
		t.Errorf("expected appended content in file, got: %s", content)
	}
}

func TestIntegrationUpdateWikiPage_MergeSection(t *testing.T) {
	_, agent := newTestWikiAgent(t)

	action := IngestAction{
		Name:    "Page",
		Page:    "entities/merge-test.md",
		Action:  "create",
		Content: "# Header\n\nExisting section.\n\n## Another\n\nMore content.\n",
	}
	if err := agent.createWikiPage(action); err != nil {
		t.Fatalf("createWikiPage error: %v", err)
	}

	updateAction := IngestAction{
		Name:         "Page",
		Page:         "entities/merge-test.md",
		Action:       "merge",
		MergeSection: "## Another",
		Content:      "Inserted before Another.\n\n",
	}
	if err := agent.updateWikiPage(updateAction); err != nil {
		t.Fatalf("updateWikiPage merge error: %v", err)
	}

	fullPath := filepath.Join(agent.wiki.WikiPath, action.Page)
	data, _ := os.ReadFile(fullPath)
	content := string(data)
	if !strings.Contains(content, "Inserted before Another.") {
		t.Errorf("expected merged content, got: %s", content)
	}
}

func TestIntegrationUpdateIndexFile_NewCategory(t *testing.T) {
	_, agent := newTestWikiAgent(t)

	updates := []struct {
		Page     string `json:"page"`
		Summary  string `json:"summary"`
		Category string `json:"category"`
	}{
		{Page: "entities/foo.md", Summary: "A test entity", Category: "entities"},
	}
	if err := agent.updateIndexFile(updates); err != nil {
		t.Fatalf("updateIndexFile error: %v", err)
	}

	data, _ := os.ReadFile(agent.wiki.IndexFilePath())
	content := string(data)
	if !strings.Contains(content, "foo") {
		t.Errorf("expected foo in index, got: %s", content)
	}
	if !strings.Contains(content, "A test entity") {
		t.Errorf("expected summary in index, got: %s", content)
	}
}

func TestIntegrationUpdateIndexFile_EmptyUpdates(t *testing.T) {
	_, agent := newTestWikiAgent(t)

	if err := agent.updateIndexFile(nil); err != nil {
		t.Fatalf("updateIndexFile with nil error: %v", err)
	}
	if err := agent.updateIndexFile([]struct {
		Page     string `json:"page"`
		Summary  string `json:"summary"`
		Category string `json:"category"`
	}{}); err != nil {
		t.Fatalf("updateIndexFile with empty slice error: %v", err)
	}
}

func TestIntegrationUpdateIndexFile_UpdatesLastUpdated(t *testing.T) {
	_, agent := newTestWikiAgent(t)

	updates := []struct {
		Page     string `json:"page"`
		Summary  string `json:"summary"`
		Category string `json:"category"`
	}{
		{Page: "entities/bar.md", Summary: "Bar entity", Category: "entities"},
	}
	if err := agent.updateIndexFile(updates); err != nil {
		t.Fatalf("updateIndexFile error: %v", err)
	}

	data, _ := os.ReadFile(agent.wiki.IndexFilePath())
	content := string(data)
	if !strings.Contains(content, "## Last Updated") {
		t.Errorf("expected Last Updated section, got: %s", content)
	}
}

func TestIntegrationAppendLogFile_AppendsEntry(t *testing.T) {
	_, agent := newTestWikiAgent(t)

	entry := "## [2025-01-01] Test entry\n- Created: entities/test.md"
	if err := agent.appendLogFile(entry); err != nil {
		t.Fatalf("appendLogFile error: %v", err)
	}

	data, _ := os.ReadFile(agent.wiki.LogFilePath())
	content := string(data)
	if !strings.Contains(content, "Test entry") {
		t.Errorf("expected log entry in file, got: %s", content)
	}
}

func TestIntegrationAppendLogFile_EmptyEntry(t *testing.T) {
	_, agent := newTestWikiAgent(t)

	if err := agent.appendLogFile(""); err != nil {
		t.Fatalf("appendLogFile with empty entry error: %v", err)
	}

	data, _ := os.ReadFile(agent.wiki.LogFilePath())
	content := string(data)
	if content != "# Wiki Log\n\n" {
		t.Errorf("expected unchanged log, got: %s", content)
	}
}

func TestIntegrationCreateWikiPage_NilFrontmatter(t *testing.T) {
	_, agent := newTestWikiAgent(t)

	action := IngestAction{
		Name:    "No FM",
		Page:    "entities/no-fm.md",
		Action:  "create",
		Content: "# No Frontmatter\n\nContent.",
		// Frontmatter is nil
	}
	if err := agent.createWikiPage(action); err != nil {
		t.Fatalf("createWikiPage with nil frontmatter error: %v", err)
	}

	fullPath := filepath.Join(agent.wiki.WikiPath, action.Page)
	data, _ := os.ReadFile(fullPath)
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		t.Errorf("expected frontmatter delimiters, got: %s", content)
	}
}

func TestIntegrationCreateWikiPage_BoolAndIntFrontmatter(t *testing.T) {
	_, agent := newTestWikiAgent(t)

	// Bool and int values trigger the default case in marshalYAML
	action := IngestAction{
		Name:    "Typed FM",
		Page:    "entities/typed-fm.md",
		Action:  "create",
		Content: "# Typed FM\n\nContent.",
		Frontmatter: map[string]interface{}{
			"type":      "entity",
			"published": true,
			"priority":  5,
		},
	}
	if err := agent.createWikiPage(action); err != nil {
		t.Fatalf("createWikiPage with typed frontmatter error: %v", err)
	}

	fullPath := filepath.Join(agent.wiki.WikiPath, action.Page)
	data, _ := os.ReadFile(fullPath)
	content := string(data)
	if !strings.Contains(content, "published: true") {
		t.Errorf("expected bool value in frontmatter, got: %s", content)
	}
	if !strings.Contains(content, "priority: 5") {
		t.Errorf("expected int value in frontmatter, got: %s", content)
	}
}

func TestIntegrationSaveQueryResult(t *testing.T) {
	_, agent := newTestWikiAgent(t)

	filename, err := agent.saveQueryResult("What is machine learning?", "Machine learning is a field of AI.", []string{"wiki/entities/ml.md"})
	if err != nil {
		t.Fatalf("saveQueryResult error: %v", err)
	}

	if filename == "" {
		t.Fatal("expected non-empty filename")
	}
	if !strings.HasPrefix(filename, "synthesis/") {
		t.Errorf("expected filename in synthesis/ dir, got %q", filename)
	}
	if !strings.HasSuffix(filename, ".md") {
		t.Errorf("expected .md extension, got %q", filename)
	}

	fullPath := filepath.Join(agent.wiki.WikiPath, filename)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Fatalf("saved file not found at %s", fullPath)
	}

	data, _ := os.ReadFile(fullPath)
	content := string(data)
	if !strings.Contains(content, "machine learning") {
		t.Errorf("expected question in saved file, got: %s", content)
	}
	if !strings.Contains(content, "type: synthesis") {
		t.Errorf("expected synthesis frontmatter, got: %s", content)
	}
	if !strings.Contains(content, "ml") {
		t.Errorf("expected source reference in file, got: %s", content)
	}
}

func TestIntegrationSearchOverlap_WithResults(t *testing.T) {
	requireTSServices(t)
	requireLLMServices(t)

	ctx := context.Background()
	defer cleanupTestData(ctx, t, testCollKey)

	tmpDir := t.TempDir()
	wc := &config.WikiConfig{
		SourceConfig: config.SourceConfig{Path: tmpDir},
		WikiDir:      "wiki",
		RawDir:       "raw",
		IndexFile:    "_index.md",
		LogFile:      "_log.md",
		GraphLinks:   true,
	}
	w, err := NewWiki("test-wiki", tmpDir, wc)
	if err != nil {
		t.Fatalf("NewWiki error: %v", err)
	}
	if err := w.Init(); err != nil {
		t.Fatalf("Init error: %v", err)
	}
	agent := NewAgent(w, testCfg, testTSClient, testLLMClient)

	// Write a wiki page with content that overlaps with the search terms
	pageRel := "entities/overlap-test.md"
	fullPath := filepath.Join(w.WikiPath, pageRel)
	os.MkdirAll(filepath.Dir(fullPath), 0755)
	os.WriteFile(fullPath, []byte("---\ntype: entity\n---\n# Machine Learning\n\nMachine learning artificial intelligence concepts.\n"), 0644)

	// Index the page (reads the existing file)
	_, err = indexWikiPage(ctx, testTSClient, testCfg.CollectionKey(w.Name), w.WikiPath, pageRel)
	if err != nil {
		t.Fatalf("indexWikiPage error: %v", err)
	}

	// Search for overlapping content — the indexed content matches these terms
	overlap := agent.searchOverlap(ctx, "machine learning artificial intelligence")

	if len(overlap) == 0 {
		t.Fatal("expected at least 1 overlapping page, got 0")
	}
	t.Logf("searchOverlap found %d overlapping pages: %v", len(overlap), overlap)
}

func TestIntegrationQuery_Basic(t *testing.T) {
	requireTSServices(t)
	requireLLMServices(t)

	ctx := context.Background()
	defer cleanupTestData(ctx, t, testCollKey)

	tmpDir := t.TempDir()
	wc := &config.WikiConfig{
		SourceConfig: config.SourceConfig{Path: tmpDir},
		WikiDir:      "wiki",
		RawDir:       "raw",
		IndexFile:    "_index.md",
		LogFile:      "_log.md",
		GraphLinks:   true,
	}
	w, err := NewWiki("test-wiki", tmpDir, wc)
	if err != nil {
		t.Fatalf("NewWiki error: %v", err)
	}
	if err := w.Init(); err != nil {
		t.Fatalf("Init error: %v", err)
	}
	agent := NewAgent(w, testCfg, testTSClient, testLLMClient)

	// Create and index a wiki page
	pageRel := "entities/query-test.md"
	fullPath := filepath.Join(w.WikiPath, pageRel)
	os.MkdirAll(filepath.Dir(fullPath), 0755)
	pageContent := "---\ntype: entity\n---\n\n# Query Test\n\nThis page is about machine learning and artificial intelligence.\n"
	os.WriteFile(fullPath, []byte(pageContent), 0644)

	// Index it in TS
	_, err = indexWikiPage(ctx, testTSClient, testCfg.CollectionKey(w.Name), w.WikiPath, pageRel)
	if err != nil {
		t.Fatalf("indexWikiPage error: %v", err)
	}

	result, err := agent.Query(ctx, "machine learning", QueryOpts{Save: false, Limit: 3})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if result.Answer == "" {
		t.Error("expected non-empty answer")
	}
	t.Logf("Query answer (%d chars): %s", len(result.Answer), result.Answer[:minInt(len(result.Answer), 200)])
}

func TestIntegrationQuery_DefaultLimit(t *testing.T) {
	requireTSServices(t)
	requireLLMServices(t)

	ctx := context.Background()
	defer cleanupTestData(ctx, t, testCollKey)

	tmpDir := t.TempDir()
	wc := &config.WikiConfig{
		SourceConfig: config.SourceConfig{Path: tmpDir},
		WikiDir:      "wiki",
		RawDir:       "raw",
		IndexFile:    "_index.md",
		LogFile:      "_log.md",
		GraphLinks:   true,
	}
	w, err := NewWiki("test-wiki", tmpDir, wc)
	if err != nil {
		t.Fatalf("NewWiki error: %v", err)
	}
	if err := w.Init(); err != nil {
		t.Fatalf("Init error: %v", err)
	}
	agent := NewAgent(w, testCfg, testTSClient, testLLMClient)

	pageRel := "entities/default-limit-test.md"
	fullPath := filepath.Join(w.WikiPath, pageRel)
	os.MkdirAll(filepath.Dir(fullPath), 0755)
	os.WriteFile(fullPath, []byte("---\ntype: entity\n---\n\n# Default Limit\nContent.\n"), 0644)

	_, err = indexWikiPage(ctx, testTSClient, testCfg.CollectionKey(w.Name), w.WikiPath, pageRel)
	if err != nil {
		t.Fatalf("indexWikiPage error: %v", err)
	}

	// Limit=0 should default to 5
	result, err := agent.Query(ctx, "test", QueryOpts{Save: false, Limit: 0})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if result.Answer == "" {
		t.Error("expected non-empty answer")
	}
}

func TestIntegrationQuery_WithSave(t *testing.T) {
	requireTSServices(t)
	requireLLMServices(t)

	ctx := context.Background()
	defer cleanupTestData(ctx, t, testCollKey)

	tmpDir := t.TempDir()
	wc := &config.WikiConfig{
		SourceConfig: config.SourceConfig{Path: tmpDir},
		WikiDir:      "wiki",
		RawDir:       "raw",
		IndexFile:    "_index.md",
		LogFile:      "_log.md",
		GraphLinks:   true,
	}
	w, err := NewWiki("test-wiki", tmpDir, wc)
	if err != nil {
		t.Fatalf("NewWiki error: %v", err)
	}
	if err := w.Init(); err != nil {
		t.Fatalf("Init error: %v", err)
	}
	agent := NewAgent(w, testCfg, testTSClient, testLLMClient)

	pageRel := "entities/query-save-test.md"
	fullPath := filepath.Join(w.WikiPath, pageRel)
	os.MkdirAll(filepath.Dir(fullPath), 0755)
	os.WriteFile(fullPath, []byte("---\ntype: entity\n---\n\n# Save Test\nContent about machine learning.\n"), 0644)

	_, err = indexWikiPage(ctx, testTSClient, testCfg.CollectionKey(w.Name), w.WikiPath, pageRel)
	if err != nil {
		t.Fatalf("indexWikiPage error: %v", err)
	}

	result, err := agent.Query(ctx, "machine learning", QueryOpts{Save: true, Limit: 3})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if result.Answer == "" {
		t.Error("expected non-empty answer")
	}
}

func TestIntegrationIngest_CreatesPagesAndUpdatesIndex(t *testing.T) {
	requireTSServices(t)
	requireLLMServices(t)

	ctx := context.Background()
	defer cleanupTestData(ctx, t, testCollKey)

	tmpDir := t.TempDir()
	wc := &config.WikiConfig{
		SourceConfig: config.SourceConfig{Path: tmpDir},
		WikiDir:      "wiki",
		RawDir:       "raw",
		IndexFile:    "_index.md",
		LogFile:      "_log.md",
		GraphLinks:   true,
	}
	w, err := NewWiki("test-wiki", tmpDir, wc)
	if err != nil {
		t.Fatalf("NewWiki error: %v", err)
	}
	if err := w.Init(); err != nil {
		t.Fatalf("Init error: %v", err)
	}
	agent := NewAgent(w, testCfg, testTSClient, testLLMClient)

	// Write source file into raw/
	sourceContent := `# Go Channels

Channels are a fundamental concurrency primitive in Go. They provide a way for goroutines to communicate with each other and synchronize their execution.

## Buffered vs Unbuffered

Unbuffered channels block the sender until a receiver is ready. Buffered channels allow sending up to the buffer capacity before blocking.

## Select Statement

The select statement lets a goroutine wait on multiple channel operations simultaneously. It picks one that is ready randomly.

## Use Cases

Channels are commonly used for:
- Pipeline patterns between goroutines
- Fan-out / fan-in patterns
- Timeout and cancellation signals
- Mutual exclusion via a single goroutine

## Best Practices

Always close channels when done sending. Use range to receive values until the channel is closed. Avoid using channels for pure mutual exclusion — use sync.Mutex instead.`
	rawPath := filepath.Join(tmpDir, "raw")
	os.MkdirAll(rawPath, 0755)
	sourceFile := filepath.Join(rawPath, "go-channels.md")
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	report, err := agent.Ingest(ctx, "go-channels.md", IngestOpts{})
	if err != nil {
		t.Fatalf("Ingest error: %v", err)
	}

	for _, e := range report.Errors {
		t.Errorf("Ingest reported error: %s", e)
	}

	total := len(report.CreatedPages) + len(report.UpdatedPages)
	if total == 0 {
		t.Error("expected at least one page to be created or updated")
	}
	t.Logf("Created: %v", report.CreatedPages)
	t.Logf("Updated: %v", report.UpdatedPages)

	// Verify pages exist on disk
	for _, p := range report.CreatedPages {
		fullPath := filepath.Join(w.WikiPath, p)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("created page not on disk: %s", fullPath)
		}
	}
	for _, p := range report.UpdatedPages {
		fullPath := filepath.Join(w.WikiPath, p)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("updated page not on disk: %s", fullPath)
		}
	}

	// Verify index file was updated
	indexData, err := os.ReadFile(w.IndexFilePath())
	if err != nil {
		t.Fatalf("ReadFile(index) error: %v", err)
	}
	indexContent := string(indexData)
	if !strings.Contains(indexContent, "go-channels") && !strings.Contains(indexContent, "Go Channels") {
		t.Errorf("expected index to reference ingested content, got: %s", indexContent)
	}

	// Verify log file was updated
	logData, err := os.ReadFile(w.LogFilePath())
	if err != nil {
		t.Fatalf("ReadFile(log) error: %v", err)
	}
	logContent := string(logData)
	if len(logContent) <= len("# Wiki Log\n\n") {
		t.Error("expected log to contain ingested entry")
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
