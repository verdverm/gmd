# Web Providers — Multi-Provider Architecture for `gmd web`

**Status: Design**

GMD is expanding from a single EXA-backed web search tool into a multi-provider
system spanning search/discovery and browser automation. Providers do not share
a feature-aligned interface — they form a Venn diagram of overlapping but
distinct capabilities. This doc maps the landscape, defines the interfaces, and
lays out the incremental implementation plan.

## Rationale

| Why Expand | What It Enables |
|---|---|
| Search-only (`exa`) is one category | `gmd web search`, `gmd web fetch`, `gmd web agent` work today |
| Browser automation is a distinct category | Screenshots, PDFs, crawls, structured data extraction, form interaction |
| MCP ecosystem growth | Agents control browsers via CDP/MCP — GMD's MCP server should expose these |
| No single provider covers all use cases | Venn diagram — pick the right tool per workflow |

The key insight: **search providers** (EXA, Tavily, SearXNG) index and retrieve
existing content, while **browser providers** (Cloudflare Browser Run,
Browserbase, Browserless) render pages in real time via headless Chrome. They
are complementary, not competing.

## Provider Landscape

### Category 0: Local Execution (no cloud, no cost)

Local execution is the foundation of the provider architecture. Every browser
command defaults to local when no cloud provider is configured.

#### Go Browser Automation Libraries — Research

| Library | Stars | Approach | CGO? | Chromium Mgmt | API Style |
|---|---|---|---|---|---|
| `go-rod/rod` | 6.9k | Pure Go CDP client | No | Auto-downloads, version-pinned | High-level fluent, auto-wait |
| `chromedp/chromedp` | 13k | Pure Go CDP client | No | System browser, no auto-dl | DSL task lists, verbose |
| `playwright-community/playwright-go` | 3.3k | Node.js RPC bridge | No (but needs Node) | Via playwright npm | Playwright API (multi-browser) |

Key differences:

- **Rod** is pure Go, auto-downloads Chromium, version-pinned, prevents zombie
  processes, auto-wait for elements, thread-safe, 100% test coverage.
- **chromedp** is less actively maintained, uses a DSL-like task system, can
  leave zombie browser processes, requires users to install Chromium separately.
- **playwright-go** requires a Node.js runtime and downloads Playwright's Node.js
  bridge (~50MB) plus browser binaries. Supports Chromium + Firefox + WebKit.

#### HTML to Markdown Libraries — Research

| Library | Stars | Approach | CGO? | Speed | Features |
|---|---|---|---|---|---|
| `JohannesKaufmann/html-to-markdown/v2` | 3.6k | Pure Go, `x/net/html` parser | No | ~25 MB/s | Plugin system, CommonMark, tables, strikethrough |
| `thorstenpfister/semantic-markdown` | newer | Pure Go, content-aware | No | fast | Main content extraction, URL refification |
| `conductor-oss/markitdown` | newer | Pure Go, multi-format | No (WASM PDF) | moderate | PDF/DOCX/HTML all in one |
| goldmark (already vendored) | 12k | MD to HTML only | No | fast | CommonMark, GFM extensions |

Key differences:

- **html-to-markdown/v2** is pure Go, built on `golang.org/x/net/html` (already
  an indirect dep), plugin architecture, goroutine-safe, ~25 MB/s.
- **semantic-markdown** adds main content extraction and URL refification
  (LLM-optimized output).
- **markitdown** handles PDF, DOCX, PPTX plus HTML, but is heavier.
- **goldmark** is MD to HTML (wrong direction for this use case).

#### Local Execution Matrix

| Provider | Screenshot | PDF | Crawl | Scrape | Markdown | JS Exec | API Key | What's Needed |
|---|---|---|---|---|---|---|---|---|
| Rod (CDP) | yes | yes | yes | yes | yes via HTML to MD | yes | none | `go-rod/rod` + Chromium (auto-dl) |
| Static fetch + HTML to MD | no | no | no | no | yes static | no | none | `html-to-markdown/v2` + `net/http` |
| net/http only | no | no | no | no | raw HTML only | no | none | nothing (stdlib) |

