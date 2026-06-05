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
   interface at runtime. `SearchProvider` for search/fetch, `BrowserProvider`
   for crawl/scrape/rendered-content.

2. **Provider roles in config.** A user configures `search: exa` and
   `browser: local` for different workflows. The `providers` object in CUE
   maps each role to a provider name.

3. **User controls provider selection.** No automatic fallback and no default
   provider. The user must explicitly set `providers.browser` /
   `providers.search` in config or pass `--provider` per command. Missing or
   unavailable providers produce errors.

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
├── rod.go            # Rod-based browser automation (future)
├── browser_linux.go  # Chromium path detection (future)
├── browser_darwin.go # Chromium path detection (future)
└── client_test.go    # Tests (unit + integration-tagged for live browser)
```

### Category 1: Search / Content Discovery

| Provider | Search | Fetch Content | Content/Markdown | Find Similar | Cost Model | API Key Needed |
|---|---|---|---|---|---|---|
| **EXA** | yes semantic + keyword | yes | no | yes | Pay-per-query | `EXA_API_KEY` |
| **Cloudflare** | no | yes | yes /content, /markdown | no | Per-browser-hour | `CLOUDFLARE_API_KEY` |
| **Tavily** | yes | yes extract | no | no | Pay-per-query | planned |
| **SearXNG** | yes self-host | no | no | no | Free (self-host) | none |

A trend across providers is `content-type: text/markdown` responses — EXA
returns markdown in its `text` field, Cloudflare's `/markdown` endpoint converts
rendered pages to markdown, and Tavily offers markdown extraction. This aligns
with GMD's LLM-oriented consumption patterns.

Cloudflare Browser Run is a **browser** product by category, but its `/content`
and `/markdown` endpoints function as an on-demand content retrieval service: any
URL → rendered markdown. It does not index the web or perform semantic search,
but it can serve as a `SearchProvider` for the `Fetch` method when live
rendering is needed. In the registry, Cloudflare registers as both `browser`
and `search` roles.

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
│  │ Content fetch      │  │ Crawl, scrape                │   │
│  │ Find similar       │  │ Content→markdown             │   │
│  │                    │  │                              │   │
│  │  ┌─────────────┐   │  │   ┌──────────────────┐      │   │
│  │  │ O V E R L A P│   │  │   │ O V E R L A P     │      │   │
│  │  │ Markdown     │   │  │   │ Content fetch     │      │   │
│  │  │ Structured   │   │  │   │ Crawl             │      │   │
│  │  │ data extract │   │  │   │ Links extraction  │      │   │
│  │  │              │   │  │   │ (Cloudflare /content│   │   │
│  │  │              │   │  │   │  in search role)   │      │   │
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
- **Search providers** retrieve pre-computed content from a web index.
- **Browser providers** render pages on demand. Cloudflare's content/markdown
  endpoints occupy the overlap: they render like a browser but return structured
  markdown suitable for search-like consumption.
- **AI browser tools** are AI-driven abstractions on top of browser providers.
  Covered in `.design/web-browser-advanced.md`.

## Interface Design

The existing `Provider` interface in `pkg/web/provider.go` covers search and fetch:

```go
type Provider interface {
    Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error)
    Fetch(ctx context.Context, urls []string) ([]SearchResult, error)
}
```

Multiple interfaces reflect the Venn diagram. A single monolithic interface
would require all providers to implement all methods.

### SearchProvider (Category 1)

```go
type SearchProvider interface {
    Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error)
    Fetch(ctx context.Context, urls []string) ([]SearchResult, error)
}
```

`SearchOptions` gains an `Extra map[string]any` field for provider-specific
parameters (e.g., EXA's `useAutoprompt`, `type`, `outputSchema`). Callers
pass keys the target provider understands; adapters ignore unknown keys.
This keeps the core interface stable while giving advanced callers an
escape hatch without leaking provider types into the interface.

Implemented by: **EXA**, **Tavily** (future), **SearXNG** (future),
**Cloudflare** (Fetch only, via `/content`/`/markdown`)

### BrowserProvider (Category 2)

```go
type BrowserProvider interface {
    GetContent(ctx context.Context, url string) (string, error)
    Crawl(ctx context.Context, startURL string, opts *CrawlOptions) ([]Page, error)
    Scrape(ctx context.Context, url string, selector string) ([]Element, error)
    Capabilities() BrowserCapabilities
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

### BrowserCapabilities

```go
type BrowserCapabilities struct {
    Crawl        bool // supports Crawl()
    Scrape       bool // supports Scrape()
    SelfHost     bool // can be self-hosted
    LocalBrowser bool // headless browser available on this machine
    LocalHTML    bool // can do static HTML→MD
    LocalCrawl   bool // can do respectful local crawling

    Features []string // e.g. "playwright", "puppeteer", "stagehand"
}
```

Session-related capabilities (CDPEndpoint, SessionRecord, LiveView, Stealth) are
in the extended `BrowserCapabilities` covered in the advanced doc.

### LocalProvider (Category 0)

`LocalProvider` implements **both** `SearchProvider` and `BrowserProvider`:

- `SearchProvider.Fetch` is served by static HTTP fetch + HTML→MD conversion
  (always available).
- `BrowserProvider.GetContent` uses the same static fetch + HTML→MD path.
- `BrowserProvider.Crawl` and `Scrape` are served when crawling is available.

```go
type LocalProvider struct {
    mdConverter HTMLToMarkdownConverter // interface — backs pure Go or subprocess impl
    // Future: rodClient *rodBrowser (when Rod is adopted)
}
```

| Runtime State | Capabilities |
|---|---|
| Default | Static fetch + HTML→MD + crawl (`SearchProvider.Fetch`, `BrowserProvider.GetContent/Crawl`) |
| `GMD_NO_BROWSER=1` or `--no-browser` flag | Static fetch + HTML→MD only |

Rod-based JS rendering is a future addition covered in the advanced doc.

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
| **B: SearchProvider + fallback** | Agent takes `SearchProvider`; uses common `SearchResult` fields. Provider-specific extras through `SearchOptions.Extra`. | When a second search provider ships (Tavily / SearXNG) |
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
            "exa":       func(cfg ProviderConfig) (any, error) { return exa.NewSearchProvider(cfg) },
            "tavily":    func(cfg ProviderConfig) (any, error) { return tavily.NewSearchProvider(cfg) },
            "searxng":   func(cfg ProviderConfig) (any, error) { return searxng.NewSearchProvider(cfg) },
            "cloudflare":func(cfg ProviderConfig) (any, error) { return cloudflare.NewSearchProvider(cfg) },
        },
        browser: map[string]ProviderConstructor{
            "local":      func(cfg ProviderConfig) (any, error) { return local.NewBrowserProvider(cfg) },
            "cloudflare": func(cfg ProviderConfig) (any, error) { return cloudflare.NewBrowserProvider(cfg) },
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
| `search` | `exa`, `tavily`, `searxng`, `cloudflare` |
| `browser` | `local`, `cloudflare` |

A provider can appear in multiple roles (`local` is search and browser;
`cloudflare` is search and browser). `Resolve` returns typed `error` values
so callers can distinguish configuration errors from runtime failures.

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
initializes providers that appear in `providers.<role>`.

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
    providers?: WebProviderRoles

    local?:      LocalConfig
    exa?:        EXAConfig
    cloudflare?: CloudflareConfig
}

