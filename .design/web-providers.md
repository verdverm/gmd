# Web Providers — Multi-Provider Architecture for `gmd web`

**Status: Proposal** — 2025-06-05

GMD is expanding from a single EXA-backed web search tool into a multi-provider
system spanning search/discovery and content retrieval. The first pass focuses on
fetch and crawl. Browser sessions, AI-driven browser control, and input
simulation are covered in `.design/web-browser-advanced.md`.

## Rationale

| Why Expand | What It Enables |
|---|---|
| Search-only (`exa`) is one category | `gmd web search`, `gmd web fetch` work today |
| Expanding search providers (Tavily, SearXNG) adds choice | Self-hosted search, different indexes, pricing flexibility |
| Browser automation is a distinct category | Crawl JS-heavy pages, extract structured data |
| MCP ecosystem growth | Agents control browsers via CDP/MCP — GMD's MCP server exposes these |
| No single provider covers all use cases | Venn diagram — pick a provider per workflow |

**Search providers** (EXA, Tavily, SearXNG) index and retrieve existing
content. **Browser providers** (Cloudflare, Browserbase, local Rod) render pages
in real time. Some providers blur the line — Cloudflare Browser Run's
`/content` and `/markdown` endpoints fetch any URL and return rendered content,
functioning as an on-demand content retrieval service even though it sits under
the "browser" product umbrella. This is why providers can register in multiple
roles.

## Design Decisions

1. **Multiple interfaces, not one.** Forcing all providers into one interface
   creates stub methods and abstraction leaks. Each command selects its
   interface at runtime. `SearchProvider` for web index search,
   `BrowserProvider` for content retrieval, crawl, scrape, and rendered-content.

2. **Named provider groups in config.** Modeled after `searchDefaults`
   (which maps preset names to lists of collection/wiki sources), provider
   groups map a preset name to `{search, browser}` provider selections.
   The global `WebConfig.group` field selects the default group; CLI
   `--provider-group` overrides per-command. This lets users define
   "quick" (exa + cloudflare), "offline" (searxng + local), and other
   workflows as named presets.

3. **User controls provider selection.** No automatic fallback. The user
   configures a provider group (search + browser) or passes
   `--provider-group` per command. Individual `--search-provider` and
   `--browser-provider` flags allow overriding one role within a group.
   Missing or unavailable providers produce errors.

4. **SDKs preferred over raw HTTP wrappers.** Cloud provider clients should use
   official Go SDKs when available and well-maintained. Fall back to raw HTTP
   only when no Go SDK exists or the SDK is unmaintained. SDKs handle auth,
   retries, rate limits, and API evolution.

5. **Provider registry is explicit, not init-time.** Provider name→constructor
   mappings live in a single central file (`pkg/web/registry.go`). Adding a
   provider means adding one line to the appropriate role map plus writing the
   provider package. No `init()` ordering, no blank imports.

6. **Robots.txt and rate limiting for local crawl.** Local crawling enforces
   robots.txt, per-domain rate limits, and configurable delays between requests.
   Cloud providers handle this on their end; the interface is provider-agnostic.

7. **Local execution is a first-class provider category.** Static HTTP fetch
   and HTML→MD conversion — with no cloud dependencies, API keys, or JS
   runtime — is independently useful for privacy-sensitive and offline
   workflows. Browser automation is additive, not a prerequisite.

8. **Credentials via environment variables.** API keys and secrets are loaded
   from env vars, never stored in CUE config files. Provider config blocks
   reference env var names as defaults. Omitting a provider block from config
   means "don't use this provider."

9. **HTML→MD conversion approach is still being evaluated.** Pure Go libraries
   (html-to-markdown, semantic-markdown) and subprocess-based converters
   (Python markitdown, JS Turndown) are both under consideration. The Go binary
   constraint (`CGO_ENABLED=0`) doesn't preclude calling external tools.

10. **Content caching at the provider level.** Not all content needs live
    retrieval. EXA returns cached/indexed content; Cloudflare renders on
    demand; Local fetches fresh each time. The `BrowserProvider.GetContent`
    method accepts a `GetContentOptions` struct with `MaxAge` — providers
    that cache content can honor it. A future local content cache (on-disk,
    keyed by URL + fetch time) can optionally sit between the caller and
    provider for offline/repeat access. Caching strategy is detailed below.

## Provider Landscape

### Category 0: Local Execution (no cloud)

Local execution supports offline and privacy-sensitive workflows. It covers
static HTTP fetch, HTML→MD conversion, and respectful crawling — all without
API keys or cloud dependencies.

#### The JS Requirement Spectrum

Not all pages need a browser. Content falls on a spectrum:

| Page Type | Example | Requires Browser? | Approach |
|---|---|---|---|
| Static HTML / SSR | blogs, docs, most server-rendered sites | No | `net/http` fetch + HTML→MD conversion |
| Hybrid (partial JS) | pages with lazy-loaded sections | Maybe | Fetch DOM, optionally hydrate with JS |
| Full SPA | React/Angular/Vue apps, infinite scroll | Yes | Headless browser (Rod / cloud provider) |

The question per request: can the content be obtained via HTTP alone, or is
JS execution required? Fetch via HTTP first; fall back to a browser when the
result is empty or placeholder content.

SPA detection is heuristic-based: an empty `<body>` or a bootstrap `<div>`
with no readable text (e.g. `<div id="root">` in React/Vue apps) indicates
content needs JS rendering. The open question is whether heuristic gating is
reliable enough, or whether a browser should always be attempted when the
initial static fetch yields no usable text. This interacts with cost — browser
rendering is slower and more expensive per request.

