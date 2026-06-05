# Web Browser Advanced — Sessions, AI Control, and Input Simulation

**Status: Proposal** — 2025-06-05

This doc covers the advanced browser automation surface: interactive sessions,
AI-driven browser control, human-like input simulation, and local headless
browser evaluation. It extends `.design/web-providers.md`, which focuses on the
first-pass fetch/crawl/search capabilities. Content moved here from that doc to
keep it focused on the deliverable scope.

## Scope Boundary

| Topic | Where |
|---|---|
| Search, fetch, crawl, provider registry, config, CLI mapping | `web-providers.md` |
| Browser sessions, CDP, AIBrowser, input simulation, Rod evaluation | This doc |

## Browser Attach Mode

Instead of launching a dedicated headless Chromium instance, Rod can attach to
an already-running Chrome/Chromium-based browser via its remote debugging port:

```
# Start Chrome with debugging enabled
chrome --remote-debugging-port=9222
```

This enables driving the user's **main browser** — logged-in sessions, saved
passwords, extensions, and cookies are all available. The browser process is
user-managed; Rod does not start or stop it.

Attach mode is activated by either:

- `--attach` CLI flag (launch mode is the default)
- `chromium_mode: "attach"` in `LocalConfig`
- Setting `chromium_remote_url: "http://localhost:9222"` to specify the debug endpoint

Attach mode is `LocalBrowser`-only; cloud providers cannot access the user's
local browser.

## Human-Like Input Simulation

CDP's `Input` domain supports raw `mouseMoved`, `mousePressed`, `keyDown`
events. Rod wraps these as `page.Mouse.Click/Drag/Scroll` and
`page.Keyboard.Press/Type/InsertText`.

For human-like movement (non-instant cursor jumps), the local provider can layer
smooth interpolation on top of Rod's mouse API:

- **Bezier curve paths** between points instead of straight lines
- **Variable velocity** — accelerate at start, decelerate near target
- **Micro-overshoot** — pass the target slightly, correct back (natural behavior)
- **Randomized timing** — realistic delays between keystrokes and clicks

This is a thin utility layer, not a library dependency. The Puppeteer ecosystem
has `ghost-cursor` as a reference implementation; the same technique ports
directly to Rod's `page.Mouse.Move()`.

## Recording Interactions

Chrome DevTools includes a built-in **Recorder** panel (DevTools → "Recorder"
tab) that captures clicks, typing, and scrolls in the user's actual browser
session. It exports recordings as:

- Puppeteer script (`.js`)
- Playwright script (`.js`)
- JSON (replayable by a custom runner)

Recordings use DOM selectors (`click button#submit`), not raw coordinates,
making them reliable across window sizes. A recorded workflow exported as JSON
could be replayed through Rod with human-like interpolation applied on top.

Playwright's `codegen` provides similar recording but launches its own browser
instance (no logged-in sessions), making Chrome DevTools Recorder the preferred
path for capturing main-browser workflows.

## Local Headless Browser: Rod Evaluation

`go-rod/rod` is a pure-Go CDP client for local browser automation.
Open questions requiring evaluation before adoption:

- Does auto-downloading ~170MB of Chromium on first use fit a CLI tool?
- Can Chromium be detected from system installs across macOS/Linux/Windows?
- What is the per-page memory footprint? Can it be pooled?
- Is the version-pinning strategy acceptable over time?
- **Attach mode off-ramp**: Rod can connect to an already-running Chrome via
  `--remote-debugging-port=9222`, avoiding the auto-download issue entirely
  for users who already have a browser. This is a lighter-weight entry point
  for local browser automation than launching a dedicated headless instance.

### Rod vs Alternatives

| Library | Stars | Approach | CGO? | Chromium Mgmt | API Style |
|---|---|---|---|---|---|
| `go-rod/rod` | 6.9k | Pure Go CDP client | No | Auto-downloads, version-pinned | High-level fluent, auto-wait |
| `chromedp/chromedp` | 13k | Pure Go CDP client | No | System browser, no auto-dl | DSL task lists, verbose |
| `playwright-community/playwright-go` | 3.3k | Node.js RPC bridge | No (but needs Node) | Via playwright npm | Playwright API (multi-browser) |

Key differences:

- **Rod** is pure Go, auto-downloads Chromium, version-pinned, prevents orphaned
  processes, auto-wait for elements, thread-safe, 100% test coverage.