WebProviderRoles: {
    search?:  string
    browser?: string
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
}

CloudflareConfig: {
    api_key:    string | *""   // from CLOUDFLARE_API_KEY env var
    account_id: string | *""   // from CLOUDFLARE_ACCOUNT_ID env var
}
```

Go-side struct:

```go
type WebConfig struct {
    Providers  *WebProviderRoles `json:"providers,omitempty"`
    Local      LocalConfig       `json:"local,omitempty"`
    EXA        EXAConfig         `json:"exa,omitempty"`
    Cloudflare CloudflareConfig  `json:"cloudflare,omitempty"`
}

type WebProviderRoles struct {
    Search  string `json:"search"`
    Browser string `json:"browser"`
}
```

API keys are loaded from environment variables in the config loading path,
matching the existing pattern for `EXA_API_KEY`.

Per-provider endpoint overrides are not included. The provider config blocks
specify identity and credentials only. A generic HTTP proxy setting may be
warranted later as a top-level config option for enterprise environments, but
per-provider endpoint customization is premature.

## CLI Command Mapping

| `gmd web` Subcommand | Interface Needed | Local | EXA | Cloudflare | Status |
|---|---|---|---|---|---|---|
| `gmd web search` | `SearchProvider` | no | yes | no | existing |
| `gmd web fetch` | `SearchProvider` or `BrowserProvider` | yes static | yes cached | yes /content | existing (EXA only) |
| `gmd web agent` | `SearchProvider` + LLM | no | yes | no | existing |
| `gmd web crawl` | `BrowserProvider` | yes | no | yes | new |
| `gmd web research` | `SearchProvider` + LLM | no | yes | planned | new |

- **Local** = execution on the user's machine (static fetch + HTML→MD, no JS).
- For `fetch`, local first tries static HTTP and converts to markdown. If the
  result is empty or consists only of an SPA bootstrap `<div>` (e.g.,
  `<div id="root">` with no readable text), and a browser is available, it
  falls back to JS rendering.
- `--provider` flag overrides the configured role per-call.
- `--max-age <duration>` flag on `fetch` controls cache freshness: content
  older than the specified duration triggers a browser fetch instead of
  using cached/indexed results.

## Provider Selection Logic

```
gmd web <command> [--provider <name>]
                    │
                    ▼
            ┌────────────────┐
            │ --provider set?│
            └────┬──────┬────┘
                 │      │
                YES     NO
                 │      │
                 ▼      ▼
          Use specified   Look up configured
          provider        provider role from
                           CUE config
                              │
                              ▼
                    ┌──────────────────┐
                    │ Provider found?  │
                    └────┬──────┬──────┘
                         │      │
                        YES     NO
                         │      │
                         ▼      ▼
                    Use it     Error:
                               "No provider configured
                                for <role>. Set with
                                gmd config or use
                                --provider <name>"
