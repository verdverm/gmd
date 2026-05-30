package ts

import (
	"context"
	"fmt"
	"strconv"

	"github.com/typesense/typesense-go/v4/typesense"
	"github.com/typesense/typesense-go/v4/typesense/api"
)

const (
	chunksCollection = "chunks"
	defaultNumDim    = 768
)

// Client wraps the Typesense client for GMD operations.
type Client struct {
	client *typesense.Client
	config Config
}

// Config holds Typesense connection parameters.
type Config struct {
	Host   string
	APIKey string
}

// ChunkDocument represents a single chunk indexed in Typesense.
type ChunkDocument struct {
	Collection  string    `json:"collection"`
	Path        string    `json:"path"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Hash        string    `json:"hash"`
	ChunkSeq    int       `json:"chunk_seq"`
	TotalChunks int       `json:"total_chunks"`
	Embedding   []float64 `json:"embedding"`
}

// New creates a new Typesense client wrapper.
func New(cfg Config) *Client {
	return &Client{
		client: typesense.NewClient(
			typesense.WithServer(cfg.Host),
			typesense.WithAPIKey(cfg.APIKey),
		),
		config: cfg,
	}
}

// EnsureSchema creates the chunks collection if it does not exist.
func (c *Client) EnsureSchema(ctx context.Context, numDim int) error {
	if numDim == 0 {
		numDim = defaultNumDim
	}

	collections, err := c.client.Collections().Retrieve(ctx, nil)
	if err != nil {
		return fmt.Errorf("retrieving collections: %w", err)
	}

	for _, col := range collections {
		if col.Name == chunksCollection {
			return nil
		}
	}

	schema := &api.CollectionSchema{
		Name: chunksCollection,
		Fields: []api.Field{
			{Name: "collection", Type: "string", Facet: boolPtr(true)},
			{Name: "path", Type: "string", Facet: boolPtr(true)},
			{Name: "title", Type: "string"},
			{Name: "content", Type: "string"},
			{Name: "hash", Type: "string"},
			{Name: "chunk_seq", Type: "int32"},
			{Name: "total_chunks", Type: "int32"},
			{Name: "embedding", Type: "float[]", NumDim: intPtr(numDim)},
		},
	}

	_, err = c.client.Collections().Create(ctx, schema)
	if err != nil {
		return fmt.Errorf("creating chunks collection: %w", err)
	}

	return nil
}

// UpsertChunks inserts or replaces a batch of chunk documents for a given path.
func (c *Client) UpsertChunks(ctx context.Context, chunks []ChunkDocument) error {
	for _, ch := range chunks {
		_, err := c.client.Collection(chunksCollection).Documents().Upsert(ctx, ch, nil)
		if err != nil {
			return fmt.Errorf("upserting chunk %s: %w", ch.Path, err)
		}
	}
	return nil
}

// DeleteChunksByPath removes all chunks for a given file path.
func (c *Client) DeleteChunksByPath(ctx context.Context, path string) error {
	filter := fmt.Sprintf("path:=%s", path)
	_, err := c.client.Collection(chunksCollection).Documents().Delete(ctx, &api.DeleteDocumentsParams{
		FilterBy: &filter,
	})
	if err != nil {
		return fmt.Errorf("deleting chunks for %s: %w", path, err)
	}
	return nil
}

// HybridSearchResult holds a single grouped search result.
type HybridSearchResult struct {
	Collection string  `json:"collection"`
	Path       string  `json:"path"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	ChunkSeq   int     `json:"chunk_seq"`
	Score      float64 `json:"score"`
}

// HybridSearchParams holds parameters for a hybrid search query.
type HybridSearchParams struct {
	Query       string
	QueryVector []float64
	Collections []string
	Limit       int
	GroupLimit  int
}

// HybridSearch performs a hybrid (text + vector) search grouped by collection and path.
func (c *Client) HybridSearch(ctx context.Context, params HybridSearchParams) ([]HybridSearchResult, error) {
	searchParams := &api.SearchCollectionParams{
		Q:          &params.Query,
		QueryBy:    stringPtr("content"),
		GroupBy:    stringPtr("collection,path"),
		GroupLimit: intPtr(params.GroupLimit),
		PerPage:    intPtr(params.Limit),
	}

	if len(params.QueryVector) > 0 {
		vec := fmt.Sprintf("embedding:(%v)", formatVector(params.QueryVector))
		searchParams.VectorQuery = &vec
	}

	if len(params.Collections) > 0 {
		filter := buildCollectionFilter(params.Collections)
		searchParams.FilterBy = &filter
	}

	resp, err := c.client.Collection(chunksCollection).Documents().Search(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("hybrid search: %w", err)
	}

	return groupedHitsToResults(resp), nil
}

// TextSearch performs a text-only search (no vector).
func (c *Client) TextSearch(ctx context.Context, params HybridSearchParams) ([]HybridSearchResult, error) {
	searchParams := &api.SearchCollectionParams{
		Q:          &params.Query,
		QueryBy:    stringPtr("content"),
		GroupBy:    stringPtr("collection,path"),
		GroupLimit: intPtr(params.GroupLimit),
		PerPage:    intPtr(params.Limit),
	}

	if len(params.Collections) > 0 {
		filter := buildCollectionFilter(params.Collections)
		searchParams.FilterBy = &filter
	}

	resp, err := c.client.Collection(chunksCollection).Documents().Search(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("text search: %w", err)
	}

	return groupedHitsToResults(resp), nil
}

func groupedHitsToResults(resp *api.SearchResult) []HybridSearchResult {
	if resp == nil || resp.GroupedHits == nil {
		return nil
	}

	var results []HybridSearchResult
	for _, group := range *resp.GroupedHits {
		if len(group.Hits) == 0 {
			continue
		}
		hit := group.Hits[0]
		r := hitToResult(hit)
		results = append(results, r)
	}
	return results
}

func hitToResult(hit api.SearchResultHit) HybridSearchResult {
	var r HybridSearchResult
	doc := hit.Document
	if doc == nil {
		return r
	}
	if v, ok := (*doc)["collection"]; ok {
		r.Collection = fmt.Sprint(v)
	}
	if v, ok := (*doc)["path"]; ok {
		r.Path = fmt.Sprint(v)
	}
	if v, ok := (*doc)["title"]; ok {
		r.Title = fmt.Sprint(v)
	}
	if v, ok := (*doc)["content"]; ok {
		r.Content = fmt.Sprint(v)
	}
	if v, ok := (*doc)["chunk_seq"]; ok {
		r.ChunkSeq = int(toFloat64(v))
	}
	if hit.TextMatchInfo != nil && hit.TextMatchInfo.Score != nil {
		if s, err := strconv.ParseFloat(*hit.TextMatchInfo.Score, 64); err == nil {
			r.Score = s
		}
	}
	return r
}

func buildCollectionFilter(collections []string) string {
	filter := "collection:=["
	for i, c := range collections {
		if i > 0 {
			filter += ","
		}
		filter += c
	}
	filter += "]"
	return filter
}

func formatVector(v []float64) string {
	if len(v) == 0 {
		return ""
	}
	b := make([]byte, 0, len(v)*10)
	for i, f := range v {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, []byte(fmt.Sprintf("%f", f))...)
	}
	return string(b)
}

func boolPtr(b bool) *bool       { return &b }
func intPtr(i int) *int          { return &i }
func stringPtr(s string) *string { return &s }

func toFloat64(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case uint64:
		return float64(x)
	case uint32:
		return float64(x)
	case int32:
		return float64(x)
	default:
		return 0
	}
}
