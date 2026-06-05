//go:build integration

package wiki

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/verdverm/gmd/pkg/chunking"
	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/ts"
	"github.com/verdverm/gmd/pkg/ts/testserver"
)

const testCollKey = "wiki-int-test"

var (
	testTSClient  *ts.Client
	testLLMClient *llm.Client
	testCfg       *config.Config
)

func TestMain(m *testing.M) {
	code := 1
	defer func() { os.Exit(code) }()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	tsSrv, err := testserver.Start(ctx, testserver.Options{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "wiki integration: FATAL: Typesense container failed (%v)\n", err)
	} else {
		defer tsSrv.Stop(ctx)

		if err := tsSrv.WaitForHealth(ctx, 30*time.Second); err != nil {
			fmt.Fprintf(os.Stderr, "wiki integration: FATAL: TS health check failed (%v)\n", err)
		} else {
			testTSClient = ts.New(ts.Config{
				Host:   tsSrv.URL(),
				APIKey: tsSrv.APIKey,
			})
			if err := testTSClient.EnsureSchema(ctx, 0, nil); err != nil {
				fmt.Fprintf(os.Stderr, "wiki integration: FATAL: TS schema failed (%v)\n", err)
				testTSClient = nil
			}
		}
	}

	cfg, err := config.Load(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "wiki integration: FATAL: LLM config load failed (%v)\n", err)
	} else {
		testLLMClient = llm.New(llm.ConfigFromProject(cfg))
		testCfg = cfg
	}

	code = m.Run()
}

func cleanupTestData(ctx context.Context, t *testing.T, collectionKey string) {
	t.Helper()
	if testTSClient == nil {
		return
	}
	if err := testTSClient.DeleteChunksByCollection(ctx, collectionKey); err != nil {
		t.Logf("cleanup: %v", err)
	}
}

func requireTSServices(t *testing.T) {
	t.Helper()
	if testTSClient == nil {
		t.Fatal("Typesense container not available — integration tests require a running Typesense instance")
	}
}

func embedOrSkip(ctx context.Context, t *testing.T, text string) []float64 {
	t.Helper()
	vec, err := testLLMClient.Embed(ctx, text)
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	return vec
}

func requireLLMServices(t *testing.T) {
	t.Helper()
	if testLLMClient == nil {
		t.Fatal("LLM services not available — integration tests require an LLM provider configured in gmd config")
	}
}

// ---------------------------------------------------------------------------
// Doctor (real TS + LLM)
// ---------------------------------------------------------------------------

func TestIntegrationDoctorRealServices(t *testing.T) {
	requireTSServices(t)
	requireLLMServices(t)

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

	result, err := Doctor(context.Background(), w, testCfg, testTSClient, testLLMClient)
	if err != nil {
		t.Fatalf("Doctor error: %v", err)
	}

	if result.WikiName != "test-wiki" {
		t.Errorf("WikiName = %q, want %q", result.WikiName, "test-wiki")
	}
	if !result.TSConnected {
		t.Error("TSConnected should be true with real client")
	}
	if len(result.LLMStatus) == 0 {
		t.Fatal("expected LLM status entries")
	}
	for _, s := range result.LLMStatus {
		if !s.OK {
			t.Errorf("LLM endpoint %s (%s): %s", s.Label, s.URL, s.Err)
		}
		if len(s.Models) == 0 && s.OK {
			t.Errorf("LLM endpoint %s: no models returned", s.Label)
		}
	}
	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
}

func TestIntegrationDoctorRealTSNilLLM(t *testing.T) {
	requireTSServices(t)

	tmpDir := t.TempDir()
	wc := &config.WikiConfig{
		SourceConfig: config.SourceConfig{Path: tmpDir},
		WikiDir:      "wiki",
		RawDir:       "raw",
		IndexFile:    "_index.md",
		LogFile:      "_log.md",
		GraphLinks:   true,
	}
	w, _ := NewWiki("test", tmpDir, wc)
	w.Init()

	result, err := Doctor(context.Background(), w, testCfg, testTSClient, nil)
	if err != nil {
		t.Fatalf("Doctor error: %v", err)
	}
	if !result.TSConnected {
		t.Error("TSConnected should be true with real TS client")
	}
	if len(result.LLMStatus) != 0 {
		t.Errorf("expected 0 LLMStatus, got %d", len(result.LLMStatus))
	}
}