Key insight: **Rod** covers the entire browser automation feature set at zero
marginal cost. The two-tier approach (Rod for browser, html-to-markdown for
static) gives a graceful degradation path:

1. Rod installed -- full browser capabilities
2. No Rod, but HTML available -- static HTML to MD
3. No Rod, no HTML fetchable -- clear error with install instructions

| Layer | Dependency | What It Enables |
|---|---|---|
| `pkg/web/local/rod.go` | `github.com/go-rod/rod` | Screenshot, PDF, crawl, scrape, markdown, JS eval |
| `pkg/web/local/html.go` | `github.com/JohannesKaufmann/html-to-markdown/v2` | Static HTML to markdown conversion |
| `pkg/web/local/fetch.go` | `net/http` (stdlib) | Static page fetch, no JS |

### Category 1: Search / Content Discovery

| Provider | Search | Fetch Content | Find Similar | Cost Model | API Key Needed |
|---|---|---|---|---|---|
| **EXA** | yes semantic + keyword | yes clean markdown | yes | Pay-per-query | `EXA_API_KEY` |
| **Tavily** | yes | yes extract | no | Pay-per-query | planned |
| **SearXNG** | yes self-host | no | no | Free (self-host) | none |

### Category 2: Browser Automation (cloud)

| Provider | Screenshot | PDF | Crawl | Scrape Elements | Markdown | Structured Data (AI) | CDP Session | Playwright | Puppeteer | Stagehand | MCP Server | Self-Host | Stealth/Anti-Bot | Session Recording | Live View |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| **Cloudflare Browser Run** | yes | yes | yes | yes | yes | yes /json | yes | yes fork | yes fork | yes | yes PW MCP | no | no docs say no | yes | yes |
| **Browserbase** | yes | yes | yes | yes | yes | yes Stagehand | yes | yes | yes | yes (owns) | yes 1-click | no | yes | yes | yes live viewer |
| **Browserless** | yes REST | yes REST | yes | yes | no | no | yes WebSocket | yes | yes | no | no | yes Docker | yes stealth flag | no | no |
| **Steel.dev** | yes | yes | yes | yes | yes | yes | yes | yes | yes | no | no | yes OSS | yes | no | yes |
| **Bright Data** | yes | yes | yes | yes | no | no | yes | yes | yes | no | no | no | yes (best) | no | no |
| **Scrapfly** | yes | yes | yes | yes | no | no | yes | yes | yes | no | no | no | yes | yes | yes manual |
| **Hyperbrowser** | yes | yes | yes | yes | no | yes | yes | yes | yes | no | no | no | yes | yes | no |

### Category 3: LLM-Centric Agent Frameworks (not raw browser APIs)

| Tool | Description | Relation |
|---|---|---|
| **Stagehand** | AI-native browser automation — `page.act("click submit")` over Playwright | Built by Browserbase, runs on any CDP browser |
| **Browser Use** | Open-source agent framework for LLM browser control | Uses CDP — can target any provider above |
| **Playwright MCP** | MCP server wrapping Playwright actions | Cloudflare ships one; runs on any Playwright-compatible provider |

### Pricing Snapshot

