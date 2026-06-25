//go:build integration

package wiki

import (
	"context"
	"testing"

	"github.com/verdverm/gmd/pkg/config"
)

func TestIntegrationWikiDoctor_WithTSAndLLM(t *testing.T) {
	c := tapeTest(t, "testdata/WikiDoctor_WithTSAndLLM.json")
	defer c.Stop()

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

	result, err := Doctor(context.Background(), w, testCfg, c.TS, c.Registry)
	if err != nil {
		t.Fatalf("Doctor error: %v", err)
	}

	if result.WikiName != "test-wiki" {
		t.Errorf("WikiName = %q, want %q", result.WikiName, "test-wiki")
	}
	if !result.TSConnected {
		t.Error("TSConnected should be true with live client")
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

func TestIntegrationWikiDoctor_WithTSNilLLM(t *testing.T) {
	c := tapeTest(t, "testdata/WikiDoctor_WithTSNilLLM.json")
	defer c.Stop()

	tmpDir := t.TempDir()
	wc := &config.WikiConfig{
		SourceConfig: config.SourceConfig{Path: tmpDir},
		WikiDir:      "wiki",
		RawDir:       "raw",
		IndexFile:    "index.md",
		LogFile:      "log.md",
		GraphLinks:   true,
	}
	w, _ := NewWiki("test", tmpDir, wc)
	w.Init()

	result, err := Doctor(context.Background(), w, testCfg, c.TS, nil)
	if err != nil {
		t.Fatalf("Doctor error: %v", err)
	}
	if !result.TSConnected {
		t.Error("TSConnected should be true with live TS client")
	}
	if len(result.LLMStatus) != 0 {
		t.Errorf("expected 0 LLMStatus, got %d", len(result.LLMStatus))
	}
}
