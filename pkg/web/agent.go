package web

import (
	"context"
	"fmt"
	"strings"

	"github.com/verdverm/gmd/pkg/exa"
	"github.com/verdverm/gmd/pkg/llm"
)

type Agent struct {
	exaClient      *exa.Client
	llmClient      *llm.Client
	maxSteps       int
	resultsPerStep int
	fetchText      bool
}

type AgentConfig struct {
	MaxSteps       int
	ResultsPerStep int
	FetchText      bool
}

type AgentResult struct {
	Answer  string
	Sources []AgentSource
}

type AgentSource struct {
	Title string
	URL   string
	Text  string
}

func NewAgent(exaClient *exa.Client, llmClient *llm.Client, cfg AgentConfig) *Agent {
	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 3
	}
	if cfg.ResultsPerStep <= 0 {
		cfg.ResultsPerStep = 5
	}
	return &Agent{
		exaClient:      exaClient,
		llmClient:      llmClient,
		maxSteps:       cfg.MaxSteps,
		resultsPerStep: cfg.ResultsPerStep,
		fetchText:      cfg.FetchText,
	}
}

func (a *Agent) Run(ctx context.Context, query string) (*AgentResult, error) {
	allResults := make([]exa.SearchResult, 0)

	searchResp, err := a.exaClient.Search(ctx, exa.SearchRequest{
		Query:      query,
		Type:       "auto",
		NumResults: a.resultsPerStep,
		Contents: &exa.ContentsOptions{
			Highlights: &exa.HighlightOpts{},
		},
		UseAutoprompt: boolPtr(true),
	})
	if err != nil {
		return nil, fmt.Errorf("initial search: %w", err)
	}
	allResults = append(allResults, searchResp.Results...)

	for step := 1; step < a.maxSteps; step++ {
		decision, queries, err := a.analyzeResults(ctx, query, allResults)
		if err != nil {
			return nil, fmt.Errorf("analyzing results at step %d: %w", step, err)
		}
		if decision == "DONE" {
			break
		}

		for _, q := range queries {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			sr, err := a.exaClient.Search(ctx, exa.SearchRequest{
				Query:      q,
				Type:       "auto",
				NumResults: a.resultsPerStep,
				Contents: &exa.ContentsOptions{
					Highlights: &exa.HighlightOpts{},
				},
				UseAutoprompt: boolPtr(true),
			})
			if err != nil {
				return nil, fmt.Errorf("follow-up search %q: %w", q, err)
			}
			allResults = append(allResults, sr.Results...)
		}
	}

	if a.fetchText {
		urls := make([]string, 0)
		seen := make(map[string]bool)
		for _, r := range allResults {
			if r.URL != "" && !seen[r.URL] {
				urls = append(urls, r.URL)
				seen[r.URL] = true
			}
		}
		if len(urls) > 0 {
			contentsResp, err := a.exaClient.GetContents(ctx, exa.ContentsRequest{
				URLs: urls,
				Text: &exa.ContentsText{
					MaxCharacters: 5000,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("fetching full text for %d URLs: %w", len(urls), err)
			}
			for i, cr := range contentsResp.Results {
				if i < len(allResults) {
					allResults[i].Text = cr.Text
				}
			}
		}
	}

	answer, err := a.synthesize(ctx, query, allResults)
	if err != nil {
		return nil, fmt.Errorf("synthesizing: %w", err)
	}

	sources := make([]AgentSource, 0, len(allResults))
	seen := make(map[string]bool)
	for _, r := range allResults {
		if r.URL == "" || seen[r.URL] {
			continue
		}
		seen[r.URL] = true
		sources = append(sources, AgentSource{
			Title: r.Title,
			URL:   r.URL,
			Text:  r.Text,
		})
	}

	return &AgentResult{
		Answer:  answer,
		Sources: sources,
	}, nil
}

func (a *Agent) analyzeResults(ctx context.Context, query string, results []exa.SearchResult) (string, []string, error) {
	resultsText := formatResultsForLLM(results)

	prompt := strings.ReplaceAll(agentSystemPrompt, "{query}", query)

	messages := []llm.ChatMessage{
		{Role: "system", Content: prompt},
		{Role: "user", Content: fmt.Sprintf("User question: %s\n\nSearch results:\n%s\n\nReview these results and decide whether to search more or synthesize an answer. Return your decision using ## ACTION (DONE or SEARCH_MORE) and if SEARCH_MORE, list queries under ## QUERIES.", query, resultsText)},
	}

	resp, err := a.llmClient.Chat(ctx, messages)
	if err != nil {
		return "", nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	decision := "DONE"
	var queries []string

	lines := strings.Split(resp, "\n")
	inAction := false
	inQueries := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "## ACTION"):
			inAction = true
			inQueries = false
		case strings.HasPrefix(trimmed, "## QUERIES"):
			inAction = false
			inQueries = true
		case inAction && trimmed == "DONE":
			decision = "DONE"
		case inAction && trimmed == "SEARCH_MORE":
			decision = "SEARCH_MORE"
		case inQueries && strings.HasPrefix(trimmed, "-"):
			queries = append(queries, strings.TrimPrefix(trimmed, "- "))
		}
	}

	return decision, queries, nil
}

func (a *Agent) synthesize(ctx context.Context, query string, results []exa.SearchResult) (string, error) {
	resultsText := formatResultsForLLM(results)

	prompt := strings.ReplaceAll(agentSynthesizePrompt, "{query}", query)
	prompt = strings.ReplaceAll(prompt, "{results}", resultsText)

	messages := []llm.ChatMessage{
		{Role: "system", Content: prompt},
		{Role: "user", Content: fmt.Sprintf("Synthesize a comprehensive answer to: %s", query)},
	}

	resp, err := a.llmClient.Summarize(ctx, messages)
	if err != nil {
		return "", err
	}
	return resp, nil
}

func formatResultsForLLM(results []exa.SearchResult) string {
	var sb strings.Builder
	for i, r := range results {
		fmt.Fprintf(&sb, "Result %d:\n", i+1)
		fmt.Fprintf(&sb, "  Title: %s\n", r.Title)
		fmt.Fprintf(&sb, "  URL: %s\n", r.URL)
		if r.Author != "" {
			fmt.Fprintf(&sb, "  Author: %s\n", r.Author)
		}
		if r.PublishedDate != nil {
			fmt.Fprintf(&sb, "  Published: %s\n", r.PublishedDate.Format("2006-01-02"))
		}
		if r.Text != "" {
			fmt.Fprintf(&sb, "  Text: %s\n", truncate(r.Text, 2000))
		}
		if len(r.Highlights) > 0 {
			fmt.Fprintf(&sb, "  Highlights:\n")
			for _, h := range r.Highlights {
				fmt.Fprintf(&sb, "    - %s\n", h)
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func boolPtr(b bool) *bool {
	return &b
}
