package tavily

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/verdverm/gmd/pkg/web"
)

const defaultTavilyBaseURL = "https://api.tavily.com"

type SearchClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	name       string
}

func NewSearchClient(cfg web.ProviderConfig) (*SearchClient, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	if apiKey == "" {
		return nil, web.ErrAuthMissing
	}
	baseURL, _ := cfg.Extra["base_url"].(string)
	if baseURL == "" {
		baseURL = defaultTavilyBaseURL
	}
	return &SearchClient{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		name:       cfg.Name,
	}, nil
}

func (c *SearchClient) Name() string { return c.name }

func (c *SearchClient) Search(ctx context.Context, opts web.SearchOptions) ([]web.SearchResult, error) {
	if c == nil {
		return nil, fmt.Errorf("tavily: nil client")
	}
	maxResults := opts.NumResults
	if maxResults <= 0 {
		maxResults = 10
	}

	body := map[string]any{
		"query":          opts.Query,
		"max_results":    maxResults,
		"search_depth":   "basic",
		"include_answer": false,
	}

	if len(opts.IncludeDomains) > 0 {
		body["include_domains"] = opts.IncludeDomains
	}
	if len(opts.ExcludeDomains) > 0 {
		body["exclude_domains"] = opts.ExcludeDomains
	}
	if depth, ok := opts.Extra["search_depth"].(string); ok && depth != "" {
		body["search_depth"] = depth
	}
	if answer, ok := opts.Extra["include_answer"].(bool); ok {
		body["include_answer"] = answer
	}
	if raw, ok := opts.Extra["include_raw_content"].(bool); ok {
		body["include_raw_content"] = raw
	}

	respBody, err := c.do(ctx, "POST", c.baseURL+"/search", body)
	if err != nil {
		return nil, web.WrapProviderError("tavily", "search failed", err)
	}

	var resp struct {
		Results []tavilyResult `json:"results"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling tavily response: %w", err)
	}

	results := make([]web.SearchResult, len(resp.Results))
	for i, r := range resp.Results {
		extra := make(map[string]any)
		if r.RawContent != "" {
			extra["raw_content"] = r.RawContent
		}
		results[i] = web.SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Content: r.Content,
			Score:   r.Score,
			Extra:   extra,
			Cost: &web.CostSummary{
				Provider: "tavily",
				Unit:     "query",
				Currency: "USD",
			},
		}
	}

	return results, nil
}

func (c *SearchClient) do(ctx context.Context, method, reqURL string, body map[string]any) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("nil response from server")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, web.ErrRateLimited
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, web.ErrAuthFailed
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("tavily API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

type tavilyResult struct {
	URL        string  `json:"url"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	RawContent string  `json:"raw_content"`
	Score      float64 `json:"score"`
}

var _ web.SearchProvider = (*SearchClient)(nil)