- **chromedp** uses a DSL-like task system, can leave orphaned browser processes,
  requires users to install Chromium separately.
- **playwright-go** requires Node.js runtime (~50MB bridge) plus browser binaries.
  Supports Chromium + Firefox + WebKit.

If Rod is adopted, it is added as a `BrowserProvider` implementation via
`pkg/web/local/rod.go`. Until the evaluation is complete, local browser
operations requiring JS rendering need a cloud provider or user-installed
Chromium + CDP connection.

## BrowserSession Interface

Sessions are CLI-agent-scoped: they live for the duration of a command and are
torn down on exit. `gmd serve` does not manage browser sessions in this phase.

```go
type BrowserSession interface {
    CDPEndpoint() string

    Navigate(ctx context.Context, url string) error
    Click(ctx context.Context, selector string) error
    Fill(ctx context.Context, selector, value string) error
    Evaluate(ctx context.Context, js string) (string, error)

    Close(ctx context.Context) error
}
```

**SessionOptions:**

```go
type SessionOptions struct {
    Timeout  time.Duration // session idle timeout
    Stealth  bool          // use stealth / evasion techniques
    Proxy    string        // proxy URL for the session
    Record   bool          // record session for replay
    LiveView bool          // enable live view URL
}
```

## AIBrowser Interface (Category 3)

AI-native browser control tools (Stagehand, Browser Use, Playwright MCP).
Distinct from the `Agent` struct in `pkg/web/agent.go` (a research agent
composing Search + LLM, not a browser controller).

```go
type AIBrowser interface {
    Provider() BrowserProvider

    Navigate(ctx context.Context, url string) error
    ToMarkdown(ctx context.Context) (string, error)
    Screenshot(ctx context.Context) ([]byte, error)

    Observe(ctx context.Context) (*PageState, error)

    Act(ctx context.Context, instruction string) (*ActionResult, error)

    ExtractJSON(ctx context.Context, schema any) (any, error)

    Execute(ctx context.Context, goal string, opts *ExecuteOptions) (*ExecuteResult, error)

    NewSession(ctx context.Context, opts *SessionOptions) (AIBrowserSession, error)
}

type AIBrowserSession interface {
    AIBrowser
    ID() string
    LiveURL() string
    Close(ctx context.Context) error
}

type PageState struct {
    URL        string
    Title      string
    Markdown   string
    Elements   []Element
    Screenshot []byte // optional, nil if not requested
    ScrollY    int
}

type ActionResult struct {
    Success     bool
    Description string
    PageState   *PageState
}

type ExecuteOptions struct {
    MaxSteps           int
    OnStep             func(int, string, *PageState)
    Timeout            time.Duration
    ObserveScreenshot  bool
}

type ExecuteResult struct {
    Success    bool
    Goal       string
    Steps      []ExecutedStep
    FinalState *PageState
}

type ExecutedStep struct {
    Index       int
    Instruction string
    Result      *ActionResult
}
```

### AIBrowser Provider Landscape

| Tool | Description | Relation |
|---|---|---|
| **Stagehand** | AI-native browser automation — `page.act("click submit")` over Playwright | Built by Browserbase, runs on any CDP browser |
| **Browser Use** | Open-source agent framework for LLM browser control | Uses CDP — can target any provider |
| **Playwright MCP** | MCP server wrapping Playwright actions | Cloudflare ships one; runs on any Playwright-compatible provider |

These tools sit on top of browser providers (CDP / Playwright) and add
AI-driven page understanding and control. They map to the `AIBrowser` interface.

## Extended BrowserCapabilities

Beyond the core `BrowserCapabilities` fields in `web-providers.md`, the advanced
surface adds:

```go
// Fields added to BrowserCapabilities for advanced browser automation:
CDPEndpoint   bool // supports CDP WebSocket sessions
SessionRecord bool // supports session recording
LiveView      bool // supports live view
Stealth       bool // supports stealth / evasion
```

## Provider Coverage (Browser Automation)

