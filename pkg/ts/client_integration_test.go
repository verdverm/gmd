//go:build integration

package ts

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/verdverm/gmd/pkg/ts/testserver"
)

const testColl = "ts-int-test"

var testTSClient *Client

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	srv, err := testserver.Start(ctx, testserver.Options{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Typesense container start failed: %v\n", err)
		os.Exit(1)
	}

	if err := srv.WaitForHealth(ctx, 30*time.Second); err != nil {
		srv.Stop(ctx)
		fmt.Fprintf(os.Stderr, "Typesense health check failed: %v\n", err)
		os.Exit(1)
	}

	testTSClient = New(Config{
		Host:   srv.URL(),
		APIKey: srv.APIKey,
	})

	if err := testTSClient.EnsureAllSchemas(ctx, 4, nil); err != nil {
		srv.Stop(ctx)
		fmt.Fprintf(os.Stderr, "Schema creation failed: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()
	srv.Stop(ctx)
	os.Exit(code)
}

func requireTS(t *testing.T) {
	t.Helper()
	if testTSClient == nil {
		t.Fatal("Typesense client not available")
	}
}

func cleanupTestData(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	_ = testTSClient.DeleteChunksByCollection(ctx, testColl)
	_ = testTSClient.DeleteDocsByCollection(ctx, testColl)
}

func makeTestChunks(path string, n int) []ChunkDocument {
	chunks := make([]ChunkDocument, n)
	for i := 0; i < n; i++ {
		chunks[i] = ChunkDocument{
			Collection:  testColl,
			Path:        path,
			Title:       fmt.Sprintf("Doc %s chunk %d", path, i),
			Content:     fmt.Sprintf("Content of chunk %d for %s.", i, path),
			Hash:        fmt.Sprintf("hash-%s-%d", path, i),
			ChunkSeq:    i,
			TotalChunks: n,
			Embedding:   []float64{float64(i), float64(i+1) * 0.5, 0.1, 0.2},
		}
	}
	return chunks
}

func makeTestDoc(path string) DocDocument {
	return DocDocument{
		Collection: testColl,
		Path:       path,
		Title:      fmt.Sprintf("Full doc %s", path),
		Content:    fmt.Sprintf("Full document content for %s.", path),
		Hash:       fmt.Sprintf("doc-hash-%s", path),
	}
}

// --- Schema ---

func TestIntegrationGetSchemaFields(t *testing.T) {
	requireTS(t)
	ctx := context.Background()

	fields, err := testTSClient.GetSchemaFields(ctx)
	if err != nil {
		t.Fatalf("GetSchemaFields: %v", err)
	}
	if len(fields) == 0 {
		t.Fatal("expected non-empty schema fields")
	}

	fieldNames := make(map[string]bool)
	for _, f := range fields {
		fieldNames[f.Name] = true
	}
	for _, name := range []string{"collection", "path", "title", "content", "hash", "chunk_seq", "total_chunks", "embedding"} {
		if !fieldNames[name] {
			t.Errorf("expected field %q in schema", name)
		}
	}
}

func TestIntegrationExtraFields(t *testing.T) {
	requireTS(t)
	ctx := context.Background()

	if err := testTSClient.EnsureSchema(ctx, 4, []SchemaField{
		{Name: "version", Type: "string"},
	}); err != nil {
		t.Fatalf("EnsureSchema with extra fields: %v", err)
	}

	fields, err := testTSClient.GetSchemaFields(ctx)
	if err != nil {
		t.Fatalf("GetSchemaFields: %v", err)
	}

	found := false
	for _, f := range fields {
		if f.Name == "version" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'version' field added to schema")
	}
}

// --- Chunks ---

func TestIntegrationChunkCRUD(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	path := "chunk-crud.md"
	chunks := makeTestChunks(path, 3)

	if err := testTSClient.UpsertChunks(ctx, chunks); err != nil {
		t.Fatalf("UpsertChunks: %v", err)
	}

	count, err := testTSClient.CountByPath(ctx, path)
	if err != nil {
		t.Fatalf("CountByPath: %v", err)
	}
	if count != 3 {
		t.Errorf("CountByPath = %d, want 3", count)
	}

	counts, err := testTSClient.CountByCollection(ctx, []string{testColl})
	if err != nil {
		t.Fatalf("CountByCollection: %v", err)
	}
	if counts[testColl] != 3 {
		t.Errorf("CountByCollection[%q] = %d, want 3", testColl, counts[testColl])
	}

	total, err := testTSClient.CollectionCount(ctx)
	if err != nil {
		t.Fatalf("CollectionCount: %v", err)
	}
	if total < 3 {
		t.Errorf("CollectionCount = %d, want >= 3", total)
	}

	hash, err := testTSClient.GetHashByPath(ctx, path)
	if err != nil {
		t.Fatalf("GetHashByPath: %v", err)
	}
	if hash == "" {
		t.Error("GetHashByPath returned empty hash")
	}
	if hash[:5] != "hash-" {
		t.Errorf("GetHashByPath = %q, want hash starting with 'hash-'", hash)
	}

	fetched, err := testTSClient.FetchChunksByPath(ctx, path)
	if err != nil {
		t.Fatalf("FetchChunksByPath: %v", err)
	}
	if len(fetched) != 3 {
		t.Fatalf("FetchChunksByPath returned %d chunks, want 3", len(fetched))
	}
	for i, f := range fetched {
		if f.ChunkSeq != i {
			t.Errorf("fetched[%d].ChunkSeq = %d, want %d", i, f.ChunkSeq, i)
		}
		if f.Path != path {
			t.Errorf("fetched[%d].Path = %q, want %q", i, f.Path, path)
		}
		if f.Collection != testColl {
			t.Errorf("fetched[%d].Collection = %q, want %q", i, f.Collection, testColl)
		}
	}

	if err := testTSClient.DeleteChunksByPath(ctx, path); err != nil {
		t.Fatalf("DeleteChunksByPath: %v", err)
	}
	count, _ = testTSClient.CountByPath(ctx, path)
	if count != 0 {
		t.Errorf("after delete, CountByPath = %d, want 0", count)
	}

	if err := testTSClient.UpsertChunks(ctx, chunks); err != nil {
		t.Fatalf("UpsertChunks (re-upsert): %v", err)
	}
	if err := testTSClient.DeleteChunksByCollection(ctx, testColl); err != nil {
		t.Fatalf("DeleteChunksByCollection: %v", err)
	}
	count, _ = testTSClient.CountByPath(ctx, path)
	if count != 0 {
		t.Errorf("after collection delete, CountByPath = %d, want 0", count)
	}
}

func TestIntegrationChunkDynamicFields(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	chunk := ChunkDocument{
		Collection:  testColl,
		Path:        "dynamic-fields.md",
		Title:       "Dynamic fields",
		Content:     "Content with custom frontmatter fields.",
		Hash:        "hash-dynamic",
		ChunkSeq:    0,
		TotalChunks: 1,
		Embedding:   []float64{0.1, 0.2, 0.3, 0.4},
		Fields: map[string]interface{}{
			"custom_field": "custom_value",
			"priority":     5,
		},
	}
	if err := testTSClient.UpsertChunks(ctx, []ChunkDocument{chunk}); err != nil {
		t.Fatalf("UpsertChunks with dynamic fields: %v", err)
	}
}

func TestIntegrationChunkLinks(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	chunk := ChunkDocument{
		Collection:  testColl,
		Path:        "links-chunk.md",
		Title:       "Chunk with links",
		Content:     "Chunk with outgoing links.",
		Hash:        "hash-links",
		ChunkSeq:    0,
		TotalChunks: 1,
		Embedding:   []float64{0.1, 0.2, 0.3, 0.4},
		Links:       []string{"other-page.md", "another-page.md"},
	}
	if err := testTSClient.UpsertChunks(ctx, []ChunkDocument{chunk}); err != nil {
		t.Fatalf("UpsertChunks with links: %v", err)
	}

	fetched, err := testTSClient.FetchChunksByPath(ctx, "links-chunk.md")
	if err != nil {
		t.Fatalf("FetchChunksByPath: %v", err)
	}
	if len(fetched) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(fetched))
	}
}