| Provider | Free Tier | Entry Paid | Billing Unit | Effective Hourly Rate |
|---|---|---|---|---|
| **Local Playwright** | unlimited (local) | $0 | per-use | **$0** (local machine) |
| **Local HTML→MD** | unlimited (local) | $0 | per-use | **$0** (local machine) |
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
│  │ Web index query    │  │ Screenshot, PDF              │   │
│  │ Content fetch      │  │ Crawl, scrape                │   │
│  │ Find similar       │  │ Form interaction             │   │
│  │                    │  │ JS execution                 │   │
│  │  ┌─────────────┐   │  │                              │   │
│  │  │ O V E R L A P│   │  │   ┌──────────────────┐      │   │
│  │  │ Markdown     │   │  │   │ O V E R L A P     │      │   │
│  │  │ Structured   │   │  │   │ Content fetch     │      │   │
│  │  │ data extract │   │  │   │ Crawl             │      │   │
│  │  │              │   │  │   │ Links extraction  │      │   │
│  │  └─────────────┘   │  │   └──────────────────┘      │   │
│  └────────────────────┘  └──────────────────────────────┘   │
│                                              │              │
│                                    ┌─────────▼────────┐     │
│                                    │ AI AGENT TOOLS    │     │
│                                    │ Stagehand         │     │
│                                    │ Browser Use       │     │
│                                    │ Playwright MCP    │     │
│                                    │ MCP Server        │     │
│                                    └──────────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

Key takeaways:
- **Search providers** do not do browser rendering. They index the web and retrieve pre-computed content.
- **Browser providers** do real-time rendering. Some overlap on fetch/crawl, but implementations differ.
- **Agent tools** (Stagehand, MCP) are abstractions on top of browser providers.
- No single provider covers all three circles.

## Interface Design

The existing `Provider` interface in `pkg/web/provider.go` is too narrow:

```go
type Provider interface {
    Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error)
    Fetch(ctx context.Context, urls []string) ([]SearchResult, error)
}
```

We need multiple interfaces reflecting the Venn diagram, not a single monolithic
interface that all providers must implement.

### LocalProvider (Category 0)

```go
type LocalProvider struct {
    // Embed BrowserProvider — backed by go-rod/rod when Chromium is available
    BrowserProvider

    // rodClient is the go-rod browser controller (nil if Chromium not found)
    rodClient *rodBrowser

    // htmlConverter converts raw HTML → Markdown (always available)
    htmlConverter *htmltomarkdown.Converter
}

// rodBrowser wraps go-rod/rod for CDP-based browser automation.
// Lazily initialized — first call downloads Chromium if needed.
type rodBrowser struct {
    browser *rod.Browser
}

// Static fetch + HTML→markdown (no browser needed)
func (p *LocalProvider) FetchStatic(ctx context.Context, url string) (*SearchResult, error)
func (p *LocalProvider) HTMLToMarkdown(ctx context.Context, html string) (string, error)
```

This provider wraps **`go-rod/rod`** for browser operations and
**`JohannesKaufmann/html-to-markdown/v2`** for HTML→markdown conversion.

| Runtime State | Rod Available? | Capabilities |
|---|---|---|
| Chromium installed or auto-downloaded | yes | Full browser: screenshot, PDF, crawl, scrape, JS eval |
| First run, no Chromium cached | yes (auto-dl) | First call downloads ~170MB Chromium, then full capability |
| `--no-browser` flag or `GMD_NO_BROWSER=1` | no (disabled) | Static fetch + HTML→MD only |
| Chromium not found and auto-dl fails | no | Fallback to static fetch; clear error with install guide |

The `LocalProvider` always exists as a fallback. Commands default to it when no
cloud provider is configured. Users get working `gmd web markdown`, `gmd web screenshot`,
etc. out of the box — Rod auto-downloads Chromium on first use.

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
    Screenshot(ctx context.Context, url string, opts *ScreenshotOptions) ([]byte, error)
    PDF(ctx context.Context, url string, opts *PDFOptions) ([]byte, error)
    FetchContent(ctx context.Context, url string) (string, error)    // rendered HTML/markdown
    Scrape(ctx context.Context, url string, selector string) ([]Element, error)
    Crawl(ctx context.Context, startURL string, opts *CrawlOptions) ([]Page, error)

    // Session-based control
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

    // Output
    Screenshot(ctx context.Context, opts *ScreenshotOptions) ([]byte, error)
    PDF(ctx context.Context, opts *PDFOptions) ([]byte, error)

    Close(ctx context.Context) error
}

