package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/verdverm/gmd/pkg/web"
)

const defaultCloudflareBaseURL = "https://api.cloudflare.com/client/v4"

type BrowserClient struct {
	apiKey     string
	accountID  string
	baseURL    string
	httpClient *http.Client
}

func NewBrowserClient(cfg web.ProviderConfig) (*BrowserClient, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	accountID, _ := cfg.Extra["account_id"].(string)
	if apiKey == "" {
		return nil, web.ErrAuthMissing
	}
	if accountID == "" {
		return nil, fmt.Errorf("gmd/web: cloudflare: account_id is required")
	}
	return &BrowserClient{
		apiKey:     apiKey,
		accountID:  accountID,
		baseURL:    defaultCloudflareBaseURL,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (c *BrowserClient) GetContent(ctx context.Context, urlStr string, opts *web.GetContentOptions) (*web.GetContentResult, error) {
	format := "markdown"
	maxChars := 5000
	if opts != nil {
		if opts.Format == "text" || opts.Format == "html" {
			format = opts.Format
		}
		if opts.MaxChars > 0 {
			maxChars = opts.MaxChars
		}
	}

	endpoint := "/markdown"
	if format == "text" {
		endpoint = "/content"
	}

	reqURL := fmt.Sprintf("%s/accounts/%s/browser-rendering%s", c.baseURL, c.accountID, endpoint)

	body := map[string]any{
		"url": urlStr,
	}
	if endpoint == "/content" && maxChars > 0 {
		body["maxChars"] = maxChars
	}

	respBody, err := c.do(ctx, "POST", reqURL, body)
	if err != nil {
		return nil, err
	}

	return &web.GetContentResult{
		Content: string(respBody),
		Cost: &web.CostSummary{
			Provider: "cloudflare",
			Unit:     "request",
			Currency: "USD",
		},
	}, nil
}

func (c *BrowserClient) Crawl(ctx context.Context, startURL string, opts *web.CrawlOptions) ([]web.Page, error) {
	maxDepth := 2
	maxPages := 20
	sameDomain := true
	if opts != nil {
		if opts.MaxDepth > 0 {
			maxDepth = opts.MaxDepth
		}
		if opts.MaxPages > 0 {
			maxPages = opts.MaxPages
		}
		sameDomain = opts.SameDomain
	}

	queue := []crawlItem{{url: startURL, depth: 0}}
	seen := map[string]bool{}
	var pages []web.Page

	for len(queue) > 0 && len(pages) < maxPages {
		select {
		case <-ctx.Done():
			return pages, ctx.Err()
		default:
		}

		item := queue[0]
		queue = queue[1:]

		if seen[item.url] {
			continue
		}
		if item.depth > maxDepth {
			continue
		}

		canonical := normalizeURL(item.url)
		if seen[canonical] {
			continue
		}
		seen[canonical] = true

		result, err := c.GetContent(ctx, item.url, nil)
		if err != nil {
			pages = append(pages, web.Page{
				URL:   item.url,
				Error: err.Error(),
				Depth: item.depth,
			})
			continue
		}

		page := web.Page{
			URL:     item.url,
			Content: result.Content,
			Depth:   item.depth,
			Status:  200,
		}

		if item.depth < maxDepth && (len(pages)+len(queue)) < maxPages {
			links := extractLinks(result.Content, item.url)
			for _, link := range links {
				if sameDomain && !isSameDomain(item.url, link) {
					continue
				}
				if !seen[normalizeURL(link)] {
					queue = append(queue, crawlItem{url: link, depth: item.depth + 1})
				}
			}
			page.Links = links
		}

		pages = append(pages, page)
	}

	return pages, nil
}

func (c *BrowserClient) Scrape(ctx context.Context, url string, selector string) ([]web.Element, error) {
	return nil, web.ErrNotSupported
}

func (c *BrowserClient) Capabilities() web.BrowserCapabilities {
	return web.BrowserCapabilities{
		GetContent: true,
		Crawl:      true,
		Scrape:     false,
		SelfHost:   false,
		Features:   []string{"browser-rendering", "markdown"},
	}
}

func (c *BrowserClient) do(ctx context.Context, method, reqURL string, body any) ([]byte, error) {
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
		return nil, fmt.Errorf("cloudflare API error %d: %s", resp.StatusCode, string(respBody))
	}

	var cfResp struct {
		Success bool            `json:"success"`
		Errors  []cfError       `json:"errors"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(respBody, &cfResp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}
	if !cfResp.Success && len(cfResp.Errors) > 0 {
		return nil, fmt.Errorf("cloudflare API error: %s (code %d)", cfResp.Errors[0].Message, cfResp.Errors[0].Code)
	}

	return cfResp.Result, nil
}

type cfError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type crawlItem struct {
	url   string
	depth int
}

func normalizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Fragment = ""
	u.RawQuery = ""
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String()
}

func isSameDomain(a, b string) bool {
	ua, err := url.Parse(a)
	if err != nil {
		return false
	}
	ub, err := url.Parse(b)
	if err != nil {
		return false
	}
	return ua.Host == ub.Host
}

func extractLinks(content, baseURL string) []string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}

	linkRE := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	matches := linkRE.FindAllStringSubmatch(content, -1)
	links := make([]string, 0, len(matches))
	seen := map[string]bool{}

	for _, m := range matches {
		href := m[2]
		if strings.HasPrefix(href, "#") {
			continue
		}
		ref, err := url.Parse(href)
		if err != nil {
			continue
		}
		resolved := base.ResolveReference(ref).String()
		if strings.HasPrefix(resolved, "http") && !seen[resolved] {
			seen[resolved] = true
			links = append(links, resolved)
		}
	}

	return links
}

var _ web.BrowserProvider = (*BrowserClient)(nil)