#### HTML to Markdown Libraries

| Library | Stars | Approach | CGO? | Features |
|---|---|---|---|---|
| `JohannesKaufmann/html-to-markdown/v2` | 3.6k | Pure Go, `x/net/html` parser | No | Plugin system, CommonMark, tables, strikethrough |
| `thorstenpfister/semantic-markdown` | newer | Pure Go, content-aware | No | Main content extraction, URL refification |
| `conductor-oss/markitdown` | newer | Pure Go, multi-format | No (WASM PDF) | PDF/DOCX/HTML all in one |

`html-to-markdown/v2` is the leading pure-Go candidate: built on
`golang.org/x/net/html` (already an indirect dep), pluggable, goroutine-safe.
Subprocess-based alternatives (Python `markitdown`, JS Turndown) are also
under evaluation for conversion quality. Writing a custom converter from
scratch is not under consideration — the maintenance burden of HTML parsing
edge cases is too high.

Subprocess calls are on the table:

| Approach | Example | Pros | Cons |
|---|---|---|---|
| Pure Go library | `html-to-markdown/v2`, `semantic-markdown` | No external deps, fast startup | Narrower feature set, less battle-tested |
| Subprocess (Python) | `markitdown` | Best-in-class conversion, active ecosystem | Requires Python runtime, IPC overhead |
| Subprocess (JS/TS) | Turndown | Mature, widely used | Requires Node.js runtime, IPC overhead |

Pure Go keeps the binary self-contained but may sacrifice conversion quality
for edge cases. Subprocess converters bring richer ecosystems but add runtime
dependencies and startup latency. A hybrid approach — prefer pure Go, with
optional subprocess fallback for problem URLs — is worth evaluating.

#### Respectful Crawling

Behaviors to enforce for local crawling:

- **robots.txt** parsing and enforcement (via `temoto/robotstxt`)
- **Per-domain rate limiting** with configurable delay
- **Queue management** with per-domain scheduling
- **User-agent** declaration and `Crawl-delay` directive support
- **sitemap.xml** discovery for seed URLs

No single Go library covers all crawling behaviors cover-to-cover. Use
`temoto/robotstxt` for robots.txt parsing and implement per-domain rate
limiting, queue management, and sitemap discovery in `pkg/web/local/` directly.

#### Local Execution Matrix

| Provider | Static Fetch | JS Render | Crawl | HTML→MD | API Key |
|---|---|---|---|---|---|
| Rod (CDP) | yes (via browser) | yes | yes | yes (via HTML→MD) | none |
| `net/http` + HTML→MD | yes | no | limited | yes | none |
| `net/http` only | yes (raw HTML) | no | limited | no | none |

Rod evaluation and local browser sessions are covered in
`.design/web-browser-advanced.md`.

#### Package Structure

```
pkg/web/local/
├── client.go         # LocalProvider struct, constructor, Capabilities()
├── fetch.go          # Static HTTP fetch via net/http
├── markdown.go       # HTML→MD conversion via html-to-markdown/v2
├── crawl.go          # Crawling with robots.txt, rate limits, queue
├── scrape.go         # CSS selector extraction via goquery
├── rod.go            # Rod-based browser automation (future)
├── browser_linux.go  # Chromium path detection (future)
├── browser_darwin.go # Chromium path detection (future)
└── client_test.go    # Tests (unit + integration-tagged for live browser)
```

### Category 1: Search / Content Discovery

| Provider | Search | Fetch Content | Content/Markdown | Find Similar | Cost Model | API Key Needed |
|---|---|---|---|---|---|---|
| **EXA** | yes semantic + keyword | yes | no | yes | Pay-per-query | `EXA_API_KEY` |
| **Cloudflare** | no | yes (any URL → rendered) | yes (`/content`, `/markdown`) | no | Per-browser-hour | `CLOUDFLARE_API_KEY` |
| **Tavily** | yes | yes extract | no | no | Pay-per-query | planned |
| **SearXNG** | yes self-host | no | no | no | Free (self-host) | none |

A trend across providers is `content-type: text/markdown` responses — EXA
returns markdown in its `text` field, Cloudflare's `/markdown` endpoint converts
rendered pages to markdown, and Tavily offers markdown extraction. This aligns
with GMD's LLM-oriented consumption patterns.

Cloudflare spans both categories. Its `/content` and `/markdown` endpoints
retrieve and render any URL on demand — content discovery without web indexing
(Category 1). Its full browser product provides automation, crawl, and scrape
(Category 2). It is listed in both the search/discovery and browser/product
tables above; content retrieval makes it a first-class member of the discovery
category even though it does not implement `SearchProvider` (web index query).

EXA also spans both roles: its `/search` endpoint implements `SearchProvider`,
and its `/contents` endpoint (cached page retrieval) implements
`BrowserProvider.GetContent`.

### Category 2: Browser Automation (cloud)

| Provider | GetContent | Crawl | Scrape | JS Render | Self-Host |
|---|---|---|---|---|---|
| **Cloudflare Browser Run** | yes | yes | yes | yes | no |
| **Browserbase** | yes | yes | yes | yes | no |
| **Browserless** | yes | yes | yes | yes | yes (Docker) |
| **Steel.dev** | yes | yes | yes | yes | yes (OSS) |
| **Bright Data** | yes | yes | yes | yes | no |
| **Scrapfly** | yes | yes | yes | yes | no |
| **Hyperbrowser** | yes | yes | yes | yes | no |