// ---------------------------------------------------------------------------
// Typesense chunk CRUD for wiki content
// ---------------------------------------------------------------------------

func TestIntegrationTSChunkCRUD(t *testing.T) {
	requireTSServices(t)
	requireLLMServices(t)

	ctx := context.Background()
	defer cleanupTestData(ctx, t, testCollKey)

	doc := ts.ChunkDocument{
		Collection:  testCollKey,
		Path:        "wiki/entities/test-entity.md",
		Title:       "Test Entity",
		Content:     "This is a test entity about machine learning and artificial intelligence.",
		ChunkSeq:    0,
		TotalChunks: 1,
	}
	doc.Embedding = embedOrSkip(ctx, t, doc.Content)

	if err := testTSClient.UpsertChunks(ctx, []ts.ChunkDocument{doc}); err != nil {
		t.Fatalf("UpsertChunks error: %v", err)
	}

	results, err := testTSClient.TextSearch(ctx, ts.HybridSearchParams{
		Query:       "machine learning",
		Collections: []string{testCollKey},
		Limit:       10,
		GroupLimit:  1,
	})
	if err != nil {
		t.Fatalf("TextSearch error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least 1 search result")
	}
	if results[0].Path != "wiki/entities/test-entity.md" {
		t.Errorf("path = %q, want %q", results[0].Path, "wiki/entities/test-entity.md")
	}
	if results[0].Collection != testCollKey {
		t.Errorf("collection = %q, want %q", results[0].Collection, testCollKey)
	}

	count, err := testTSClient.CountByPath(ctx, "wiki/entities/test-entity.md")
	if err != nil {
		t.Fatalf("CountByPath error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	if err := testTSClient.DeleteChunksByPath(ctx, "wiki/entities/test-entity.md"); err != nil {
		t.Fatalf("DeleteChunksByPath error: %v", err)
	}

	count, _ = testTSClient.CountByPath(ctx, "wiki/entities/test-entity.md")
	if count != 0 {
		t.Errorf("expected count 0 after delete, got %d", count)
	}
}

func TestIntegrationTSSearchFiltersByCollection(t *testing.T) {
	requireTSServices(t)
	requireLLMServices(t)

	ctx := context.Background()
	defer cleanupTestData(ctx, t, testCollKey)

	docs := []ts.ChunkDocument{
		{
			Collection:  testCollKey,
			Path:        "wiki/entities/alpha.md",
			Title:       "Alpha",
			Content:     "Alpha is about machine learning models.",
			ChunkSeq:    0,
			TotalChunks: 1,
		},
		{
			Collection:  testCollKey + "-other",
			Path:        "other/data.md",
			Title:       "Other",
			Content:     "Other data about machine learning.",
			ChunkSeq:    0,
			TotalChunks: 1,
		},
	}
	for i := range docs {
		docs[i].Embedding = embedOrSkip(ctx, t, docs[i].Content)
	}

	if err := testTSClient.UpsertChunks(ctx, docs); err != nil {
		t.Fatalf("UpsertChunks error: %v", err)
	}
	defer testTSClient.DeleteChunksByCollection(ctx, testCollKey+"-other")

	results, err := testTSClient.TextSearch(ctx, ts.HybridSearchParams{
		Query:       "machine learning",
		Collections: []string{testCollKey},
		Limit:       10,
		GroupLimit:  1,
	})
	if err != nil {
		t.Fatalf("TextSearch error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result (filtered by collection), got %d", len(results))
	}
	if results[0].Collection != testCollKey {
		t.Errorf("collection = %q, want %q", results[0].Collection, testCollKey)
	}
}

func TestIntegrationTSHybridSearch(t *testing.T) {
	requireTSServices(t)
	requireLLMServices(t)

	ctx := context.Background()
	defer cleanupTestData(ctx, t, testCollKey)

	doc := ts.ChunkDocument{
		Collection:  testCollKey,
		Path:        "wiki/concepts/test-concept.md",
		Title:       "Test Concept",
		Content:     "A concept about reinforcement learning in autonomous systems.",
		ChunkSeq:    0,
		TotalChunks: 1,
	}
	doc.Embedding = embedOrSkip(ctx, t, doc.Content)
	if err := testTSClient.UpsertChunks(ctx, []ts.ChunkDocument{doc}); err != nil {
		t.Fatalf("UpsertChunks error: %v", err)
	}

	results, err := testTSClient.HybridSearch(ctx, ts.HybridSearchParams{
		Query:       "autonomous systems",
		Collections: []string{testCollKey},
		Limit:       10,
		GroupLimit:  1,
	})
	if err != nil {
		t.Fatalf("HybridSearch error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least 1 hybrid search result")
	}
	if results[0].Path != "wiki/concepts/test-concept.md" {
		t.Errorf("path = %q, want %q", results[0].Path, "wiki/concepts/test-concept.md")
	}
}

func TestIntegrationTSMultiChunkPerPath(t *testing.T) {
	requireTSServices(t)
	requireLLMServices(t)

	ctx := context.Background()
	defer cleanupTestData(ctx, t, testCollKey)

	docs := []ts.ChunkDocument{
		{Collection: testCollKey, Path: "wiki/entities/long.md", Title: "Long", Content: "First chunk of a long document.", ChunkSeq: 0, TotalChunks: 2},
		{Collection: testCollKey, Path: "wiki/entities/long.md", Title: "Long", Content: "Second chunk of a long document.", ChunkSeq: 1, TotalChunks: 2},
	}
	for i := range docs {
		docs[i].Embedding = embedOrSkip(ctx, t, docs[i].Content)
	}
	if err := testTSClient.UpsertChunks(ctx, docs); err != nil {
		t.Fatalf("UpsertChunks error: %v", err)
	}

	results, err := testTSClient.TextSearch(ctx, ts.HybridSearchParams{
		Query:       "long document",
		Collections: []string{testCollKey},
		Limit:       10,
		GroupLimit:  2,
	})
	if err != nil {
		t.Fatalf("TextSearch error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 grouped result, got %d", len(results))
	}
	if results[0].Path != "wiki/entities/long.md" {
		t.Errorf("path = %q, want %q", results[0].Path, "wiki/entities/long.md")
	}
}

func TestIntegrationTSVectorSearch(t *testing.T) {
	requireTSServices(t)
	requireLLMServices(t)

	ctx := context.Background()
	defer cleanupTestData(ctx, t, testCollKey)

	doc := ts.ChunkDocument{
		Collection:  testCollKey,
		Path:        "wiki/entities/vector-test.md",
		Title:       "Vector Test",
		Content:     "Vector search is important for semantic retrieval in RAG systems.",
		ChunkSeq:    0,
		TotalChunks: 1,
	}
	doc.Embedding = embedOrSkip(ctx, t, doc.Content)
	if err := testTSClient.UpsertChunks(ctx, []ts.ChunkDocument{doc}); err != nil {
		t.Fatalf("UpsertChunks error: %v", err)
	}

	queryVec := embedOrSkip(ctx, t, "semantic retrieval with vectors")

	results, err := testTSClient.VectorSearch(ctx, ts.HybridSearchParams{
		QueryVector: queryVec,
		Collections: []string{testCollKey},
		Limit:       10,
		GroupLimit:  1,
	})
	if err != nil {
		t.Fatalf("VectorSearch error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least 1 vector search result")
	}
	if results[0].Path != "wiki/entities/vector-test.md" {
		t.Errorf("path = %q, want %q", results[0].Path, "wiki/entities/vector-test.md")
	}
}

// ---------------------------------------------------------------------------
// LLM integration
// ---------------------------------------------------------------------------

func TestIntegrationLLMEmbed(t *testing.T) {
	requireLLMServices(t)

	ctx := context.Background()
	vec, err := testLLMClient.Embed(ctx, "machine learning")
	if err != nil {
		t.Fatalf("Embed error: %v", err)
	}
	if len(vec) == 0 {
		t.Fatal("expected non-empty embedding vector")
	}
	t.Logf("embedding dimension: %d", len(vec))
}

func TestIntegrationLLMChat(t *testing.T) {
	requireLLMServices(t)

	ctx := context.Background()
	resp, err := testLLMClient.Chat(ctx, []llm.ChatMessage{
		{Role: "system", Content: "You are a helpful assistant. Answer concisely."},
		{Role: "user", Content: "Say hello world in one word."},
	})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	if resp == "" {
		t.Fatal("expected non-empty chat response")
	}
	t.Logf("chat response: %s", resp)
}

// ---------------------------------------------------------------------------
// Full pipeline: index wiki content in TS, then search it
// ---------------------------------------------------------------------------

func TestIntegrationWikiIndexAndSearch(t *testing.T) {
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

	action := IngestAction{
		Name:    "Machine Learning",
		Page:    "entities/machine-learning.md",
		Action:  "create",
		Content: "# Machine Learning\n\nMachine learning is a subset of artificial intelligence.",
		Frontmatter: map[string]interface{}{
			"type": "entity",
			"tags": []interface{}{"ai", "ml"},
		},
	}
	if err := agent.createWikiPage(action); err != nil {
		t.Fatalf("createWikiPage error: %v", err)
	}

	chunks, err := indexWikiPage(ctx, testTSClient, testCfg.CollectionKey(w.Name), w.WikiPath, "entities/machine-learning.md")
	if err != nil {
		t.Fatalf("indexWikiPage error: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk")
	}

	results, err := testTSClient.TextSearch(ctx, ts.HybridSearchParams{
		Query:       "artificial intelligence",
		Collections: []string{testCfg.CollectionKey(w.Name)},
		Limit:       10,
		GroupLimit:  1,
	})
	if err != nil {
		t.Fatalf("TextSearch error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results after indexing wiki page")
	}
}

func TestIntegrationWikiCollectionCounts(t *testing.T) {
	requireTSServices(t)
	requireLLMServices(t)

	ctx := context.Background()
	defer cleanupTestData(ctx, t, testCollKey)

	docs := []ts.ChunkDocument{
		{Collection: testCollKey, Path: "wiki/a.md", Title: "A", Content: "alpha content", ChunkSeq: 0, TotalChunks: 1},
		{Collection: testCollKey, Path: "wiki/b.md", Title: "B", Content: "beta content", ChunkSeq: 0, TotalChunks: 1},
	}
	for i := range docs {
		docs[i].Embedding = embedOrSkip(ctx, t, docs[i].Content)
	}
	if err := testTSClient.UpsertChunks(ctx, docs); err != nil {
		t.Fatalf("UpsertChunks error: %v", err)
	}

	counts, err := testTSClient.CountByCollection(ctx, []string{testCollKey, "nonexistent"})
	if err != nil {
		t.Fatalf("CountByCollection error: %v", err)
	}
	if counts[testCollKey] != 2 {
		t.Errorf("expected count 2 for collection, got %d", counts[testCollKey])
	}
	if counts["nonexistent"] != 0 {
		t.Errorf("expected count 0 for nonexistent, got %d", counts["nonexistent"])
	}
}

// ---------------------------------------------------------------------------
// Init error paths
// ---------------------------------------------------------------------------

func TestIntegrationInit_ErrorMkdirAll(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "raw"), []byte("blocking file"), 0644)
	w, _ := NewWiki("test", tmpDir, &config.WikiConfig{
		SourceConfig: config.SourceConfig{Path: tmpDir},
		WikiDir:      "wiki",
		RawDir:       "raw",
		IndexFile:    "_index.md",
		LogFile:      "_log.md",
		GraphLinks:   true,
	})
	err := w.Init()
	if err == nil {
		t.Fatal("expected MkdirAll error")
	}
}

