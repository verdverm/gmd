# Test Data Capture & Mock Strategy

**Created:** 2026-06-13
**Last updated:** 2026-06-13
**Phase:** Design
**Status:** Draft

## Context

The gmd test suite is split into:

- **Unit tests** (`make test`) — fast, no external deps, limited coverage for packages touching Typesense/LLMs.
- **Integration tests** (`make test.integration`) — slow (~2-3min), require Docker + LLM API keys, fragile.

The main pain points:

1. `pkg/ts/` — No unit tests for search, schema, CRUD. All 735 lines of tests are integration-only.
2. `pkg/llm/` — No unit tests at all. Embedding, chat, rerank all call real APIs.
3. `pkg/wiki/` — ~1700 lines of integration tests, mostly testing LLM agent + Typesense interaction.
4. `pkg/search/` — Pipeline tests use hand-crafted structs; never test with real-looking data.
5. `pkg/web/` — Provider unit tests use `httptest.Server` with hand-crafted responses, not real API shapes.

**Goal:** Capture real request/response data from integration tests into per-package `testdata/` directories, then replay in unit tests — enabling fast, deterministic, parallel tests covering the same codepaths.

## Design: Sequential Tape Recording at the HTTP Transport Layer

### Why sequential tapes, not content-hash matching

Content-hash matching fails for stateful sequences. Example:

```
1. UpsertChunks([{path: "crud.md", ...}])  → 200 OK
2. CountByPath("crud.md")                   → count=3
3. DeleteChunksByPath("crud.md")            → 200 OK
4. CountByPath("crud.md")                   → count=0
```

Steps 2 and 4 have identical requests but different responses. A sequential tape replays exchanges in order, handling stateful sequences naturally.

### Why HTTP transport, not interfaces

- `ts.Client` has ~80 methods. An interface large enough to cover real usage would be unwieldy.
- `llm.Client` wraps `openai-go` SDK types. Interfacing these would require wrapping every return type.
- HTTP-level recording captures exact wire bytes with zero interface changes — only additive fields on config structs.

### Why tape wrapping, not replacement

The `openai-go` SDK's `option.WithHTTPClient` **replaces** the HTTP client entirely — it does not wrap or merge transports. When service-account auth sets a GCP-authenticated transport, adding a second `WithHTTPClient` would overwrite it and break auth. The tape must therefore accept a parent `http.RoundTripper` to wrap, so the auth transport remains in the chain beneath the tape.

```go
// Correct: tape wraps the auth transport
parentTransport := authTransport  // GCP credentials
tapeTransport := tape.Transport(parentTransport)

// Wrong: replaces auth transport
opts = append(opts, option.WithHTTPClient(tapeHTTPClient))
```

The tape's `Transport(parent http.RoundTripper)` method returns a RoundTripper that records/replays exchanges, then delegates actual network calls to `parent`. For non-auth setups (apikey, none), `parent` is `http.DefaultTransport`.

- **NewReplayTape** returns error if the file doesn't exist — tests `t.Fatal` on it. No skipping, no hiding.

## Architecture

```
pkg/testutil/
  tape.go              # Tape type: records or replays sequential HTTP exchanges
  tape_test.go         # Self-tests (see Tape Self-Tests below)

pkg/ts/testdata/       # Typesense tapes (next to pkg/ts/*_test.go)
  001_chunk_crud.json
  002_text_search.json
  003_hybrid_search.json
  004_edge_cases.json

pkg/llm/testdata/      # LLM tapes (next to pkg/llm/*_test.go)
  001_embed.json
  002_chat_expand.json
  003_rerank.json

pkg/wiki/testdata/     # Wiki integration tapes
  001_ingest_flow.json
  002_query_flow.json

pkg/web/fusion/testdata/                         # Fusion tapes
  001_multisearch.json

pkg/web/providers/exa/testdata/                  # EXA tapes
  001_search.json
  002_browser.json

pkg/web/providers/tavily/testdata/               # Tavily tapes
  001_search.json

pkg/web/providers/searxng/testdata/              # SearXNG tapes
  001_search.json

pkg/web/providers/cloudflare/testdata/           # Cloudflare tapes
  001_search.json
  002_crawl.json
```

