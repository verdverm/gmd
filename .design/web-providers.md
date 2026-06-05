# Web Providers — Multi-Provider Architecture for `gmd web`

**Status: Proposal** — 2025-06-05

GMD is expanding from a single EXA-backed web search tool into a multi-provider
system spanning search/discovery and browser automation. Providers do not share
a feature-aligned interface — they form a Venn diagram of overlapping but
distinct capabilities. This doc maps the landscape, defines the interfaces, and
lays out the incremental implementation plan.

## Rationale

| Why Expand | What It Enables |
|---|---|
| Search-only (`exa`) is one category | `gmd web search`, `gmd web fetch` work today |
| Browser automation is a distinct category | Crawl JS-heavy pages, extract structured data, interact with forms |
| MCP ecosystem growth | Agents control browsers via CDP/MCP — GMD's MCP server exposes these |
| No single provider covers all use cases | Venn diagram — pick a provider per workflow |

**Search providers** (EXA, Tavily, SearXNG) index and retrieve
existing content, while **browser providers** (Cloudflare Browser Run,
Browserbase, Browserless) render pages in real time via headless Chrome. They
are complementary, not competing.

## Provider Landscape

### Category 0: Local Execution (no cloud)

Local execution supports offline and privacy-sensitive
workflows. It covers a spectrum from HTTP fetch to
full browser rendering, with increasing dependency weight at each level.

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

#### Go Browser Automation Libraries — Research

| Library | Stars | Approach | CGO? | Chromium Mgmt | API Style |
|---|---|---|---|---|---|
| `go-rod/rod` | 6.9k | Pure Go CDP client | No | Auto-downloads, version-pinned | High-level fluent, auto-wait |
| `chromedp/chromedp` | 13k | Pure Go CDP client | No | System browser, no auto-dl | DSL task lists, verbose |
| `playwright-community/playwright-go` | 3.3k | Node.js RPC bridge | No (but needs Node) | Via playwright npm | Playwright API (multi-browser) |

Key differences:

- **Rod** is pure Go, auto-downloads Chromium, version-pinned, prevents orphaned
  processes, auto-wait for elements, thread-safe, 100% test coverage. Viability
  for this project is under evaluation. Auto-downloading a ~170MB binary adds
  startup overhead for a CLI tool.
- **chromedp** uses a DSL-like task system, can leave orphaned browser processes,
  requires users to install Chromium separately.
- **playwright-go** requires Node.js runtime (~50MB bridge) plus browser binaries.
  Supports Chromium + Firefox + WebKit.

#### HTML to Markdown Libraries — Research

| Library | Stars | Approach | CGO? | Speed | Features |
|---|---|---|---|---|---|
| `JohannesKaufmann/html-to-markdown/v2` | 3.6k | Pure Go, `x/net/html` parser | No | ~25 MB/s | Plugin system, CommonMark, tables, strikethrough |
| `thorstenpfister/semantic-markdown` | newer | Pure Go, content-aware | No | fast | Main content extraction, URL refification |
| `conductor-oss/markitdown` | newer | Pure Go, multi-format | No (WASM PDF) | moderate | PDF/DOCX/HTML all in one |

Key differences:

- **html-to-markdown/v2** is pure Go, built on `golang.org/x/net/html` (an
  indirect dep), plugin architecture, goroutine-safe, ~25 MB/s.
- **semantic-markdown** adds main content extraction and URL refification
  (output formatted for LLM consumption).
- **markitdown** handles PDF, DOCX, PPTX plus HTML, with a larger dependency
  footprint.

#### Respectful Crawling — Research

For local crawling, behaviors to enforce:

- **robots.txt** parsing and enforcement
- **Per-domain rate limiting** (configurable delay between requests to same host)
- **Queue management** with per-domain scheduling
- **User-agent** declaration and `Crawl-delay` directive support
- **sitemap.xml** discovery and parsing for seed URLs

Go library candidates:

| Library | Description | Status |
|---|---|---|
| `temoto/robotstxt` | robots.txt parser (Google's spec) | Active |
| `gocolly/colly` | Full scraping framework with rate limiting, caching, robots.txt | Large dependency tree; risk of CGO from transitive deps |
| `PuerkitoBio/gocrawl` | Crawler with robots.txt + delay + max parallelism | Last updated 2018 |
| `crawshaw/littleboss` | Small crawler | Last updated 2018 |

No single Go library covers all behaviors. Plan: use
`temoto/robotstxt` for robots.txt and implement per-domain rate limiting +
queue management in `pkg/web/local/` directly.

#### Local Execution Matrix

| Provider | Static Fetch | JS Render | Crawl | HTML→MD | API Key | What's Needed |
|---|---|---|---|---|---|---|
| Rod (CDP) | yes (via browser) | yes | yes | yes via HTML→MD | none | `go-rod/rod` + Chromium |
| `net/http` + HTML→MD | yes | no | limited | yes | none | `html-to-markdown/v2` |
| `net/http` only | yes (raw HTML) | no | limited | no | none | nothing (stdlib) |

#### Package Structure

```
pkg/web/local/
├── client.go         # LocalProvider struct, constructor, Capabilities()
├── rod.go            # Rod-based browser automation (if Rod is adopted)
├── fetch.go          # Static HTTP fetch via net/http
├── markdown.go       # HTML→MD conversion via html-to-markdown/v2
├── crawl.go          # Crawling with robots.txt, rate limits, queue
├── browser_linux.go  # Chromium path detection (Linux)
├── browser_darwin.go # Chromium path detection (macOS)
└── client_test.go    # Tests (unit + integration-tagged for live browser)
```

### Category 1: Search / Content Discovery

| Provider | Search | Fetch Content | Find Similar | Cost Model | API Key Needed |
|---|---|---|---|---|---|
| **EXA** | yes semantic + keyword | yes markdown | yes | Pay-per-query | `EXA_API_KEY` |
| **Tavily** | yes | yes extract | no | Pay-per-query | planned |
| **SearXNG** | yes self-host | no | no | Free (self-host) | none |

### Category 2: Browser Automation (cloud)

| Provider | Crawl | Scrape | JS Render | Structured Data | CDP | Stealth | Self-Host | API / SDK Docs |
|---|---|---|---|---|---|---|---|---|
| **Cloudflare Browser Run** | yes | yes | yes | yes /json | yes | no (docs say no) | no | [REST API](https://developers.cloudflare.com/browser-rendering/) |
| **Browserbase** | yes | yes | yes | yes Stagehand | yes | yes | no | [API docs](https://docs.browserbase.com/), [Go SDK](https://pkg.go.dev/github.com/browserbase/browserbase-go) |
| **Browserless** | yes | yes | yes | no | yes WebSocket | yes stealth flag | yes Docker | [REST API](https://docs.browserless.io/) |
| **Steel.dev** | yes | yes | yes | yes | yes | yes | yes OSS | [API docs](https://docs.steel.dev/) |
| **Bright Data** | yes | yes | yes | no | yes | yes | no | [API docs](https://docs.brightdata.com/) |
| **Scrapfly** | yes | yes | yes | no | yes | yes | no | [API docs](https://scrapfly.io/docs/) |
| **Hyperbrowser** | yes | yes | yes | yes | yes | yes | no | [API docs](https://docs.hyperbrowser.ai/) |

### Category 3: LLM-Centric Agent Frameworks (not raw browser APIs)

These tools sit on top of browser providers (CDP / Playwright) and add
AI-driven page understanding and control. They map to the `AIBrowser`
interface (see Interface Design below).

| Tool | Description | Relation |
|---|---|---|
| **Stagehand** | AI-native browser automation — `page.act("click submit")` over Playwright | Built by Browserbase, runs on any CDP browser |
| **Browser Use** | Open-source agent framework for LLM browser control | Uses CDP — can target any provider above |
| **Playwright MCP** | MCP server wrapping Playwright actions | Cloudflare ships one; runs on any Playwright-compatible provider |

### Pricing Snapshot

| Provider | Free Tier | Entry Paid | Billing Unit | Effective Hourly Rate |
|---|---|---|---|---|
| **Local Browser (Rod)** | local only | $0 | per-use | — |
| **Local HTML→MD** | local only | $0 | per-use | — |
| EXA | 1000 queries/mo | pay-as-you-go | per-query | ~$0.003/query |
| Cloudflare Browser Run | 10 min/day (Free) or 10 hrs/mo (Paid) | Workers Paid $5/mo | per-browser-hour | **$0.09/hr** |
| Browserbase | 1000 min/mo | $20/mo (100 hrs) | per-minute | ~$0.10–0.12/hr |
| Browserless | 1000 units/mo | $25/mo (annual) | 30s connection units | ~$0.23/hr equiv |
| Steel.dev | 100 hrs/mo | $29/mo (290 hrs) | credit-based | ~$0.10/hr |
| Scrapfly | 1000 credits | $30/mo (200k credits) | credits (time+bw) | varies |

## Venn Diagram: Feature Categories

```
┌─────────────────────────────────────────────────────────────┐
│                     SEARCH / DISCOVERY                       │
│  EXA, Tavily, SearXNG                                       │
│  ┌────────────────────┐  ┌──────────────────────────────┐   │
│  │ Semantic search    │  │ Browser automation           │   │
│  │ Web index query    │  │ JS rendering                 │   │
│  │ Content fetch      │  │ Crawl, scrape                │   │
│  │ Find similar       │  │ Form interaction             │   │
│  │                    │  │                              │   │
│  │  ┌─────────────┐   │  │   ┌──────────────────┐      │   │
│  │  │ O V E R L A P│   │  │   │ O V E R L A P     │      │   │
│  │  │ Markdown     │   │  │   │ Content fetch     │      │   │
│  │  │ Structured   │   │  │   │ Crawl             │      │   │
│  │  │ data extract │   │  │   │ Links extraction  │      │   │
│  │  │              │   │  │   └──────────────────┘      │   │
│  │  └─────────────┘   │  └──────────────────────────────┘   │
│  └────────────────────┘              │                       │
│                                      │                       │
│                            ┌─────────▼────────┐              │
│                            │ AI BROWSER TOOLS  │              │
│                            │ (AIBrowser)       │              │
│                            │ Stagehand         │              │
│                            │ Browser Use       │              │
│                            │ Playwright MCP    │              │
│                            │ MCP Server        │              │
│                            └──────────────────┘              │
└─────────────────────────────────────────────────────────────┘
```
- **Search providers** do not do browser rendering. They index the web and retrieve pre-computed content.
- **Browser providers** do real-time rendering. Some overlap on fetch/crawl, but implementations differ.
- **AI browser tools** (Stagehand, MCP) are AI-driven abstractions on top of browser providers, modeled as
  the `AIBrowser` interface (Category 3).
- No single provider covers all three circles.

## Interface Design

The existing `Provider` interface in `pkg/web/provider.go` covers search and fetch:

```go
type Provider interface {
    Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error)
    Fetch(ctx context.Context, urls []string) ([]SearchResult, error)
}
```

Multiple interfaces reflect the Venn diagram. A single monolithic
interface would require all providers to implement all methods.

### LocalProvider (Category 0)

`LocalProvider` implements **both** `SearchProvider` and `BrowserProvider`:

- `SearchProvider.Fetch` is served by static HTTP fetch + HTML→MD conversion
  (always available).
- `BrowserProvider` methods (Crawl, Scrape, FetchContent, NewSession) are served
  by Rod when Chromium is available. When Chromium is absent, `Capabilities()`
  reports `LocalBrowser: false` and callers can check before dispatching.

```go
type LocalProvider struct {
    // rodClient is the go-rod browser controller (nil if Chromium not found)
    rodClient *rodBrowser

    // htmlConverter converts raw HTML → Markdown (always available)
    htmlConverter *htmltomarkdown.Converter
}

// rodBrowser wraps go-rod/rod for CDP-based browser automation.
// Initialized only when the user has Chromium installed and configured.
type rodBrowser struct {
    browser *rod.Browser
}
```

| Runtime State | Rod Available? | Capabilities |
|---|---|---|
| User has Chromium installed; path configured | yes | Full browser: crawl, scrape, JS eval, plus static fetch + HTML→MD |
| No Chromium installed | no | Static fetch + HTML→MD only (`SearchProvider.Fetch`) |
| `GMD_NO_BROWSER=1` or `--no-browser` flag | no (disabled) | Static fetch + HTML→MD only |

Rod's `launcher.NewBrowser()` can auto-download Chromium (~170MB). This
behavior is opt-in. Users who want local browser automation
install Chromium or set `chromium_auto_download: true` in config.
Documentation covers setup paths per platform.

### SearchProvider (Category 1)

```go
type SearchProvider interface {
    Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error)
    Fetch(ctx context.Context, urls []string) ([]SearchResult, error)
}
```

Implemented by: **EXA**, **Tavily** (future), **SearXNG** (future)

### BrowserProvider (Category 2)

```go
type BrowserProvider interface {
    // Core: fetch rendered content from a URL
    FetchContent(ctx context.Context, url string) (string, error)    // rendered HTML/markdown

    // Core: crawl from a start URL, following links within scope
    Crawl(ctx context.Context, startURL string, opts *CrawlOptions) ([]Page, error)

    // Scrape structured elements from a page
    Scrape(ctx context.Context, url string, selector string) ([]Element, error)

    // Session-based control (for agent workflows)
    NewSession(ctx context.Context, opts *SessionOptions) (BrowserSession, error)

    // Capability introspection
    Capabilities() BrowserCapabilities
}

type BrowserSession interface {
    // CDP / WebSocket endpoint for direct control
    CDPEndpoint() string

    // High-level actions (delegated to provider's Playwright/Puppeteer if supported)
    Navigate(ctx context.Context, url string) error
    Click(ctx context.Context, selector string) error
    Fill(ctx context.Context, selector, value string) error
    Evaluate(ctx context.Context, js string) (string, error)

    Close(ctx context.Context) error
}
```

**Supporting types:**

```go
// CrawlOptions configures a crawl job.
type CrawlOptions struct {
    MaxDepth       int           // maximum link depth from start URL (default: 3)
    MaxPages       int           // maximum total pages to crawl (default: 50)
    SameDomain     bool          // stay within same domain (default: true)
    IncludePattern string        // URL path glob to include
    ExcludePattern string        // URL path glob to exclude
    Stealth        bool          // use stealth / evasion techniques
    Timeout        time.Duration // per-page timeout
}

// SessionOptions configures a browser session.
type SessionOptions struct {
    Timeout  time.Duration // session idle timeout
    Stealth  bool          // use stealth / evasion techniques
    Proxy    string        // proxy URL for the session
    Record   bool          // record session for replay
    LiveView bool          // enable live view URL
}

// Element represents a scraped DOM element.
type Element struct {
    Tag   string            // e.g. "div", "a", "span"
    Text  string            // visible text content
    HTML  string            // inner HTML
    Attrs map[string]string // element attributes
}

// Page represents a crawled or rendered page.
type Page struct {
    URL     string   // final URL after redirects
    Title   string   // page title
    Content string   // rendered HTML or markdown (provider-dependent)
    Status  int      // HTTP status code
    Depth   int      // crawl depth from root
    Links   []string // outbound links discovered on the page
    Error   string   // non-empty if page failed
}
```

**BrowserCapabilities:**

```go
// BrowserCapabilities describes what a browser provider can do.
// Stable fields are defined as booleans. Optional or provider-specific
// features (third-party frameworks, experimental capabilities) go in
// the Features set to avoid constantly expanding the struct.
type BrowserCapabilities struct {
    Crawl         bool // supports Crawl()
    Scrape        bool // supports Scrape()
    CDPEndpoint   bool // supports CDP WebSocket sessions
    SessionRecord bool // supports session recording
    LiveView      bool // supports live view
    Stealth       bool // supports stealth / evasion
    SelfHost      bool // can be self-hosted
    LocalBrowser  bool // headless browser available on this machine
    LocalHTML     bool // can do static HTML→MD via html-to-markdown/v2
    LocalCrawl    bool // can do respectful local crawling (robots.txt, rate limits)

    // Optional features provided by the provider.
    // Examples: "playwright", "puppeteer", "stagehand", "mcp-server"
    Features []string
}
```

### AIBrowser (Category 3)

AI-native browser control tools (Stagehand, Browser Use, Playwright MCP).
Implements the `AIBrowser` interface, distinct from the existing `Agent` struct
in `pkg/web/agent.go` (which is a research agent composing Search + LLM, not a
browser controller).

```go
type AIBrowser interface {
    // Provider returns the underlying browser provider for raw access
    Provider() BrowserProvider

    // Extracted structured data via natural language
    ExtractJSON(ctx context.Context, url string, schema any) (any, error)

    // Act on a page using natural language (Stagehand-style)
    Act(ctx context.Context, instruction string) error

    // Extract markdown from rendered page
    ToMarkdown(ctx context.Context, url string) (string, error)
}
```

### Multiple Interfaces

With a single interface:

- Every provider stubs most methods with `return nil, ErrNotSupported`
- The interface becomes a compatibility matrix, not a behavioral contract
- The `gmd web` commands require provider-specific knowledge
- Providers' pricing models differ (per-query vs per-minute vs per-session)
- Implementation effort: every new provider implements every method

Commands declare which interface they need, and config selects both a
provider **and a capability mode** (e.g., `--provider cloudflare --action crawl`).

## Provider Registry

Provider names map to constructors via a registry. Each provider package
registers itself at init time, keeping the mapping alongside the
implementation and avoiding a central switch statement.

```go
// pkg/web/registry.go

var Registry = &ProviderRegistry{
    search:    map[string]func(ProviderConfig) (SearchProvider, error){},
    browser:   map[string]func(ProviderConfig) (BrowserProvider, error){},
    aibrowser: map[string]func(ProviderConfig) (AIBrowser, error){},
}

func (r *ProviderRegistry) RegisterSearch(name string, ctor func(ProviderConfig) (SearchProvider, error))
func (r *ProviderRegistry) RegisterBrowser(name string, ctor func(ProviderConfig) (BrowserProvider, error))
func (r *ProviderRegistry) RegisterAIBrowser(name string, ctor func(ProviderConfig) (AIBrowser, error))
func (r *ProviderRegistry) Resolve(role, name string, cfg ProviderConfig) (any, error)
func (r *ProviderRegistry) ValidateName(role, name string) error
```

Each provider package calls the appropriate `RegisterX` in its `init()`:

```go
// pkg/web/exa/registry.go
func init() {
    web.Registry.RegisterSearch("exa", newEXASearchProvider)
}

// pkg/web/local/registry.go
func init() {
    web.Registry.RegisterSearch("local", newLocalSearchProvider)
    web.Registry.RegisterBrowser("local", newLocalBrowserProvider)
}

// pkg/web/cloudflare/registry.go
func init() {
    web.Registry.RegisterBrowser("cloudflare", newCloudflareBrowserProvider)
    web.Registry.RegisterAIBrowser("cloudflare", newCloudflareAIBrowserProvider)
}
```

**Supported provider names per role:**

| Role | Valid Names |
|---|---|
| `search` | `exa`, `tavily`, `searxng` |
| `browser` | `local`, `cloudflare`, `browserbase`, `browserless`, `steel` |
| `aibrowser` | `cloudflare`, `browserbase` |

A provider can register for multiple roles. For example, `local` registers
as a search provider (static fetch) and a browser provider (Rod); `cloudflare`
registers as both browser and aibrowser.

`Resolve` returns typed `error` values — `ErrProviderNotFound`,
`ErrProviderNotRegistered` — so callers can distinguish configuration errors
from runtime failures.

## Config Evolution

Current (`pkg/config/schema/types.cue`):
```cue
WebConfig: {
    provider?: string | *"exa"  // active provider: exa, tavily, searxng, ...
    exa?:      EXAConfig
}
```

Proposed:
```cue
WebConfig: {
    // Active provider(s) with role assignment.
    // Examples:
    //   provider: "exa"
    //   provider: "local"
    //   provider: "cloudflare"
    //   providers: {
    //     search:   "exa"
    //     browser:  "local"          // local browser
    //     aibrowser:"cloudflare"
    //   }
    provider?:  string | *"exa"
    providers?: WebProviderRoles

    local?:       LocalConfig
    exa?:         EXAConfig
    cloudflare?:  CloudflareConfig
    browserbase?: BrowserbaseConfig
    browserless?: BrowserlessConfig
}

WebProviderRoles: {
    search?:    string // which provider handles SearchProvider
    browser?:   string // which provider handles BrowserProvider
    aibrowser?: string // which provider handles AIBrowser
}

LocalConfig: {
    // Chromium path: empty = auto-detect from system installs.
    // Set explicitly to use a specific Chromium/Chrome installation.
    chromium_path?: string | *""

    // Opt-in: allow Rod to auto-download Chromium (~170MB) on first use.
    chromium_auto_download?: bool | *false

    // Disable browser automation entirely (static fetch + HTML→MD only)
    no_browser?:    bool   | *false

    // Maximum bytes for static HTTP fetch (default: 10MB)
    html_max_size?: int    | *10485760
}

CloudflareConfig: {
    api_key:    string | *""     // from CLOUDFLARE_API_KEY env var
    account_id: string | *""     // from CLOUDFLARE_ACCOUNT_ID env var
}

BrowserbaseConfig: {
    api_key: string | *""        // from BROWSERBASE_API_KEY env var
    project_id: string | *""
}
```

The Go-side `WebConfig` struct mirrors this with one struct per provider:

```go
type WebConfig struct {
    Provider    string              `json:"provider"`
    Providers   *WebProviderRoles   `json:"providers,omitempty"`
    Local       LocalConfig         `json:"local,omitempty"`
    EXA         EXAConfig           `json:"exa,omitempty"`
    Cloudflare  CloudflareConfig    `json:"cloudflare,omitempty"`
    Browserbase BrowserbaseConfig   `json:"browserbase,omitempty"`
}

type WebProviderRoles struct {
    Search    string `json:"search"`
    Browser   string `json:"browser"`
    AIBrowser string `json:"aibrowser"`
}
```

API keys are loaded from environment variables (e.g., `CLOUDFLARE_API_KEY`,
`BROWSERBASE_API_KEY`) in the config loading path, matching the existing
pattern for `EXA_API_KEY`.

## CLI Command Mapping

The `gmd web` subcommands focus on four workflows:

| `gmd web` Subcommand | Interface Needed | Local | EXA | Cloudflare | Browserbase | Status |
|---|---|---|---|---|---|---|
| `gmd web search` | `SearchProvider` | no | yes | no | no | **existing** |
| `gmd web fetch` | `SearchProvider` + `BrowserProvider` | yes static/rod | yes cached | yes /content | yes | **existing** (EXA only) |
| `gmd web agent` | `SearchProvider` + LLM | no | yes | no | no | **existing** |
| `gmd web crawl` | `BrowserProvider` | yes rod | no | yes /crawl | yes | **new** |
| `gmd web research` | `SearchProvider` + LLM | no | yes | planned | planned | **new** |

- **Local** = execution on the user's machine (requires
  user-installed Chromium for browser ops).
- For `fetch`, local first tries static HTTP; if the response requires JS
  rendering and a browser is available, it falls back to Rod. Otherwise it
  returns the static content as-is.
- `--provider` flag overrides the configured default per-call.
- `--live` flag on `fetch` forces browser rendering even for static pages.

## Implementation Phases

### Phase W7: Interface Refinement (this sprint)

- [ ] Split `Provider` into `SearchProvider` / `BrowserProvider` / `AIBrowser`
- [ ] Define `BrowserCapabilities` struct for runtime introspection
- [ ] Define `CrawlOptions`, `SessionOptions`, `Element`, `Page` types
- [ ] Implement provider registry (`pkg/web/registry.go`)
- [ ] Update existing `exa` package to implement `SearchProvider` (with adapter)
- [ ] Refactor `pkg/web/agent.go` to use `SearchProvider` instead of `*exa.Client` directly
- [ ] Update CLI commands to use the new interfaces (not hardcoded `*exa.Client`)
- [ ] Add `Capabilities()` check before dispatching to a browser provider
- [ ] Update CUE schema with `WebProviderRoles`; add Go-side `WebProviderRoles` struct
- [ ] Add compile-time interface satisfaction checks per provider

### Phase W8: Local Provider (basic)

Local provider for static fetch and HTML→MD conversion. No browser dependency
required. Rod integration is explored but not the initial deliverable.

#### New Dependencies to Add to `go.mod`

```
github.com/JohannesKaufmann/html-to-markdown/v2 v2.5.1  // HTML→MD, pure Go
github.com/temoto/robotstxt                             // robots.txt parsing
```

Both meet the `CGO_ENABLED=0` constraint. `golang.org/x/net` is already an
indirect dep.

#### Package Structure

```
pkg/web/local/
├── client.go         # LocalProvider struct, constructor, Capabilities()
├── fetch.go          # Static HTTP fetch via net/http
├── markdown.go       # HTML→MD conversion via html-to-markdown/v2
├── crawl.go          # Crawling (robots.txt, rate limits, queue)
├── registry.go       # init() — registers "local" for search + browser roles
├── browser.go        # Chromium path detection + Rod wrapper (future)
└── client_test.go    # Tests (unit + integration-tagged)
```

#### Implementation Checklist

- [ ] `go get github.com/JohannesKaufmann/html-to-markdown/v2@v2.5.1`
- [ ] `go get github.com/temoto/robotstxt`
- [ ] Create `pkg/web/local/client.go` — `LocalProvider` struct, `NewLocalProvider()`, `Capabilities()`
- [ ] Create `pkg/web/local/fetch.go` — `FetchStatic(ctx, url)` via `net/http`, respects robots.txt, timeout, max size
- [ ] Create `pkg/web/local/markdown.go` — `HTMLToMarkdown(ctx, html)` using `html-to-markdown/v2` with tables + strikethrough plugins
- [ ] Create `pkg/web/local/crawl.go` — crawling with:
  - robots.txt parsing and enforcement (via `temoto/robotstxt`)
  - Per-domain rate limiting with configurable delay
  - Max depth, same-domain constraint
  - Cycle detection via URL canonicalization
  - Sitemap discovery for seed URLs
- [ ] Create `pkg/web/local/registry.go` — register "local" for search + browser roles
- [ ] Write tests: unit tests for fetch/markdown, integration-tagged tests for crawl

#### Rod Exploration (future phase, not W8 deliverable)

`go-rod/rod` is a pure-Go CDP client for local browser automation.
Open questions requiring evaluation:

- Does auto-downloading ~170MB of Chromium on first use fit a CLI tool?
- Can Chromium be detected from system installs across macOS/Linux/Windows?
- What is the per-page memory footprint? Can it be pooled?
- Is the version-pinning strategy acceptable over time?

If Rod is adopted, it is added as a `BrowserProvider` implementation in a
future phase (`pkg/web/local/rod.go`). Until then, local browser operations
require a cloud provider or user-installed Chromium + CDP connection.

#### HTML→Markdown Integration Pattern

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

### Phase W9: Cloudflare Browser Run Provider

- [ ] Create `pkg/web/cloudflare/client.go` — thin HTTP wrapper over Quick Actions REST API
- [ ] Create `pkg/web/cloudflare/registry.go` — register "cloudflare" for browser + aibrowser roles
- [ ] Implement `BrowserProvider` (FetchContent, Crawl, Scrape)
- [ ] Implement `SearchProvider.Fetch` via /content and /markdown endpoints
- [ ] Implement `AIBrowser.ExtractJSON` via /json endpoint
- [ ] Support CDP session creation for agent workflows
- [ ] Add `gmd web crawl` command
- [ ] Add `gmd web research` (search + LLM, EXA-backed initially)

### Phase W10: Browserbase Provider

- [ ] Create `pkg/web/browserbase/client.go` — wrapper over Browserbase API / Go SDK
- [ ] Create `pkg/web/browserbase/registry.go` — register "browserbase"
- [ ] Implement `BrowserProvider` using Browserbase's CDP/Playwright/Stagehand endpoints
- [ ] Use Stagehand for AI-native extract / act capabilities
- [ ] MCP server integration

### Phase W11: Additional Providers

Explore and implement based on user demand and feature coverage:

- **Browserless** (`pkg/web/browserless/`) — self-host option, Docker image available
- **Steel.dev** (`pkg/web/steel/`) — self-host option, credit-based pricing
- **Tavily** (`pkg/web/tavily/`) — additional search provider
- **SearXNG** (`pkg/web/searxng/`) — self-hosted search

### Phase W12: Research Agent

- [ ] `gmd web research` — deep research using search provider + LLM
- [ ] Sub-question generation, cross-referencing, citation tracking
- [ ] Works with any SearchProvider (EXA primary, others secondary)
- [ ] Can optionally use browser provider for live-fetch sources

## Provider Selection Logic

The user controls provider selection via two mechanisms:

1. **`--provider <name>` flag** — per-command override
2. **Config `providers` roles** — defaults for search, browser, aibrowser

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
                          CUE config (or default)
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

No automatic fallback. The user declares which provider handles each
role. If the configured provider is unavailable at runtime, the command errors
rather than switching to a different provider.

Supported provider names: `exa`, `tavily`, `searxng`, `cloudflare`, `browserbase`,
`browserless`, `steel`, `local`.

## Testing Strategy

### Unit Tests (`make test` — no external deps)

- **Cloud providers**: `httptest.Server` mocks replaying recorded API responses
  per endpoint. Each provider package ships response fixtures.
- **Registry**: Test registration, resolution, missing-provider errors, and
  duplicate-registration detection.
- **Config → provider mapping**: Verify that CUE config provider names route
  to correct constructors.
- **Local HTML→MD**: Convert known HTML fixtures to markdown, verify output.

### Integration Tests (`//go:build integration`, `make test.integration`)

- **EXA**: Live smoke test (search + fetch), skipped if `EXA_API_KEY` unset.
- **Local crawl**: Crawl a local HTTP test server serving controlled pages with
  robots.txt and rate-limit behavior.
- **Remote providers** (Cloudflare, Browserbase): Opt-in via env vars. Skipped
  if API keys are absent.

### Contract Tests (compile-time)

```go
var _ SearchProvider  = (*exa.SearchClient)(nil)
var _ BrowserProvider = (*local.LocalProvider)(nil)
var _ AIBrowser       = (*cloudflare.AIBrowserClient)(nil)
```

### Test Fixtures

```
pkg/web/testdata/           # shared HTML fixtures
pkg/web/exa/testdata/       # EXA API response recordings
pkg/web/local/testdata/     # local crawl test server pages
pkg/web/cloudflare/testdata/# Cloudflare API response recordings
```

## Design Decisions

1. **Multiple interfaces, not one.** Forcing all providers into one interface
   creates stub methods and abstraction leaks. Each command selects its
   interface at runtime.

2. **Capability introspection.** `BrowserCapabilities` lets commands fail early
   with actionable messages ("Provider X does not support crawl") rather than
   runtime errors.

3. **Provider roles in config.** A user configures `search: exa` and
   `browser: cloudflare` for different workflows. The `providers` object in CUE
   supports this; a single `provider` string keeps BC.

4. **User controls provider selection.** No automatic fallback. The user sets
   `providers.browser` / `providers.search` / `providers.aibrowser` in config or
   passes `--provider` per command. Missing or unavailable providers produce errors.

5. **No SDK deps.** Browser providers are wrapped with `net/http`.
   Cloudflare, Browserbase, and Browserless all have REST APIs or CDP WebSocket
   endpoints. Browserbase also offers a Go SDK.

6. **EXA is the default search provider.** EXA's neural index supports semantic
   search and content discovery. Browser providers do not index the web — they
   render pages on demand. They complement, not replace, EXA.

7. **Cloudflare is the first cloud browser provider to be implemented.**
   $0.09/hr browser time, 10 hrs/mo free tier, Quick Actions REST API + CDP +
   Playwright/Puppeteer/Stagehand support, and existing Cloudflare docs on disk.

8. **CDP is the common connector.** Every browser provider exposes CDP
   WebSocket endpoints. The `BrowserSession.CDPEndpoint()` return value is a
   `ws://` URL that any CDP client (Chrome DevTools, Playwright, Puppeteer,
   browser-use, MCP) can connect to.

9. **Local provider is user-installed, not auto-managed.** The tool does not
   download or manage Chromium automatically. Users install Chromium and
   configure the path. Documentation covers setup. Rod's auto-download
   capability is explored as an opt-in future feature.

10. **Robots.txt and rate limiting.** Local crawling enforces robots.txt,
    per-domain rate limits, and configurable delays between requests. Cloud
    providers handle this on their end; the interface is provider-agnostic.

11. **Provider registry with init-time registration.** Each provider package
    registers itself via `init()`, keeping the name→constructor mapping
    alongside the implementation. Central registry validates names and resolves
    constructors by role.

## Open Questions

- **Browser provider vs search provider for `gmd web fetch`?**
  Depends on the `--live` flag — EXA for cached content, browser for freshly
  rendered. Without `--live`, use the configured search provider.

- **How does cost feedback work for per-minute providers?**
  Implement `printBrowserCost()` analogous to `printCost()` for EXA. Providers
  report cost in different units (minutes vs credits vs queries).

- **What about providers that support both categories?**
  (e.g., Scrapfly has search-like crawling AND browser rendering)
  They implement multiple interfaces. The registry supports multiple roles per name.

- **Is Rod suitable for a CLI tool?**
  Concerns: ~170MB download on first use, version-pinned Chromium drifting over
  time, per-page memory footprint. Evaluate with a prototype. Alternatives:
  `chromedp` or manual Chromium + CDP.

- **What local options exist that don't require a browser?**
  `net/http` + HTML→MD covers static sites. For hybrid pages (partial JS),
  evaluate lightweight DOM hydration without a full browser. The spectrum
  from static → SPA determines when a browser is required.

- **Are there Go libraries for respectful crawling?**
  `temoto/robotstxt` handles robots.txt parsing. No single Go library covers
  per-domain rate limiting + queue management + sitemap discovery.
  Plan to build the rate-limit/queue layer in `pkg/web/local/crawl.go`.

- **What granularity for `BrowserCapabilities`?**
  Coarse granularity limits error specificity. Fine granularity increases config
  surface. Stable features are booleans; optional/provider-specific features use
  the `Features []string` set. Add new stable fields only when a capability is
  needed by command dispatch logic across multiple providers.

- **Does `gmd web research` need its own provider interface or is it composition**
  **of SearchProvider + BrowserProvider + LLM?**
  Composition — research is a workflow over SearchProvider + BrowserProvider +
  LLM. Some providers may offer research-specific endpoints in the future.

- **Should cloud provider configs include endpoint overrides?**
  For self-hosted (SearXNG, Browserless Docker) and proxies.
  In a follow-on config update, not the initial schema.

- **Error taxonomy.**
  Define error types: `ErrNotSupported`, `ErrAuthFailed`, `ErrRateLimited`,
  `ErrTimeout`, `ErrBrowserNotAvailable` so callers can distinguish retryable
  from fatal errors.

- **SSRF protection for local fetch.**
  Static HTTP fetch blocks private/loopback IP ranges to prevent SSRF.

- **Chromium in CI/Docker.**
  If Rod is adopted, detection in `browser_linux.go` handles the
  `chromedp/headless-shell` Docker image and system-installed Chrome. Rod's
  auto-download works in Docker with writable disk.
