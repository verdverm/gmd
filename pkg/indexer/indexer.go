package indexer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/verdverm/gmd/pkg/chunking"
	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/ts"
)

// TSClient defines the Typesense operations needed by the indexer.
type TSClient interface {
	GetHashByPath(ctx context.Context, path string) (string, error)
	DeleteChunksByPath(ctx context.Context, path string) error
	UpsertChunks(ctx context.Context, chunks []ts.ChunkDocument) error
	SearchDistinctPaths(ctx context.Context, filter string) ([]string, error)
}

type ProgressFn func(msg string)

type Indexer struct {
	cfg  *config.Config
	ts   TSClient
	llm  *llm.Client
	fsys fs.FS
}

func New(cfg *config.Config, tsClient TSClient, llmClient *llm.Client) *Indexer {
	return &Indexer{
		cfg: cfg,
		ts:  tsClient,
		llm: llmClient,
	}
}

func (idx *Indexer) WithFS(fsys fs.FS) *Indexer {
	idx.fsys = fsys
	return idx
}

func (idx *Indexer) rootFS() fs.FS {
	if idx.fsys != nil {
		return idx.fsys
	}
	return os.DirFS("/")
}

type fileStatus int

const (
	statusNew fileStatus = iota
	statusChanged
	statusUnchanged
)

type fileInfo struct {
	path    string
	relPath string
	hash    string
	status  fileStatus
}

type IndexResult struct {
	Collection string
	TotalFiles int
	Indexed    int
	Skipped    int
	ChunkCount int
	Errors     []string
}

func (idx *Indexer) UpdateAll(ctx context.Context, progress ProgressFn) (*IndexResult, error) {
	result := &IndexResult{}
	root := idx.cfg.ProjectRoot

	for name, col := range idx.cfg.Collections {
		colResult := idx.updateCollection(ctx, name, col, root, progress)
		result.TotalFiles += colResult.TotalFiles
		result.Indexed += colResult.Indexed
		result.Skipped += colResult.Skipped
		result.ChunkCount += colResult.ChunkCount
		result.Errors = append(result.Errors, colResult.Errors...)
	}

	return result, nil
}

func (idx *Indexer) updateCollection(ctx context.Context, name string, col config.CollectionConfig, root string, progress ProgressFn) *IndexResult {
	result := &IndexResult{Collection: name}

	colPath := col.Path
	if !filepath.IsAbs(colPath) {
		colPath = filepath.Join(root, colPath)
	}
	colPath = filepath.Clean(colPath)

	if progress != nil {
		progress(fmt.Sprintf("[%s] Scanning %s", name, colPath))
	}

	subRoot := strings.TrimLeft(colPath, "/")
	fsys := idx.rootFS()
	if subRoot != "" {
		var err error
		fsys, err = fs.Sub(fsys, subRoot)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("[%s] FS error: %v", name, err))
			return result
		}
	}

	files, err := scanFilesFS(fsys, ".", col.Pattern, col.Ignore)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("[%s] scan error: %v", name, err))
		return result
	}

	result.TotalFiles = len(files)

	colRel := colPath
	if root != "" {
		if r, err := filepath.Rel(root, colPath); err == nil {
			colRel = r
		}
	}

	var allChunks []ts.ChunkDocument
	var indexed int
	var skipped int

	for _, fi := range files {
		if err := ctx.Err(); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("[%s] cancelled: %v", name, err))
			break
		}

		hash, err := fileHashFS(fsys, fi)
		if err != nil {
			fullPath := filepath.Join(colPath, fi)
			result.Errors = append(result.Errors, fmt.Sprintf("[%s] hash error %s: %v", name, fullPath, err))
			continue
		}

		relPath := filepath.Join(colRel, fi)

		existingHash, err := idx.ts.GetHashByPath(ctx, relPath)
		if err != nil {
			fullPath := filepath.Join(colPath, fi)
			result.Errors = append(result.Errors, fmt.Sprintf("[%s] query error %s: %v", name, fullPath, err))
			continue
		}

		if existingHash == hash {
			skipped++
			continue
		}

		if progress != nil {
			action := "Indexing"
			if existingHash != "" {
				action = "Re-indexing"
			}
			progress(fmt.Sprintf("[%s] %s %s", name, action, relPath))
		}

		if existingHash != "" {
			if err := idx.ts.DeleteChunksByPath(ctx, relPath); err != nil {
				fullPath := filepath.Join(colPath, fi)
				result.Errors = append(result.Errors, fmt.Sprintf("[%s] delete error %s: %v", name, fullPath, err))
				continue
			}
		}

		chunks, err := idx.processFile(ctx, fsys, fi, relPath, name, hash)
		if err != nil {
			fullPath := filepath.Join(colPath, fi)
			result.Errors = append(result.Errors, fmt.Sprintf("[%s] process error %s: %v", name, fullPath, err))
			continue
		}

		if len(chunks) > 0 {
			if err := idx.ts.UpsertChunks(ctx, chunks); err != nil {
				fullPath := filepath.Join(colPath, fi)
				result.Errors = append(result.Errors, fmt.Sprintf("[%s] upsert error %s: %v", name, fullPath, err))
				continue
			}
		}

		allChunks = append(allChunks, chunks...)
		indexed++
	}

	result.Indexed = indexed
	result.Skipped = skipped
	result.ChunkCount = len(allChunks)

	if progress != nil {
		progress(fmt.Sprintf("[%s] Done: %d indexed, %d skipped, %d chunks", name, indexed, skipped, len(allChunks)))
	}

	return result
}

