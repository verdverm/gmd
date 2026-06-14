package ts

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/typesense/typesense-go/v4/typesense"
	"github.com/typesense/typesense-go/v4/typesense/api"
)

// SchemaField defines a custom field to add to the Typesense chunks collection schema.
type SchemaField struct {
	Name  string
	Type  string
	Facet bool
	Sort  bool
}

const (
	chunksCollection = "chunks"
	docsCollection   = "documents"
	defaultNumDim    = 768
)

// Client wraps the Typesense client for GMD operations.
type Client struct {
	client *typesense.Client
	config Config
}

// Config holds Typesense connection parameters.
type Config struct {
	Host       string
	APIKey     string
	HTTPClient *http.Client
}

// ChunkDocument represents a single chunk indexed in Typesense.
type ChunkDocument struct {
	Collection  string                 `json:"collection"`
	Path        string                 `json:"path"`
	Title       string                 `json:"title"`
	Content     string                 `json:"content"`
	Hash        string                 `json:"hash"`
	ChunkSeq    int                    `json:"chunk_seq"`
	TotalChunks int                    `json:"total_chunks"`
	Embedding   []float64              `json:"embedding"`
	Links       []string               `json:"links,omitempty"`
	Fields      map[string]interface{} `json:"-"` // dynamic frontmatter fields, merged at upsert time
}

// ToMap converts the document (including dynamic Fields) to a flat map for Typesense upsert.
func (d *ChunkDocument) ToMap() map[string]interface{} {
	m := map[string]interface{}{
		"collection":   d.Collection,
		"path":         d.Path,
		"title":        d.Title,
		"content":      d.Content,
		"hash":         d.Hash,
		"chunk_seq":    d.ChunkSeq,
		"total_chunks": d.TotalChunks,
		"embedding":    d.Embedding,
	}
	if d.Links != nil {
		m["links"] = d.Links
	}
	for k, v := range d.Fields {
		m[k] = v
	}
	return m
}

// DocDocument represents a full document indexed in Typesense (not chunked).
type DocDocument struct {
	Collection string                 `json:"collection"`
	Path       string                 `json:"path"`
	Title      string                 `json:"title"`
	Content    string                 `json:"content"`
	Hash       string                 `json:"hash"`
	Links      []string               `json:"links,omitempty"`
	Fields     map[string]interface{} `json:"-"`
}

// ToMap converts the DocDocument (including dynamic Fields) to a flat map for Typesense upsert.
func (d *DocDocument) ToMap() map[string]interface{} {
	m := map[string]interface{}{
		"collection": d.Collection,
		"path":       d.Path,
		"title":      d.Title,
		"content":    d.Content,
		"hash":       d.Hash,
	}
	if d.Links != nil {
		m["links"] = d.Links
	}
	for k, v := range d.Fields {
		m[k] = v
	}
	return m
}

// New creates a new Typesense client wrapper.
func New(cfg Config) *Client {
	opts := []typesense.ClientOption{
		typesense.WithServer(cfg.Host),
		typesense.WithAPIKey(cfg.APIKey),
	}
	if cfg.HTTPClient != nil {
		opts = append(opts, typesense.WithCustomHTTPClient(cfg.HTTPClient))
	}
	return &Client{
		client: typesense.NewClient(opts...),
		config: cfg,
	}
}

// EnsureSchema creates the chunks collection (or adds missing fields) with base fields
// plus any extra fields derived from collection-level frontmatter configs.
func (c *Client) EnsureSchema(ctx context.Context, embedDim int, extraFields []SchemaField) error {
	if embedDim == 0 {
		embedDim = defaultNumDim
	}
	return c.ensureCollection(ctx, chunksCollection, buildChunkFields(embedDim, extraFields))
}

// EnsureDocSchema creates the documents collection (or adds missing fields).
func (c *Client) EnsureDocSchema(ctx context.Context, extraFields []SchemaField) error {
	return c.ensureCollection(ctx, docsCollection, buildDocFields(extraFields))
}