```

No automatic fallback. The user declares which provider handles each role. If
the configured provider is unavailable at runtime, the command errors rather
than switching to a different provider.

Supported provider names: `exa`, `tavily`, `searxng`, `cloudflare`, `local`.

## Implementation Phases

### Phase 1: Interface Refinement & Foundations

Goal: split the single `Provider` interface, build the registry, wire EXA as the
first adapter, and update CLI commands without breaking existing functionality.

- [ ] Split `Provider` into `SearchProvider` and `BrowserProvider` interfaces
- [ ] Add `Extra map[string]any` to `SearchOptions`
- [ ] Define `BrowserCapabilities` struct (Crawl, Scrape, SelfHost, LocalBrowser, LocalHTML, LocalCrawl, Features)
- [ ] Define `CrawlOptions`, `Page`, `Element` types
- [ ] Implement provider registry (`pkg/web/registry.go`) — explicit map, no `init()`
- [ ] Define error taxonomy sentinels (`pkg/web/errors.go`)
- [ ] Create EXA adapter (`pkg/web/exa/adapter.go`) implementing `SearchProvider` over `*exa.Client`
- [ ] Update CLI commands (`cmd/gmd/web_*.go`) to use `SearchProvider` interface
- [ ] Add `Capabilities()` check before dispatching to a browser provider
- [ ] Update CUE schema with `WebProviderRoles`; add Go-side struct
- [ ] Keep `pkg/web/agent.go` EXA-specific (option A)

**Testing:**
- Compile-time interface assertions: `var _ SearchProvider = (*exa.SearchAdapter)(nil)`
- Registry unit tests: resolution, missing providers, unknown names
- `httptest.Server` mocks for EXA adapter with recorded API response fixtures
- CLI integration: existing `gmd web search/fetch/agent` commands continue to work

### Phase 2: Cloudflare Provider

Goal: implement Cloudflare Browser Run as both a `BrowserProvider` (GetContent,
Crawl) and a `SearchProvider` (Fetch via `/content` and `/markdown` endpoints).
Cloudflare is the first new provider after EXA, proving the multi-provider
architecture works end-to-end.

- [ ] Create `pkg/web/cloudflare/client.go` — thin HTTP wrapper over Quick Actions REST API
- [ ] Implement `BrowserProvider.GetContent` via `/content` and `/markdown`
- [ ] Implement `SearchProvider.Fetch` via the same endpoints (Cloudflare as content-only search)
- [ ] Implement `BrowserProvider.Crawl`
- [ ] Add `gmd web crawl` command (or wire into existing fetch with `--max-age`)
- [ ] Register `cloudflare` in both `search` and `browser` roles

**Testing:**
- Unit: `httptest.Server` mocks with recorded Cloudflare API response fixtures
- Integration (`//go:build integration`): live smoke test, skipped if `CLOUDFLARE_API_KEY` unset
- Contract: `var _ SearchProvider = (*cloudflare.SearchClient)(nil)`, `var _ BrowserProvider = (*cloudflare.BrowserClient)(nil)`

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