Provider details (session support, stealth, live view, CDP) are covered in
`.design/web-browser-advanced.md`.

### Category 3: LLM-Centric Agent Frameworks

Stagehand, Browser Use, and Playwright MCP sit on top of browser providers and
add AI-driven page understanding and control. These map to the `AIBrowser`
interface. Covered in `.design/web-browser-advanced.md`.

### Pricing Snapshot

| Provider | Free Tier | Entry Paid | Billing Unit | Effective Rate |
|---|---|---|---|---|
| **Local HTML→MD** | local only | $0 | per-use | — |
| EXA | 1000 queries/mo | pay-as-you-go | per-query | ~$0.003/query |
| Cloudflare Browser Run | 10 min/day (Free) or 10 hrs/mo (Paid) | Workers Paid $5/mo | per-browser-hour | $0.09/hr |
| Browserbase | 1000 min/mo | $20/mo (100 hrs) | per-minute | ~$0.10–0.12/hr |
| Browserless | 1000 units/mo | $25/mo (annual) | 30s connection units | ~$0.23/hr equiv |
| Steel.dev | 100 hrs/mo | $29/mo (290 hrs) | credit-based | ~$0.10/hr |

## Venn Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     SEARCH / DISCOVERY                       │
│  EXA, Tavily, SearXNG                                       │
│  ┌────────────────────┐  ┌──────────────────────────────┐   │
│  │ Semantic search    │  │ Browser automation           │   │
│  │ Web index query    │  │ JS rendering                 │   │
│  │ Find similar       │  │ Crawl, scrape                │   │
│  │                    │  │ Content→markdown             │   │
│  │                    │  │                              │   │
│  │  ┌─────────────┐   │  │   ┌──────────────────┐      │   │
│  │  │ O V E R L A P│   │  │   │ O V E R L A P     │      │   │
│  │  │ EXA /contents│   │  │   │ Content fetch     │      │   │
│  │  │ (cached page │   │  │   │ Crawl             │      │   │
│  │  │  retrieval)  │   │  │   │ Links extraction  │      │   │
│  │  │              │   │  │   │ EXA GetContents,  │      │   │
│  │  │              │   │  │   │ Cloudflare /content│     │   │
│  │  └─────────────┘   │  │   └──────────────────┘      │   │
│  └────────────────────┘  └──────────────────────────────┘   │
│                                      │                       │
│                            ┌─────────▼────────┐              │
│                            │ AI BROWSER TOOLS  │              │
│                            │ (advanced doc)    │              │
│                            │ Stagehand,        │              │
│                            │ Browser Use,      │              │
│                            │ Playwright MCP    │              │
│                            └──────────────────┘              │
└─────────────────────────────────────────────────────────────┘
```
- **Search providers** (EXA, Tavily, SearXNG): query web indexes, return ranked
  results. EXA's `/contents` endpoint (cached page retrieval) maps to
  `BrowserProvider`.
- **Browser providers** (Local, Cloudflare, EXA): retrieve and render content
  on demand via `GetContent`, with different freshness/latency tradeoffs.
- **AI browser tools** (Stagehand, Browser Use, Playwright MCP): AI-driven
  abstractions on top of browser providers. Covered in
  `.design/web-browser-advanced.md`.

## Interface Design

The existing `Provider` interface in `pkg/web/provider.go` bundles search and fetch:

```go
type Provider interface {
    Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error)
    Fetch(ctx context.Context, urls []string) ([]SearchResult, error)
}
```

This conflates two concerns: querying a web index (search) and retrieving
page content (fetch). Several providers can do one but not the other:
SearXNG searches but doesn't fetch, Cloudflare fetches but doesn't search,
Local fetches but doesn't search. Forcing both methods into a single
interface requires stub implementations that return `ErrNotSupported`.

The fix: `SearchProvider` does search only. `BrowserProvider` owns content
retrieval (`GetContent`). Commands that need both (e.g. `gmd web research`)
compose the two interfaces.

### SearchProvider (Category 1)

```go
type SearchProvider interface {
    Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error)
}
```

`SearchOptions` gains an `Extra map[string]any` field for provider-specific
parameters (e.g., EXA's `useAutoprompt`, `type`, `outputSchema`). Callers
pass keys the target provider understands; adapters ignore unknown keys.

`SearchResult` also gains `Extra` so provider-specific response fields
(`Author`, `PublishedDate`, `Highlights` from EXA) flow through to callers
that know which provider they're talking to, without leaking provider types
into the interface:

```go
type SearchResult struct {
    Title   string
    URL     string
    Content string
    Score   float64
    Extra   map[string]any // provider-specific fields (Author, PublishedDate, Highlights, etc.)
}

type SearchOptions struct {
    Query          string
    NumResults     int
    IncludeDomains []string
    ExcludeDomains []string
    Extra          map[string]any // provider-specific params (useAutoprompt, type, outputSchema, etc.)
}
```

Implemented by: **EXA**, **Tavily**, **SearXNG**.

### BrowserProvider (Category 0 + 2)

```go
type BrowserProvider interface {
    GetContent(ctx context.Context, url string, opts *GetContentOptions) (string, error)
    Crawl(ctx context.Context, startURL string, opts *CrawlOptions) ([]Page, error)
    Scrape(ctx context.Context, url string, selector string) ([]Element, error)
    Capabilities() BrowserCapabilities
}
```

`GetContent` subsumes the old `Fetch`. Every browser provider implements it:
EXA via its `/contents` cached-page endpoint, Cloudflare via `/content` and
`/markdown` live rendering, Local via `net/http` + HTML→MD. Providers that
support additional methods (Crawl, Scrape) advertise that in
`Capabilities()`; commands check capabilities before calling those methods.

**GetContentOptions:**

```go
type GetContentOptions struct {
    Format     string        // "text", "markdown", "html" (default: "markdown")
    MaxChars   int           // max characters to return (0 = unlimited)
    MaxAge     time.Duration // prefer live fetch if cached content is older than this
    Extra      map[string]any
}
```

**Supporting types:**

```go
type CrawlOptions struct {
    MaxDepth       int
    MaxPages       int
    SameDomain     bool
    IncludePattern string
    ExcludePattern string
    Timeout        time.Duration
}