Each `testdata/` directory sits alongside the `_test.go` files that use it.

### Core Types

```go
// pkg/testutil/tape.go

type Exchange struct {
    Request struct {
        Method  string            `json:"method"`
        URL     string            `json:"url"`
        Headers map[string]string `json:"headers"`
        Body    string            `json:"body"`
    } `json:"request"`
    Response struct {
        StatusCode int               `json:"status_code"`
        Headers    map[string]string `json:"headers"`
        Body       string            `json:"body"`
    } `json:"response"`
}

type Mode int

const (
    ModeRecord Mode = iota  // capture exchanges from real API calls
    ModeReplay              // serve pre-recorded exchanges in FIFO order
)

// Tape is a sequential recorder/replayer of HTTP exchanges.
// Each tape file contains an ordered []Exchange array.
// In Record mode, exchanges are appended as real API calls happen.
// In Replay mode, the tape is drained in FIFO order.
type Tape struct {
    mu        sync.Mutex
    mode      Mode
    filePath  string
    upstream  *url.URL
    parent    http.RoundTripper  // wrapped transport (auth layer or http.DefaultTransport)
    exchanges []Exchange
    pos       int                // replay position
    recording bool               // gated by Start()/Stop()
}

// NewTape creates a tape backed by filePath.
// parent is the upstream transport to wrap (nil means http.DefaultTransport).
func NewTape(filePath string, upstreamURL string, parent http.RoundTripper, mode Mode) *Tape

// NewReplayTape loads a pre-recorded tape from filePath for replay.
func NewReplayTape(filePath string) (*Tape, error)

// Start begins capturing. In Record mode, subsequent calls through
// Transport() are appended. In Replay mode, position resets to 0.
func (t *Tape) Start()

// Stop ends capture. In Record mode, writes exchanges to filePath
// (creating testdata/ if needed). In Replay mode, no-op.
func (t *Tape) Stop() error

// Transport returns an http.RoundTripper that records/replays exchanges.
// Uses t.parent for upstream calls in Record mode.
func (t *Tape) Transport() http.RoundTripper
```

### Start/Stop Gates

`Tape.Start()` / `Tape.Stop()` bracket what gets recorded. TestMain schema setup happens outside the gate and is not captured:

```go
func TestMain(m *testing.M) {
    // schema creation here — not recorded
    m.Run()
}

func TestIntegrationChunkCRUD(t *testing.T) {
    tape := testutil.NewTape("testdata/001_chunk_crud.json", srv.URL(), nil, testutil.ModeRecord)
    tape.Start()
    defer func() {
        if err := tape.Stop(); err != nil {
            t.Fatal(err)
        }
    }()
    // TS calls in this function only are recorded
}
```

Note: `defer tape.Stop()` alone discards the error return. Use `defer func() { if err := tape.Stop(); err != nil { t.Fatal(err) } }()` to catch write failures, JSON marshal errors, or tape validation failures.

### Tape Self-Tests

`pkg/testutil/tape_test.go` must cover:

1. **Record-then-replay round-trip** — Record N exchanges via `httptest.Server`, replay them, verify identical content (method, URL, status, headers, body).
2. **Header stripping** — Verify `Authorization`, `x-api-key` (lowercase), `X-TYPESENSE-API-KEY` (mixed case), `Cookie`, and `Set-Cookie` are stripped from recorded exchanges. Test case-insensitive matching.
3. **Response header stripping** — Verify `Set-Cookie` and echoed auth headers are stripped from recorded response headers.
4. **Parent directory creation** — `Stop()` in Record mode creates missing `testdata/` directories.
5. **Tape exhaustion** — Replay beyond recorded length returns an error containing "tape exhausted" and the position number.
6. **Empty tape** — Replay with zero exchanges returns an error on the first call.
7. **Start/Stop gate** — A request made before `Start()` is not recorded. A request after `Stop()` is not recorded.
8. **Response body re-readability** — The recorded response body can be read once (matching `http.Response.Body` semantics — `io.NopCloser` behavior).
9. **Large response body round-trip** — A 1MB response is recorded and replayed correctly.
10. **Invalid tape JSON** — `NewReplayTape` returns an error (not a panic) when the file contains malformed JSON or a non-array root. Corrupted files from merge conflicts are caught at load time.
11. **File write failure** — If the tape file path is unwritable (e.g., read-only parent), `Stop()` returns an error.