type BrowserCapabilities struct {
    Screenshot    bool
    PDF           bool
    Crawl         bool
    Scrape        bool
    CDPEndpoint   bool
    Playwright    bool
    Puppeteer     bool
    Stagehand     bool
    MCPServer     bool
    SessionRecord bool
    LiveView      bool
    Stealth       bool
    SelfHost      bool
    // Local execution flags
    LocalPlaywright bool // Playwright browser available on this machine
    LocalHTML       bool // can do static HTML→markdown via goldmark
}
```

### AgentProvider (Category 3)

```go
type AgentProvider interface {
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

### Why Not One Interface?

If we crammed everything into one interface:

- Every provider stubs 80% of methods with `return nil, ErrNotSupported`
- Interface becomes a compatibility spreadsheet, not an abstraction
- The `gmd web` commands would need provider-specific knowledge anyway
- Providers' pricing models differ (per-query vs per-minute vs per-session)
- Implementation effort: every new provider needs to implement every method

Instead, commands declare which interface they need, and config selects both a
provider **and a capability mode** (e.g., `--provider cloudflare --action crawl`).

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
    //     search:  "exa"
    //     browser: "local"          // zero-cost local Playwright
    //     agent:   "cloudflare"
    //   }
    provider?:  string | *"exa"
    providers?: WebProviderRoles

    local?:      LocalConfig
    exa?:        EXAConfig
    cloudflare?: CloudflareConfig
    browserbase?: BrowserbaseConfig
    browserless?: BrowserlessConfig
}

WebProviderRoles: {
    search?:  string // which provider handles SearchProvider
    browser?: string // which provider handles BrowserProvider
    agent?:   string // which provider handles AgentProvider
}

LocalConfig: {
    // Chromium path: empty = auto-detect or auto-download via Rod
    // Set explicitly to use a specific Chromium/Chrome installation
    chromium_path?: string | *""

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

## CLI Command Mapping

| `gmd web` Subcommand | Interface Needed | Local | EXA | Cloudflare | Browserbase |
|---|---|---|---|---|---|---|
| `gmd web search` | `SearchProvider` | no | yes | no | no |
| `gmd web fetch` | `SearchProvider` | yes fetch HTML | yes | yes /content | yes |
| `gmd web agent` | `SearchProvider` + LLM | no | yes | planned browser+LLM | yes |
| `gmd web screenshot` | `BrowserProvider` | yes Playwright | no | yes | yes |
| `gmd web pdf` | `BrowserProvider` | yes Playwright | no | yes | yes |
| `gmd web crawl` | `BrowserProvider` | yes Playwright | no | yes /crawl | yes |
| `gmd web scrape` | `BrowserProvider` | yes Playwright | no | yes /scrape | yes |
| `gmd web markdown` | `BrowserProvider` | yes HTML→MD / Playwright | no | yes /markdown | yes |
| `gmd web session` | `BrowserProvider` | no | no | yes CDP | yes |
| `gmd web extract` | `AgentProvider` | no | no | yes /json | yes Stagehand |
| `gmd web research` | `SearchProvider` + LLM | no | planned planned | planned | planned |

Key: **Local** = zero-cost, zero-API-key execution on the user's machine.
Commands default to local when no cloud provider is configured, falling back
gracefully if Playwright isn't installed.

## Implementation Phases

### Phase W7: Local Provider (foundation)

The local provider comes first because it's zero-cost, zero-API-key, and makes
every browser command useful out of the box. It uses **`go-rod/rod`** for
browser automation (pure Go CDP, auto-downloads Chromium) and
**`JohannesKaufmann/html-to-markdown/v2`** for static HTML→MD.

#### New Dependencies to Add to `go.mod`

```
github.com/go-rod/rod v0.116.2              // Pure Go CDP client, no CGO
github.com/JohannesKaufmann/html-to-markdown/v2 v2.5.1  // HTML→MD, pure Go
```

Both meet the `CGO_ENABLED=0` constraint. Neither requires Node.js, external
binaries, or runtime services. `golang.org/x/net` is already an indirect dep.

#### Package Structure

```
pkg/web/local/
├── client.go          # LocalProvider struct, constructor, Capabilities()
├── rod.go             # Rod-based browser automation (screenshot, pdf, crawl, scrape, js)
├── fetch.go           # Static HTTP fetch via net/http
├── markdown.go        # HTML→MD conversion via html-to-markdown/v2
├── browser_linux.go   # Chromium path detection (Linux)
├── browser_darwin.go  # Chromium path detection (macOS)
├── browser_windows.go # Chromium path detection (Windows)
└── client_test.go     # Tests (unit + integration-tagged for live browser)
```

#### Key Decisions

1. **Rod for all browser ops.** One library for screenshot, PDF, scrape,
   network interception, JS evaluation. No separate Playwright or Puppeteer
   wrapper needed — Rod handles everything via CDP directly.

2. **Auto-download Chromium.** Rod's `launcher.NewBrowser()` downloads the
   matching Chromium binary on first use (~170MB). Subsequent launches use the
   cached binary. Users don't install anything manually.

3. **Graceful degradation.** If Rod can't find or download Chromium (air-gapped,
   no disk space), fall back to static `net/http` fetch. If HTML is returned,
   convert to Markdown. If the page requires JS (SPA, infinite scroll), error
   with clear instructions.