func (idx *Indexer) processFile(ctx context.Context, fsys fs.FS, fsPath, relPath, collection, hash string) ([]ts.ChunkDocument, error) {
	data, err := fs.ReadFile(fsys, fsPath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	content := string(data)
	chunkCfg := idx.chunkConfig()

	rawChunks := chunking.ChunkMarkdown(content, chunkCfg)

	texts := make([]string, len(rawChunks))
	for i, c := range rawChunks {
		texts[i] = c.Content
	}

	var embeddings [][]float64
	if len(texts) > 0 {
		var err error
		embeddings, err = idx.llm.EmbedBatch(ctx, texts)
		if err != nil {
			return nil, fmt.Errorf("embedding chunks: %w", err)
		}
	}

	docChunks := make([]ts.ChunkDocument, len(rawChunks))
	for i, c := range rawChunks {
		docChunks[i] = ts.ChunkDocument{
			Collection:  idx.cfg.CollectionKey(collection),
			Path:        relPath,
			Title:       c.Title,
			Content:     c.Content,
			Hash:        hash,
			ChunkSeq:    c.ChunkSeq,
			TotalChunks: c.TotalChunks,
		}
		if i < len(embeddings) {
			docChunks[i].Embedding = embeddings[i]
		}
	}

	return docChunks, nil
}

func (idx *Indexer) chunkConfig() chunking.Config {
	return chunking.Config{
		TargetTokens: idx.cfg.Pipeline.Chunk.TargetTokens,
		Overlap:      idx.cfg.Pipeline.Chunk.Overlap,
		HeadingWeights: chunking.HeadingWeights{
			H1: idx.cfg.Pipeline.Chunk.HeadingWeights.H1,
			H2: idx.cfg.Pipeline.Chunk.HeadingWeights.H2,
			H3: idx.cfg.Pipeline.Chunk.HeadingWeights.H3,
			H4: idx.cfg.Pipeline.Chunk.HeadingWeights.H4,
			H5: idx.cfg.Pipeline.Chunk.HeadingWeights.H5,
			H6: idx.cfg.Pipeline.Chunk.HeadingWeights.H6,
		},
		CodeFenceWeight: idx.cfg.Pipeline.Chunk.CodeFenceWeight,
		NewlineWeight:   idx.cfg.Pipeline.Chunk.NewlineWeight,
	}
}

func scanFilesFS(fsys fs.FS, root, pattern string, ignore []string) ([]string, error) {
	info, err := fs.Stat(fsys, root)
	if err != nil {
		return nil, fmt.Errorf("accessing collection path: %w", err)
	}
	if !info.IsDir() {
		return []string{root}, nil
	}

	pattern = filepath.Join(root, pattern)
	matches, err := doublestar.Glob(fsys, pattern)
	if err != nil {
		return nil, fmt.Errorf("globbing %s: %w", pattern, err)
	}
	if len(matches) == 0 {
		return nil, nil
	}

	if len(ignore) == 0 {
		return matches, nil
	}

	files := make([]string, 0, len(matches))
	for _, m := range matches {
		ignored := false
		for _, ig := range ignore {
			if matched, _ := doublestar.Match(ig, m); matched {
				ignored = true
				break
			}
			if strings.HasPrefix(m, ig) {
				ignored = true
				break
			}
		}
		if !ignored {
			files = append(files, m)
		}
	}
	return files, nil
}

func fileHashFS(fsys fs.FS, path string) (string, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h), nil
}

func fileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h), nil
}

// StalePaths finds paths indexed in Typesense that no longer exist on disk.
func (idx *Indexer) StalePaths(ctx context.Context, collection string) ([]string, error) {
	col, ok := idx.cfg.Collections[collection]
	if !ok {
		return nil, fmt.Errorf("collection %q not found", collection)
	}

	filter := fmt.Sprintf("collection:=%s", idx.cfg.CollectionKey(collection))
	indexedPaths, err := idx.ts.SearchDistinctPaths(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("searching indexed paths: %w", err)
	}

	colPath := col.Path
	if !filepath.IsAbs(colPath) {
		colPath = filepath.Join(idx.cfg.ProjectRoot, colPath)
	}
	colPath = filepath.Clean(colPath)

	var stale []string
	for _, p := range indexedPaths {
		fullPath := filepath.Join(colPath, p)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			stale = append(stale, p)
		}
	}
	return stale, nil
}

// CleanupDeleted removes all chunks for files that no longer exist on disk.
func (idx *Indexer) CleanupDeleted(ctx context.Context, collection string) (int, error) {
	stale, err := idx.StalePaths(ctx, collection)
	if err != nil {
		return 0, err
	}

	var deleted int
	for _, p := range stale {
		if err := idx.ts.DeleteChunksByPath(ctx, p); err != nil {
			return deleted, fmt.Errorf("deleting %s: %w", p, err)
		}
		deleted++
	}
	return deleted, nil
}

// CleanupAllCollections runs cleanup for all configured collections.
func (idx *Indexer) CleanupAllCollections(ctx context.Context, progress ProgressFn) map[string]int {
	results := make(map[string]int)
	for name := range idx.cfg.Collections {
		if progress != nil {
			progress(fmt.Sprintf("[%s] Cleaning up stale chunks...", name))
		}
		deleted, err := idx.CleanupDeleted(ctx, name)
		if err != nil && progress != nil {
			progress(fmt.Sprintf("[%s] Error: %v", name, err))
		}
		results[name] = deleted
	}
	return results
}
