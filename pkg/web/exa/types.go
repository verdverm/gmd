package exa

import (
	"encoding/json"
	"time"
)

type SearchRequest struct {
	Query              string           `json:"query"`
	Type               string           `json:"type,omitempty"`
	NumResults         int              `json:"numResults,omitempty"`
	IncludeDomains     []string         `json:"includeDomains,omitempty"`
	ExcludeDomains     []string         `json:"excludeDomains,omitempty"`
	StartPublishedDate *time.Time       `json:"startPublishedDate,omitempty"`
	EndPublishedDate   *time.Time       `json:"endPublishedDate,omitempty"`
	Category           string           `json:"category,omitempty"`
	AdditionalQueries  []string         `json:"additionalQueries,omitempty"`
	Contents           *ContentsOptions `json:"contents,omitempty"`
	UseAutoprompt      *bool            `json:"useAutoprompt,omitempty"`
	OutputSchema       any              `json:"outputSchema,omitempty"`
	SystemPrompt       string           `json:"systemPrompt,omitempty"`
	Moderation         *bool            `json:"moderation,omitempty"`
}

type SearchResponse struct {
	Results          []SearchResult `json:"results"`
	AutopromptString string         `json:"autopromptString,omitempty"`
	RequestID        string         `json:"requestId"`
	CostDollars      *CostDollars   `json:"costDollars,omitempty"`
}

type SearchResult struct {
	ID              string         `json:"id"`
	Title           string         `json:"title"`
	URL             string         `json:"url"`
	PublishedDate   *time.Time     `json:"publishedDate,omitempty"`
	Author          string         `json:"author,omitempty"`
	Score           *float64       `json:"score,omitempty"`
	Text            string         `json:"text,omitempty"`
	Highlights      []string       `json:"highlights,omitempty"`
	HighlightScores []float64      `json:"highlightScores,omitempty"`
	Summary         string         `json:"summary,omitempty"`
	Subpages        []SearchResult `json:"subpages,omitempty"`
}

type ContentsRequest struct {
	URLs          []string       `json:"urls"`
	Text          *ContentsText  `json:"text,omitempty"`
	Highlights    *HighlightOpts `json:"highlights,omitempty"`
	Summary       *SummaryOpts   `json:"summary,omitempty"`
	Subpages      int            `json:"subpages,omitempty"`
	SubpageTarget []string       `json:"subpageTarget,omitempty"`
	Extras        *ExtrasOpts    `json:"extras,omitempty"`
	Livecrawl     string         `json:"livecrawl,omitempty"`
	MaxAgeHours   *int           `json:"maxAgeHours,omitempty"`
}

type ContentsText struct {
	MaxCharacters   int  `json:"maxCharacters,omitempty"`
	IncludeHTMLTags bool `json:"includeHtmlTags,omitempty"`
}

type HighlightOpts struct {
	Query *string `json:"query,omitempty"`
}

type SummaryOpts struct {
	Query  string `json:"query,omitempty"`
	Schema any    `json:"schema,omitempty"`
}

type ExtrasOpts struct {
	Links        bool `json:"links,omitempty"`
	SubpageLinks bool `json:"subpageLinks,omitempty"`
	PageQuality  bool `json:"pageQuality,omitempty"`
	TitleQuality bool `json:"titleQuality,omitempty"`
}

type ContentsResponse struct {
	Results     []SearchResult `json:"results"`
	CostDollars *CostDollars   `json:"costDollars,omitempty"`
}

type FindSimilarRequest struct {
	URL            string           `json:"url"`
	NumResults     int              `json:"numResults,omitempty"`
	IncludeDomains []string         `json:"includeDomains,omitempty"`
	ExcludeDomains []string         `json:"excludeDomains,omitempty"`
	Contents       *ContentsOptions `json:"contents,omitempty"`
}

type FindSimilarResponse struct {
	Results     []SearchResult `json:"results"`
	CostDollars *CostDollars   `json:"costDollars,omitempty"`
}

type AnswerRequest struct {
	Query  string `json:"query"`
	Stream bool   `json:"stream,omitempty"`
	Text   bool   `json:"text,omitempty"`
}

type AnswerResponse struct {
	Answer      string         `json:"answer"`
	Citations   []SearchResult `json:"citations"`
	CostDollars *CostDollars   `json:"costDollars,omitempty"`
}

type ContentsOptions struct {
	Text        *ContentsText  `json:"text,omitempty"`
	Highlights  *HighlightOpts `json:"highlights,omitempty"`
	Summary     *SummaryOpts   `json:"summary,omitempty"`
	Livecrawl   string         `json:"livecrawl,omitempty"`
	MaxAgeHours *int           `json:"maxAgeHours,omitempty"`
}

type CostDollars struct {
	Total    float64         `json:"total"`
	Search   json.RawMessage `json:"search,omitempty"`
	Contents json.RawMessage `json:"contents,omitempty"`
	Answer   json.RawMessage `json:"answer,omitempty"`
}