### Security: Header Stripping

The RoundTripper strips these sensitive headers from **both request and response** during recording:

**Request headers:**
- `Authorization` — OpenAI-compatible APIs and some web providers
- `X-TYPESENSE-API-KEY` — Typesense API
- `X-Api-Key` — EXA
- `X-Auth-Key` — potential Cloudflare/SearXNG variants
- `Cookie` — some providers may use cookie auth

**Response headers:**
- `Set-Cookie` — echoed by some auth setups
- `X-TYPESENSE-API-KEY` — echoed by internal Typesense configurations
- `X-Api-Key`, `X-Auth-Key`, `Authorization` — mirrored in some proxy setups

Matching is case-insensitive — the Typesense SDK sends `x-typesense-api-key` (lowercase), EXA sends `x-api-key` (lowercase). The strip function normalizes header names via `strings.EqualFold`.

Rather than hardcoding a list, use a `stripHeaders` set passed at construction time, defaulting to `{"Authorization", "X-TYPESENSE-API-KEY", "X-Api-Key", "X-Auth-Key", "Cookie", "Set-Cookie"}`.

Playback tests use fake keys (`"test-key"`). Header values in committed tape files contain no secrets.

## Required Structural Changes

### `pkg/ts/client.go` — Config + New()

```go
type Config struct {
    Host       string
    APIKey     string
    HTTPClient *http.Client  // ADD: optional, for test recording
}

func New(cfg Config) *Client {
    opts := []typesense.ClientOption{
        typesense.WithServer(cfg.Host),
        typesense.WithAPIKey(cfg.APIKey),
    }
    if cfg.HTTPClient != nil {
        opts = append(opts, typesense.WithCustomHTTPClient(cfg.HTTPClient))
    }
    return &Client{client: typesense.NewClient(opts...), config: cfg}
}
```

### `pkg/llm/builder.go` — ProviderConfig + BuildClient()

```go
type ProviderConfig struct {
    Name       string
    BaseURL    string
    Auth       string
    AuthData   map[string]string
    HTTPClient *http.Client  // ADD: optional, for test recording
}
```

**Critical: `BuildClient()` must wrap, not replace.** The service-account path already calls `option.WithHTTPClient(authHTTPClient)`. Adding another `WithHTTPClient` would overwrite it. Instead, `ProviderConfig.HTTPClient` wraps the entire client *after* construction:

```go
func BuildClient(provider ProviderConfig) (*openai.Client, error) {
    // ... existing auth logic (lines 37-67) ...

    client := openai.NewClient(opts...)

    // If test recording is active, wrap the client's HTTP transport.
    // The tape transport wraps the auth transport (for service-account) or
    // http.DefaultTransport (for apikey/none).
    if provider.HTTPClient != nil {
        // Replace the internal http.Client with the tape-wrapped one
        // openai-go doesn't expose a way to swap the transport post-construction,
        // so we create a new client with the tape transport replacing the
        // auth transport beneath it, then re-apply auth headers if needed.
        //
        // Simpler approach: the HTTPClient field on ProviderConfig is ONLY
        // set in test mode. In test mode, Auth is "apikey" with a fake key,
        // not "service-account". So the service-account path is never
        // triggered during recording. No conflict.
        clientOpt := option.WithHTTPClient(provider.HTTPClient)
        client = openai.NewClient(opts..., clientOpt)
    }

    return &client, nil
}
```