4. **Lazy browser init.** The Chromium process starts on the first browser
   operation, not at command startup. Commands like `gmd web markdown` on a
   static HTML page never spin up a browser.

5. **`--no-browser` / `GMD_NO_BROWSER=1`.** Skip Rod initialization entirely.
   Useful for CI, constrained environments, or users who only want static fetch.

6. **HTML→MD uses html-to-markdown/v2, not goldmark.** Goldmark is MD→HTML
   (the wrong direction). `html-to-markdown/v2` uses `golang.org/x/net/html`
   (already vendored) and produces CommonMark-compliant output.

#### Implementation Checklist

- [ ] `go get github.com/go-rod/rod@v0.116.2`
- [ ] `go get github.com/JohannesKaufmann/html-to-markdown/v2@v2.5.1`
- [ ] Create `pkg/web/local/client.go` — `LocalProvider` struct, `NewLocalProvider()`, `Capabilities()` with runtime detection
- [ ] Create `pkg/web/local/fetch.go` — `FetchStatic(ctx, url)` via `net/http`, respects robots.txt, timeout, max size
- [ ] Create `pkg/web/local/markdown.go` — `HTMLToMarkdown(ctx, html)` using `html-to-markdown/v2` with tables + strikethrough plugins
- [ ] Create `pkg/web/local/rod.go` — Rod wrapper implementing `BrowserProvider`:
  - `Screenshot(ctx, url, opts) ([]byte, error)` — full page or viewport
  - `PDF(ctx, url, opts) ([]byte, error)` — A4/letter, margin config
  - `FetchContent(ctx, url) (string, error)` — rendered HTML after JS execution
  - `Scrape(ctx, url, selector) ([]Element, error)` — CSS selector-based extraction
  - `Crawl(ctx, startURL, opts) ([]Page, error)` — follow links, configurable depth
  - `NewSession(ctx, opts) (BrowserSession, error)` — CDP WebSocket endpoint for interactive use
  - `Evaluate(ctx, js) (string, error)` — arbitrary JS execution in page context
- [ ] Create platform-specific Chromium detection files
- [ ] Write tests: unit tests for fetch/markdown, integration-tagged tests for Rod
- [ ] Wire `LocalProvider` into command auto-selection as the default when no cloud provider is configured

#### Rod API Integration Pattern