Goal: deliver the local provider for static HTTP fetch, HTML→MD conversion, and
respectful crawling. No browser dependency, no JS rendering.

#### New Dependencies (candidates)

```
github.com/JohannesKaufmann/html-to-markdown/v2 v2.5.1   (or subprocess approach)
github.com/temoto/robotstxt
```

Both meet the `CGO_ENABLED=0` constraint. The HTML→MD library is not final —
see Open Questions for the pure-Go vs. subprocess evaluation.

#### Package: `pkg/web/local/`

```
pkg/web/local/
├── client.go         # LocalProvider struct, constructor, Capabilities()
├── fetch.go          # Static HTTP fetch via net/http
├── markdown.go       # HTML→MD conversion
├── crawl.go          # Crawling (robots.txt, rate limits, queue)
└── client_test.go    # Tests
```

#### Checklist

- [ ] Resolve HTML→MD conversion approach (pure Go library or subprocess)
- [ ] `go get github.com/temoto/robotstxt`
- [ ] `pkg/web/local/client.go` — `LocalProvider` struct, `NewLocalProvider()`, `Capabilities()` implements both `SearchProvider` and `BrowserProvider`
- [ ] `pkg/web/local/fetch.go` — `FetchStatic(ctx, url)` via `net/http`, SSRF protection, timeout, max size
- [ ] `pkg/web/local/markdown.go` — `HTMLToMarkdown(ctx, html)` using chosen converter
- [ ] `pkg/web/local/crawl.go` — crawling with:
  - robots.txt parsing and enforcement (`temoto/robotstxt`)
  - Per-domain rate limiting with configurable delay
  - Max depth, same-domain constraint
  - Cycle detection via URL canonicalization
  - Sitemap discovery for seed URLs
- [ ] Register `local` in the provider registry (both `search` and `browser` roles)

**Testing:**
- Unit: HTML fixtures → markdown output verification
- Unit: mock HTTP server for fetch tests (timeout, redirect, SSRF block)
- Integration (`//go:build integration`): crawl a local HTTP test server with
  robots.txt, rate limits, and multi-page link graphs
- Contract: `var _ SearchProvider = (*local.LocalProvider)(nil)`, `var _ BrowserProvider = (*local.LocalProvider)(nil)`

### Phase 5: Research Agent

Goal: build `gmd web research` — deep research using SearchProvider + LLM, and
refactor the existing agent to the provider interface.

Research is a workflow composed over existing interfaces (SearchProvider +
BrowserProvider + LLM), not a new provider interface. Some providers may offer
research-specific endpoints in the future, but the initial implementation uses
the same provider dispatch as other commands.

- [ ] `gmd web research` command — deep research agent loop
  - Sub-question generation, cross-referencing, citation tracking
  - Works with any `SearchProvider`
  - Can optionally use `BrowserProvider` for live-fetch sources
- [ ] Refactor `pkg/web/agent.go` to use `SearchProvider` interface (option B from Agent Refactoring)

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
var _ SearchProvider  = (*exa.SearchAdapter)(nil)
var _ SearchProvider  = (*cloudflare.SearchClient)(nil)
var _ SearchProvider  = (*tavily.SearchClient)(nil)
var _ SearchProvider  = (*searxng.SearchClient)(nil)
var _ SearchProvider  = (*local.LocalProvider)(nil)
var _ BrowserProvider = (*cloudflare.BrowserClient)(nil)
var _ BrowserProvider = (*local.LocalProvider)(nil)
```