The simpler approach is correct: `ProviderConfig.HTTPClient` is only ever non-nil in test code. Test code uses `Auth: "apikey"` with a fake key buffered by the tape. The service-account path (`gmd/llm/auth.BuildHTTPClient`) is never exercised in tests, so the conflict never arises. No wrapping complexity needed.

### `pkg/llm/client.go` — newOpenAIClient()

```go
func newOpenAIClient(baseURL, apiKey string, httpClient *http.Client) openai.Client {
    opts := []option.RequestOption{option.WithBaseURL(baseURL)}
    if apiKey != "" {
        opts = append(opts, option.WithAPIKey(apiKey))
    }
    if httpClient != nil {
        opts = append(opts, option.WithHTTPClient(httpClient))
    }
    return openai.NewClient(opts...)
}
```

The single call site at `client.go:231` passes `nil`. Must be updated to pass `nil` as the third `*http.Client` argument.

### `pkg/web/config.go` — ProviderConfig

```go
type ProviderConfig struct {
    Name       string
    Extra      map[string]any
    HTTPClient *http.Client  // ADD: optional, for test recording
}
```

This propagates to all web providers since every constructor accepts `web.ProviderConfig`.

### Web Provider Constructors

Every provider creates a private `*http.Client` internally. When `cfg.HTTPClient` is non-nil, it replaces the internally-created client:

| Package | Constructor | Change |
|---|---|---|
| `pkg/web/providers/tavily` | `NewSearchClient(cfg)` → line 36 | Use `cfg.HTTPClient` if non-nil |
| `pkg/web/providers/searxng` | `NewSearchClient(cfg)` → line 30 | Use `cfg.HTTPClient` if non-nil |
| `pkg/web/providers/cloudflare` | `NewBrowserClient(cfg)` → line 40 | Use `cfg.HTTPClient` if non-nil |
| `pkg/web/providers/exa` | `NewSearchAdapter(cfg)` → search.go:16 | Pass through to exaclient |
| `pkg/web/providers/exa` | `NewBrowserAdapter(cfg)` → browser.go:14 | Pass through to exaclient |

### EXA Two-Layer Architecture

EXA has a separate low-level client package (`pkg/web/exa/`) that the provider adapters delegate to. Both layers need changes:

**`pkg/web/exa/client.go`** — low-level client:
```go
func New(apiKey string) *Client { ... }
func NewWithServer(apiKey, baseURL string) *Client { ... }
```
Both create `http.Client{Timeout: defaultTimeout}` internally. Add `*http.Client` params:
```go
func New(apiKey string, httpClient *http.Client) *Client { ... }
func NewWithServer(apiKey, baseURL string, httpClient *http.Client) *Client { ... }
```
When `httpClient` is non-nil, use it; otherwise create the default `http.Client{Timeout: defaultTimeout}`.

**`pkg/web/providers/exa/search.go`** — adapter (line 22):
```go
func NewSearchAdapter(cfg web.ProviderConfig) (*SearchAdapter, error) {
    // existing apiKey/baseURL extraction
    return &SearchAdapter{
        client: exaclient.NewWithServer(apiKey, baseURL, cfg.HTTPClient),
        name:   cfg.Name,
    }, nil
}
```

Same pattern for `NewBrowserAdapter` at `browser.go:20`.

When `cfg.HTTPClient` is nil (production), all behavior is unchanged.

## Implementation Plan

### Phase 1: Tape Infrastructure (`pkg/testutil/tape.go`)

Single ~250-line file, no dependencies beyond stdlib.

**Record mode RoundTripper:**
1. Buffer request body (read bytes, restore with `io.NopCloser(bytes.NewReader(bodyBytes))`)
2. Forward to upstream via `t.parent.RoundTrip(req)`
3. Buffer response body
4. Strip sensitive headers (from configured set)
5. Append exchange to `t.exchanges`
6. Return response with buffered body

