package searxng

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/verdverm/gmd/pkg/web"
)

const defaultSearXNGPath = "/search"

type SearchClient struct {
	baseURL    string
	httpClient *http.Client
	name       string
}

func NewSearchClient(cfg web.ProviderConfig) (*SearchClient, error) {
	baseURL, _ := cfg.Extra["base_url"].(string)
	if baseURL == "" {
		return nil, fmt.Errorf("gmd/web: searxng: base_url is required")
	}
	return &SearchClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		name:       cfg.Name,
	}, nil
}

func (c *SearchClient) Name() string { return c.name }

func (c *SearchClient) Search(ctx context.Context, opts web.SearchOptions) ([]web.SearchResult, error) {
	params := url.Values{}
	params.Set("q", opts.Query)
	params.Set("format", "json")

	if opts.NumResults > 0 {
		params.Set("limit", fmt.Sprintf("%d", opts.NumResults))
	}
	if categories, ok := opts.Extra["categories"].(string); ok && categories != "" {
		params.Set("categories", categories)
	}
	if engines, ok := opts.Extra["engines"].(string); ok && engines != "" {
		params.Set("engines", engines)
	}
	if lang, ok := opts.Extra["language"].(string); ok && lang != "" {
		params.Set("language", lang)
	}

	reqURL := fmt.Sprintf("%s%s?%s", c.baseURL, defaultSearXNGPath, params.Encode())

	respBody, err := c.do(ctx, reqURL)
	if err != nil {
		return nil, web.WrapProviderError("searxng", "search failed", err)
	}

	var resp struct {
		Results []searxngResult `json:"results"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling searxng response: %w", err)
	}

	results := make([]web.SearchResult, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = web.SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Content: r.Content,
			Score:   float64(len(resp.Results)-i) / float64(len(resp.Results)),
			Extra: map[string]any{
				"engine":      r.Engine,
				"engines":     r.Engines,
				"published":   r.PublishedDate,
				"category":    r.Category,
				"score_searx": r.Score,
			},
			Cost: &web.CostSummary{
				Provider: "searxng",
				Unit:     "query",
				Currency: "USD",
			},
		}
	}

	return results, nil
}

func (c *SearchClient) do(ctx context.Context, reqURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("searxng API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

type searxngResult struct {
	URL           string   `json:"url"`
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	Engine        string   `json:"engine"`
	Engines       []string `json:"engines"`
	PublishedDate string   `json:"publishedDate"`
	Category      string   `json:"category"`
	Score         float64  `json:"score"`
}

var _ web.SearchProvider = (*SearchClient)(nil)