// --- Documents ---

func TestIntegrationDocCRUD(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	path := "doc-crud.md"
	doc := makeTestDoc(path)

	if err := testTSClient.UpsertDoc(ctx, doc); err != nil {
		t.Fatalf("UpsertDoc: %v", err)
	}

	fetched, err := testTSClient.FetchDocByPath(ctx, path)
	if err != nil {
		t.Fatalf("FetchDocByPath: %v", err)
	}
	if fetched == nil {
		t.Fatal("FetchDocByPath returned nil")
	}
	if fetched.Path != path {
		t.Errorf("Path = %q, want %q", fetched.Path, path)
	}
	if fetched.Collection != testColl {
		t.Errorf("Collection = %q, want %q", fetched.Collection, testColl)
	}
	if fetched.Title != doc.Title {
		t.Errorf("Title = %q, want %q", fetched.Title, doc.Title)
	}
	if fetched.Content != doc.Content {
		t.Errorf("Content = %q, want %q", fetched.Content, doc.Content)
	}
	if fetched.Hash != doc.Hash {
		t.Errorf("Hash = %q, want %q", fetched.Hash, doc.Hash)
	}

	counts, err := testTSClient.CountDocsByCollection(ctx, []string{testColl})
	if err != nil {
		t.Fatalf("CountDocsByCollection: %v", err)
	}
	if counts[testColl] != 1 {
		t.Errorf("CountDocsByCollection[%q] = %d, want 1", testColl, counts[testColl])
	}

	total, err := testTSClient.DocCollectionCount(ctx)
	if err != nil {
		t.Fatalf("DocCollectionCount: %v", err)
	}
	if total < 1 {
		t.Errorf("DocCollectionCount = %d, want >= 1", total)
	}

	if err := testTSClient.DeleteDocByPath(ctx, path); err != nil {
		t.Fatalf("DeleteDocByPath: %v", err)
	}
	fetched, _ = testTSClient.FetchDocByPath(ctx, path)
	if fetched != nil {
		t.Error("expected nil after DeleteDocByPath")
	}

	if err := testTSClient.UpsertDoc(ctx, doc); err != nil {
		t.Fatalf("UpsertDoc (re-upsert): %v", err)
	}
	if err := testTSClient.DeleteDocsByCollection(ctx, testColl); err != nil {
		t.Fatalf("DeleteDocsByCollection: %v", err)
	}
	fetched, _ = testTSClient.FetchDocByPath(ctx, path)
	if fetched != nil {
		t.Error("expected nil after DeleteDocsByCollection")
	}
}