func TestIntegrationInit_ErrorWriteIndex(t *testing.T) {
	tmpDir := t.TempDir()
	for _, dir := range wikiDirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	wikiDir := filepath.Join(tmpDir, "wiki")
	// 0555 = read+execute (needed for MkdirAll path traversal) but no write
	if err := os.Chmod(wikiDir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(wikiDir, 0755) })

	w, _ := NewWiki("test", tmpDir, &config.WikiConfig{
		SourceConfig: config.SourceConfig{Path: tmpDir},
		WikiDir:      "wiki",
		RawDir:       "raw",
		IndexFile:    "_index.md",
		LogFile:      "_log.md",
		GraphLinks:   true,
	})
	err := w.Init()
	if err == nil {
		t.Fatal("expected error for index file write failure")
	}
}

func TestIntegrationInit_ErrorWriteLog(t *testing.T) {
	tmpDir := t.TempDir()
	for _, dir := range wikiDirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	indexPath := filepath.Join(tmpDir, "wiki", "_index.md")
	if err := os.WriteFile(indexPath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	wikiDir := filepath.Join(tmpDir, "wiki")
	if err := os.Chmod(wikiDir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(wikiDir, 0755) })

	w, _ := NewWiki("test", tmpDir, &config.WikiConfig{
		SourceConfig: config.SourceConfig{Path: tmpDir},
		WikiDir:      "wiki",
		RawDir:       "raw",
		IndexFile:    "_index.md",
		LogFile:      "_log.md",
		GraphLinks:   true,
	})
	err := w.Init()
	if err == nil {
		t.Fatal("expected error for log file write failure")
	}
}

