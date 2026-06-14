package wiki

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/ts"
)

func storePath(relPath string) string {
	return filepath.Join("wiki", relPath)
}

func indexTapedWikiPage(ctx context.Context, tsClient *ts.Client, llmClient *llm.Client, collectionKey, wikiPath, relPath string) ([]ts.ChunkDocument, error) {
	fullPath := filepath.Join(wikiPath, relPath)
	os.MkdirAll(filepath.Dir(fullPath), 0755)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		data = []byte(fmt.Sprintf("---\ntype: entity\n---\n# Test\n\nContent for %s.\n", relPath))
		os.WriteFile(fullPath, data, 0644)
	}
	_, stripped, _ := ParseFrontmatter(string(data))

	if err := tsClient.DeleteChunksByPath(ctx, storePath(relPath)); err != nil {
		return nil, fmt.Errorf("delete existing: %w", err)
	}

	vec, err := llmClient.Embed(ctx, stripped)
	if err != nil {
		return nil, fmt.Errorf("embed content: %w", err)
	}

	tsPath := storePath(relPath)
	doc := ts.ChunkDocument{
		Collection:  collectionKey,
		Path:        tsPath,
		Title:       "Test",
		Content:     stripped,
		ChunkSeq:    0,
		TotalChunks: 1,
		Embedding:   vec,
	}

	if err := tsClient.UpsertChunks(ctx, []ts.ChunkDocument{doc}); err != nil {
		return nil, fmt.Errorf("upsert: %w", err)
	}

	return []ts.ChunkDocument{doc}, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func newTestWikiAgent(t *testing.T) (*Wiki, *Agent) {
	t.Helper()
	tmpDir := t.TempDir()
	w, err := NewWiki("test-wiki", tmpDir, &config.WikiConfig{
		SourceConfig: config.SourceConfig{Path: tmpDir},
		WikiDir:      "wiki",
		RawDir:       "raw",
		IndexFile:    "index.md",
		LogFile:      "log.md",
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