**Replay mode RoundTripper:**
1. Pop next exchange from `t.exchanges[t.pos]`, increment `t.pos`
2. Construct `*http.Response` from stored status, headers, body
3. If `t.pos >= len(t.exchanges)`, return error ("tape exhausted at position N")

**`Stop()` in Record mode:**
- Creates `testdata/` directory if missing via `os.MkdirAll(filepath.Dir(t.filePath), 0755)`
- Writes `t.exchanges` as JSON array to `t.filePath`

**`Stop()` in Replay mode:**
- No-op. (Unconsumed exchanges are fine — the test exercises a subset of the recorded flow.)

**Concurrency:** `sync.Mutex` protects the exchanges slice. Not designed for concurrent use — each test function creates its own `Tape`.

### Phase 2: Add HTTPClient Fields to Config Structs

- `pkg/ts/client.go` — `Config.HTTPClient` + `New()` wiring
- `pkg/llm/builder.go` — `ProviderConfig.HTTPClient` + `BuildClient()` wiring
- `pkg/llm/client.go` — `newOpenAIClient` optional `*http.Client` param
- All web providers — `HTTPClient` field on each provider's config struct + constructor wiring

### Phase 3: Wire Tapes into Integration Tests

Recording is on by default. The integration test code checks `GMD_NORECORD=1` to skip recording:

```go
func maybeNewTape(t *testing.T, filePath, upstreamURL string) *testutil.Tape {
    if os.Getenv("GMD_NORECORD") == "1" {
        return nil
    }
    return testutil.NewTape(filePath, upstreamURL, nil, testutil.ModeRecord)
}
```

Integration test functions call `tape.Start()` / `defer tape.Stop()` to bracket recording. When tape is nil (opt-out), the test runs as before with no recording.

```go
// pkg/ts/client_integration_test.go
func TestIntegrationChunkCRUD(t *testing.T) {
    tape := maybeNewTape(t, "testdata/001_chunk_crud.json", srv.URL())
    if tape != nil {
        tape.Start()
        defer tape.Stop()
    }
    // ... existing test logic ...
}
```

Target tapes per package (each in its own `testdata/` alongside test files):

| Package | Tapes |
|---|---|
| `pkg/ts/` | Chunk CRUD, text search, hybrid search, vector search, doc CRUD, empty results |
| `pkg/llm/` | Embed (single + batch), chat (expansion), rerank |
| `pkg/wiki/` | Ingest flow, query flow, lint |
| `pkg/web/fusion/` | MultiSearch fan-out + partial failure |
| `pkg/web/providers/exa/` | Search, browser |
| `pkg/web/providers/tavily/` | Search |
| `pkg/web/providers/searxng/` | Search |
| `pkg/web/providers/cloudflare/` | Search, crawl |

### Phase 4: Write Unit Tests Using Replay

```go
func TestHybridSearch(t *testing.T) {
    tape, err := testutil.NewReplayTape("testdata/003_hybrid_search.json")
    if err != nil {
        t.Fatal(err) // fail hard — testdata is committed, absence is a bug
    }
    tape.Start()
    defer tape.Stop()

    client := ts.New(ts.Config{
        Host:       "http://unused",
        APIKey:     "test-key",
        HTTPClient: &http.Client{Transport: tape.Transport()},
    })

    results, err := client.HybridSearch(t.Context(), ts.HybridSearchParams{
        Query:       "deploy",
        Collections: []string{"docs"},
        Limit:       10,
        GroupLimit:  3,
    })
    if err != nil {
        t.Fatal(err)
    }
    if len(results) == 0 {
        t.Error("expected non-empty results")
    }
    for _, r := range results {
        if r.Path == "" || r.Content == "" {
            t.Errorf("result missing required field: %+v", r)
        }
    }
}
```

**Assertion strategy:**
- Typesense responses: structural validation (non-empty fields, correct types, result counts).
- LLM chat/rerank: structural only — `len(resp.Choices) > 0`, content non-empty. Never assert on semantic content.
- LLM embeddings: assert correct dimension and finite float values. Skip per-value epsilon comparison.
- Web provider responses: structural validation against the real response shape.

