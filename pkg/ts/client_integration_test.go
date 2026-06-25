//go:build integration

package ts

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/verdverm/gmd/pkg/testutil"
	"github.com/verdverm/gmd/pkg/ts/testserver"
)

var testTSClient *Client
var testServerURL string

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

	testServerURL = srv.URL()
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

func maybeNewTape(t *testing.T, filePath string) *testutil.Tape {
	t.Helper()
	if os.Getenv("GMD_NORECORD") == "1" {
		return nil
	}
	return testutil.NewTape(filePath, testServerURL, nil, testutil.ModeRecord)
}

func newTapeClient(t *testing.T, tape *testutil.Tape) *Client {
	t.Helper()
	return New(Config{
		Host:       testServerURL,
		APIKey:     testserver.DefaultAPIKey,
		HTTPClient: &http.Client{Transport: tape.Transport()},
	})
}

func TestIntegrationTS_GetSchemaFields(t *testing.T) {
	requireTS(t)
	ctx := context.Background()

	tape := maybeNewTape(t, "testdata/TS_GetSchemaFields.json")
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
	}

	client := testTSClient
	if tape != nil {
		client = newTapeClient(t, tape)
	}

	fields, err := client.GetSchemaFields(ctx)
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

func TestIntegrationTS_ExtraFields(t *testing.T) {
	requireTS(t)
	ctx := context.Background()

	tape := maybeNewTape(t, "testdata/TS_ExtraFields.json")
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
	}

	client := testTSClient
	if tape != nil {
		client = newTapeClient(t, tape)
	}

	if err := client.EnsureSchema(ctx, 4, []SchemaField{
		{Name: "version", Type: "string"},
	}); err != nil {
		t.Fatalf("EnsureSchema with extra fields: %v", err)
	}

	fields, err := client.GetSchemaFields(ctx)
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

func TestIntegrationTS_ChunkCRUD(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	tape := maybeNewTape(t, "testdata/TS_ChunkCRUD.json")
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
	}

	client := testTSClient
	if tape != nil {
		client = newTapeClient(t, tape)
	}

	path := "chunk-crud.md"
	chunks := makeTestChunks(path, 3)

	if err := client.UpsertChunks(ctx, chunks); err != nil {
		t.Fatalf("UpsertChunks: %v", err)
	}

	count, err := client.CountByPath(ctx, path)
	if err != nil {
		t.Fatalf("CountByPath: %v", err)
	}
	if count != 3 {
		t.Errorf("CountByPath = %d, want 3", count)
	}

	counts, err := client.CountByCollection(ctx, []string{testColl})
	if err != nil {
		t.Fatalf("CountByCollection: %v", err)
	}
	if counts[testColl] != 3 {
		t.Errorf("CountByCollection[%q] = %d, want 3", testColl, counts[testColl])
	}

	total, err := client.CollectionCount(ctx)
	if err != nil {
		t.Fatalf("CollectionCount: %v", err)
	}
	if total < 3 {
		t.Errorf("CollectionCount = %d, want >= 3", total)
	}

	hash, err := client.GetHashByPath(ctx, path)
	if err != nil {
		t.Fatalf("GetHashByPath: %v", err)
	}
	if hash == "" {
		t.Error("GetHashByPath returned empty hash")
	}
	if hash[:5] != "hash-" {
		t.Errorf("GetHashByPath = %q, want hash starting with 'hash-'", hash)
	}

	fetched, err := client.FetchChunksByPath(ctx, path)
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

	if err := client.DeleteChunksByPath(ctx, path); err != nil {
		t.Fatalf("DeleteChunksByPath: %v", err)
	}
	count, _ = client.CountByPath(ctx, path)
	if count != 0 {
		t.Errorf("after delete, CountByPath = %d, want 0", count)
	}

	if err := client.UpsertChunks(ctx, chunks); err != nil {
		t.Fatalf("UpsertChunks (re-upsert): %v", err)
	}
	if err := client.DeleteChunksByCollection(ctx, testColl); err != nil {
		t.Fatalf("DeleteChunksByCollection: %v", err)
	}
	count, _ = client.CountByPath(ctx, path)
	if count != 0 {
		t.Errorf("after collection delete, CountByPath = %d, want 0", count)
	}
}

func TestIntegrationTS_ChunkDynamicFields(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	tape := maybeNewTape(t, "testdata/TS_ChunkDynamicFields.json")
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
	}

	client := testTSClient
	if tape != nil {
		client = newTapeClient(t, tape)
	}

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
	if err := client.UpsertChunks(ctx, []ChunkDocument{chunk}); err != nil {
		t.Fatalf("UpsertChunks with dynamic fields: %v", err)
	}
}

