package web

import "context"

type SearchResult struct {
	Title   string
	URL     string
	Content string
	Score   float64
}

type SearchOptions struct {
	Query          string
	NumResults     int
	IncludeDomains []string
	ExcludeDomains []string
}

type Provider interface {
	Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error)
	Fetch(ctx context.Context, urls []string) ([]SearchResult, error)
}
