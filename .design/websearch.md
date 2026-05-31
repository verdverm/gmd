# 15. Web Search & Research (EXA Integration)

**Status: Design**

GMD adds four web commands backed by [EXA](https://exa.ai) — a neural web search
API that indexes and embeds the entire web. These commands sit alongside the local
search pipeline (`gmd query`) and the wiki subsystem (`gmd wiki`), sharing LLM
infrastructure and optionally feeding into wiki for persistent knowledge.

## 15a. Architecture Overview

```
                    ┌─────────────────────────────────┐
                    │          gmd web ...             │
                    ├───────────┬──────────┬───────────┤
                    │  fetch    │  search  │ agent     │  → research
                    │  (no LLM) │ (no LLM) │ (LLM)    │     (LLM + skills)
                    └─────┬─────┴────┬─────┴─────┬─────┘
                          │          │           │
                    ┌─────▼──────────▼───────────▼──────┐
                    │         pkg/exa/client.go         │
                    │  Thin HTTP wrapper over EXA API   │
                    │  Search / Contents / FindSimilar  │
                    └────────────────┬──────────────────┘
                                     │
                          ┌──────────▼──────────┐
                          │    EXA API           │
                          │  api.exa.ai          │
                          │  x-api-key header    │
                          └─────────────────────┘
```

| Layer | Owned By | What It Does |
|---|---|---|
| `pkg/exa/client.go` | GMD | Thin `net/http` wrapper around EXA REST API. No SDK dependency — the community Go SDK is outdated. |
| `cmd/gmd/web_fetch.go` | GMD | Fetch clean markdown/text for one or more URLs |
| `cmd/gmd/web_search.go` | GMD | Neural web search with type/length/domain filtering |
| `pkg/web/agent.go` | GMD | Agent loop: LLM decides next searches, reads results, synthesizes |
| `pkg/web/research.go` | GMD | Deep research with sub-questions, cross-referencing, assumption validation |
| `pkg/config/schema/config.cue` | GMD | `exa` config block (API key, endpoint) |

### Design Tier

| Command | LLM Needed? | EXA Calls Per Run | Output |
|---|---|---|---|
| `gmd web fetch <url>` | No | 1 | Markdown/text content |
| `gmd web search <query>` | No | 1-2 | Ranked results + content |
| `gmd web agent <query>` | Yes (expansion model) | 2-6 | Synthesized answer + sources |
| `gmd web research <query>` | Yes (expansion model) | 8-30 | Deep research report with citations |

### Relationship to wiki

These are separate but complementary subsystems:

| Web Feature | Wiki Integration |
|---|---|
| `gmd web research` | `--save` or `--wiki <name>` writes final report to `wiki/synthesis/` |
| `gmd web search` | Wiki lint gap analysis: "find web sources for missing topics" |
| `gmd web fetch` | Ingest arbitrary URLs into raw/ for wiki processing |
| `gmd web agent` | Interactive exploration that can feed into wiki pages |

The web commands are **not dependent on wiki** and wiki is **not dependent on
web**. They share only the LLM client (`pkg/llm`) and config infrastructure.
The integration points are opt-in flags (`--wiki`, `--save`).

## 15b. EXA API Client (`pkg/exa/`)

We do **not** use the community Go SDK (`github.com/amalucelli/exa-go`). It
lags behind the current Exa API (missing `deep`, `deep-lite`, `deep-reasoning`,
`outputSchema`, `systemPrompt`, `stream`). Instead we write a thin `net/http`
wrapper that maps directly to the REST API.

### `pkg/exa/client.go`

```go
package exa

type Client struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
}

func New(apiKey string) *Client

// Search performs a neural web search.
// Uses POST /search with JSON body matching the EXA API spec.
func (c *Client) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error)

// GetContents fetches clean content for one or more URLs.
// Uses POST /contents with JSON body.
func (c *Client) GetContents(ctx context.Context, req ContentsRequest) (*ContentsResponse, error)

// FindSimilar finds pages semantically similar to a given URL.
// Uses POST /findSimilar with JSON body.
func (c *Client) FindSimilar(ctx context.Context, req FindSimilarRequest) (*FindSimilarResponse, error)

// Answer generates an LLM answer with citations.
// Uses POST /answer. Retained for completeness; agent/research commands
// use their own LLM prompts instead.
func (c *Client) Answer(ctx context.Context, req AnswerRequest) (*AnswerResponse, error)
```

### `pkg/exa/types.go`

All types mirror the EXA REST API JSON structure directly — no abstractions, no
transformations:

```go
// === Search ===

type SearchRequest struct {
    Query              string          `json:"query"`
    Type               string          `json:"type,omitempty"`        // auto, fast, instant, deep-lite, deep, deep-reasoning
    NumResults         int             `json:"numResults,omitempty"`
    IncludeDomains     []string        `json:"includeDomains,omitempty"`
    ExcludeDomains     []string        `json:"excludeDomains,omitempty"`
    StartPublishedDate *time.Time      `json:"startPublishedDate,omitempty"`
    EndPublishedDate   *time.Time      `json:"endPublishedDate,omitempty"`
    Category           string          `json:"category,omitempty"`    // company, research paper, news, personal site, financial report
    Contents           *ContentsOptions `json:"contents,omitempty"`
    UseAutoprompt      *bool           `json:"useAutoprompt,omitempty"`
    OutputSchema       any             `json:"outputSchema,omitempty"`
    SystemPrompt       string          `json:"systemPrompt,omitempty"`
}

type SearchResponse struct {
    Results         []SearchResult `json:"results"`
    AutopromptString string        `json:"autopromptString,omitempty"`
    RequestID       string         `json:"requestId"`
    CostDollars     *CostDollars   `json:"costDollars,omitempty"`
}

type SearchResult struct {
    ID              string        `json:"id"`
    Title           string        `json:"title"`
    URL             string        `json:"url"`
    PublishedDate   *time.Time    `json:"publishedDate,omitempty"`
    Author          string        `json:"author,omitempty"`
    Score           *float64      `json:"score,omitempty"`
    Text            string        `json:"text,omitempty"`
    Highlights      []string      `json:"highlights,omitempty"`
    HighlightScores []float64     `json:"highlightScores,omitempty"`
    Summary         string        `json:"summary,omitempty"`
    Subpages        []SearchResult `json:"subpages,omitempty"`
}

// === Contents (fetch) ===

type ContentsRequest struct {
    URLs              []string        `json:"urls"`
    Text              *ContentsText   `json:"text,omitempty"`
    Highlights        *HighlightOpts  `json:"highlights,omitempty"`
    Summary           *SummaryOpts    `json:"summary,omitempty"`
    Subpages          int             `json:"subpages,omitempty"`
    SubpageTarget     []string        `json:"subpageTarget,omitempty"`
    Extras            *ExtrasOpts     `json:"extras,omitempty"`
    Livecrawl         string          `json:"livecrawl,omitempty"`    // "always", "fallback"
    MaxAgeHours       *int            `json:"maxAgeHours,omitempty"`
}

type ContentsText struct {
    MaxCharacters   int  `json:"maxCharacters,omitempty"`
    IncludeHtmlTags bool `json:"includeHtmlTags,omitempty"`
}

type HighlightOpts struct {
    Query *string `json:"query,omitempty"`
    // When query is unset, Exa returns default highlights.
}

type SummaryOpts struct {
    Query  string `json:"query,omitempty"`
    Schema any    `json:"schema,omitempty"`
}

type ExtrasOpts struct {
    Links         bool `json:"links,omitempty"`
    SubpageLinks  bool `json:"subpageLinks,omitempty"`
    PageQuality   bool `json:"pageQuality,omitempty"`
    TitleQuality  bool `json:"titleQuality,omitempty"`
}

type ContentsResponse struct {
    Results     []SearchResult `json:"results"`
    CostDollars *CostDollars   `json:"costDollars,omitempty"`
}

// === Find Similar ===

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

// === Answer ===

type AnswerRequest struct {
    Query  string `json:"query"`
    Stream bool   `json:"stream,omitempty"`
    Text   bool   `json:"text,omitempty"`
}

type AnswerResponse struct {
    Answer    string         `json:"answer"`
    Citations []SearchResult `json:"citations"`
    CostDollars *CostDollars `json:"costDollars,omitempty"`
}

// === Shared ===

type ContentsOptions struct {
    Text       *ContentsText   `json:"text,omitempty"`
    Highlights *HighlightOpts  `json:"highlights,omitempty"`
    Summary    *SummaryOpts    `json:"summary,omitempty"`
    Livecrawl  string          `json:"livecrawl,omitempty"`
    MaxAgeHours *int           `json:"maxAgeHours,omitempty"`
}

type CostDollars struct {
    Total     float64 `json:"total"`
    Search    float64 `json:"search,omitempty"`
    Contents  float64 `json:"contents,omitempty"`
    Answer    float64 `json:"answer,omitempty"`
}

// RequestID for cost tracking in EXA dashboard
```

### Error handling

```go
type APIError struct {
    StatusCode int
    Message    string
    Body       string
}

func (e *APIError) Error() string

func IsRateLimit(err error) bool
func IsAuthError(err error) bool
```

### Timeout / retry

- All requests use a context with timeout (30s default, configurable)
- Rate-limit (429) retries with exponential backoff: 1s, 2s, 4s, 8s, max 3 retries
- Other 4xx errors returned immediately; 5xx retried once

## 15c. CLI Commands

### `gmd web fetch <url> [url2 ...]`

Fetches clean content from one or more URLs. No LLM involved. Thin wrapper over
`POST /contents`.

```
Flags:
  --format text|markdown      Output format (default: markdown)
  --highlights                Return highlights only (no full text)
  --summary <query>           Return LLM-generated summary targeting query
  --max-chars N               Max characters per page (default: 5000)
  --output stdout|file         Write to stdout or file(s) (default: stdout)
  -o, --outdir DIR            Output directory for --output file
  --json                      Output raw JSON from EXA API

Examples:
  gmd web fetch https://example.com/article
  gmd web fetch https://a.com https://b.com --max-chars 2000
  gmd web fetch https://example.com --summary "key claims about"
  gmd web fetch https://example.com --output file -o ./fetched/
```

Implementation: `cmd/gmd/web_fetch.go`

```go
var webFetchCmd = &cobra.Command{
    Use:   "fetch <url> [url2 ...]",
    Short: "Fetch clean content from URLs via EXA",
    Args:  cobra.MinimumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        cfg := getConfig()
        client := exa.New(cfg.EXA.APIKey)
        // Build ContentsRequest with Text/Highlights/Summary from flags
        // Call client.GetContents()
        // Print results
    },
}
```

### `gmd web search <query>`

Neural web search. No LLM involved. Thin wrapper over `POST /search`.

```
Flags:
  --type auto|fast|instant|deep-lite|deep|deep-reasoning   (default: auto)
  -n, --limit N               Max results (default: 10)
  --domain DOMAIN             Require results from domain (repeatable)
  --exclude-domain DOMAIN     Exclude domain (repeatable)
  --date-start DATE           Filter by publish date start (ISO 8601)
  --date-end DATE             Filter by publish date end (ISO 8601)
  --category CAT              company, research paper, news, personal site, financial report
  --text                      Return full text content
  --highlights                Return highlights only
  --summary <query>           Return LLM summary targeting query
  --max-chars N               Max characters when --text (default: 5000)
  --json                      Output raw JSON
  --no-autoprompt             Disable EXA's query rewriting

Examples:
  gmd web search "transformer architecture"
  gmd web search "golang generics" --type deep --limit 5 --text
  gmd web search "startup funding" --category company --date-start 2026-01-01
  gmd web search "kubernetes" --domain kubernetes.io --highlights
```

Implementation: `cmd/gmd/web_search.go`

```go
var webSearchCmd = &cobra.Command{
    Use:   "search <query>",
    Short: "Search the web via EXA neural search",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        cfg := getConfig()
        client := exa.New(cfg.EXA.APIKey)
        // Build SearchRequest from flags
        // Call client.Search()
        // Print results
    },
}
```

### `gmd web agent <query>`

Searching agent with LLM orchestration. Multi-step: search → analyze results
→ optionally search deeper → synthesize. The LLM acts as the "brain" deciding
what to search for next, while EXA does the actual retrieval.

```
Flags:
  -n, --limit N               Max results per search step (default: 5)
  --steps N                   Max search steps (default: 3)
  --depth shallow|medium|deep Research depth — influences step count and
                              result detail (default: medium)
  --text                      Fetch full text for results (not just highlights)
  --output json|markdown      Output format (default: markdown)
  --json                      Short for --output json
  -s, --save                  Save result to wiki/synthesis/ (requires wiki)
  --wiki NAME                 Wiki name to save to (default: first wiki found)

Examples:
  gmd web agent "what are the latest developments in Go 1.24?"
  gmd web agent "compare Nuxt 4 vs Next.js 16" --depth deep --save
  gmd web agent "rust async runtime performance" --steps 5 --text
```

**Agent loop** (`pkg/web/agent.go`):

```
1. INITIAL SEARCH
   → call exa.Search(query, type=auto, highlights=true, numResults=N)

2. ANALYZE (LLM)
   → system prompt: "You are a research analyst. Given search results,
     identify what's covered and what's missing."
   → LLM decides: DONE (sufficient info) or SEARCH_MORE (with refined queries)

3. SEARCH MORE (if needed, up to --steps times)
   → call exa.Search for each refined query
   → Optionally call exa.GetContents for deep reading of top results

4. SYNTHESIZE (LLM)
   → system prompt: "Synthesize a comprehensive answer from all gathered
     sources. Cite every claim with [Source Title](URL). Note contradictions.
     Flag any unsupported claims."
   → LLM returns final answer

5. SAVE (if --save)
   → Write to wiki/synthesis/YYYY-MM-DD-query-slug.md
   → Update Typesense index
```

### `gmd web research <query>`

Deep research agent. This is the most sophisticated command — it performs
multi-layered exploration with assumption testing, cross-validation, and
structured reporting.

```
Flags:
  --depth shallow|medium|deep Research depth (default: medium)
  --max-sources N             Max unique sources to consult (default: 20)
  --output stdout|file        Output destination (default: stdout)
  -o, --out FILE              File to write report to
  --format json|markdown      Output format (default: markdown)
  -s, --save                  Save to wiki (requires wiki)
  --wiki NAME                 Wiki name

Examples:
  gmd web research "environmental impact of EV batteries"
  gmd web research "WebAssembly in production" --depth deep -o wasm-report.md
  gmd web research "post-quantum cryptography standards" --save
```

**Research pipeline** (`pkg/web/research.go`):

```
Phase 1: DECOMPOSE
  → LLM breaks query into sub-questions (3-8 depending on depth)
  → Each sub-question is a self-contained research unit
  → LLM generates initial search queries for each sub-question

Phase 2: EXPLORE (parallel per sub-question)
  → For each sub-question:
      a. Search EXA (type depends on depth: auto/deep/deep-reasoning)
      b. Fetch full text of top 2-5 results per sub-question
      c. Extract key claims, data points, citations from each source
      d. Track source provenance (URL, date, author, publication)

Phase 3: CROSS-REFERENCE
  → LLM compares findings across sub-questions
  → Identifies:
      - Corroborated claims (supported by 2+ independent sources)
      - Contested claims (conflicting evidence)
      - Single-source claims (needs flagging)
      - Knowledge gaps (sub-questions with thin coverage)

Phase 4: VALIDATE ASSUMPTIONS
  → LLM lists implicit assumptions in current findings
  → For each assumption: search EXA for contrary evidence
  → Flag assumptions as: verified / plausible / unverified / contested

Phase 5: GAP FILL
  → LLM identifies gaps from Phase 3
  → Generates targeted follow-up searches
  → (Up to 30% of total source budget allocated to gap filling)

Phase 6: SYNTHESIZE
  → LLM writes structured research report:

    # Research Report: <query>
    **Date:** <ISO date> | **Sources consulted:** N | **Depth:** <level>

    ## Executive Summary
    (2-3 paragraph synthesis of findings)

    ## Key Findings
    1. **Finding title** — evidence with citations
    2. ...

    ## Evidence Map
    | Claim | Sources | Confidence | Notes |
    |---|---|---|---|
    | ... | [Source A](url), [Source B](url) | High/Med/Low | ... |

    ## Contradictions & Debates
    - **Topic:** Position A vs Position B — sources, analysis

    ## Assumptions Validated
    | Assumption | Status | Evidence |
    |---|---|---|
    | ... | Verified/Plausible/Unverified/Contested | ... |

    ## Knowledge Gaps
    - **Gap:** what's missing, why it matters, how to fill

    ## Source List
    1. [Title](URL) — Author, Date, Publication — relevance notes
    2. ...

    ## Methodology Notes
    (Search strategy, depth level, source selection criteria)

Phase 7: SAVE (if --save/--wiki)
  → Write report to wiki/synthesis/
  → Extract entities/concepts found during research
  → Offer to create stub wiki pages for significant findings
```

**Depth levels:**

| Level | Sub-Questions | Max Sources | Search Type | Validation | Best For |
|---|---|---|---|---|---|
| `shallow` | 3-4 | 8 | `auto` | No | Quick overview, fact checks |
| `medium` | 5-6 | 20 | `deep-lite` | Light (key assumptions) | Analysis, comparisons |
| `deep` | 6-8 | 35 | `deep` | Full pass | Investment decisions, academic research |

## 15d. Agent & Research Prompt Design

### Agent system prompt (`pkg/web/prompts.go`)

```go
const agentSystemPrompt = `You are a research analyst agent backed by EXA neural web search.
Your goal is to answer the user's question by searching the web, reading results,
and synthesizing a comprehensive answer.

Workflow:
1. Review the initial search results carefully
2. Identify what's covered and what substantive information is missing
3. If gaps exist, formulate precise follow-up search queries
4. When sufficient information is gathered, synthesize your answer

Synthesis guidelines:
- Cite every factual claim with [Source Title](URL)
- When sources disagree, present both perspectives
- Distinguish between established facts and emerging/contested claims
- Note when a claim comes from a single source
- Be transparent about uncertainty — say "I found no evidence for..." not "X is false"
- Prefer primary sources (original research, official docs) over secondary
- Note source recency and relevance

You have access to these search types:
- auto: balanced speed/quality
- deep: multi-step reasoning (use for complex questions)
- deep-reasoning: maximum depth (use for research-level questions)

Output format: Markdown with sections, inline citations, and a sources list.`
```

### Research decompose prompt

```go
const researchDecomposePrompt = `Decompose the following research question into sub-questions.
Each sub-question should be self-contained and answerable through web search.

Research question: {query}

Return a JSON array of sub-questions. For each:
{
  "question": "the sub-question",
  "rationale": "why this needs its own search",
  "search_query": "optimized EXA search query (be specific, use keywords)",
  "priority": "high|medium|low"
}

Guidelines:
- Cover different dimensions: technical, historical, comparative, critical
- Include a "contrary evidence" question to find opposing views
- Include a "latest developments" question for timeliness
- Generate appropriate search queries (EXA works well with natural language queries)`
```

### Research validate prompt

```go
const researchValidatePrompt = `Review these research findings and identify implicit assumptions.
For each assumption, determine if the sources validate it or if contrary evidence exists.

Findings:
{findings}

Return JSON array:
{
  "assumptions": [
    {
      "assumption": "the implicit assumption",
      "source_claim": "what in the findings implies this",
      "status": "verified|plausible|unverified|contested",
      "evidence": "brief evidence summary",
      "search_query": "query to validate or refute this assumption"
    }
  ]
}`
```

### Research synthesis prompt

```go
const researchSynthesisPrompt = `Synthesize the following research findings into a structured report.

Research question: {query}
Depth: {depth}
Sources consulted: {source_count}
Sub-question findings:
{sub_question_results}
Cross-reference analysis:
{cross_ref}
Assumption validation:
{assumptions}

Generate a comprehensive research report with these sections:
1. Executive Summary (2-3 paragraphs)
2. Key Findings (numbered, with citations)
3. Evidence Map (table: claim, sources, confidence, notes)
4. Contradictions & Debates (positions, sources, analysis)
5. Assumptions Validated (table: assumption, status, evidence)
6. Knowledge Gaps (what's missing, why it matters)
7. Source List (numbered, with relevance notes)
8. Methodology Notes

Writing guidelines:
- Cite every claim: [Source N]
- Use confidence ratings: High (3+ independent sources), Medium (2 sources), Low (1 source or tentative)
- Be precise about what is and isn't known
- Highlight practical implications
- Note temporal context (when sources were published)
- Keep the report self-contained (someone should understand it without reading sources)`
```

## 15e. Config Integration

### CUE schema addition (`pkg/config/schema/config.cue`)

```cue
// EXAConfig defines the EXA web search API settings.
EXAConfig: {
    api_key:  string | *""         // from EXA_API_KEY env var
}
```

### CUE types addition (`pkg/config/schema/types.cue`)

Add to `ProjectConfig`:
```cue
ProjectConfig: {
    project?:    string
    llm:         LLMConfig
    typesense:   TypesenseConfig
    exa?:        EXAConfig
    pipeline?:   PipelineConfig
    collections: [string]: CollectionConfig
}
```

### Go config addition (`pkg/config/config.go`)

```go
type Config struct {
    // ... existing fields ...
    EXA         EXAConfig                    `json:"exa,omitempty"`
}

type EXAConfig struct {
    APIKey string `json:"-"`  // from EXA_API_KEY env var
}
```

In `Load()`:
```go
cfg.EXA.APIKey = os.Getenv("EXA_API_KEY")
```

No CUE config file changes needed for typical use — API key comes from env var.
Users can set `exa: api_key: "sk-..."` in `config.cue` instead if they prefer.

## 15f. Integration with wiki

### `--save` / `--wiki` flag

When `--save` or `--wiki <name>` is passed to `gmd web agent` or `gmd web research`:

1. Resolve wiki collection (by `--wiki` name, or first wiki-enabled collection)
2. Write the agent/research output to `wiki/synthesis/YYYY-MM-DD-query-slug.md`
   with frontmatter:
   ```yaml
   ---
   type: synthesis
   tags: [web-research, <query-keywords>]
   sources: [<url1>, <url2>, ...]
   date: 2026-05-31
   tool: gmd web research
   ---
   ```
3. If research mode: offer to extract entities/concepts into stub wiki pages
4. Auto-trigger `gmd update` for the wiki collection so the new page is indexed

### Wiki lint gap analysis (future)

The `gmd wiki lint` gap analysis prompt (`agent_prompts.go:LintGapPrompt`)
already mentions "suggested web search queries to fill the gaps". Once web
commands exist, this can be enhanced to:

1. Lint identifies gaps → outputs EXA search queries
2. User runs `gmd web research` with those queries
3. Results auto-saved to wiki

### Wiki ingest from URLs (future)

`gmd wiki ingest https://example.com/article` is already planned but returns
"not yet implemented". The `pkg/exa` client's `GetContents` method provides
the clean markdown extraction needed to implement this.

## 15g. Implementation Phases

### Phase W1: EXA Client (`pkg/exa/`)

**Files:**
```
pkg/exa/
├── client.go      # HTTP client, New(), Search(), GetContents(), FindSimilar(), Answer()
├── types.go       # All request/response structs
└── exa_test.go    # Tests with mock HTTP server
```

- `net/http` stdlib only, no external dependencies
- JSON marshal/unmarshal for request/response
- Rate limit retry with exponential backoff
- Context propagation for timeouts
- Unit tests with `httptest.Server`

### Phase W2: CLI Commands — fetch + search (no LLM)

**Files:**
```
cmd/gmd/
├── web.go             # Parent "web" command (just shows help)
├── web_fetch.go       # gmd web fetch
└── web_search.go      # gmd web search
```

- Both commands are pure wrappers — no LLM, no agent logic
- Config from env var `EXA_API_KEY`, validated at runtime
- Output formatting: CLI text table (default), JSON (`--json`)
- Standard Cobra flag patterns from existing commands

### Phase W3: Agent loop (`pkg/web/agent.go`)

**Files:**
```
pkg/web/
├── agent.go           # Agent loop: search → analyze → search more → synthesize
├── prompts.go         # Embedded prompt constants
└── agent_test.go      # Tests
cmd/gmd/web_agent.go   # CLI command
```

- Agent struct holds `*exa.Client`, `*llm.Client`, max steps, max results per step
- Loop:
  1. Initial search (EXA)
  2. LLM analysis: parse results, decide next action (DONE / SEARCH_MORE with queries)
  3. Follow-up searches (EXA) if needed
  4. LLM synthesis: final answer with citations
- LLM responses parsed from structured markdown (the LLM is instructed to output
  a `## ACTION` section with `DONE` or `SEARCH_MORE` and `## QUERIES` with newline-separated queries)
- Max search steps enforced; after max steps, synthesize anyway

### Phase W4: Deep research agent (`pkg/web/research.go`)

**Files:**
```
pkg/web/
├── research.go        # Research pipeline: decompose → explore → cross-ref → validate → fill → synthesize
└── research_test.go   # Tests
cmd/gmd/web_research.go # CLI command
```

- Research struct holds `*exa.Client`, `*llm.Client`, config (depth, max sources)
- Pipeline phases executed sequentially with LLM at decision points
- Parallel searches within Phase 2 (Explore) using `golang.org/x/sync/errgroup`
- Structured output: report sections as defined above
- Wiki integration: `--save` writes to wiki collection

### Phase W5: Config + Polish

- Add `EXAConfig` to CUE schema and Go config
- Add `--wiki` flag integration to research/agent commands
- Cost tracking: print EXA cost per command run (`$0.0023 for 5 searches`)
- Graceful degradation: if `EXA_API_KEY` is missing, print clear error
- `gmd doctor` extension: check EXA API connectivity
- Integration tests with recorded HTTP fixtures

### Phase W6: Advanced Features (future)

- **Streaming output:** `--stream` flag for agent/research to show intermediate results
- **Multi-source batch:** research across both local Typesense index AND web
- **Saved research sessions:** `gmd web research --session myproject` stores intermediate state, can resume
- **Citation quality scoring:** evaluate source authority, recency, independence
- **Web → wiki pipeline:** `gmd web research "topic" --save --create-entities` auto-creates wiki entity pages for discovered concepts
- **Search history:** `gmd web history` shows past web searches with cost

## 15h. File Layout

```
.design/websearch.md              # This document

pkg/exa/
├── client.go                     # HTTP client: Search, GetContents, FindSimilar, Answer
├── types.go                      # Request/response structs
└── exa_test.go                   # Tests with httptest.Server

pkg/web/
├── agent.go                      # Agent loop: search → analyze → search more → synthesize
├── research.go                   # Research pipeline: decompose → explore → cross-ref → validate → fill → synthesize
├── prompts.go                    # LLM prompts (//go:embed optional for longer prompts)
├── agent_test.go                 # Tests
└── research_test.go              # Tests

cmd/gmd/
├── web.go                        # Parent "web" command
├── web_fetch.go                  # gmd web fetch
├── web_search.go                 # gmd web search
├── web_agent.go                  # gmd web agent
└── web_research.go               # gmd web research

pkg/config/schema/
├── types.cue                     # Add EXAConfig type
└── config.cue                    # Add exa: {} block

pkg/config/config.go              # Add EXAConfig to Config struct + env var loading
```

## 15i. Key Design Decisions

### 15j. No SDK dependency — thin HTTP wrapper

The community Exa Go SDK is outdated (missing `deep`, `deep-reasoning`,
`outputSchema`, `systemPrompt`, `stream`). A thin `net/http` wrapper is ~200
lines of code and directly mirrors the REST API. This is faster to implement,
keeps dependencies minimal, and never falls out of sync with the API.

### 15k. Fetch and search are LLM-free

These are pure API wrappers. No tokens burned, no latency from LLM calls, no
config complexity. The user can pipe results to any LLM they want. This keeps
the tool composable.

### 15l. Agent and research use GMD's existing LLM client

No separate LLM configuration. The agent and research commands use the same
`pkg/llm.Client` (expansion model for chat) that `gmd query` and `gmd wiki`
use. This means local LLMs (Ollama, vLLM) work out of the box.

### 15m. Research is structured but not rigid

The research pipeline has fixed phases, but each phase is LLM-driven. The LLM
decides what sub-questions to ask, what assumptions to test, and what gaps to
fill. The structure provides reliability; the LLM provides intelligence.

### 15n. Wiki integration is opt-in

Web commands work standalone. `--save` adds wiki persistence. No cyclical
dependency: `pkg/web` does not import `pkg/wiki`; CLI commands that need both
wire them together.

### 15o. EXA API key from env var, not CUE config file

`EXA_API_KEY` environment variable is the primary config method. Users can
override in CUE config if they prefer. Same pattern as `OPENAI_API_KEY` and
`GMD_TYPESENSE_API_KEY`.

### 15p. Cost transparency

Every command prints EXA cost at the end:
```
$ gmd web search "golang generics" --type deep --limit 5 --text
...results...
Cost: $0.0047 (5 searches @ deep, 5 content fetches)
```
This prevents bill shock and helps users understand the cost of different
search types.

## 15q. Priorities & Dependencies

| Priority | Task | Depends On |
|---|---|---|
| 1 | `pkg/exa/` client + types | Nothing (stdlib only) |
| 2 | `gmd web fetch` + `gmd web search` | EXA client |
| 3 | EXA config (CUE schema + Go struct) | Nothing (can do in parallel with #1) |
| 4 | `pkg/web/agent.go` (agent loop) | EXA client + LLM client |
| 5 | `gmd web agent` CLI | Agent loop |
| 6 | `pkg/web/research.go` (deep research) | Agent loop (shares search+LLM pattern) |
| 7 | `gmd web research` CLI | Research pipeline |
| 8 | `--save` / `--wiki` integration | Wiki collection detection |
| 9 | Wiki lint gap → web search | Both subsystems exist |
| 10 | Wiki ingest from URL via EXA | EXA client + wiki agent |
| 11 | Streaming output (`--stream`) | Agent loop |
| 12 | Saved research sessions | Research pipeline |

**Quick wins (no dependencies beyond Phase W1):** `gmd web fetch` and `gmd web
search` — pure API wrappers, ~100 lines each, immediately useful.

**Core value:** `gmd web agent` and `gmd web research` — these are the
differentiated features that go beyond a simple API wrapper. The agent provides
guided exploration; the research pipeline provides structured, validated,
citation-backed reports.