**Tape exhausted error:** If replay makes more calls than recorded, `Transport()` returns `fmt.Errorf("tape exhausted at position %d", pos)`, failing the test.

### Phase 5: Makefile Integration

The existing `test` and `test.integration` targets stay unchanged. A new `ci` target chains them — no target shadowing, no breaking changes:

```makefile
# Existing rules — unchanged
test:
	go test ./... -v -count=1

test.integration: clean-ts
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -p 1 ./... -v -count=1 -tags=integration

# Opt out of recording during development (new)
test.integration.norecord:
	GMD_NORECORD=1 $(MAKE) test.integration

# CI: replay unit tests (make test) then re-record tapes (make test.integration)
ci: test test.integration
```

The `check` target (currently `tidy gofmt lint lint-all vulncheck test`) is unchanged. `ci` is the new target that validates tape freshness.

### Phase 6: Coexistence with Hand-Written Mocks

For packages using narrow interfaces (e.g., `pkg/indexer/` uses 6 TS methods), the existing `mockTSClient` pattern remains:

| Use case | Best approach |
|---|---|
| Full search pipeline (20+ TS + LLM methods) | Replay tape |
| Indexer cleanup logic (6 TS methods) | Hand-written mock struct |
| LLM agent workflows (wiki) | Replay tape |
| Web provider response parsing | Replay tape |
| Config, chunking, formatting | Pure functions, no mocking |

## Design Considerations

### Why per-package testdata

- No cross-package dependency from `pkg/ts/` to `pkg/testutil/` beyond `_test.go` imports.
- Each package owns its tapes — clear ownership, easy re-recording per-package.
- Go ignores `testdata/` for package compilation. `go test` runs with CWD set to the package directory, so `"testdata/001_chunk_crud.json"` resolves correctly.

### LLM non-determinism

Chat/rerank responses vary. Tapes record one concrete response. Assertions validate structure, not content. Provider response format changes are caught by re-recording on `make test.integration`.

### EXA retry loop

`pkg/web/exa/client.go:88-155` retries up to 3 times on 429s/5xxs. The retry loop still executes during replay. The tape feeds responses in FIFO order:

```
Recording:
  Attempt 0: HTTP 429 → tape records exchange[0]
  Attempt 1: HTTP 429 → tape records exchange[1]
  Attempt 2: HTTP 200 → tape records exchange[2]

Replay (429s only):
  Attempt 0: tape serves exchange[0] (429) → loop retries (always for 429s)
  Attempt 1: tape serves exchange[1] (429) → loop retries
  Attempt 2: tape serves exchange[2] (200) → success
```

This works for 429s because the loop unconditionally retries them (line 122-128). However, for 5xx errors the loop only retries on the first attempt (`attempt == 0`, line 131-140):

```
Recording with 5xx:
  Attempt 0: HTTP 502 → tape records exchange[0]
  Attempt 1: HTTP 502 → tape records exchange[1]
  Attempt 2: HTTP 200 → tape records exchange[2]

Replay (5xx):
  Attempt 0: tape serves exchange[0] (502) → loop retries (attempt == 0)
  Attempt 1: tape serves exchange[1] (502) → loop returns error (attempt != 0), test fails
```

The third entry (200) is never consumed. **Mitigation:** During recording, use `GMD_NORECORD` or manually record only against a stable API that doesn't return 5xx errors. For tests that need to exercise retry behavior, use `httptest.Server` directly (existing pattern in `exa/client_test.go`) rather than depending on real API jitter.

If the retry logic itself changes (maxRetries, backoff, which status codes trigger retry), previously recorded tapes break. This is an explicit coupling — tape tests validate the retry logic indirectly, so retry changes require re-recording.

### Out-of-order requests in replay