```go
package local

import (
    "github.com/go-rod/rod"
    "github.com/go-rod/rod/lib/launcher"
)

type rodBrowser struct {
    browser *rod.Browser
}

func newRodBrowser(ctx context.Context) (*rod.Browser, error) {
    // Auto-download Chromium if not present, then launch
    path, found := launcher.LookPath()
    if !found {
        // Download matching Chromium for this Rod version
        u := launcher.New().Bin("").MustGet()
        path = u
    }
    return rod.New().ControlURL(
        launcher.New().
            Headless(true).
            NoSandbox(true).
            Bin(path).
            MustLaunch(),
    ).Context(ctx).Connect()
}

func (b *rodBrowser) Screenshot(ctx context.Context, url string, opts *ScreenshotOptions) ([]byte, error) {
    page := b.browser.MustPage(url)
    defer page.MustClose()
    page.MustWaitStable()  // auto-wait for network idle
    return page.Screenshot(true, nil)
}

func (b *rodBrowser) PDF(ctx context.Context, url string, opts *PDFOptions) ([]byte, error) {
    page := b.browser.MustPage(url)
    defer page.MustClose()
    page.MustWaitStable()
    return page.PDF(&proto.PagePrintToPDF{
        PaperWidth:  8.27,  // A4
        PaperHeight: 11.69,
    })
}
```

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

### Phase W9: Interface Refinement (this sprint)

- [ ] Split `Provider` into `SearchProvider` / `BrowserProvider` / `AgentProvider`
- [ ] Define `BrowserCapabilities` struct for runtime introspection
- [ ] Update existing `exa` package to implement `SearchProvider`
- [ ] Update CLI commands to use the new interfaces (not hardcoded `*exa.Client`)
- [ ] Add `Capabilities()` check to `gmd web fetch` when using a browser provider
- [ ] Update CUE schema with `WebProviderRoles`

### Phase W10: Cloudflare Browser Run Provider

- [ ] Create `pkg/web/cloudflare/client.go` — thin HTTP wrapper over Quick Actions REST API
- [ ] Implement `BrowserProvider` (Screenshot, PDF, Crawl, Scrape, FetchContent)
- [ ] Implement `SearchProvider.Fetch` via /content and /markdown endpoints
- [ ] Implement `AgentProvider.ExtractJSON` via /json endpoint
- [ ] Support CDP session creation for direct browser control
- [ ] Add `gmd web screenshot`, `gmd web pdf`, `gmd web crawl`, `gmd web scrape` commands
- [ ] Add `gmd web session` for CDP-based interactive sessions

### Phase W11: Browserbase Provider

- [ ] Create `pkg/web/browserbase/client.go` — wrapper over Browserbase API
- [ ] Implement `BrowserProvider` using Browserbase's CDP/Playwright/Stagehand endpoints
- [ ] Leverage Stagehand for AI-native extract / act capabilities
- [ ] MCP server integration

### Phase W12: Browserless Provider (self-host)

- [ ] Create `pkg/web/browserless/client.go` — wrapper over Browserless API
- [ ] Implement `BrowserProvider` focusing on CDP/WebSocket connection
- [ ] Optional: self-hosted Docker management subcommands

### Phase W13: Research Agent

- [ ] `gmd web research` — deep research using search provider + LLM
- [ ] Sub-question generation, cross-referencing, citation tracking
- [ ] Works with any SearchProvider (EXA primary, others secondary)
- [ ] Can optionally use browser provider for live-fetch sources

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
          Use specified   │
          cloud provider  │
                          │
                          ▼
              ┌──────────────────────┐
              │ Command needs:       │
              │  browser?            │
              └────┬──────┬──────────┘
                   │      │
                  YES     NO
                   │      │
                   ▼      ▼
         ┌────────────┐   Use configured
         │ Playwright │   search provider
         │ installed? │   (default: exa)
         └────┬───┬───┘
              │   │
             YES  NO
              │   │
              ▼   ▼
         Local    ┌──────────────────┐
         (free)   │ cloud provider   │
                  │ configured?      │
                  └────┬──────┬──────┘
                       │      │
                      YES     NO
                       │      │
                       ▼      ▼
                 Use cloud   Error:
                 provider    "Install Playwright
                             or configure a cloud
                             provider"