// EnsureAllSchemas ensures both the chunks and documents collections exist.
func (c *Client) EnsureAllSchemas(ctx context.Context, embedDim int, extraFields []SchemaField) error {
	if err := c.EnsureSchema(ctx, embedDim, extraFields); err != nil {
		return err
	}
	return c.EnsureDocSchema(ctx, extraFields)
}

// ensureCollection creates or updates a single Typesense collection.
func (c *Client) ensureCollection(ctx context.Context, name string, desired []api.Field) error {
	collections, err := c.client.Collections().Retrieve(ctx, nil)
	if err != nil {
		return fmt.Errorf("retrieving collections: %w", err)
	}

	for _, col := range collections {
		if col.Name == name {
			return c.updateCollectionSchema(ctx, name, col.Fields, desired)
		}
	}

	return c.createCollection(ctx, name, desired)
}

// buildChunkFields returns the field schema for the chunks collection.
func buildChunkFields(embedDim int, extraFields []SchemaField) []api.Field {
	fields := []api.Field{
		{Name: "collection", Type: "string", Facet: boolPtr(true)},
		{Name: "path", Type: "string", Facet: boolPtr(true)},
		{Name: "title", Type: "string"},
		{Name: "content", Type: "string"},
		{Name: "hash", Type: "string"},
		{Name: "chunk_seq", Type: "int32"},
		{Name: "total_chunks", Type: "int32"},
		{Name: "embedding", Type: "float[]", NumDim: intPtr(embedDim)},
		{Name: "links", Type: "string[]", Facet: boolPtr(true), Optional: boolPtr(true)},
	}
	for _, sf := range extraFields {
		fields = append(fields, api.Field{
			Name:     sf.Name,
			Type:     sf.Type,
			Facet:    boolPtr(sf.Facet),
			Sort:     boolPtr(sf.Sort),
			Optional: boolPtr(true),
		})
	}
	return fields
}

// buildDocFields returns the field schema for the documents collection.
func buildDocFields(extraFields []SchemaField) []api.Field {
	fields := []api.Field{
		{Name: "collection", Type: "string", Facet: boolPtr(true)},
		{Name: "path", Type: "string", Facet: boolPtr(true)},
		{Name: "title", Type: "string"},
		{Name: "content", Type: "string"},
		{Name: "hash", Type: "string"},
		{Name: "links", Type: "string[]", Facet: boolPtr(true), Optional: boolPtr(true)},
	}
	for _, sf := range extraFields {
		fields = append(fields, api.Field{
			Name:     sf.Name,
			Type:     sf.Type,
			Facet:    boolPtr(sf.Facet),
			Sort:     boolPtr(sf.Sort),
			Optional: boolPtr(true),
		})
	}
	return fields
}

// updateCollectionSchema adds any desired fields that don't already exist on the collection.
func (c *Client) updateCollectionSchema(ctx context.Context, name string, existing []api.Field, desired []api.Field) error {
	existingMap := make(map[string]bool)
	for _, f := range existing {
		existingMap[f.Name] = true
	}
	var newFields []api.Field
	for _, f := range desired {
		if !existingMap[f.Name] {
			newFields = append(newFields, f)
		}
	}
	if len(newFields) == 0 {
		return nil
	}
	_, err := c.client.Collection(name).Update(ctx, &api.CollectionUpdateSchema{Fields: newFields})
	if err != nil {
		return fmt.Errorf("updating schema: %w", err)
	}
	return nil
}

// createCollection creates a new Typesense collection with the given fields.
func (c *Client) createCollection(ctx context.Context, name string, fields []api.Field) error {
	_, err := c.client.Collections().Create(ctx, &api.CollectionSchema{
		Name:   name,
		Fields: fields,
	})
	if err != nil {
		return fmt.Errorf("creating %s collection: %w", name, err)
	}
	return nil
}

// UpsertChunks inserts or replaces a batch of chunk documents.
// Dynamic Fields on each ChunkDocument are merged into the Typesense document.
func (c *Client) UpsertChunks(ctx context.Context, chunks []ChunkDocument) error {
	for _, ch := range chunks {
		doc := ch.ToMap()
		_, err := c.client.Collection(chunksCollection).Documents().Upsert(ctx, doc, &api.DocumentIndexParameters{})
		if err != nil {
			return fmt.Errorf("upserting chunk %s: %w", ch.Path, err)
		}
	}
	return nil
}

