package indexer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/verdverm/gmd/chunking"
	"github.com/verdverm/gmd/config"
	"github.com/verdverm/gmd/llm"
	"github.com/verdverm/gmd/ts"
)

type ProgressFn func(msg string)

type Indexer struct {
	cfg *config.Config
	ts  *ts.Client
	llm *llm.Client
}

func New(cfg *config.Config, tsClient *ts.Client, llmClient *llm.Client) *Indexer {
	return &Indexer{
		cfg: cfg,
		ts:  tsClient,
		llm: llmClient,
	}
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

	files, err := scanFiles(colPath, col.Pattern, col.Ignore)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("[%s] scan error: %v", name, err))
		return result
	}

	result.TotalFiles = len(files)

	var allChunks []ts.ChunkDocument
	var indexed int
	var skipped int

	for _, fi := range files {
		if err := ctx.Err(); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("[%s] cancelled: %v", name, err))
			break
		}

		hash, err := fileHash(fi)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("[%s] hash error %s: %v", name, fi, err))
			continue
		}

		relPath := fi
		if root != "" {
			r, err := filepath.Rel(root, fi)
			if err == nil {
				relPath = r
			}
		}

		existingHash, err := idx.ts.GetHashByPath(ctx, relPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("[%s] query error %s: %v", name, fi, err))
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
				result.Errors = append(result.Errors, fmt.Sprintf("[%s] delete error %s: %v", name, fi, err))
				continue
			}
		}

		chunks, err := idx.processFile(ctx, fi, relPath, name, hash)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("[%s] process error %s: %v", name, fi, err))
			continue
		}

		if len(chunks) > 0 {
			if err := idx.ts.UpsertChunks(ctx, chunks); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("[%s] upsert error %s: %v", name, fi, err))
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

func (idx *Indexer) processFile(ctx context.Context, absPath, relPath, collection, hash string) ([]ts.ChunkDocument, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	content := string(data)
	chunkCfg := chunking.Config{
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
			Collection:  collection,
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

func scanFiles(root, pattern string, ignore []string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("accessing collection path %s: %w", root, err)
	}
	if !info.IsDir() {
		return []string{root}, nil
	}

	ignoreSet := make(map[string]bool)
	for _, ig := range ignore {
		ignoreSet[ig] = true
	}

	var files []string

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		if rel == "." {
			return nil
		}

		for ig := range ignoreSet {
			if matched, _ := filepath.Match(ig, rel); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasPrefix(rel, ig) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.IsDir() {
			return nil
		}

		matched, err := filepath.Match(pattern, rel)
		if err != nil {
			return err
		}
		if matched {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", root, err)
	}

	return files, nil
}

func fileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h), nil
}
