//go:build integration

package wiki

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/testutil"
	"github.com/verdverm/gmd/pkg/ts"
)

func maybeNewTape(t *testing.T, filePath string) *testutil.Tape {
	t.Helper()
	if os.Getenv("GMD_NORECORD") == "1" {
		return nil
	}
	return testutil.NewTape(filePath, "", nil, testutil.ModeRecord)
}

func buildTapedRegistry(t *testing.T, tape *testutil.Tape) *llm.Registry {
	t.Helper()
	var opts []llm.RegistryOption
	for name := range testCfg.LLM.Providers {
		opts = append(opts, llm.WithProviderTransport(name, &http.Client{Transport: tape.Transport()}))
	}
	reg, err := llm.NewRegistry(context.Background(), testCfg, opts...)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	return reg
}

func buildTapedTSCWikiClient(t *testing.T, tape *testutil.Tape) *ts.Client {
	t.Helper()
	return ts.New(ts.Config{
		Host:       TestTSSrvURL,
		APIKey:     TestTSSrvKey,
		HTTPClient: &http.Client{Transport: tape.Transport()},
	})
}

func TestIntegrationQueryFlow_Record(t *testing.T) {
	requireTSServices(t)
	requireLLMServices(t)

	ctx := context.Background()

	tape := maybeNewTape(t, "testdata/query_flow.json")

	var (
		tsClient  *ts.Client
		embedder  llm.Embedder
		chatModel llm.ChatModel
	)

	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
		tsClient = buildTapedTSCWikiClient(t, tape)
		reg := buildTapedRegistry(t, tape)
		embedder = reg.Embedder()
		chatModel = reg.Model(llm.RoleGeneralBig)
	} else {
		tsClient = testTSClient
		embedder = testRegistry.Embedder()
		chatModel = testRegistry.Model(llm.RoleGeneralBig)
	}

	defer func() {
		if err := tsClient.DeleteChunksByCollection(ctx, testCollKey); err != nil {
			t.Logf("cleanup: %v", err)
		}
	}()

	tmpDir := t.TempDir()
	wc := &config.WikiConfig{
		SourceConfig: config.SourceConfig{Path: tmpDir},
		WikiDir:      "wiki",
		RawDir:       "raw",
		IndexFile:    "index.md",
		LogFile:      "log.md",
		GraphLinks:   true,
	}
	w, err := NewWiki("test-wiki", tmpDir, wc)
	if err != nil {
		t.Fatalf("NewWiki error: %v", err)
	}
	if err := w.Init(); err != nil {
		t.Fatalf("Init error: %v", err)
	}
	agent := NewAgent(w, testCfg, tsClient, chatModel)

	pageRel := "entities/query-test.md"
	fullPath := filepath.Join(w.WikiPath, pageRel)
	os.MkdirAll(filepath.Dir(fullPath), 0755)
	pageContent := "---\ntype: entity\n---\n\n# Query Test\n\nThis page is about machine learning and artificial intelligence.\n"
	os.WriteFile(fullPath, []byte(pageContent), 0644)

	_, err = indexTapedWikiPage(ctx, tsClient, embedder, testCfg.CollectionKey(w.Name), w.WikiPath, pageRel)
	if err != nil {
		t.Fatalf("indexTapedWikiPage error: %v", err)
	}

	result, err := agent.Query(ctx, "machine learning", QueryOpts{Save: false, Limit: 3})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if result.Answer == "" {
		t.Error("expected non-empty answer")
	}
}

func TestIntegrationIngestFlow_Record(t *testing.T) {
	requireTSServices(t)
	requireLLMServices(t)

	tape := maybeNewTape(t, "testdata/ingest_flow.json")
	if tape == nil {
		t.Skip("GMD_NORECORD=1, skipping tape recording")
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	tapedTS := buildTapedTSCWikiClient(t, tape)
	tapedRegistry := buildTapedRegistry(t, tape)

	ctx := context.Background()
	defer func() {
		if err := tapedTS.DeleteChunksByCollection(ctx, testCollKey); err != nil {
			t.Logf("cleanup: %v", err)
		}
	}()

	tmpDir := t.TempDir()
	wc := &config.WikiConfig{
		SourceConfig: config.SourceConfig{Path: tmpDir},
		WikiDir:      "wiki",
		RawDir:       "raw",
		IndexFile:    "index.md",
		LogFile:      "log.md",
		GraphLinks:   true,
	}
	w, err := NewWiki("test-wiki", tmpDir, wc)
	if err != nil {
		t.Fatalf("NewWiki error: %v", err)
	}
	if err := w.Init(); err != nil {
		t.Fatalf("Init error: %v", err)
	}
	agent := NewAgent(w, testCfg, tapedTS, tapedRegistry.Model(llm.RoleGeneralBig))

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
}

func TestIntegrationLintContentFlow_Record(t *testing.T) {
	requireLLMServices(t)

	tape := maybeNewTape(t, "testdata/lint_content.json")
	if tape == nil {
		t.Skip("GMD_NORECORD=1, skipping tape recording")
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	tapedRegistry := buildTapedRegistry(t, tape)

	_, agent := newTestWikiAgent(t)
	agent.chat = tapedRegistry.Model(llm.RoleGeneralBig)

	os.MkdirAll(filepath.Join(agent.wiki.WikiPath, "entities"), 0755)
	os.WriteFile(filepath.Join(agent.wiki.WikiPath, "entities", "a.md"), []byte("# Page A\nMachine learning is a field of AI.\n"), 0644)
	os.WriteFile(filepath.Join(agent.wiki.WikiPath, "entities", "b.md"), []byte("# Page B\nDeep learning uses neural networks.\n"), 0644)

	result := &LintResult{}
	agent.lintContent(context.Background(), result)

	t.Logf("lintContent found %d contradictions", len(result.Contradictions))
	for _, c := range result.Contradictions {
		t.Logf("  %s vs %s", c.PageA, c.PageB)
	}
}