// UpsertDoc inserts or replaces a single full document.
func (c *Client) UpsertDoc(ctx context.Context, doc DocDocument) error {
	_, err := c.client.Collection(docsCollection).Documents().Upsert(ctx, doc.ToMap(), &api.DocumentIndexParameters{})
	if err != nil {
		return fmt.Errorf("upserting doc %s: %w", doc.Path, err)
	}
	return nil
}

// FetchDocByPath retrieves the full document for a given path from the documents collection.
func (c *Client) FetchDocByPath(ctx context.Context, path string) (*DocDocument, error) {
	searchParams := &api.SearchCollectionParams{
		Q:        stringPtr(""),
		QueryBy:  stringPtr("content"),
		FilterBy: stringPtr(fmt.Sprintf("path:=%s", path)),
		PerPage:  intPtr(1),
	}

	resp, err := c.client.Collection(docsCollection).Documents().Search(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("fetching doc %s: %w", path, err)
	}
	if resp.Hits == nil || len(*resp.Hits) == 0 {
		return nil, nil
	}

	hit := (*resp.Hits)[0]
	if hit.Document == nil {
		return nil, nil
	}
	docMap := *hit.Document

	doc := &DocDocument{}
	if v, ok := docMap["collection"]; ok {
		doc.Collection = fmt.Sprint(v)
	}
	if v, ok := docMap["path"]; ok {
		doc.Path = fmt.Sprint(v)
	}
	if v, ok := docMap["title"]; ok {
		doc.Title = fmt.Sprint(v)
	}
	if v, ok := docMap["content"]; ok {
		doc.Content = fmt.Sprint(v)
	}
	if v, ok := docMap["hash"]; ok {
		doc.Hash = fmt.Sprint(v)
	}
	if v, ok := docMap["links"]; ok {
		switch links := v.(type) {
		case []interface{}:
			for _, l := range links {
				doc.Links = append(doc.Links, fmt.Sprint(l))
			}
		}
	}
	return doc, nil
}

// FetchDocs retrieves documents matching a path query.
// If the query contains glob characters (* ? [), it's treated as a path filter.
// Otherwise, exact match is attempted first, then prefix match.
func (c *Client) FetchDocs(ctx context.Context, query string) ([]DocDocument, error) {
	if hasGlobChars(query) {
		return c.searchDocsByPattern(ctx, query)
	}

	doc, err := c.FetchDocByPath(ctx, query)
	if err != nil {
		return nil, err
	}
	if doc != nil {
		return []DocDocument{*doc}, nil
	}

	return c.searchDocsByPattern(ctx, query+"*")
}

// searchDocsByPattern searches the documents collection with a path filter.
func (c *Client) searchDocsByPattern(ctx context.Context, filter string) ([]DocDocument, error) {
	perPage := 250
	page := 1
	f := fmt.Sprintf("path:%s", filter)
	var all []DocDocument

	for {
		searchParams := &api.SearchCollectionParams{
			Q:        stringPtr(""),
			QueryBy:  stringPtr("content"),
			FilterBy: &f,
			PerPage:  intPtr(perPage),
			Page:     &page,
		}

		resp, err := c.client.Collection(docsCollection).Documents().Search(ctx, searchParams)
		if err != nil {
			return nil, fmt.Errorf("searching docs by pattern: %w", err)
		}

		if resp.Hits == nil || len(*resp.Hits) == 0 {
			break
		}

		for _, hit := range *resp.Hits {
			if hit.Document == nil {
				continue
			}
			docMap := *hit.Document
			var doc DocDocument
			if v, ok := docMap["collection"]; ok {
				doc.Collection = fmt.Sprint(v)
			}
			if v, ok := docMap["path"]; ok {
				doc.Path = fmt.Sprint(v)
			}
			if v, ok := docMap["title"]; ok {
				doc.Title = fmt.Sprint(v)
			}
			if v, ok := docMap["content"]; ok {
				doc.Content = fmt.Sprint(v)
			}
			if v, ok := docMap["hash"]; ok {
				doc.Hash = fmt.Sprint(v)
			}
			if v, ok := docMap["links"]; ok {
				switch links := v.(type) {
				case []interface{}:
					for _, l := range links {
						doc.Links = append(doc.Links, fmt.Sprint(l))
					}
				}
			}
			all = append(all, doc)
		}

		if len(*resp.Hits) < perPage {
			break
		}
		page++
	}

	return all, nil
}

