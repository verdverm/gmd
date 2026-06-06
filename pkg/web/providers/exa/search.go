package exa

import (
	"context"
	"time"

	"github.com/verdverm/gmd/pkg/web"
	exaclient "github.com/verdverm/gmd/pkg/web/exa"
)

type SearchAdapter struct {
	client *exaclient.Client
}

func NewSearchAdapter(cfg web.ProviderConfig) (*SearchAdapter, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	baseURL, _ := cfg.Extra["base_url"].(string)
	if baseURL != "" {
		return &SearchAdapter{client: exaclient.NewWithServer(apiKey, baseURL)}, nil
	}
	return &SearchAdapter{client: exaclient.New(apiKey)}, nil
}

func (a *SearchAdapter) Search(ctx context.Context, opts web.SearchOptions) ([]web.SearchResult, error) {
	req := exaclient.SearchRequest{
		Query:      opts.Query,
		NumResults: opts.NumResults,
	}

	if len(opts.IncludeDomains) > 0 {
		req.IncludeDomains = opts.IncludeDomains
	}
	if len(opts.ExcludeDomains) > 0 {
		req.ExcludeDomains = opts.ExcludeDomains
	}

	if extra, ok := opts.Extra["search_type"].(string); ok && extra != "" {
		req.Type = extra
	}
	if extra, ok := opts.Extra["use_autoprompt"].(bool); ok {
		req.UseAutoprompt = &extra
	}
	if extra, ok := opts.Extra["category"].(string); ok && extra != "" {
		req.Category = extra
	}
	if extra, ok := opts.Extra["start_published_date"].(string); ok && extra != "" {
		if t, err := parseDateExtra(extra); err == nil {
			req.StartPublishedDate = &t
		}
	}
	if extra, ok := opts.Extra["end_published_date"].(string); ok && extra != "" {
		if t, err := parseDateExtra(extra); err == nil {
			req.EndPublishedDate = &t
		}
	}

	withText, _ := opts.Extra["with_text"].(bool)
	withHighlights, _ := opts.Extra["with_highlights"].(bool)
	maxChars, _ := opts.Extra["max_chars"].(int)

	if withText {
		req.Contents = &exaclient.ContentsOptions{
			Text: &exaclient.ContentsText{
				MaxCharacters: maxChars,
			},
		}
	} else if withHighlights || (!withText && maxChars == 0) {
		req.Contents = &exaclient.ContentsOptions{
			Highlights: &exaclient.HighlightOpts{},
		}
	}

	resp, err := a.client.Search(ctx, req)
	if err != nil {
		return nil, err
	}

	results := make([]web.SearchResult, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = web.SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Content: r.Text,
			Extra:   make(map[string]any),
		}
		if r.Score != nil {
			results[i].Score = *r.Score
		}
		if r.Author != "" {
			results[i].Extra["author"] = r.Author
		}
		if r.PublishedDate != nil {
			results[i].Extra["published_date"] = r.PublishedDate.Format("2006-01-02")
		}
		if len(r.Highlights) > 0 {
			results[i].Extra["highlights"] = r.Highlights
		}
		if r.Summary != "" {
			results[i].Extra["summary"] = r.Summary
		}
		if c := exaCostSummary(resp.CostDollars); c != nil {
			results[i].Cost = c
		}
	}

	return results, nil
}

func exaCostSummary(cost *exaclient.CostDollars) *web.CostSummary {
	if cost == nil {
		return nil
	}
	return &web.CostSummary{
		Provider: "exa",
		Cost:     cost.Total,
		Unit:     "query",
		Currency: "USD",
	}
}

func parseDateExtra(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

var _ web.SearchProvider = (*SearchAdapter)(nil)