type Page struct {
    URL     string
    Title   string
    Content string   // rendered HTML or markdown
    Status  int
    Depth   int
    Links   []string
    Error   string
}

type Element struct {
    Tag   string
    Text  string
    HTML  string
    Attrs map[string]string
}
```

`NewSession`/`BrowserSession` (CDP sessions, interactive control) are part
of the advanced browser surface — see `.design/web-browser-advanced.md`.

Implemented by: **Local**, **Cloudflare**, **EXA** (GetContent only; Crawl
and Scrape return `ErrNotSupported`).

### BrowserCapabilities

```go
type BrowserCapabilities struct {
    GetContent   bool // supports GetContent() — always true for any BrowserProvider
    Crawl        bool // supports Crawl()
    Scrape       bool // supports Scrape()
    SelfHost     bool // can be self-hosted
    LocalBrowser bool // headless browser available on this machine
    LocalHTML    bool // can do static HTML→MD without a browser
    LocalCrawl   bool // can do respectful local crawling

    Features []string // e.g. "playwright", "puppeteer", "stagehand"
}
```

Session-related capabilities (CDPEndpoint, SessionRecord, LiveView, Stealth) are
in the extended `BrowserCapabilities` covered in the advanced doc.

### LocalProvider (Category 0)

`LocalProvider` implements `BrowserProvider`:

- `GetContent`: static HTTP fetch + HTML→MD conversion (always available).
- `Crawl`: respectful crawling with robots.txt enforcement, per-domain rate
  limits, and sitemap discovery.
- `Scrape`: CSS selector extraction via `goquery` (which wraps
  `golang.org/x/net/html`, already an indirect dependency). No JS needed;
  works on static HTML.

```go
type LocalProvider struct {
    httpClient  *http.Client
    mdConverter HTMLToMarkdownConverter // interface — backs pure Go or subprocess impl
    // Future: rodClient *rodBrowser (when Rod is adopted)
}
```

| Runtime State | Capabilities |
|---|---|
| Default | Static fetch + HTML→MD + crawl + scrape |
| `GMD_NO_BROWSER=1` or `--no-browser` flag | Static fetch + HTML→MD only (Crawl/Scrape false) |

Rod-based JS rendering is a future addition covered in the advanced doc.
`LocalProvider` does NOT implement `SearchProvider` — it cannot query a web
index.

### LocalProvider Dependencies

```
github.com/JohannesKaufmann/html-to-markdown/v2   # HTML→MD (or subprocess alternative)
github.com/temoto/robotstxt                        # robots.txt parsing
github.com/PuerkitoBio/goquery                     # CSS selector support for Scrape()
```

All three meet the `CGO_ENABLED=0` constraint. `goquery` wraps
`golang.org/x/net/html` (already an indirect dependency of the project via
other packages).

### HTML→Markdown Integration (Reference)

If the pure-Go path is chosen, the integration looks like this:

```go
package local

import (
    "github.com/JohannesKaufmann/html-to-markdown/v2"
    "github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
    "github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
    "github.com/JohannesKaufmann/html-to-markdown/v2/plugin/strikethrough"
)

func newConverter() *converter.Converter {
    return converter.NewConverter(
        converter.WithPlugins(
            commonmark.NewCommonmarkPlugin(
                commonmark.WithStrongDelimiter("**"),
                commonmark.WithEmphasisDelimiter("_"),
            ),
            table.NewTablePlugin(),
            strikethrough.NewStrikethroughPlugin(),
        ),
    )
}
```

For subprocess-based converters, the integration is an `exec.Command` wrapper
with stdin/stdout streaming rather than a direct library call.
```

### Agent Refactoring

The existing `pkg/web/agent.go` hardcodes `*exa.Client` and uses
EXA-specific result fields (`Author`, `PublishedDate`, `Highlights`).

| Approach | Description | When |
|---|---|---|
| **A: Stay EXA-specific** | Agent keeps `*exa.Client` directly. No interface abstraction. | Now — no other search providers exist yet |
| **B: SearchProvider + Extra** | Agent takes `SearchProvider`; uses common `SearchResult` fields. Provider-specific extras accessed through `SearchResult.Extra["highlights"]` etc. and `SearchOptions.Extra` for params. | When a second search provider ships (Tavily / SearXNG) |
| **C: Provider-specific agents** | `NewEXAAgent`, `NewTavilyAgent` — each optimized for its provider. | If providers diverge too much for a single agent shape |

**Recommendation:** Start with **A** (no change to agent.go in Phase 1). Move to **B**
when a second search provider lands. Fall back to **C** if needed.

### Error Taxonomy

