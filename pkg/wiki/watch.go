package wiki

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/verdverm/gmd/pkg/chunking"
	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/ts"
)

type Watcher struct {
	wiki      *Wiki
	cfg       *config.Config
	tsClient  *ts.Client
	llmClient *llm.Client
	agent     *Agent
}

func NewWatcher(wiki *Wiki, cfg *config.Config, tsClient *ts.Client, llmClient *llm.Client) *Watcher {
	return &Watcher{
		wiki:      wiki,
		cfg:       cfg,
		tsClient:  tsClient,
		llmClient: llmClient,
		agent:     NewAgent(wiki, cfg, tsClient, llmClient),
	}
}

func (w *Watcher) Watch(ctx context.Context) error {
	fmt.Println("Wiki watch mode started. Watching raw/ and wiki/ for changes...")
	fmt.Println("Press Ctrl+C to stop.")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			w.checkRaw(ctx)
			w.checkWiki(ctx)
		}
	}
}

func (w *Watcher) checkRaw(ctx context.Context) {
	entries, err := os.ReadDir(w.wiki.RawPath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") && !strings.HasSuffix(entry.Name(), ".txt") {
			continue
		}

		report, err := w.agent.Ingest(ctx, entry.Name(), IngestOpts{Batch: false})
		if err != nil {
			fmt.Printf("  ingest error: %v\n", err)
			continue
		}

		created := len(report.CreatedPages)
		updated := len(report.UpdatedPages)
		flagged := len(report.Contradictions)

		if created+updated > 0 {
			fmt.Printf("  Ingested %s -> +%d pages, ~%d, !%d\n",
				entry.Name(), created, updated, flagged)
		}
	}
}

func (w *Watcher) checkWiki(ctx context.Context) {
	filepath.Walk(w.wiki.WikiPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		if strings.HasPrefix(filepath.Base(path), "_") {
			return nil
		}

		return nil
	})
}

func indexFile(ctx context.Context, tsClient *ts.Client, cfg *config.Config, wikiName string, relPath string) error {
	fullPath := filepath.Join(cfg.ProjectRoot, relPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	content := string(data)

	_, stripped, _ := ParseFrontmatter(content)
	links := chunking.ExtractWikilinks(stripped)
	chunks := chunking.ChunkMarkdown(stripped, chunking.DefaultConfig())

	for i := range chunks {
		chunks[i].Links = links
	}

	collectionKey := cfg.CollectionKey(wikiName)

	_ = tsClient.DeleteChunksByPath(ctx, relPath)
	_ = tsClient.DeleteDocByPath(ctx, relPath)

	var tsDocs []ts.ChunkDocument
	var title string
	for i, c := range chunks {
		if i == 0 {
			title = c.Title
		}
		tsDocs = append(tsDocs, ts.ChunkDocument{
			Collection:  collectionKey,
			Path:        relPath,
			Title:       c.Title,
			Content:     c.Content,
			ChunkSeq:    c.ChunkSeq,
			TotalChunks: c.TotalChunks,
		})
	}

	if len(tsDocs) > 0 {
		if err := tsClient.UpsertChunks(ctx, tsDocs); err != nil {
			return err
		}
	}

	return tsClient.UpsertDoc(ctx, ts.DocDocument{
		Collection: collectionKey,
		Path:       relPath,
		Title:      title,
		Content:    stripped,
		Links:      links,
	})
}