func TestIntegrationInit_ErrorWriteSchema(t *testing.T) {
	tmpDir := t.TempDir()
	for _, dir := range wikiDirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	indexPath := filepath.Join(tmpDir, "wiki", "_index.md")
	if err := os.WriteFile(indexPath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(tmpDir, "wiki", "_log.md")
	if err := os.WriteFile(logPath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	// 0555 = read+execute but no write — MkdirAll can stat existing dirs, WriteFile fails
	if err := os.Chmod(tmpDir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(tmpDir, 0755) })

	w, _ := NewWiki("test", tmpDir, &config.WikiConfig{
		SourceConfig: config.SourceConfig{Path: tmpDir},
		WikiDir:      "wiki",
		RawDir:       "raw",
		IndexFile:    "_index.md",
		LogFile:      "_log.md",
		GraphLinks:   true,
	})
	err := w.Init()
	if err == nil {
		t.Fatal("expected error for schema file write failure")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// storePath converts a relative page path (e.g. "entities/foo.md") to the path
// the production indexer would store — relative to the project root with a "wiki/"
// prefix, matching how readWikiPage (which uses a.wiki.Path) resolves it.
func storePath(relPath string) string {
	return filepath.Join("wiki", relPath)
}

func indexWikiPage(ctx context.Context, c *ts.Client, collectionKey, wikiPath, relPath string) ([]ts.ChunkDocument, error) {
	fullPath := filepath.Join(wikiPath, relPath)
	os.MkdirAll(filepath.Dir(fullPath), 0755)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		data = []byte(fmt.Sprintf("---\ntype: entity\n---\n# Test\n\nContent for %s.\n", relPath))
		os.WriteFile(fullPath, data, 0644)
	}
	_, stripped, _ := ParseFrontmatter(string(data))

	if err := c.DeleteChunksByPath(ctx, storePath(relPath)); err != nil {
		return nil, fmt.Errorf("delete existing: %w", err)
	}

	vec, err := testLLMClient.Embed(ctx, stripped)
	if err != nil {
		return nil, fmt.Errorf("embed content: %w", err)
	}

	tsPath := storePath(relPath)
	extractedLinks := chunking.ExtractWikilinks(stripped)
	doc := ts.ChunkDocument{
		Collection:  collectionKey,
		Path:        tsPath,
		Title:       "Test",
		Content:     stripped,
		ChunkSeq:    0,
		TotalChunks: 1,
		Embedding:   vec,
		Links:       extractedLinks,
	}

	if err := c.UpsertChunks(ctx, []ts.ChunkDocument{doc}); err != nil {
		return nil, fmt.Errorf("upsert: %w", err)
	}

	return []ts.ChunkDocument{doc}, nil
}