func TestIntegrationDocLinks(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	doc := DocDocument{
		Collection: testColl,
		Path:       "doc-with-links.md",
		Title:      "Doc with links",
		Content:    "Document with outgoing links.",
		Hash:       "doc-hash-links",
		Links:      []string{"ref1.md", "ref2.md"},
	}
	if err := testTSClient.UpsertDoc(ctx, doc); err != nil {
		t.Fatalf("UpsertDoc: %v", err)
	}

	fetched, err := testTSClient.FetchDocByPath(ctx, "doc-with-links.md")
	if err != nil {
		t.Fatalf("FetchDocByPath: %v", err)
	}
	if fetched == nil {
		t.Fatal("FetchDocByPath returned nil")
	}
	if len(fetched.Links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(fetched.Links))
	}
	if fetched.Links[0] != "ref1.md" {
		t.Errorf("Links[0] = %q, want %q", fetched.Links[0], "ref1.md")
	}
	if fetched.Links[1] != "ref2.md" {
		t.Errorf("Links[1] = %q, want %q", fetched.Links[1], "ref2.md")
	}
}

func TestIntegrationFetchDocs(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	paths := []string{
		"alpha/one.md",
		"alpha/two.md",
		"beta/one.md",
	}
	for _, p := range paths {
		doc := makeTestDoc(p)
		if err := testTSClient.UpsertDoc(ctx, doc); err != nil {
			t.Fatalf("UpsertDoc %q: %v", p, err)
		}
	}

	results, err := testTSClient.FetchDocs(ctx, "alpha/one.md")
	if err != nil {
		t.Fatalf("FetchDocs exact: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("exact match: expected 1, got %d", len(results))
	}
	if results[0].Path != "alpha/one.md" {
		t.Errorf("Path = %q, want %q", results[0].Path, "alpha/one.md")
	}

	results, err = testTSClient.FetchDocs(ctx, "alpha/*")
	if err != nil {
		t.Fatalf("FetchDocs glob: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("glob match: expected 2, got %d", len(results))
	}

	results, err = testTSClient.FetchDocs(ctx, "alpha/")
	if err != nil {
		t.Fatalf("FetchDocs prefix: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("prefix match: expected 2, got %d", len(results))
	}
}

// --- Search ---

func TestIntegrationTextSearch(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	chunks := []ChunkDocument{
		{
			Collection:  testColl,
			Path:        "search-text.md",
			Title:       "Search test",
			Content:     "The quick brown fox jumps over the lazy dog.",
			Hash:        "hash-st-0",
			ChunkSeq:    0,
			TotalChunks: 2,
			Embedding:   []float64{0.1, 0.2, 0.3, 0.4},
		},
		{
			Collection:  testColl,
			Path:        "search-text.md",
			Title:       "Search test continued",
			Content:     "The lazy dog sleeps peacefully all day long.",
			Hash:        "hash-st-1",
			ChunkSeq:    1,
			TotalChunks: 2,
			Embedding:   []float64{0.5, 0.6, 0.7, 0.8},
		},
	}
	if err := testTSClient.UpsertChunks(ctx, chunks); err != nil {
		t.Fatalf("UpsertChunks: %v", err)
	}

	results, err := testTSClient.TextSearch(ctx, HybridSearchParams{
		Query:       "fox",
		Collections: []string{testColl},
		GroupLimit:  10,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("TextSearch: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("TextSearch returned 0 results for 'fox'")
	}
	for _, r := range results {
		if r.Collection != testColl {
			t.Errorf("result Collection = %q, want %q", r.Collection, testColl)
		}
	}

	results, err = testTSClient.TextSearch(ctx, HybridSearchParams{
		Query:       "nonexistenttermzzz",
		Collections: []string{testColl},
		GroupLimit:  10,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("TextSearch (no match): %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for nonexistent term, got %d", len(results))
	}

	results, err = testTSClient.TextSearch(ctx, HybridSearchParams{
		Query:      "",
		FilterBy:   fmt.Sprintf("collection:=%s", testColl),
		GroupLimit: 10,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("TextSearch with FilterBy: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("TextSearch with FilterBy returned 0 results")
	}
}

func TestIntegrationHybridSearch(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	chunks := []ChunkDocument{
		{
			Collection:  testColl,
			Path:        "hybrid-test.md",
			Title:       "Hybrid test",
			Content:     "Alpha beta gamma delta epsilon.",
			Hash:        "hash-hybrid-0",
			ChunkSeq:    0,
			TotalChunks: 1,
			Embedding:   []float64{0.9, 0.1, 0.1, 0.1},
		},
	}
	if err := testTSClient.UpsertChunks(ctx, chunks); err != nil {
		t.Fatalf("UpsertChunks: %v", err)
	}

	results, err := testTSClient.HybridSearch(ctx, HybridSearchParams{
		Query:       "alpha",
		QueryVector: []float64{0.8, 0.2, 0.1, 0.1},
		Collections: []string{testColl},
		GroupLimit:  10,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("HybridSearch (with vector): %v", err)
	}
	if len(results) == 0 {
		t.Fatal("HybridSearch with vector returned 0 results")
	}

	results, err = testTSClient.HybridSearch(ctx, HybridSearchParams{
		Query:       "beta",
		Collections: []string{testColl},
		GroupLimit:  10,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("HybridSearch (text-only): %v", err)
	}
	if len(results) == 0 {
		t.Fatal("HybridSearch text-only returned 0 results")
	}
}

func TestIntegrationVectorSearch(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	chunks := []ChunkDocument{
		{
			Collection:  testColl,
			Path:        "vector-test.md",
			Title:       "Vector test",
			Content:     "Vector similarity search content.",
			Hash:        "hash-vector-0",
			ChunkSeq:    0,
			TotalChunks: 1,
			Embedding:   []float64{0.9, 0.1, 0.1, 0.1},
		},
	}
	if err := testTSClient.UpsertChunks(ctx, chunks); err != nil {
		t.Fatalf("UpsertChunks: %v", err)
	}

	results, err := testTSClient.VectorSearch(ctx, HybridSearchParams{
		QueryVector: []float64{0.85, 0.15, 0.1, 0.1},
		Collections: []string{testColl},
		GroupLimit:  10,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("VectorSearch: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("VectorSearch returned 0 results")
	}
	if results[0].Path != "vector-test.md" {
		t.Errorf("result Path = %q, want %q", results[0].Path, "vector-test.md")
	}
}

// --- Path operations ---

func TestIntegrationSearchDistinctPaths(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	paths := []string{"distinct/a.md", "distinct/b.md", "other/c.md"}
	for _, p := range paths {
		if err := testTSClient.UpsertChunks(ctx, makeTestChunks(p, 1)); err != nil {
			t.Fatalf("UpsertChunks %q: %v", p, err)
		}
	}

	allPaths, err := testTSClient.SearchDistinctPaths(ctx, "")
	if err != nil {
		t.Fatalf("SearchDistinctPaths: %v", err)
	}
	if len(allPaths) < 3 {
		t.Fatalf("expected >= 3 distinct paths, got %d", len(allPaths))
	}

	filtered, err := testTSClient.SearchDistinctPaths(ctx, "collection:=ts-int-test")
	if err != nil {
		t.Fatalf("SearchDistinctPaths (filtered): %v", err)
	}
	if len(filtered) < 3 {
		t.Errorf("expected >= 3 filtered paths, got %d", len(filtered))
	}
}

func TestIntegrationListDocuments(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	paths := []string{"list/doc1.md", "list/doc2.md"}
	for _, p := range paths {
		if err := testTSClient.UpsertChunks(ctx, makeTestChunks(p, 1)); err != nil {
			t.Fatalf("UpsertChunks %q: %v", p, err)
		}
	}

	results, err := testTSClient.ListDocuments(ctx, []string{testColl})
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(results))
	}
}

func TestIntegrationSearchChunksByPath(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	path := "search-by-path.md"
	if err := testTSClient.UpsertChunks(ctx, makeTestChunks(path, 2)); err != nil {
		t.Fatalf("UpsertChunks: %v", err)
	}

	results, err := testTSClient.SearchChunksByPath(ctx, fmt.Sprintf("path:=%s", path), 10)
	if err != nil {
		t.Fatalf("SearchChunksByPath: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("SearchChunksByPath returned 0 results")
	}
	for _, r := range results {
		if r.Path != path {
			t.Errorf("result Path = %q, want %q", r.Path, path)
		}
	}
}

// --- Edge cases ---

func TestIntegrationNonExistentPaths(t *testing.T) {
	requireTS(t)
	ctx := context.Background()

	count, err := testTSClient.CountByPath(ctx, "nonexistent-path.md")
	if err != nil {
		t.Fatalf("CountByPath: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	hash, err := testTSClient.GetHashByPath(ctx, "nonexistent-path.md")
	if err != nil {
		t.Fatalf("GetHashByPath: %v", err)
	}
	if hash != "" {
		t.Errorf("expected empty hash, got %q", hash)
	}

	doc, err := testTSClient.FetchDocByPath(ctx, "nonexistent-path.md")
	if err != nil {
		t.Fatalf("FetchDocByPath: %v", err)
	}
	if doc != nil {
		t.Error("expected nil for non-existent doc")
	}
}

func TestIntegrationEmptyCollectionSearch(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	results, err := testTSClient.TextSearch(ctx, HybridSearchParams{
		Query:       "anything",
		Collections: []string{testColl},
		GroupLimit:  10,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("TextSearch on empty collection: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results on empty collection, got %d", len(results))
	}

	paths, err := testTSClient.SearchDistinctPaths(ctx, "collection:=ts-int-test")
	if err != nil {
		t.Fatalf("SearchDistinctPaths on empty: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected 0 paths on empty collection, got %d", len(paths))
	}
}