func hasGlobChars(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

// DeleteDocByPath removes a full document for a given path from the documents collection.
func (c *Client) DeleteDocByPath(ctx context.Context, path string) error {
	filter := fmt.Sprintf("path:=%s", path)
	batchSize := 10000
	_, err := c.client.Collection(docsCollection).Documents().Delete(ctx, &api.DeleteDocumentsParams{
		FilterBy:  &filter,
		BatchSize: &batchSize,
	})
	if err != nil {
		return fmt.Errorf("deleting doc %s: %w", path, err)
	}
	return nil
}

// DeleteDocsByCollection removes all documents for a given collection name from the documents collection.
func (c *Client) DeleteDocsByCollection(ctx context.Context, name string) error {
	filter := fmt.Sprintf("collection:=%s", name)
	batchSize := 10000
	_, err := c.client.Collection(docsCollection).Documents().Delete(ctx, &api.DeleteDocumentsParams{
		FilterBy:  &filter,
		BatchSize: &batchSize,
	})
	if err != nil {
		return fmt.Errorf("deleting docs for collection %s: %w", name, err)
	}
	return nil
}

// CountDocsByCollection returns the number of full documents for each given collection name.
func (c *Client) CountDocsByCollection(ctx context.Context, names []string) (map[string]int64, error) {
	result := make(map[string]int64)
	for _, name := range names {
		filter := fmt.Sprintf("collection:=%s", name)
		searchParams := &api.SearchCollectionParams{
			Q:        stringPtr(""),
			QueryBy:  stringPtr("content"),
			FilterBy: &filter,
			PerPage:  intPtr(0),
		}
		resp, err := c.client.Collection(docsCollection).Documents().Search(ctx, searchParams)
		if err != nil {
			return nil, err
		}
		if resp.Found != nil {
			result[name] = int64(*resp.Found)
		}
	}
	return result, nil
}

// CountByCollection returns the number of chunk documents for each given collection name.
func (c *Client) CountByCollection(ctx context.Context, names []string) (map[string]int64, error) {
	result := make(map[string]int64)
	for _, name := range names {
		count, err := c.countByField(ctx, "collection", name)
		if err != nil {
			return nil, err
		}
		result[name] = count
	}
	return result, nil
}

// CountByPath returns the number of chunk documents for a given path.
func (c *Client) CountByPath(ctx context.Context, path string) (int64, error) {
	filter := fmt.Sprintf("path:=%s", path)
	limit := 0
	searchParams := &api.SearchCollectionParams{
		Q:        stringPtr(""),
		QueryBy:  stringPtr("content"),
		FilterBy: &filter,
		PerPage:  &limit,
	}
	resp, err := c.client.Collection(chunksCollection).Documents().Search(ctx, searchParams)
	if err != nil {
		return 0, err
	}
	if resp.Found != nil {
		return int64(*resp.Found), nil
	}
	return 0, nil
}

func (c *Client) countByField(ctx context.Context, field, value string) (int64, error) {
	filter := fmt.Sprintf("%s:=%s", field, value)
	limit := 0
	searchParams := &api.SearchCollectionParams{
		Q:        stringPtr(""),
		QueryBy:  stringPtr("content"),
		FilterBy: &filter,
		PerPage:  &limit,
	}
	resp, err := c.client.Collection(chunksCollection).Documents().Search(ctx, searchParams)
	if err != nil {
		return 0, err
	}
	if resp.Found != nil {
		return int64(*resp.Found), nil
	}
	return 0, nil
}

// GetSchemaFields returns the current field definitions of the chunks collection.
func (c *Client) GetSchemaFields(ctx context.Context) ([]api.Field, error) {
	collections, err := c.client.Collections().Retrieve(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("retrieving collections: %w", err)
	}
	for _, col := range collections {
		if col.Name == chunksCollection {
			return col.Fields, nil
		}
	}
	return nil, nil
}

// CollectionCount returns the total number of documents in the chunks collection.
func (c *Client) CollectionCount(ctx context.Context) (int64, error) {
	return c.collectionDocCount(ctx, chunksCollection)
}

// DocCollectionCount returns the total number of documents in the documents collection.
func (c *Client) DocCollectionCount(ctx context.Context) (int64, error) {
	return c.collectionDocCount(ctx, docsCollection)
}

func (c *Client) collectionDocCount(ctx context.Context, name string) (int64, error) {
	collections, err := c.client.Collections().Retrieve(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("retrieving collections: %w", err)
	}
	for _, col := range collections {
		if col.Name == name {
			if col.NumDocuments != nil {
				return *col.NumDocuments, nil
			}
			return 0, nil
		}
	}
	return 0, nil
}

// GetHashByPath returns the hash of the first chunk document for a given path.
// Returns empty string if no chunks exist for that path.
func (c *Client) GetHashByPath(ctx context.Context, path string) (string, error) {
	filter := fmt.Sprintf("path:=%s", path)
	limit := 1
	searchParams := &api.SearchCollectionParams{
		Q:        stringPtr(""),
		QueryBy:  stringPtr("content"),
		FilterBy: &filter,
		PerPage:  &limit,
	}

	resp, err := c.client.Collection(chunksCollection).Documents().Search(ctx, searchParams)
	if err != nil {
		return "", fmt.Errorf("searching hash for %s: %w", path, err)
	}

	if resp.Hits == nil || len(*resp.Hits) == 0 {
		return "", nil
	}

	doc := (*resp.Hits)[0].Document
	if doc == nil {
		return "", nil
	}
	if v, ok := (*doc)["hash"]; ok {
		return fmt.Sprint(v), nil
	}

	return "", nil
}

// DeleteChunksByPath removes all chunks for a given file path.
func (c *Client) DeleteChunksByPath(ctx context.Context, path string) error {
	filter := fmt.Sprintf("path:=%s", path)
	batchSize := 10000
	_, err := c.client.Collection(chunksCollection).Documents().Delete(ctx, &api.DeleteDocumentsParams{
		FilterBy:  &filter,
		BatchSize: &batchSize,
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
	FilterBy    string
	Limit       int
	GroupLimit  int
}

// HybridSearch performs a hybrid (text + vector) search grouped by collection and path.
// Uses POST MultiSearch when a vector query is present to avoid the 4000-char URL limit.
func (c *Client) HybridSearch(ctx context.Context, params HybridSearchParams) ([]HybridSearchResult, error) {
	if len(params.QueryVector) > 0 {
		return c.multiSearch(ctx, params, params.Query)
	}

	searchParams := &api.SearchCollectionParams{
		Q:          &params.Query,
		QueryBy:    stringPtr("content"),
		GroupBy:    stringPtr("collection,path"),
		GroupLimit: intPtr(params.GroupLimit),
		PerPage:    intPtr(params.Limit),
	}

	if params.FilterBy != "" {
		searchParams.FilterBy = &params.FilterBy
	} else if len(params.Collections) > 0 {
		filter := buildCollectionFilter(params.Collections)
		searchParams.FilterBy = &filter
	}

	resp, err := c.client.Collection(chunksCollection).Documents().Search(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("hybrid search: %w", err)
	}

	return groupedHitsToResults(resp), nil
}

// VectorSearch performs a vector-only search using the POST-based MultiSearch endpoint.
func (c *Client) VectorSearch(ctx context.Context, params HybridSearchParams) ([]HybridSearchResult, error) {
	return c.multiSearch(ctx, params, "*")
}

// multiSearch is the unified MultiSearch POST implementation used by both
// HybridSearch (with a text query) and VectorSearch (with "*" as fallback text).
func (c *Client) multiSearch(ctx context.Context, params HybridSearchParams, textQuery string) ([]HybridSearchResult, error) {
	collection := chunksCollection
	vec := fmt.Sprintf("embedding:([%v])", formatVector(params.QueryVector))

	searchItem := api.MultiSearchCollectionParameters{
		Collection:  &collection,
		Q:           &textQuery,
		QueryBy:     stringPtr("content"),
		VectorQuery: &vec,
		GroupBy:     stringPtr("collection,path"),
		GroupLimit:  intPtr(params.GroupLimit),
		PerPage:     intPtr(params.Limit),
	}

	if params.FilterBy != "" {
		searchItem.FilterBy = &params.FilterBy
	} else if len(params.Collections) > 0 {
		filter := buildCollectionFilter(params.Collections)
		searchItem.FilterBy = &filter
	}

	body := api.MultiSearchSearchesParameter{
		Searches: []api.MultiSearchCollectionParameters{searchItem},
	}

	resp, err := c.client.MultiSearch.Perform(ctx, nil, body)
	if err != nil {
		return nil, fmt.Errorf("multi search: %w", err)
	}

	if len(resp.Results) == 0 {
		return nil, nil
	}

	resultItem := resp.Results[0]
	if resultItem.GroupedHits == nil {
		return nil, nil
	}

	results := make([]HybridSearchResult, 0, len(*resultItem.GroupedHits))
	for _, group := range *resultItem.GroupedHits {
		if len(group.Hits) == 0 {
			continue
		}
		hit := group.Hits[0]
		r := hitToResult(hit)
		results = append(results, r)
	}
	return results, nil
}

// TextSearch performs a text-only search (no vector).
func (c *Client) TextSearch(ctx context.Context, params HybridSearchParams) ([]HybridSearchResult, error) {
	perPage := params.Limit
	if perPage > 250 {
		perPage = 250
	}

	page := 1
	var allResults []HybridSearchResult

	for {
		searchParams := &api.SearchCollectionParams{
			Q:          &params.Query,
			QueryBy:    stringPtr("content"),
			GroupBy:    stringPtr("collection,path"),
			GroupLimit: intPtr(params.GroupLimit),
			PerPage:    intPtr(perPage),
			Page:       &page,
		}

		if params.FilterBy != "" {
			searchParams.FilterBy = &params.FilterBy
		} else if len(params.Collections) > 0 {
			filter := buildCollectionFilter(params.Collections)
			searchParams.FilterBy = &filter
		}

		resp, err := c.client.Collection(chunksCollection).Documents().Search(ctx, searchParams)
		if err != nil {
			return nil, fmt.Errorf("text search: %w", err)
		}

		results := groupedHitsToResults(resp)
		allResults = append(allResults, results...)

		if len(results) < perPage {
			break
		}
		if params.Limit > 0 && len(allResults) >= params.Limit {
			allResults = allResults[:params.Limit]
			break
		}
		page++
	}

	return allResults, nil
}

func groupedHitsToResults(resp *api.SearchResult) []HybridSearchResult {
	if resp == nil || resp.GroupedHits == nil {
		return nil
	}

	results := make([]HybridSearchResult, 0, len(*resp.GroupedHits))
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
	if r.Score == 0 && hit.VectorDistance != nil {
		r.Score = 1.0 - float64(*hit.VectorDistance)
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
	b := make([]byte, 0, len(v)*6)
	for i, f := range v {
		if i > 0 {
			b = append(b, ',')
		}
		b = strconv.AppendFloat(b, f, 'f', 2, 64)
	}
	return string(b)
}

// DeleteChunksByCollection removes all chunks for a given collection name.
func (c *Client) DeleteChunksByCollection(ctx context.Context, name string) error {
	filter := fmt.Sprintf("collection:=%s", name)
	batchSize := 10000
	_, err := c.client.Collection(chunksCollection).Documents().Delete(ctx, &api.DeleteDocumentsParams{
		FilterBy:  &filter,
		BatchSize: &batchSize,
	})
	if err != nil {
		return fmt.Errorf("deleting chunks for collection %s: %w", name, err)
	}
	return nil
}

// SearchDistinctPaths returns all distinct document paths in Typesense.
// Optional filter can restrict by collection or other fields.
func (c *Client) SearchDistinctPaths(ctx context.Context, filter string) ([]string, error) {
	perPage := 250
	groupLimit := 1
	var allPaths []string
	seen := make(map[string]bool)
	page := 1

	for {
		searchParams := &api.SearchCollectionParams{
			Q:          stringPtr(""),
			QueryBy:    stringPtr("content"),
			GroupBy:    stringPtr("path"),
			GroupLimit: &groupLimit,
			PerPage:    &perPage,
			Page:       &page,
		}
		if filter != "" {
			searchParams.FilterBy = &filter
		}

		resp, err := c.client.Collection(chunksCollection).Documents().Search(ctx, searchParams)
		if err != nil {
			return nil, fmt.Errorf("searching distinct paths: %w", err)
		}

		if resp.GroupedHits == nil || len(*resp.GroupedHits) == 0 {
			break
		}

		for _, group := range *resp.GroupedHits {
			if len(group.Hits) == 0 {
				continue
			}
			doc := group.Hits[0].Document
			if doc == nil {
				continue
			}
			if v, ok := (*doc)["path"]; ok {
				p := fmt.Sprint(v)
				if !seen[p] {
					allPaths = append(allPaths, p)
					seen[p] = true
				}
			}
		}

		if len(*resp.GroupedHits) < perPage {
			break
		}
		page++
	}

	return allPaths, nil
}

// ListDocuments returns all indexed documents (one per path) optionally filtered by collections.
// Results are grouped by collection and path (one hit per group) and paginated internally.
func (c *Client) ListDocuments(ctx context.Context, collections []string) ([]HybridSearchResult, error) {
	return c.TextSearch(ctx, HybridSearchParams{
		Query:       "",
		Collections: collections,
		Limit:       10000,
		GroupLimit:  1,
	})
}

// SearchChunksByPath searches for chunks matching a given path filter.
func (c *Client) SearchChunksByPath(ctx context.Context, filter string, limit int) ([]HybridSearchResult, error) {
	searchParams := &api.SearchCollectionParams{
		Q:          stringPtr(""),
		QueryBy:    stringPtr("content"),
		GroupBy:    stringPtr("collection,path"),
		GroupLimit: intPtr(10),
		PerPage:    intPtr(limit),
	}
	if filter != "" {
		searchParams.FilterBy = &filter
	}

	resp, err := c.client.Collection(chunksCollection).Documents().Search(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("searching chunks by path: %w", err)
	}

	return groupedHitsToResults(resp), nil
}

// FetchChunksByPath retrieves all chunks for a given path, sorted by chunk_seq ascending.
// Paginates internally to support files with more than 250 chunks.
func (c *Client) FetchChunksByPath(ctx context.Context, path string) ([]HybridSearchResult, error) {
	perPage := 250
	page := 1
	var allResults []HybridSearchResult

	for {
		searchParams := &api.SearchCollectionParams{
			Q:        stringPtr(""),
			QueryBy:  stringPtr("content"),
			FilterBy: stringPtr(fmt.Sprintf("path:=%s", path)),
			SortBy:   stringPtr("chunk_seq:asc"),
			PerPage:  intPtr(perPage),
			Page:     &page,
		}

		resp, err := c.client.Collection(chunksCollection).Documents().Search(ctx, searchParams)
		if err != nil {
			return nil, fmt.Errorf("fetching chunks: %w", err)
		}

		if resp.Hits == nil || len(*resp.Hits) == 0 {
			break
		}

		for _, hit := range *resp.Hits {
			allResults = append(allResults, hitToResult(hit))
		}

		if len(*resp.Hits) < perPage {
			break
		}
		page++
	}

	return allResults, nil
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