| Provider | Crawl | Scrape | JS Render | CDP | Stealth | LiveView | Self-Host |
|---|---|---|---|---|---|---|---|
| **Local (Rod)** | yes | yes | yes | yes | no | yes (attach) | yes |
| **Cloudflare Browser Run** | yes | yes | yes | yes | no | no | no |
| **Browserbase** | yes | yes | yes | yes | yes | yes | no |
| **Browserless** | yes | yes | yes | yes | yes | no | yes (Docker) |
| **Steel.dev** | yes | yes | yes | yes | yes | no | yes (OSS) |
| **Bright Data** | yes | yes | yes | yes | yes | no | no |
| **Scrapfly** | yes | yes | yes | yes | yes | no | no |
| **Hyperbrowser** | yes | yes | yes | yes | yes | no | no |

## Credentials

| Provider | Account | Env Vars | Notes |
|---|---|---|---|
| **Browserbase** | [browserbase.com](https://browserbase.com) | `BROWSERBASE_API_KEY`, `BROWSERBASE_PROJECT_ID` | Free tier: 1000 min/mo |
| **Browserless** | [browserless.io](https://browserless.io) | `BROWSERLESS_API_KEY` | Free tier: 1000 units/mo; self-host via Docker |
| **Steel.dev** | [steel.dev](https://steel.dev) | `STEEL_API_KEY` | Free tier: 100 hrs/mo; self-host OSS |
| **Bright Data** | [brightdata.com](https://brightdata.com) | `BRIGHTDATA_API_KEY` | Pay-as-you-go |
| **Scrapfly** | [scrapfly.io](https://scrapfly.io) | `SCRAPFLY_API_KEY` | Free tier: 1000 credits/mo |
| **Hyperbrowser** | [hyperbrowser.ai](https://hyperbrowser.ai) | `HYPERBROWSER_API_KEY` | Pay-as-you-go |

## Config

```cue
BrowserbaseConfig: {
    api_key:    string | *""   // from BROWSERBASE_API_KEY env var
    project_id: string | *""
}
```

Go-side:

```go
type BrowserbaseConfig struct {
    APIKey    string `json:"api_key"`
    ProjectID string `json:"project_id"`
}
```

Register in the registry's `browser` role:

```go
"browserbase": func(cfg ProviderConfig) (any, error) { return browserbase.NewBrowserProvider(cfg) },
```

## Design Decisions (Advanced)

1. **CDP is the common connector.** Every browser provider exposes CDP
   WebSocket endpoints. The `BrowserSession.CDPEndpoint()` return value is a
   `ws://` URL that any CDP client (Chrome DevTools, Playwright, Puppeteer,
   browser-use, MCP) can connect to.

2. **AIBrowser is a separate interface,** not folded into BrowserProvider.
   AI-native tools (Stagehand, Browser Use) add LLM-driven reasoning on top of
   raw browser control. Treating them as a distinct interface keeps the
   BrowserProvider surface clean.

3. **Local browser is user-installed, not auto-managed.** The tool does not
   download or manage Chromium automatically. Users install Chromium and
   configure the path. Rod's auto-download capability is explored as an opt-in
   future feature.

## Open Questions

- **Is Rod suitable for a CLI tool?**
  Concerns: ~170MB download on first use, version-pinned Chromium drifting over
  time, per-page memory footprint. Evaluate with a prototype.

- **Chromium in CI/Docker.**
  If Rod is adopted, detection in `browser_linux.go` handles the
  `chromedp/headless-shell` Docker image and system-installed Chrome. Rod's
  auto-download works in Docker with writable disk.

- **How does cost feedback work for per-minute providers?**
  Providers that bill by time (Cloudflare, Browserbase) need a cost model
  distinct from per-query providers (EXA). A `CostSummary` struct with a `Unit`
  field (query vs minute vs credit) lets `web.go` display costs generically.

## Implementation Phases (Future)

Browser sessions, AIBrowser, and input simulation are not in scope for the
initial fetch/crawl implementation. They will be phased in after the core
provider architecture is stable:

| Phase | Scope |
|---|---|
| Phase | Scope |
|---|---|---|
| **Phase 6: Browserbase Provider** | `BrowserProvider` implementation, `pkg/web/browserbase/`, REST API client |
| **Phase 7: Local Browser** | Rod integration, headless Chromium launch, CDP endpoint |
| **Phase 8: Session Management** | BrowserSession interface, attach mode, `gmd web session` command |
| **Phase 9: AIBrowser** | Stagehand / Browser Use integration, Execute loop, Observe/Act |
| **Phase 10: Input Simulation** | Bezier mouse paths, human-like typing, click patterns |
| **Phase 11: Recording & Replay** | Chrome DevTools Recorder import, replay through Rod |