```

The `--provider` flag overrides auto-selection. When set to `"local"`, it forces
local execution and errors if Playwright isn't installed.

## Design Decisions

1. **Multiple interfaces, not one.** The Venn diagram is real — forcing all providers
   into one interface creates stub methods and abstraction leaks. Each command
   selects the right interface at runtime.

2. **Capability introspection.** `BrowserCapabilities` lets commands fail early
   with clear messages ("Provider X does not support screenshots") rather than
   runtime errors.

3. **Provider roles in config.** A user might want `search: exa` and
   `browser: cloudflare` for different workflows. The `providers` object in CUE
   supports this; a single `provider` string keeps BC.

4. **No SDK deps.** Like EXA, browser providers are wrapped with `net/http`.
   Cloudflare, Browserbase, and Browserless all have REST APIs or CDP WebSocket
   endpoints that don't require vendored SDKs.

5. **EXA stays primary for search.** EXA's neural index is uniquely suited for
   semantic search and content discovery. Browser providers do not index the web
   — they render pages on demand. They complement, not replace, EXA.

6. **Cloudflare is first browser provider.** Lowest effective hourly rate
   ($0.09/hr), largest free tier, broadest feature set (Quick Actions + CDP +
   Playwright/Puppeteer/Stagehand), and GMD already has Cloudflare docs on disk.

7. **Local is the default for browser commands.** `gmd web screenshot`, `gmd web pdf`,
   `gmd web scrape`, and `gmd web markdown` default to local Rod when available.
   Cloud providers are opt-in via `--provider` or config. This means:
   - Zero cost for basic usage
   - Works offline
   - No API key registration friction
   - Rod auto-downloads Chromium on first use — no manual setup
   - Clear fallback messages ("Install Chromium with `gmd web install` or
     use `--provider cloudflare` to use the cloud")

8. **Rod over chromedp or playwright-go.** Rod is pure Go, CGO-free, auto-downloads
   Chromium, prevents zombie processes, and has a clean high-level API. chromedp is
   less actively maintained and leaks processes. playwright-go requires Node.js.
   For a CLI tool targeting content extraction, Chrome-only via Rod is the right
   tradeoff.

8. **CDP is the universal connector.** Every browser provider exposes CDP
   WebSocket endpoints. The `BrowserSession.CDPEndpoint()` return value is a
   `ws://` URL that any CDP client (Chrome DevTools, Playwright, Puppeteer,
   browser-use, MCP) can connect to directly.

## Open Questions

- Should `gmd web fetch` use the search provider or the browser provider?
  **Answer:** Depends on the command's `--live` flag — EXA for cached content,
  browser for freshly rendered.
- How does cost feedback work for per-minute providers? Add a `printBrowserCost()`
  analogous to `printCost()` for EXA, but providers report cost differently
  (minutes vs credits vs queries).
- What about providers that support both categories (e.g., Scrapfly has search-like
  crawling AND browser rendering)? They implement multiple interfaces.
- Rod vs chromedp resolved: Rod chosen. Pure Go, no CGO, auto-downloads Chromium,
  zombie process prevention, thread-safe, actively maintained.
- html-to-markdown/v2 vs goldmark resolved: html-to-markdown/v2 converts HTML→MD
  (the direction we need). Goldmark is MD→HTML (used elsewhere in GMD).
  Both are pure Go, no CGO. `golang.org/x/net/html` already an indirect dep.
- Should local browser commands check for Rod at startup or lazily?
  **Answer:** Lazy — first browser operation triggers Chromium launch.
  `gmd web fetch` on static HTML never starts a browser.
- `gmd web install` subcommand to pre-download Chromium? Users can run
  `gmd web install` to download Chromium upfront (useful for CI, air-gapped
  prep, or avoiding first-use latency). Under the hood: `rod.launcher.NewBin()`.
- How to handle Chromium in CI/Docker? Detection in `browser_linux.go` can
  look for the `chromedp/headless-shell` Docker image or system-installed
  Chrome. Rod's auto-download works in Docker too.
