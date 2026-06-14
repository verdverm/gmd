package wiki

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/testutil"
	"github.com/verdverm/gmd/pkg/ts"
)

func buildReplayLLMClient(t *testing.T, tape *testutil.Tape) *llm.Client {
	providers := map[string]llm.ProviderConfig{
		"test": {
			Name:       "test",
			BaseURL:    "https://api.openai.com/v1",
			Auth:       "apikey",
			AuthData:   map[string]string{"api_key": "test-key"},
			HTTPClient: &http.Client{Transport: tape.Transport()},
		},
	}
	profile := llm.Profile{
		Embedding:   llm.RoleConfig{Provider: "test", Model: "text-embedding-3-small"},
		Expansion:   llm.RoleConfig{Provider: "test", Model: "gpt-4o-mini"},
		Summarizing: llm.RoleConfig{Provider: "test", Model: "gpt-4o-mini"},
	}
	c, err := llm.BuildAllClients(providers, profile)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func buildReplayTSCClient(tape *testutil.Tape) *ts.Client {
	return ts.New(ts.Config{
		Host:       "http://unused",
		APIKey:     "test-key",
		HTTPClient: &http.Client{Transport: tape.Transport()},
	})
}

func TestQueryFlow_Replay(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/query_flow.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	tapedTS := buildReplayTSCClient(tape)
	tapedLLM := buildReplayLLMClient(t, tape)

	ctx := t.Context()
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
	testCfgLocal := &config.Config{
		Collections: map[string]config.CollectionConfig{
			"test-wiki": {
				SourceConfig: config.SourceConfig{Path: tmpDir},
			},
		},
	}
	agent := NewAgent(w, testCfgLocal, tapedTS, tapedLLM)

	pageRel := "entities/query-test.md"
	fullPath := filepath.Join(w.WikiPath, pageRel)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte("---\ntype: entity\n---\n\n# Query Test\n\nThis page is about machine learning and artificial intelligence.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = indexTapedWikiPage(ctx, tapedTS, tapedLLM, testCfgLocal.CollectionKey(w.Name), w.WikiPath, pageRel)
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
	t.Logf("Query answer (%d chars): %s", len(result.Answer), result.Answer[:minInt(len(result.Answer), 200)])
}

func TestIngestFlow_Replay(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/ingest_flow.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	tapedTS := buildReplayTSCClient(tape)
	tapedLLM := buildReplayLLMClient(t, tape)

	ctx := t.Context()
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
	testCfgLocal := &config.Config{
		Collections: map[string]config.CollectionConfig{
			"test-wiki": {
				SourceConfig: config.SourceConfig{Path: tmpDir},
			},
		},
	}
	agent := NewAgent(w, testCfgLocal, tapedTS, tapedLLM)

	rawPath := filepath.Join(tmpDir, "raw")
	if err := os.MkdirAll(rawPath, 0755); err != nil {
		t.Fatal(err)
	}
	sourceFile := filepath.Join(rawPath, "go-channels.md")
	sourceContent := "# Go Channels\n\nChannels are a fundamental concurrency primitive in Go. They provide a way for goroutines to communicate with each other and synchronize their execution.\n\n## Buffered vs Unbuffered\n\nUnbuffered channels block the sender until a receiver is ready. Buffered channels allow sending up to the buffer capacity before blocking.\n\n## Select Statement\n\nThe select statement lets a goroutine wait on multiple channel operations simultaneously. It picks one that is ready randomly.\n\n## Use Cases\n\nChannels are commonly used for:\n- Pipeline patterns between goroutines\n- Fan-out / fan-in patterns\n- Timeout and cancellation signals\n- Mutual exclusion via a single goroutine\n\n## Best Practices\n\nAlways close channels when done sending. Use range to receive values until the channel is closed. Avoid using channels for pure mutual exclusion — use sync.Mutex instead."
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatal(err)
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

func TestLintContentFlow_Replay(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/lint_content.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	tapedLLM := buildReplayLLMClient(t, tape)

	_, agent := newTestWikiAgent(t)
	agent.llmClient = tapedLLM

	if err := os.MkdirAll(filepath.Join(agent.wiki.WikiPath, "entities"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agent.wiki.WikiPath, "entities", "a.md"), []byte("# Page A\nMachine learning is a field of AI.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agent.wiki.WikiPath, "entities", "b.md"), []byte("# Page B\nDeep learning uses neural networks.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := &LintResult{}
	agent.lintContent(t.Context(), result)

	if len(result.Contradictions) == 0 {
		t.Log("no contradictions found (expected)")
	}
	for _, c := range result.Contradictions {
		t.Logf("  %s vs %s", c.PageA, c.PageB)
	}
}
