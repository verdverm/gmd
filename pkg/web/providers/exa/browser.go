package exa

import (
	"context"

	"github.com/verdverm/gmd/pkg/web"
)

type BrowserAdapter struct {
	client *Client
}

func NewBrowserAdapter(cfg web.ProviderConfig) (*BrowserAdapter, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	baseURL, _ := cfg.Extra["base_url"].(string)
	if baseURL != "" {
		return &BrowserAdapter{client: NewWithServer(apiKey, baseURL, cfg.HTTPClient)}, nil
	}
	return &BrowserAdapter{client: New(apiKey, cfg.HTTPClient)}, nil
}

func (a *BrowserAdapter) GetContent(ctx context.Context, url string, opts *web.GetContentOptions) (*web.GetContentResult, error) {
	req := ContentsRequest{
		URLs: []string{url},
	}

	maxChars := 5000
	if opts != nil {
		if opts.MaxChars > 0 {
			maxChars = opts.MaxChars
		}
		if opts.MaxAge > 0 {
			hours := int(opts.MaxAge.Hours())
			req.MaxAgeHours = &hours
		}
	}

	req.Text = &ContentsText{
		MaxCharacters: maxChars,
	}

	resp, err := a.client.GetContents(ctx, req)
	if err != nil {
		return nil, err
	}

	var content string
	if len(resp.Results) > 0 {
		content = resp.Results[0].Text
	}

	extra := make(map[string]any)
	if len(resp.Results) > 0 {
		if resp.Results[0].Summary != "" {
			extra["summary"] = resp.Results[0].Summary
		}
	}

	return &web.GetContentResult{
		Content: content,
		Cost:    exaCostSummary(resp.CostDollars),
		Extra:   extra,
	}, nil
}

func (a *BrowserAdapter) Crawl(ctx context.Context, startURL string, opts *web.CrawlOptions) ([]web.Page, error) {
	return nil, web.ErrNotSupported
}

func (a *BrowserAdapter) Scrape(ctx context.Context, url string, selector string) ([]web.Element, error) {
	return nil, web.ErrNotSupported
}

func (a *BrowserAdapter) Capabilities() web.BrowserCapabilities {
	return web.BrowserCapabilities{
		GetContent: true,
		Crawl:      false,
		Scrape:     false,
		SelfHost:   false,
		Features:   []string{"exa-contents"},
	}
}

var _ web.BrowserProvider = (*BrowserAdapter)(nil)
