package ts

import (
	"net/http"
	"testing"

	"github.com/verdverm/gmd/pkg/testutil"
)

func replayClient(tape *testutil.Tape) *Client {
	return New(Config{
		Host:       "http://unused",
		APIKey:     "test-key",
		HTTPClient: &http.Client{Transport: tape.Transport()},
	})
}

func TestReplayDemo(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/replay_demo.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)

	count, err := client.CountByPath(t.Context(), "chunk-crud.md")
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}

	_, err = client.CountByPath(t.Context(), "another.md")
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

//nolint:gocyclo
func TestReplayChunkCRUD(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/001_chunk_crud.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)
	ctx := t.Context()

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

	fetched, err := client.FetchChunksByPath(ctx, path)
	if err != nil {
		t.Fatalf("FetchChunksByPath: %v", err)
	}
	if len(fetched) != 3 {
		t.Errorf("FetchChunksByPath returned %d chunks, want 3", len(fetched))
	}
	for i, f := range fetched {
		if f.ChunkSeq != i {
			t.Errorf("fetched[%d].ChunkSeq = %d, want %d", i, f.ChunkSeq, i)
		}
		if f.Path != path {
			t.Errorf("fetched[%d].Path = %q, want %q", i, f.Path, path)
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

	count, err = client.CountByPath(ctx, path)
	if err != nil {
		t.Fatalf("CountByPath: %v", err)
	}
	if count != 0 {
		t.Errorf("after collection delete, CountByPath = %d, want 0", count)
	}

	_, err = client.CountByPath(ctx, "extra.md")
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

func TestReplayTextSearch(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/002_text_search.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)
	ctx := t.Context()

	searchChunks := []ChunkDocument{
		{Collection: testColl, Path: "search-text.md", Title: "Search test", Content: "The quick brown fox jumps over the lazy dog.", Hash: "hash-st-0", ChunkSeq: 0, TotalChunks: 2, Embedding: []float64{0.1, 0.2, 0.3, 0.4}},
		{Collection: testColl, Path: "search-text.md", Title: "Search test continued", Content: "The lazy dog sleeps peacefully all day long.", Hash: "hash-st-1", ChunkSeq: 1, TotalChunks: 2, Embedding: []float64{0.5, 0.6, 0.7, 0.8}},
	}
	if err := client.UpsertChunks(ctx, searchChunks); err != nil {
		t.Fatalf("UpsertChunks: %v", err)
	}

	results, err := client.TextSearch(ctx, HybridSearchParams{
		Query: "fox", Collections: []string{testColl}, GroupLimit: 10, Limit: 10,
	})
	if err != nil {
		t.Fatalf("TextSearch: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("TextSearch returned 0 results for 'fox'")
	}

	results, err = client.TextSearch(ctx, HybridSearchParams{
		Query: "nonexistenttermzzz", Collections: []string{testColl}, GroupLimit: 10, Limit: 10,
	})
	if err != nil {
		t.Fatalf("TextSearch (no match): %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}

	results, err = client.TextSearch(ctx, HybridSearchParams{
		Query: "", FilterBy: "collection:=ts-int-test", GroupLimit: 10, Limit: 10,
	})
	if err != nil {
		t.Fatalf("TextSearch with FilterBy: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("TextSearch with FilterBy returned 0 results")
	}

	_, err = client.TextSearch(ctx, HybridSearchParams{
		Query: "extra", Collections: []string{testColl}, GroupLimit: 10, Limit: 10,
	})
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

func TestReplayHybridSearch(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/003_hybrid_search.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)
	ctx := t.Context()

	chunks := []ChunkDocument{
		{Collection: testColl, Path: "hybrid-test.md", Title: "Hybrid test", Content: "Alpha beta gamma delta epsilon.", Hash: "hash-hybrid-0", ChunkSeq: 0, TotalChunks: 1, Embedding: []float64{0.9, 0.1, 0.1, 0.1}},
	}
	if err := client.UpsertChunks(ctx, chunks); err != nil {
		t.Fatalf("UpsertChunks: %v", err)
	}

	results, err := client.HybridSearch(ctx, HybridSearchParams{
		Query: "alpha", QueryVector: []float64{0.8, 0.2, 0.1, 0.1}, Collections: []string{testColl}, GroupLimit: 10, Limit: 10,
	})
	if err != nil {
		t.Fatalf("HybridSearch (with vector): %v", err)
	}
	if len(results) == 0 {
		t.Fatal("HybridSearch with vector returned 0 results")
	}

	results, err = client.HybridSearch(ctx, HybridSearchParams{
		Query: "beta", Collections: []string{testColl}, GroupLimit: 10, Limit: 10,
	})
	if err != nil {
		t.Fatalf("HybridSearch (text-only): %v", err)
	}
	if len(results) == 0 {
		t.Fatal("HybridSearch text-only returned 0 results")
	}

	_, err = client.HybridSearch(ctx, HybridSearchParams{
		Query: "extra", Collections: []string{testColl}, GroupLimit: 10, Limit: 10,
	})
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

func TestReplayVectorSearch(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/004_vector_search.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)
	ctx := t.Context()

	chunks := []ChunkDocument{
		{Collection: testColl, Path: "vector-test.md", Title: "Vector test", Content: "Vector similarity search content.", Hash: "hash-vector-0", ChunkSeq: 0, TotalChunks: 1, Embedding: []float64{0.9, 0.1, 0.1, 0.1}},
	}
	if err := client.UpsertChunks(ctx, chunks); err != nil {
		t.Fatalf("UpsertChunks: %v", err)
	}

	results, err := client.VectorSearch(ctx, HybridSearchParams{
		QueryVector: []float64{0.85, 0.15, 0.1, 0.1}, Collections: []string{testColl}, GroupLimit: 10, Limit: 10,
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

	_, err = client.VectorSearch(ctx, HybridSearchParams{
		QueryVector: []float64{0.1, 0.1, 0.1, 0.1}, Collections: []string{testColl}, GroupLimit: 10, Limit: 10,
	})
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

//nolint:gocyclo
func TestReplayDocCRUD(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/005_doc_crud.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)
	ctx := t.Context()
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
	fetched, err = client.FetchDocByPath(ctx, path)
	if err != nil {
		t.Fatalf("FetchDocByPath: %v", err)
	}
	if fetched != nil {
		t.Error("expected nil after DeleteDocsByCollection")
	}

	_, err = client.FetchDocByPath(ctx, "extra.md")
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

func TestReplayEmptyCollectionSearch(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/006_empty_results.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)
	ctx := t.Context()

	results, err := client.TextSearch(ctx, HybridSearchParams{
		Query: "anything", Collections: []string{testColl}, GroupLimit: 10, Limit: 10,
	})
	if err != nil {
		t.Fatalf("TextSearch: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}

	paths, err := client.SearchDistinctPaths(ctx, "collection:=ts-int-test")
	if err != nil {
		t.Fatalf("SearchDistinctPaths: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected 0 paths, got %d", len(paths))
	}

	_, err = client.SearchDistinctPaths(ctx, "collection:=ts-int-test")
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

func TestReplayGetSchemaFields(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/007_schema_fields.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)
	ctx := t.Context()

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

	_, err = client.GetSchemaFields(ctx)
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

func TestReplayExtraFields(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/008_extra_fields.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)
	ctx := t.Context()

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

	_, err = client.GetSchemaFields(ctx)
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

func TestReplayChunkDynamicFields(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/009_dynamic_fields.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)
	ctx := t.Context()

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

	err = client.UpsertChunks(ctx, []ChunkDocument{chunk})
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

func TestReplayChunkLinks(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/010_chunk_links.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)
	ctx := t.Context()

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

	_, err = client.FetchChunksByPath(ctx, "links-chunk.md")
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

func TestReplayDocLinks(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/011_doc_links.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)
	ctx := t.Context()

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

	_, err = client.FetchDocByPath(ctx, "doc-with-links.md")
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

func TestReplayFetchDocs(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/012_fetch_docs.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)
	ctx := t.Context()

	paths := []string{"alpha/one.md", "alpha/two.md", "beta/one.md"}
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

	_, err = client.FetchDocs(ctx, "extra")
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

func TestReplaySearchDistinctPaths(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/013_search_distinct_paths.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)
	ctx := t.Context()

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

	_, err = client.SearchDistinctPaths(ctx, "")
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

func TestReplayListDocuments(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/014_list_documents.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)
	ctx := t.Context()

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

	_, err = client.ListDocuments(ctx, []string{testColl})
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

func TestReplaySearchChunksByPath(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/015_search_chunks_by_path.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)
	ctx := t.Context()

	path := "search-by-path.md"
	if err := client.UpsertChunks(ctx, makeTestChunks(path, 2)); err != nil {
		t.Fatalf("UpsertChunks: %v", err)
	}

	results, err := client.SearchChunksByPath(ctx, "path:=search-by-path.md", 10)
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

	_, err = client.SearchChunksByPath(ctx, "path:=search-by-path.md", 10)
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

func TestReplayNonExistentPaths(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/016_non_existent_paths.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := replayClient(tape)
	ctx := t.Context()

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

	_, err = client.CountByPath(ctx, "nonexistent-path.md")
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}