func TestIntegrationTS_ChunkLinks(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	tape := maybeNewTape(t, "testdata/TS_ChunkLinks.json")
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
	}

	client := testTSClient
	if tape != nil {
		client = newTapeClient(t, tape)
	}

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
	if err := client.UpsertChunks(ctx, []ChunkDocument{chunk}); err != nil {
		t.Fatalf("UpsertChunks with links: %v", err)
	}

	fetched, err := client.FetchChunksByPath(ctx, "links-chunk.md")
	if err != nil {
		t.Fatalf("FetchChunksByPath: %v", err)
	}
	if len(fetched) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(fetched))
	}
}

// --- Documents ---

func TestIntegrationTS_DocCRUD(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	tape := maybeNewTape(t, "testdata/TS_DocCRUD.json")
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
	}

	client := testTSClient
	if tape != nil {
		client = newTapeClient(t, tape)
	}

	path := "doc-crud.md"
	doc := makeTestDoc(path)

	if err := client.UpsertDoc(ctx, doc); err != nil {
		t.Fatalf("UpsertDoc: %v", err)
	}

	fetched, err := client.FetchDocByPath(ctx, path)
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

	counts, err := client.CountDocsByCollection(ctx, []string{testColl})
	if err != nil {
		t.Fatalf("CountDocsByCollection: %v", err)
	}
	if counts[testColl] != 1 {
		t.Errorf("CountDocsByCollection[%q] = %d, want 1", testColl, counts[testColl])
	}

	total, err := client.DocCollectionCount(ctx)
	if err != nil {
		t.Fatalf("DocCollectionCount: %v", err)
	}
	if total < 1 {
		t.Errorf("DocCollectionCount = %d, want >= 1", total)
	}

	if err := client.DeleteDocByPath(ctx, path); err != nil {
		t.Fatalf("DeleteDocByPath: %v", err)
	}
	fetched, _ = client.FetchDocByPath(ctx, path)
	if fetched != nil {
		t.Error("expected nil after DeleteDocByPath")
	}

	if err := client.UpsertDoc(ctx, doc); err != nil {
		t.Fatalf("UpsertDoc (re-upsert): %v", err)
	}
	if err := client.DeleteDocsByCollection(ctx, testColl); err != nil {
		t.Fatalf("DeleteDocsByCollection: %v", err)
	}
	fetched, _ = client.FetchDocByPath(ctx, path)
	if fetched != nil {
		t.Error("expected nil after DeleteDocsByCollection")
	}
}

func TestIntegrationTS_DocLinks(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	tape := maybeNewTape(t, "testdata/TS_DocLinks.json")
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
	}

	client := testTSClient
	if tape != nil {
		client = newTapeClient(t, tape)
	}

	doc := DocDocument{
		Collection: testColl,
		Path:       "doc-with-links.md",
		Title:      "Doc with links",
		Content:    "Document with outgoing links.",
		Hash:       "doc-hash-links",
		Links:      []string{"ref1.md", "ref2.md"},
	}
	if err := client.UpsertDoc(ctx, doc); err != nil {
		t.Fatalf("UpsertDoc: %v", err)
	}

	fetched, err := client.FetchDocByPath(ctx, "doc-with-links.md")
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