```go
// pkg/web/errors.go

var (
    ErrNotSupported        = errors.New("gmd/web: operation not supported by provider")
    ErrProviderNotFound    = errors.New("gmd/web: provider not found in registry")
    ErrBrowserNotAvailable = errors.New("gmd/web: browser not available on this machine")
    ErrAuthFailed          = errors.New("gmd/web: authentication failed")
    ErrRateLimited         = errors.New("gmd/web: rate limited by provider")
    ErrTimeout             = errors.New("gmd/web: request timed out")
    ErrSSRFBlocked         = errors.New("gmd/web: request blocked — private/internal IP")
)

type ProviderError struct {
    Provider string
    Err      error
    Detail   string
}
func (e *ProviderError) Error() string { ... }
func (e *ProviderError) Unwrap() error { return e.Err }
```

Existing EXA-specific helpers (`IsRateLimit`, `IsAuthError`) stay in the
`exa` package. The registry's `Resolve` returns `ErrProviderNotFound`. Commands
check `errors.Is(err, ErrNotSupported)` after `Capabilities()` checks and wrap
with `ProviderError` for actionable messages.

### Cost Display

```go
type CostSummary struct {
    Provider string
    Cost     float64
    Unit     string // "query", "minute", "credit"
    Currency string // "USD"
}
```

Providers return `CostSummary` alongside results. The CLI renders it
generically, replacing the EXA-specific `printCost` pattern. Billing models
differ per provider (per-query, per-minute, credit-based) — `Unit` makes the
distinction visible without leaking provider logic into display code.

## Caching Strategy

Content retrieval sits on a freshness spectrum. Different providers occupy
different points on it, and the caching strategy must account for all of them
without tying the interface to any one model.

### Provider Freshness Models

| Provider | Content Source | Freshness | Latency |
|---|---|---|---|
| EXA `/contents` | Indexed crawl cache | Hours to days old | Low (pre-indexed) |
| Cloudflare `/content` | Live browser render | Real-time | Medium (browser startup + render) |
| Local `net/http` | Live HTTP fetch | Real-time | Low (single HTTP round-trip) |

### Two-Layer Caching

**Layer 1 — Provider-level:** Providers that cache internally (EXA) honor
`GetContentOptions.MaxAge`. If the cached content is older than `MaxAge`,
EXA triggers a live crawl (via its `livecrawl` parameter). Cloudflare and
Local always fetch live (MaxAge is a no-op).

**Layer 2 — Application-level (future):** An optional local disk cache in
`pkg/web/cache/` sits between the caller and provider. Cache entries are
keyed by `(provider, url, format)`, stored as markdown files under
`~/.cache/gmd/web/`. On read, if a cache entry is fresher than
`MaxAge`, it's returned without calling the provider. On write, successful
provider responses populate the cache.

```
Caller → [cache check] → [cache hit, fresh] → return
                       → [cache miss / stale] → Provider.GetContent() → [write cache] → return
```

This layer is optional and orthogonal to the provider interfaces — providers
don't know about it. The cache is enabled/disabled in `LocalConfig`:

```cue
LocalConfig: {
    // ...
    cache_enabled?: bool | *false
    cache_dir?:     string | *"~/.cache/gmd/web"
    cache_max_size?: int   | *536870912  // 512 MB
    cache_ttl?:      string | *"24h"     // default TTL for cached entries
}
```

### Cache-Aware GetContentOptions

`GetContentOptions.MaxAge` drives both layers:

- `MaxAge = 0`: always prefer live/fresh content (cache bypass)
- `MaxAge > 0`: cached content up to this age is acceptable
- `MaxAge < 0` (e.g. `-1`): prefer cached content regardless of age
  (offline mode)

The CLI's `--max-age` flag on `gmd web fetch` maps to this field.

## Provider Registry

Provider names map to constructors via a central, explicit map — no `init()`
magic, no blank imports. Each provider package exports a constructor; the
registry file imports those packages and wires them up directly.

```go
// pkg/web/registry.go

type ProviderConstructor func(cfg ProviderConfig) (any, error)

type ProviderRegistry struct {
    search  map[string]ProviderConstructor
    browser map[string]ProviderConstructor
}

func NewRegistry() *ProviderRegistry {
    return &ProviderRegistry{
        search: map[string]ProviderConstructor{
            "exa":     func(cfg ProviderConfig) (any, error) { return exa.NewSearchProvider(cfg) },
            "tavily":  func(cfg ProviderConfig) (any, error) { return tavily.NewSearchProvider(cfg) },
            "searxng": func(cfg ProviderConfig) (any, error) { return searxng.NewSearchProvider(cfg) },
        },
        browser: map[string]ProviderConstructor{
            "exa":        func(cfg ProviderConfig) (any, error) { return exa.NewBrowserProvider(cfg) },
            "cloudflare": func(cfg ProviderConfig) (any, error) { return cloudflare.NewBrowserProvider(cfg) },
            "local":      func(cfg ProviderConfig) (any, error) { return local.NewBrowserProvider(cfg) },
        },
    }
}

func (r *ProviderRegistry) Resolve(role, name string, cfg ProviderConfig) (any, error)
func (r *ProviderRegistry) ValidateName(role, name string) error
```

Each role map is built once at startup. Adding a new provider means adding
one line to the appropriate role map and writing the provider package.

**Supported provider names per role:**

| Role | Valid Names |
|---|---|
| `search` | `exa`, `tavily`, `searxng` |
| `browser` | `exa`, `cloudflare`, `local` |

**Cross-category providers:** Cloudflare appears in both discovery and
browser tables above; it does not implement `SearchProvider` but its content
retrieval endpoints (`/content`, `/markdown`) make it a full participant in
the discovery workflow. The registry tracks it under the `browser` role since
that's the interface it implements. EXA is the only provider registered in
both `search` and `browser` roles.

## Required Credentials per Provider

