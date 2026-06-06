package web

import (
	"context"
	"time"
)

// SearchResult is a unified search result from any search provider.
type SearchResult struct {
	Title   string
	URL     string
	Content string
	Score   float64
	Cost    *CostSummary
	Extra   map[string]any
}

// SearchOptions are parameters for a search query.
type SearchOptions struct {
	Query          string
	NumResults     int
	IncludeDomains []string
	ExcludeDomains []string
	Extra          map[string]any
}

// SearchProvider queries a web index.
type SearchProvider interface {
	Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error)
}

// BrowserProvider retrieves and renders web content.
type BrowserProvider interface {
	GetContent(ctx context.Context, url string, opts *GetContentOptions) (*GetContentResult, error)
	Crawl(ctx context.Context, startURL string, opts *CrawlOptions) ([]Page, error)
	Scrape(ctx context.Context, url string, selector string) ([]Element, error)
	Capabilities() BrowserCapabilities
}

// GetContentOptions control content retrieval.
type GetContentOptions struct {
	Format   string
	MaxChars int
	MaxAge   time.Duration
	Extra    map[string]any
}

// GetContentResult holds retrieved content.
type GetContentResult struct {
	Content string
	Cost    *CostSummary
	Extra   map[string]any
}

// CrawlOptions control web crawling.
type CrawlOptions struct {
	MaxDepth       int
	MaxPages       int
	SameDomain     bool
	IncludePattern string
	ExcludePattern string
	Timeout        time.Duration
	Extra          map[string]any
}

// Page represents a crawled page.
type Page struct {
	URL     string
	Title   string
	Content string
	Status  int
	Depth   int
	Links   []string
	Error   string
	Extra   map[string]any
}

// Element represents a scraped element.
type Element struct {
	Tag   string
	Text  string
	HTML  string
	Attrs map[string]string
	Extra map[string]any
}

// BrowserCapabilities describes what a BrowserProvider can do.
type BrowserCapabilities struct {
	GetContent   bool
	Crawl        bool
	Scrape       bool
	SelfHost     bool
	LocalBrowser bool
	LocalHTML    bool
	LocalCrawl   bool
	Features     []string
}

// CostSummary is a provider-reported cost for an operation.
type CostSummary struct {
	Provider string
	Cost     float64
	Unit     string
	Currency string
}

// Deprecated: Provider is the old combined interface. Use SearchProvider and BrowserProvider instead.
type Provider interface {
	Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error)
	Fetch(ctx context.Context, urls []string) ([]SearchResult, error)
}