func TestIntegrationTS_FetchDocs(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	tape := maybeNewTape(t, "testdata/TS_FetchDocs.json")
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
	}

	client := testTSClient
	if tape != nil {
		client = newTapeClient(t, tape)
	}

	paths := []string{
		"alpha/one.md",
		"alpha/two.md",
		"beta/one.md",
	}
	for _, p := range paths {
		doc := makeTestDoc(p)
		if err := client.UpsertDoc(ctx, doc); err != nil {
			t.Fatalf("UpsertDoc %q: %v", p, err)
		}
	}

	results, err := client.FetchDocs(ctx, "alpha/one.md")
	if err != nil {
		t.Fatalf("FetchDocs exact: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("exact match: expected 1, got %d", len(results))
	}
	if results[0].Path != "alpha/one.md" {
		t.Errorf("Path = %q, want %q", results[0].Path, "alpha/one.md")
	}

	results, err = client.FetchDocs(ctx, "alpha/*")
	if err != nil {
		t.Fatalf("FetchDocs glob: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("glob match: expected 2, got %d", len(results))
	}

	results, err = client.FetchDocs(ctx, "alpha/")
	if err != nil {
		t.Fatalf("FetchDocs prefix: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("prefix match: expected 2, got %d", len(results))
	}
}

// --- Search ---

func TestIntegrationTS_TextSearch(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	tape := maybeNewTape(t, "testdata/TS_TextSearch.json")
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
	}

	client := testTSClient
	if tape != nil {
		client = newTapeClient(t, tape)
	}

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
	if err := client.UpsertChunks(ctx, chunks); err != nil {
		t.Fatalf("UpsertChunks: %v", err)
	}

	results, err := client.TextSearch(ctx, HybridSearchParams{
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

	results, err = client.TextSearch(ctx, HybridSearchParams{
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

	results, err = client.TextSearch(ctx, HybridSearchParams{
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

func TestIntegrationTS_HybridSearch(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	tape := maybeNewTape(t, "testdata/TS_HybridSearch.json")
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
	}

	client := testTSClient
	if tape != nil {
		client = newTapeClient(t, tape)
	}

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
	if err := client.UpsertChunks(ctx, chunks); err != nil {
		t.Fatalf("UpsertChunks: %v", err)
	}

	results, err := client.HybridSearch(ctx, HybridSearchParams{
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

	results, err = client.HybridSearch(ctx, HybridSearchParams{
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

func TestIntegrationTS_VectorSearch(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	tape := maybeNewTape(t, "testdata/TS_VectorSearch.json")
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
	}

	client := testTSClient
	if tape != nil {
		client = newTapeClient(t, tape)
	}

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
	if err := client.UpsertChunks(ctx, chunks); err != nil {
		t.Fatalf("UpsertChunks: %v", err)
	}

	results, err := client.VectorSearch(ctx, HybridSearchParams{
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

func TestIntegrationTS_SearchDistinctPaths(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	tape := maybeNewTape(t, "testdata/TS_SearchDistinctPaths.json")
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
	}

	client := testTSClient
	if tape != nil {
		client = newTapeClient(t, tape)
	}

	paths := []string{"distinct/a.md", "distinct/b.md", "other/c.md"}
	for _, p := range paths {
		if err := client.UpsertChunks(ctx, makeTestChunks(p, 1)); err != nil {
			t.Fatalf("UpsertChunks %q: %v", p, err)
		}
	}

	allPaths, err := client.SearchDistinctPaths(ctx, "")
	if err != nil {
		t.Fatalf("SearchDistinctPaths: %v", err)
	}
	if len(allPaths) < 3 {
		t.Fatalf("expected >= 3 distinct paths, got %d", len(allPaths))
	}

	filtered, err := client.SearchDistinctPaths(ctx, "collection:=ts-int-test")
	if err != nil {
		t.Fatalf("SearchDistinctPaths (filtered): %v", err)
	}
	if len(filtered) < 3 {
		t.Errorf("expected >= 3 filtered paths, got %d", len(filtered))
	}
}

func TestIntegrationTS_ListDocuments(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	tape := maybeNewTape(t, "testdata/TS_ListDocuments.json")
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
	}

	client := testTSClient
	if tape != nil {
		client = newTapeClient(t, tape)
	}

	paths := []string{"list/doc1.md", "list/doc2.md"}
	for _, p := range paths {
		if err := client.UpsertChunks(ctx, makeTestChunks(p, 1)); err != nil {
			t.Fatalf("UpsertChunks %q: %v", p, err)
		}
	}

	results, err := client.ListDocuments(ctx, []string{testColl})
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(results))
	}
}

func TestIntegrationTS_SearchChunksByPath(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	tape := maybeNewTape(t, "testdata/TS_SearchChunksByPath.json")
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
	}

	client := testTSClient
	if tape != nil {
		client = newTapeClient(t, tape)
	}

	path := "search-by-path.md"
	if err := client.UpsertChunks(ctx, makeTestChunks(path, 2)); err != nil {
		t.Fatalf("UpsertChunks: %v", err)
	}

	results, err := client.SearchChunksByPath(ctx, fmt.Sprintf("path:=%s", path), 10)
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

func TestIntegrationTS_NonExistentPaths(t *testing.T) {
	requireTS(t)
	ctx := context.Background()

	tape := maybeNewTape(t, "testdata/TS_NonExistentPaths.json")
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
	}

	client := testTSClient
	if tape != nil {
		client = newTapeClient(t, tape)
	}

	count, err := client.CountByPath(ctx, "nonexistent-path.md")
	if err != nil {
		t.Fatalf("CountByPath: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	hash, err := client.GetHashByPath(ctx, "nonexistent-path.md")
	if err != nil {
		t.Fatalf("GetHashByPath: %v", err)
	}
	if hash != "" {
		t.Errorf("expected empty hash, got %q", hash)
	}

	doc, err := client.FetchDocByPath(ctx, "nonexistent-path.md")
	if err != nil {
		t.Fatalf("FetchDocByPath: %v", err)
	}
	if doc != nil {
		t.Error("expected nil for non-existent doc")
	}
}

func TestIntegrationTS_EmptyCollectionSearch(t *testing.T) {
	requireTS(t)
	ctx := context.Background()
	defer cleanupTestData(t)

	tape := maybeNewTape(t, "testdata/TS_EmptyCollectionSearch.json")
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
	}

	client := testTSClient
	if tape != nil {
		client = newTapeClient(t, tape)
	}

	results, err := client.TextSearch(ctx, HybridSearchParams{
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

	paths, err := client.SearchDistinctPaths(ctx, "collection:=ts-int-test")
	if err != nil {
		t.Fatalf("SearchDistinctPaths on empty: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected 0 paths on empty collection, got %d", len(paths))
	}
}