The tape is strictly sequential. If a test refactoring changes the call order, the replay will serve wrong responses or exhaust prematurely. This is an acceptable tradeoff: tape tests are coupled to the recorded test flow, and changes to that flow require re-recording (which happens on the next `make test.integration` run).

### Typesense pagination: multi-exchange per method call

Several `ts.Client` methods paginate internally and make multiple HTTP calls per single Go method call:

- `TextSearch` — paginates when results span >250 per page
- `FetchChunksByPath` — paginates for files with >250 chunks
- `SearchDistinctPaths` — paginates per 250 grouped hits
- `FetchDocs` → `searchDocsByPattern` — paginates across matching docs

Each paginated call produces N sequential HTTP exchanges. The tape records these in order, and replay serves them in FIFO order — each `RoundTrip` call consumes one exchange. A single `client.TextSearch(query)` that paginates over 3 pages consumes 3 tape entries. This is deliberate and expected. If the tape has fewer entries than the pagination requires, replay fails with "tape exhausted."

Integration tests should use small page sizes (e.g., `limit=5`) to keep tapes compact while still exercising pagination logic.

### Upstream errors during recording

If the real upstream API returns a 4xx/5xx during recording, the tape captures the error response with its status code and body. A replay test that asserts on success-structure (e.g., `len(resp.Choices) > 0`) will fail against a 400/500-shaped response. This is correct behavior — it signals that the recorded response was not a successful one. To avoid false failures:
- Record tapes against a known-good, non-rate-limited API state.
- For tests that exercise error handling, use `httptest.Server` with hand-crafted error responses instead.

### `NewReplayTape` JSON validation