| Provider | Account | Env Vars | Notes |
|---|---|---|---|
| **EXA** | [exa.ai](https://exa.ai) | `EXA_API_KEY` | Free tier: 1000 queries/mo |
| **Cloudflare Browser Run** | [dash.cloudflare.com](https://dash.cloudflare.com) | `CLOUDFLARE_API_KEY`, `CLOUDFLARE_ACCOUNT_ID` | Workers Paid $5/mo; 10 hrs/mo free |
| **Tavily** | [tavily.com](https://tavily.com) | `TAVILY_API_KEY` | Pay-per-query |
| **SearXNG** | self-host | none | Set `searxng.base_url` in config |
| **Local** | none | none | No credentials needed |

Each provider config block in CUE references these env vars as default values.
Omitting a config block means "don't use this provider" — the tool only
initializes providers referenced by the active provider group.

## Config Evolution

Current (`pkg/config/schema/types.cue`):
```cue
WebConfig: {
    provider?: string | *"exa"
    exa?:      EXAConfig
}
```

Proposed:
```cue
WebConfig: {
    group?:  string | *"default"  // active provider group
    groups?: [string]: WebProviderGroup

    local?:      LocalConfig
    exa?:        EXAConfig
    cloudflare?: CloudflareConfig
}

WebProviderGroup: {
    search?:  string  // provider name for search role (exa, tavily, searxng)
    browser?: string  // provider name for browser role (exa, cloudflare, local)
}

LocalConfig: {
    chromium_path?: string | *""

    // Disable browser automation (static fetch + HTML→MD only)
    no_browser?:    bool   | *false

    // Maximum bytes for static HTTP fetch (default: 10MB)
    html_max_size?: int    | *10485760

    // Crawl tuning: minimum delay between requests to the same domain (ms)
    crawl_delay_ms?: int | *1000

    // Maximum number of domains crawled concurrently
    max_concurrent_domains?: int | *2

    // Maximum pages to fetch per domain during a crawl
    max_pages_per_domain?: int | *200

    // Cache tuning
    cache_enabled?:  bool   | *false
    cache_dir?:      string | *"~/.cache/gmd/web"
    cache_max_size?: int    | *536870912   // 512 MB
    cache_ttl?:      string | *"24h"
}

CloudflareConfig: {
    api_key:    string | *""   // from CLOUDFLARE_API_KEY env var
    account_id: string | *""   // from CLOUDFLARE_ACCOUNT_ID env var
}
```

Go-side struct:

```go
type WebConfig struct {
    Group      string                        `json:"group"`
    Groups     map[string]WebProviderGroup   `json:"groups,omitempty"`
    Local      LocalConfig                   `json:"local,omitempty"`
    EXA        EXAConfig                     `json:"exa,omitempty"`
    Cloudflare CloudflareConfig              `json:"cloudflare,omitempty"`
}

type WebProviderGroup struct {
    Search  string `json:"search,omitempty"`
    Browser string `json:"browser,omitempty"`
}
```

API keys are loaded from environment variables in the config loading path,
matching the existing pattern for `EXA_API_KEY`.

### Provider Group Design Rationale

Named groups follow the `searchDefaults` pattern already used for
collections and wikis. `searchDefaults` maps a preset name to a list of
source names; `WebConfig.groups` maps a preset name to `{search, browser}`
role selections. Both are flat maps of `string` → structured value, with a
top-level field (`group` / implicit default) selecting the active preset.

This supports common workflows:

| Group Name | Config | Use Case |
|---|---|---|
| `default` | `{search: "exa", browser: "cloudflare"}` | Full-featured: indexed search + live rendering |
| `offline` | `{search: "searxng", browser: "local"}` | Self-hosted search + static HTTP fetch |
| `quick` | `{search: "exa", browser: "exa"}` | Single-provider simplicity, no live rendering |
| `research` | `{search: "tavily", browser: "cloudflare"}` | Alternative index + live content for deep research |

The `--provider-group` flag overrides the configured group per-command.
Individual `--search-provider` and `--browser-provider` flags override one
role within the active group without changing the other.

Per-provider endpoint overrides are not included. The provider config blocks
specify identity and credentials only. A generic HTTP proxy setting may be
warranted later as a top-level config option for enterprise environments, but
per-provider endpoint customization is premature.

## CLI Command Mapping

| `gmd web` Subcommand | Interface Needed | Local | EXA | Cloudflare | Status |
|---|---|---|---|---|---|
| `gmd web search` | `SearchProvider` | no | yes | no | existing |
| `gmd web fetch` | `BrowserProvider.GetContent` | yes static | yes cached | yes /content | existing (EXA only) |
| `gmd web agent` | `SearchProvider` + LLM | no | yes | no | existing |
| `gmd web crawl` | `BrowserProvider.Crawl` | yes | no | yes | new |
| `gmd web research` | `SearchProvider` + `BrowserProvider` + LLM | search: no, browser: yes | yes | planned | new |

### Fetch vs. Crawl

| Command | Intent | Scope | Recursive? | Use Case |
|---|---|---|---|---|
| `gmd web fetch` | Retrieve content for specific, known URLs | Explicit URL list | No — exactly the given URLs | "Get me the content of these 3 pages" |
| `gmd web crawl` | Discover and retrieve pages starting from a seed URL | Seed URL + discovered links | Yes — bounded by depth/pages/domain | "Starting from this page, grab everything relevant" |

`fetch` is point retrieval: the caller provides the URLs. `crawl` is
graph traversal: the provider discovers URLs by following links. They share
the same `GetContent` mechanism under the hood, but the control flow differs:
fetch iterates a flat list, crawl manages a queue with depth and dedup.

- **Local** = execution on the user's machine (static fetch + HTML→MD, no JS).
- For `fetch`, local first tries static HTTP and converts to markdown. If the
  result is empty or consists only of an SPA bootstrap `<div>` (e.g.,
  `<div id="root">` with no readable text), and a browser is available, it
  falls back to JS rendering.
- `--search-provider` and `--browser-provider` flags override individual
  roles in the active provider group per-call.
- `--provider-group <name>` overrides the entire active group per-call.
- `--max-age <duration>` flag on `fetch` controls cache freshness: content
  older than the specified duration triggers a browser fetch instead of
  using cached/indexed results.

## Provider Selection Logic

```
gmd web <command> [--provider-group <name>] [--search-provider <name>] [--browser-provider <name>]
                    │
                    ▼
            ┌──────────────────────┐
            │ --provider-group set?│
            └────┬──────┬──────────┘
                 │      │
                YES     NO
                 │      │
                 ▼      ▼
          Use specified   Look up configured
          provider group   group from WebConfig.group
          from WebConfig.   (default: "default")
          groups[<name>]    │
                 │          ▼
                 ▼    ┌──────────────────────┐
                 │    │ --search-provider or │
                 │    │ --browser-provider?  │
                 │    └────┬──────┬──────────┘
                 │         │      │
                 │        YES     NO
                 │         │      │
                 │         ▼      ▼
                 │    Override   Use group's
                 │    one role   role selection
                 │    │          │
                 └──────────────┘
                        │
                        ▼
              ┌──────────────────┐
              │ Resolve provider │
              │ from registry    │
              └────┬──────┬──────┘
                   │      │
                  found  not found
                   │      │
                   ▼      ▼
               Use it    Error:
                         "No provider <name>
                          configured for <role>.
                          Available: <list>"
```

No automatic fallback. The user declares which provider handles each role
via a named group. CLI flags override individual roles or the entire group.
If the resolved provider is unavailable at runtime, the command errors rather
than switching to a different provider.

Supported provider names: `exa`, `tavily`, `searxng`, `cloudflare`, `local`.

## Implementation Phases

### Phase 1: Interface Refinement & Foundations

Goal: split the single `Provider` interface, build the registry, wire EXA as the
first adapter for both roles, and update CLI commands without breaking existing
functionality.

- [ ] Deprecate `Provider`; define `SearchProvider` (search-only) and `BrowserProvider` (GetContent, Crawl, Scrape, Capabilities)
- [ ] Add `Extra map[string]any` to `SearchResult` and `SearchOptions`
- [ ] Define `GetContentOptions` struct (Format, MaxChars, MaxAge, Extra)
- [ ] Define `BrowserCapabilities` struct (GetContent, Crawl, Scrape, SelfHost, LocalBrowser, LocalHTML, LocalCrawl, Features)
- [ ] Define `CrawlOptions`, `Page`, `Element` types
- [ ] Implement provider registry (`pkg/web/registry.go`) — explicit map, no `init()`
- [ ] Define error taxonomy sentinels (`pkg/web/errors.go`)
- [ ] Implement `ProviderError` wrapping in command dispatch — wrap provider errors before user display so messages always include the provider name
- [ ] Implement `CostSummary` return from providers (providers return cost alongside results)
- [ ] Add generic cost display in CLI output (`pkg/output/`) — replaces EXA-specific `printCost`
- [ ] Create EXA search adapter (`pkg/web/exa/search.go`) implementing `SearchProvider` over `*exa.Client`
- [ ] Create EXA browser adapter (`pkg/web/exa/browser.go`) implementing `BrowserProvider` over `*exa.Client` (GetContent only; Crawl/Scrape return ErrNotSupported)
- [ ] Update CLI commands (`cmd/gmd/web_*.go`) to use typed provider interfaces
- [ ] Wire `gmd web fetch` to use `BrowserProvider.GetContent` instead of direct EXA client
- [ ] Add `Capabilities()` check before dispatching to Crawl/Scrape
- [ ] Update CUE schema with `WebProviderGroup` and `WebConfig.groups`; add Go-side struct
- [ ] Add `--provider-group`, `--search-provider`, `--browser-provider` flags
- [ ] Keep `pkg/web/agent.go` EXA-specific (option A)

**Testing:**
- Compile-time interface assertions: `var _ SearchProvider = (*exa.SearchAdapter)(nil)`, `var _ BrowserProvider = (*exa.BrowserAdapter)(nil)`
- Registry unit tests: resolution, missing providers, unknown names, cross-role lookups
- `httptest.Server` mocks for EXA search and contents adapters with recorded API response fixtures
- CLI integration: existing `gmd web search/fetch/agent` commands continue to work

### Phase 2: Cloudflare Provider

Goal: implement Cloudflare Browser Run as a `BrowserProvider` (GetContent,
Crawl). Cloudflare is the first new provider after EXA, proving the
multi-provider architecture works end-to-end.

- [ ] Create `pkg/web/cloudflare/client.go` — thin HTTP wrapper over Quick Actions REST API
- [ ] Implement `BrowserProvider.GetContent` via `/content` and `/markdown`
- [ ] Implement `BrowserProvider.Crawl`
- [ ] Add `gmd web crawl` command
- [ ] Register `cloudflare` in the `browser` role only

**Testing:**
- Unit: `httptest.Server` mocks with recorded Cloudflare API response fixtures
- Integration (`//go:build integration`): live smoke test, skipped if `CLOUDFLARE_API_KEY` unset
- Contract: `var _ BrowserProvider = (*cloudflare.BrowserClient)(nil)`

### Phase 3: Additional Search Providers (Tavily, SearXNG)

Goal: expand search provider coverage. Tavily and SearXNG are both pure search
providers — they implement `SearchProvider` only, adding choice without
introducing new interface shapes.

- [ ] Tavily provider (`pkg/web/tavily/`) — `SearchProvider`
- [ ] SearXNG provider (`pkg/web/searxng/`) — `SearchProvider`
- [ ] Register both in the `search` role

**Testing:**
- Unit: `httptest.Server` mocks with recorded API response fixtures
- Integration: live tests for each, skipped if API keys / instance absent
- Contract: `var _ SearchProvider = (*tavily.SearchClient)(nil)`, `var _ SearchProvider = (*searxng.SearchClient)(nil)`

### Phase 4: Local Provider — Fetch & Crawl

Goal: deliver the local provider for static HTTP fetch, HTML→MD conversion,
respectful crawling, and CSS selector scraping. No browser dependency, no JS
rendering.

#### New Dependencies

```
github.com/JohannesKaufmann/html-to-markdown/v2   (or subprocess approach)
github.com/temoto/robotstxt
github.com/PuerkitoBio/goquery
```

All three meet the `CGO_ENABLED=0` constraint. The HTML→MD library is not
final — see Open Questions for the pure-Go vs. subprocess evaluation.
`goquery` wraps `golang.org/x/net/html` (already an indirect dependency)
to provide jQuery-style CSS selector support for `BrowserProvider.Scrape()`.

#### Package: `pkg/web/local/`

Package structure defined in [Category 0 Package Structure](#package-structure)
above. Phase 4 delivers the files listed there except `rod.go` and
`browser_*.go` (deferred to the advanced browser doc).

#### Checklist

- [ ] Resolve HTML→MD conversion approach (pure Go library or subprocess)
- [ ] `go get github.com/temoto/robotstxt github.com/PuerkitoBio/goquery`
- [ ] `pkg/web/local/client.go` — `LocalProvider` struct, `NewLocalProvider()`, `Capabilities()` implements `BrowserProvider`
- [ ] `pkg/web/local/fetch.go` — `GetContent(ctx, url, opts)` via `net/http`, SSRF protection, timeout, max size, HTML→MD conversion
- [ ] `pkg/web/local/markdown.go` — `HTMLToMarkdown(ctx, html)` using chosen converter
- [ ] `pkg/web/local/crawl.go` — crawling with:
  - robots.txt parsing and enforcement (`temoto/robotstxt`)
  - Per-domain rate limiting with configurable delay
  - Max depth, same-domain constraint
  - Cycle detection via URL canonicalization
  - Sitemap discovery for seed URLs
- [ ] `pkg/web/local/scrape.go` — `Scrape(ctx, url, selector)` using `goquery` for CSS selector matching on static HTML
- [ ] Register `local` in the provider registry (`browser` role only)
- [ ] Implement Layer 2 local content cache (`pkg/web/cache/`) — disk-backed, keyed by `(provider, url, format)`, honoring `MaxAge`/`cache_ttl` from `LocalConfig`

**Testing:**
- Unit: HTML fixtures → markdown output verification
- Unit: HTML fixture → CSS selector extraction via goquery
- Unit: mock HTTP server for fetch tests (timeout, redirect, SSRF block)
- Integration (`//go:build integration`): crawl a local HTTP test server with
  robots.txt, rate limits, and multi-page link graphs
- Contract: `var _ BrowserProvider = (*local.LocalProvider)(nil)`

### Phase 5: Research Agent

Goal: build `gmd web research` — deep research using SearchProvider + LLM, and
refactor the existing agent to the provider interface.

Research is a workflow composed over existing interfaces (SearchProvider +
BrowserProvider + LLM), not a new provider interface. Some providers may offer
research-specific endpoints in the future, but the initial implementation uses
the same provider dispatch as other commands.

- [ ] `gmd web research` command — deep research agent loop
  - Sub-question generation, cross-referencing, citation tracking
  - Uses `SearchProvider` for discovery and `BrowserProvider` for live-fetch sources
  - Works with any provider combination in the active group
- [ ] Refactor `pkg/web/agent.go` to use `SearchProvider` interface (option B from Agent Refactoring)
  - EXA-specific fields accessed through `SearchResult.Extra`

**Testing:**
- Unit: mock providers for research agent workflow tests
- Integration: live research runs against EXA, skipped if `EXA_API_KEY` unset

### Test Fixtures Structure

```
pkg/web/testdata/           # shared HTML fixtures
pkg/web/exa/testdata/       # EXA API response recordings
pkg/web/cloudflare/testdata/# Cloudflare API response recordings
pkg/web/local/testdata/     # crawl test server pages, robots.txt fixtures
pkg/web/tavily/testdata/    # Tavily API response recordings
pkg/web/searxng/testdata/   # SearXNG API response recordings
```

### Contract Tests (compile-time)

```go
// Search providers
var _ SearchProvider  = (*exa.SearchAdapter)(nil)
var _ SearchProvider  = (*tavily.SearchClient)(nil)
var _ SearchProvider  = (*searxng.SearchClient)(nil)

// Browser providers
var _ BrowserProvider = (*exa.BrowserAdapter)(nil)
var _ BrowserProvider = (*cloudflare.BrowserClient)(nil)
var _ BrowserProvider = (*local.LocalProvider)(nil)
```