`NewReplayTape` must validate more than file existence:
1. File exists and is readable.
2. File contains valid JSON (return parse error, don't panic).
3. JSON root is an array (not an object or scalar).

A tape file corrupted by a merge conflict would fail step 2, producing a descriptive error. The replay test's `t.Fatal` on the tape load error catches this before any test logic runs.

### Working directory and file paths

`go test` sets the working directory to the package under test. `testdata/001_chunk_crud.json` resolves to `pkg/ts/testdata/001_chunk_crud.json` when running `go test ./pkg/ts/`. `Tape.Stop()` writes to this path, creating `testdata/` via `os.MkdirAll` if needed.

### Host URL in replay mode

The design sets `Host: "http://unused"` in replay mode. The `typesense.NewClient()` call does not validate the URL or pre-connect — it simply stores it. So `"http://unused"` is safe. If a future SDK version validates URLs, `"http://127.0.0.1:1"` or `"http://localhost:9"` (unused port) can be used.

## Summary

6-phase plan to add HTTP-level sequential tape recording/replay to gmd's test suite. Per-package `testdata/` directories hold committed JSON tape files. Integration tests record by default (opt-out via `GMD_NORECORD`). Unit tests replay tapes for fast, deterministic, parallel tests covering the same codepaths as integration. `make ci` chains replay-then-record for CI validation; `check` validates replay only. Hand-written mocks coexist for narrow interfaces; tapes cover broad API surfaces (Typesense, LLM, web providers, wiki).

### Tape Freshness and CI

CI runs `make ci` which does `test` (replay) then `test.integration` (record). If integration passes but replay fails, the replay logic regressed — not the data. Tapes are always committed after CI passes.

**Constraint:** The `check` target (`check: tidy gofmt lint lint-all vulncheck test`) runs `test` (replay unit tests) but NOT `test.integration`. A developer running `make check` before pushing may have stale tapes from a prior recording session that pass replay but fail against the current code structure (e.g., if call order changed or new endpoints were added). The `check` target should ideally include a `tape-fresh` verification hook, or the CI pipeline must be the authoritative freshness gate — tapes validated via `make ci` before merge, not via `make check`.

## Operational Notes

### `pkg/web/providers/local/`

This package exists in the tree but has no HTTP client yet. When it gains one, the same `cfg.HTTPClient` injection pattern applies.

### Web Crawl Tape Size

Cloudflare's `Crawl` makes N sequential `GetContent` calls per depth level. A crawl of depth 2 with 20 pages produces ~20 exchange entries per depth. Tape files may exceed 1MB for complex crawls. Use a minimal depth (1) and a constrained seed URL for crawl tapes. The `002_crawl.json` tape should target a small, stable page set.

Link extraction is content-dependent: if the seed page's rendered markdown varies between recordings (dynamic content, timestamps, ads), extracted links differ → different subsequent `GetContent` calls → different tape shapes. Use a static, version-pinned seed URL for crawl recording tapes.

### Page Limit and Metadata Endpoints

For tapes targeting list/query endpoints (e.g. Typesense `ListDocs`, `CountByPath`), limit parameters in integration tests to keep tape files small. Metadata-only endpoints (schema info, collection stats) can be combined into a single tape.

### Tape File Merge Conflicts

Tapes are JSON arrays, not line-oriented. When two branches modify the same test and both re-record the tape, the merge conflict is two different JSON arrays — effectively unresolvable without re-recording. Mitigations:
- Name tapes per-test-function (e.g., `TestChunkCRUD.json`) rather than shared tapes, minimizing overlap.
- Pretty-print JSON (indented) so diffs are at least partially readable.
- In practice, developers resolve conflicts by accepting one branch's tape and re-running `make test.integration`.

### Tape Versioning

The `Exchange` struct has no version field. If fields are added (e.g., `Response.TrailerHeaders`), existing tape files must be migrated or re-recorded. For the initial implementation this is acceptable — old tapes are simply deleted and re-recorded. Future work: add an optional `"version": 1` field and a version-check on `NewReplayTape`.

## Alternatives Considered

| Approach | Rejected because |
|---|---|
| **Interface-based mocks** (`gomock`, `moq`) | `ts.Client` has ~80 methods. `llm.Client` wraps `openai-go` SDK types that would require wrapping every return type. Interface surface is too large. |
| **Content-hash matching** (VCR-style) | Fails for stateful sequences where the same request yields different responses (e.g., `CountByPath` before/after delete). |
| **Standalone proxy recorder** (mitmproxy, `go-vcr`) | Adds external dependency. Doesn't integrate with test code gating (`Start/Stop`). Requires TLS cert management for HTTPS endpoints. |
| **Contract-based testing** (OpenAPI specs) | Typesense and all LLM providers would need maintained OpenAPI specs. No spec exists for many providers. |
| **Generating fake responses** (hand-crafted structs) | Already used in `pkg/web/` unit tests. Works for structural validation but doesn't test client deserialization against real API shapes. Tapes complement this approach. |

## Risks and Mitigations

| Risk | Severity | Mitigation |
|---|---|---|
| EXA 5xx retries recorded in tape break replay | Low | Use stable API for recording; test retry logic via `httptest.Server` (existing pattern); the recording path avoids this by simply not recording during API instability |
| Response headers leak secrets | High | Strip `Set-Cookie` and echoed auth headers from responses, case-insensitive |
| Stale tapes pass `make check` | Medium | CI runs `make ci` (replay then record) as the authoritative freshness gate |
| Tape merge conflicts in git | Low | Per-test-function naming; pretty-printed JSON for readability |
| Tape size bloat (web crawl) | Low | Constrain depth and seed pages; size guard in recording |
| Retry logic changes break tapes | Medium | Document coupling; re-recording required; CI detects this automatically |
| Test call order changes break replay | Medium | Accepted tradeoff — tapes are deliberately coupled to their recording flow |
| JSON marshal failure during Stop() | Low | Error returned from Stop(); test code checks it (defer func pattern) |
| `pkg/web/providers/local/` unaddressed | Low | No HTTP client yet; tape pattern applies when one is added |

## Large Response Policy

If a response body exceeds 1MB during recording, the tape truncates and logs a warning. Response bodies that large are typically web crawl results or file downloads — not useful for structural assertion tests. The size guard prevents tape file bloat.
