# Web Result Persistence

| Field | Value |
|---|---|
| Created | 2026-06-10 |
| Updated | 2026-06-14 |
| Phase | Design |
| Status | Updated (review feedback v2 incorporated) |

## Context

`gmd web` commands (`fetch`, `crawl`, `search`) currently output results to stdout or JSON
and then discard them. The `web fetch --output file` flag writes to an explicit directory, but
`crawl` and `search` have no disk persistence. The `web agent --save` flag is parsed but unimplemented.

Users need results automatically persisted to disk so they can reference, search, and re-process
web content across sessions without re-running commands.

## Goal

Automatically persist all `gmd web fetch/crawl/search` results to disk by default. Results are
written to a configurable directory (default: `.gmd/web/`) after each command completes. Users
can opt out per-invocation with `--no-persist`.

## Summary

Add a `persistence` config block under `web` in the CUE schema. On every `web fetch`, `web crawl`,
`web search`, and `web agent` invocation, after the command completes its main work, serialize
results into a timestamped subdirectory under the persistence root. Each persisted result includes
structured JSON metadata alongside human-readable content files.

Search results save both the fused/merged output AND the raw per-provider results (before dedup).
Agent runs capture the full multi-step trail: intermediate search results per step, the final
synthesis, and all source content. A `--caller` flag on web commands lets external AI agents mark
their invocations, distinguishing them from human-triggered runs and internal gmd agents.

## Key Decisions

1. **On by default.** Persistence is opt-out (`--no-persist`), matching the user's "automatically
   (in the background)" intent. Config has `persistence.enabled: *true`.

2. **Separate from LocalConfig cache.** The existing `local.cache_*` fields are about HTTP-level
   caching for content freshness within a local browser provider. Persistence is different: it
   saves final processed results for all providers (EXA, Cloudflare, Tavily, etc.) and across sessions.

3. **Project-relative default with global fallback.** Defaults to `.gmd/web/` resolved against
   `cfg.ProjectRoot` (set by `config.Load()`). If no project root is found, falls back to
   `os.UserCacheDir()/gmd/web/` (e.g. `~/Library/Caches/gmd/web/` on macOS,
   `~/.cache/gmd/web/` on Linux). This mirrors the existing global config directory pattern
   and prevents scattering results across arbitrary CWDs. Uses `filepath.Join` for
   cross-platform path construction.

4. **Timestamped subdirectories.** Each invocation creates `fetch/<iso8601>-<slug>/`,
   `crawl/<iso8601>-<slug>/`, or `search/<iso8601>-<slug>/`. Avoids filename collisions and
   provides natural chronology.

5. **JSON + content dual format.** Each result saves a `result.json` (full structured data) plus
   extracted content files (`.md`). The JSON preserves everything for programmatic access; the
   content files enable direct reading and markdown tooling.

6. **No cross-command dedup or cache index.** Persistence is append-only with no shared index.
   An index/manifest (`.gmd/web/index.json`) could be added later if querying persisted results
   is desired.

7. **Raw per-provider results saved alongside fused output.** For `web search`, the fused/deduped
   `fusion.Result` is saved as the primary result, AND each provider's raw `[]SearchResult` (before
   any dedup or merge) is saved under `raw/<provider>.json`. This preserves the full unmodified
   provider response for audit, debugging, and downstream processing.

8. **Agent persistence included in v1.** `web agent` runs multi-step LLM-orchestrated searches.
   Each step's intermediate search results, the final `AgentResult`, and all source content are
   persisted. The `--save`/`--wiki` flags (already parsed) target wiki integration and are
   separate from web persistence; `--no-persist` controls web persistence for agent commands.

9. **Index-prefixed filenames with adequate digit counts.** Crawl pages use 6-digit zero-padded
   prefixes (`000001-`, `000002-`) to accommodate potentially hundreds of pages. Search result
   files use 4-digit prefixes (`0001-`, `0002-`) as search results per invocation are typically
   under 100.

10. **Original query preserved verbatim.** The exact query string passed by the user (or agent)
    is stored in `query.txt` inside each search/agent persistence directory. Fusion operations
    (query expansion, autoprompt) do not modify the persisted original.

11. **Caller tracking via `--caller` flag.** A persistent string flag on `webCmd` records who
    triggered the command. Default is `"human"`. External AI agents set `--caller=<agent-name>`
    to mark their invocations. Internal gmd components (e.g. `gmd wiki ingest`) set
    `--caller=gmd-wiki-ingest`. This value is written into `metadata.json` alongside each
    persisted result, enabling filtering and attribution when results are later queried.

12. **All invocation flags captured for full reproducibility.** metadata.json captures every
    CLI flag value for the command, plus the resolved LLM profile and provider group. This means
    any persisted result can be exactly reproduced by reading its metadata.json. Flags include
    not just search parameters but also output format (`--json`, `--output`), format options
    (`--format`), and all provider-specific settings (domains, date ranges, autoprompt, etc.).

13. **Single unified slugify function.** The existing `slugify()` in `cmd/gmd/web.go` (60-char
    limit) is moved to `pkg/web/slug.go` as a shared utility with a 100-char limit. The old
    60-char limit is increased to 100. The persist package imports this single implementation.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ CLI commands (cmd/gmd/web_*.go)                             │
│                                                             │
│  1. Execute main operation:                                 │
│     search  → fusion.Run() → raw per-provider results       │
│     fetch   → bp.GetContent()                               │
│     crawl   → bp.Crawl()                                    │
│     agent   → agent.Run() → intermediate steps              │
│  2. If persistence enabled:                                 │
│      call pkg/web/persist.Save*()                           │
│  3. Print results to stdout (existing behavior)             │
│                                                             │
│  --no-persist flag skips step 2                             │
│  --caller flag attached to metadata.json                    │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ pkg/web/persist/                                            │
│                                                             │
│  PersistFetchResult(dir, url, result, caller) error         │
│  PersistCrawlResult(dir, url, pages, caller) error          │
│  PersistSearchResult(dir, query, result, rawResults, caller)│
│  PersistAgentResult(dir, query, result, steps, caller)      │
│                                                             │
│  Each function:                                             │
│    - Creates <dir>/<type>/<timestamp>-<slug>/               │
│    - Writes result.json + metadata.json + query.txt         │
│    - Writes content*.md files                               │
│    - For search: writes raw/<provider>.json                 │
│    - For agent: writes steps/0001-search.json etc.          │
│    - Returns error or nil                                   │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ Config (pkg/config/)                                        │
│                                                             │
│  WebPersistenceConfig {                                     │
│    enabled: bool | *true                                    │
│    dir:     string | *".gmd/web"                            │
│  }                                                          │
│                                                             │
│  WebConfig.persistence: WebPersistenceConfig                │
└─────────────────────────────────────────────────────────────┘
```

## Directory Layout

```
.gmd/web/                            # within a project (cfg.ProjectRoot/.gmd/web/)
OR <UserCacheDir>/gmd/web/           # global fallback (no project root)
  fetch/
    2026-06-10T15_30_00_123456789Z-example-com-article/
      result.json       # Full web.GetContentResult marshaled
      metadata.json     # {timestamp, caller, url, provider, flags: {...}}
      content.md        # markdown content (or .txt for text format)
    ...
  crawl/
    2026-06-10T15_35_00_987654321Z-docs-example-com/
      result.json       # []web.Page marshaled
      metadata.json     # {timestamp, caller, start_url, depth, maxPages, ...}
      pages/
        000001-index.md
        000002-getting-started.md
        ...
    ...
  search/
    2026-06-10T15_40_00_456789123Z-search-query-text/
      result.json       # fusion.Result marshaled (deduped results)
      metadata.json     # {timestamp, caller, query, providers, dedup, synthesize, ...}
      query.txt         # original query string, unmodified (raw text file)
      answer.md         # Synthesized answer, if present
      results/
        0001-result-title.md    # Per-result content with metadata header
        0002-another-title.md
        ...
      raw/
        exa.json                # Raw []SearchResult from EXA (before dedup)
        tavily.json             # Raw []SearchResult from Tavily (before dedup)
        searxng.json            # Raw []SearchResult from SearXNG (before dedup)
        <provider>.json         # One file per search provider in the invocation
    ...
  agent/
    2026-06-10T16_00_00_789012345Z-agent-query-slug/
      result.json       # AgentResult marshaled {answer, sources}
      metadata.json     # {timestamp, caller, query, maxSteps, resultsPerStep, ...}
      query.txt         # original query string, unmodified
      answer.md         # Final synthesized answer
      steps/
        0001-search.json     # Provider search response JSON from step 1
        0002-search.json     # Provider search response from step 2
        ...
      sources/
        0001-0001-source-title.md  # Step 1, result 1
        0001-0002-another-title.md # Step 1, result 2
        0002-0001-source-title.md  # Step 2, result 1
        ...
    ...

```

## Configuration

### CUE schema additions (`pkg/config/embeds/types.cue`)

Add after the `WebSearchConfig` definition and before the `WebConfig` definition:

```cue
// WebPersistenceConfig controls automatic persistence of web results to disk.
// dir is relative to cfg.ProjectRoot when inside a project.
// When no project root is found, dir is ignored and UserCacheDir/gmd/web/ is used.
WebPersistenceConfig: {
  enabled:  bool   | *true
  dir:      string | *".gmd/web"
}
```

Then add the field to `WebConfig` (after the `search` field):

```cue
WebConfig: {
  // ... existing fields ...
  search:       WebSearchConfig
  persistence?: WebPersistenceConfig   // NEW
}
```

### Go struct additions (`pkg/config/config.go`)

```go
type WebPersistenceConfig struct {
  Enabled bool   `json:"enabled"`
  Dir     string `json:"dir"`
}

// Add to WebConfig:
type WebConfig struct {
  // ... existing fields ...
  Persistence WebPersistenceConfig `json:"persistence,omitempty"`
}
```

## CLI Integration

### Shared flags (`cmd/gmd/web.go`)

Add persistent flags on `webCmd`:

```go
var (
    webNoPersist  bool
    webPersistDir string
    webCaller     string
)

// in init():
webCmd.PersistentFlags().BoolVar(&webNoPersist, "no-persist", false,
    "Skip persisting results to disk")
webCmd.PersistentFlags().StringVar(&webPersistDir, "persist-dir", "",
    "Override persistence directory")
webCmd.PersistentFlags().StringVar(&webCaller, "caller", "human",
    "Caller identifier for attribution (e.g. 'my-agent', 'gmd-wiki-ingest')")
```

The `--caller` flag defaults to `"human"`. External AI agents calling `gmd web search`
should set `--caller=<agent-name>` so their results are distinguishable. Internal gmd
components (e.g. `gmd wiki ingest` performing web searches, `gmd web agent`) set
caller programmatically rather than via CLI.

### Per-command persistence calls

**web_fetch.go** — Inside the `for _, urlStr := range args` loop, after each `GetContent` call:
```go
if !webNoPersist && cfg.Web.Persistence.Enabled {
    persistDir := resolvePersistDir(cmd, cfg)
    if err := persist.PersistFetchResult(persistDir, urlStr, result, webCaller); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: persist failed: %v\n", err)
    }
}
```
NOTE: Each URL gets its own timestamped directory. For multi-URL fetch invocations
this creates N directories (each looks the same as a single `gmd web fetch <url>` call).

**web_crawl.go** — After the `bp.Crawl()` call and before printing results:
```go
if !webNoPersist && cfg.Web.Persistence.Enabled {
    persistDir := resolvePersistDir(cmd, cfg)
    if err := persist.PersistCrawlResult(persistDir, args[0], pages, webCaller); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: persist failed: %v\n", err)
    }
}
```

**web_search.go** — After `fusion.Run()` returns and before printing. The fusion.Result
will carry per-provider raw results and failure info (added in Phase 3):
```go
if !webNoPersist && cfg.Web.Persistence.Enabled {
    persistDir := resolvePersistDir(cmd, cfg)
    if err := persist.PersistSearchResult(persistDir, args[0], result, result.Raw, webCaller); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: persist failed: %v\n", err)
    }
}
```
`result.Raw` is a `map[string][]web.SearchResult` keyed by provider name, populated by
`fusion.Run()` before the dedup step. The `_provider` key is stripped from each result
before marshaling to `raw/<provider>.json`. Failed providers appear in the `Failures` map
on the result but are omitted from `Raw`.

**web_agent.go** — After `agent.Run()` returns and before printing. The AgentResult
will carry step data (added in Phase 3):
```go
if !webNoPersist && cfg.Web.Persistence.Enabled {
    persistDir := resolvePersistDir(cmd, cfg)
    if err := persist.PersistAgentResult(persistDir, args[0], result, result.Steps, "gmd-agent"); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: persist failed: %v\n", err)
    }
}
```
The caller is hardcoded to `"gmd-agent"` for internal agent runs (the user-set
`--caller` flag is ignored since this is gmd's own agent, not an external one). The existing
`--save`/`--wiki` flags target wiki integration (saving agent output as wiki pages) and
are separate from web persistence — they are not affected by this feature.
`result.Steps` is an `[]json.RawMessage` field on the returned `AgentResult`, populated
by `agent.Run()` during its multi-step loop. Each element is the full provider search
response JSON from one step.

### Helper function

```go
// resolvePersistDir returns the absolute persistence directory.
// When inside a project: resolves persistence.dir relative to cfg.ProjectRoot.
// When outside a project: uses os.UserCacheDir()/gmd/web/ as the global fallback.
// Precedence: CLI --persist-dir > config web.persistence.dir (only in projects).
// An absolute CLI --persist-dir bypasses all resolution logic.
func resolvePersistDir(cmd *cobra.Command, cfg *config.Config) string {
    dir := cfg.Web.Persistence.Dir
    if cmd.Flags().Changed("persist-dir") {
        dir = webPersistDir
    }
    if !filepath.IsAbs(dir) {
        if cfg.ProjectRoot != "" {
            dir = filepath.Join(cfg.ProjectRoot, dir)
        } else {
            cacheDir, err := os.UserCacheDir()
            if err != nil {
                cacheDir = filepath.Join(os.TempDir(), "gmd")
            }
            dir = filepath.Join(cacheDir, "gmd", "web")
        }
    }
    return dir
}
```

The `cmd.Flags().Changed()` pattern matches the existing convention used for `--dedup`,
`--synthesize`, and other flags in `web_search.go`.

## Package `pkg/web/persist/`

### Files

```
pkg/web/persist/
  persist.go        # PersistFetchResult, PersistCrawlResult, PersistSearchResult, PersistAgentResult
  persist_test.go   # Unit tests
```

Files are written with `0644` permissions, directories with `0755`, matching the
existing convention in `cmd/gmd/web_fetch.go` and the project's Go standard library
defaults.

### Functions

```go
package persist

// PersistFetchResult saves content from a single URL fetch.
// Creates: <dir>/fetch/<timestamp>-<slug>/result.json, metadata.json, content.<ext>
func PersistFetchResult(dir string, url string, result *web.GetContentResult, caller string) error

// PersistCrawlResult saves all pages from a crawl.
// Creates: <dir>/crawl/<timestamp>-<slug>/result.json, metadata.json, pages/*.md
func PersistCrawlResult(dir string, startURL string, pages []web.Page, caller string) error

// PersistSearchResult saves fused search results AND raw per-provider results.
// Creates: <dir>/search/<timestamp>-<slug>/result.json, metadata.json, query.txt,
//          answer.md, results/*.md, raw/<provider>.json
// rawResults is a map of provider name → raw []SearchResult (before dedup/fusion).
func PersistSearchResult(dir string, query string, result *fusion.Result, rawResults map[string][]web.SearchResult, caller string) error

// PersistAgentResult saves agent multi-step search results.
// Creates: <dir>/agent/<timestamp>-<slug>/result.json, metadata.json, query.txt,
//          answer.md, steps/*.json, sources/*.md
// steps contains the full provider search response for each agent step
// (e.g. exa.SearchResponse), serialized as JSON. The function accepts
// json.RawMessage to stay provider-agnostic.
func PersistAgentResult(dir string, query string, result *web.AgentResult, steps []json.RawMessage, caller string) error
```

### Internal helpers

```go
// timestamp generates an ISO8601-like timestamp suitable for filenames.
// Uses nanoseconds to avoid collisions between concurrent gmd processes.
func timestamp() string // "2006-01-02T15_04_05_000000000Z" (colons/periods -> underscores)

// urlSlug extracts a short slug from a URL (host + path tail).
func urlSlug(rawURL string) string

// slugify trims, lowercases, replaces spaces with dashes, limits length.
// Truncates to 100 chars. Implemented as a shared utility in pkg/web/slug.go
// (moved from cmd/gmd/web.go, limit increased from 60 to 100).
func slugify(s string) string
```

### result.json schema

Each `result.json` is the JSON-marshaled struct from the provider/fusion/agent layer:

- **fetch**: `web.GetContentResult` — `{"content": "...", "cost": {...}, "extra": {...}}`
  Note: `GetContentResult` does not include a URL field. The URL is stored in
  `metadata.json.url`. Code reading only `result.json` needs `metadata.json` for
  URL attribution.
- **crawl**: `[]web.Page` — array of pages, each with URL, title, content, depth, etc.
- **search**: `fusion.Result` — `{"answer": "...", "results": [...], "costs": [...]}`
- **agent**: `web.AgentResult` — `{"answer": "...", "sources": [...]}`

No custom schema needed; reusing existing types ensures forward compatibility.

### metadata.json schema

Each persistence directory includes a `metadata.json` capturing every aspect of the
invocation for full reproducibility. Fields shared by all commands: `timestamp`, `caller`,
`command`, `providerGroup`. The `flags` object contains every non-default CLI flag value.

```json
{
  "timestamp": "2026-06-10T15:30:00.123456789Z",
  "caller": "human",
  "command": "search",
  "providerGroup": "default",
  "query": "original query text",
  "providers": ["exa", "tavily"],
  "llmProfile": "default",
  "llmModel": "gpt-4o",
  "failures": {
    "searxng": "connection refused"
  },
  "flags": {
    "dedup": "heuristic",
    "synthesize": false,
    "limit": 10,
    "text": false,
    "type": "auto",
    "highlights": false,
    "maxChars": 5000,
    "noAutoprompt": false,
    "domains": [],
    "excludeDomains": [],
    "dateStart": "",
    "dateEnd": "",
    "additionalQueries": [],
    "systemPrompt": "",
    "noModeration": false,
    "json": false,
    "noSynthesize": false,
    "synthesisPrompt": ""
  }
}
```

The `flags` object captures every CLI flag for the command with its resolved value
(CLI override or config default). This means a persisted result can be exactly
reproduced by reading its `metadata.json` and passing the same flags.

Flag schemas by command:

- **fetch**: `{"format": "markdown", "maxChars": 5000, "highlights": false, "summary": "...", "maxAge": 0, "json": false, "output": "stdout", "outdir": "."}`
- **crawl**: `{"depth": 2, "maxPages": 20, "sameDomain": true, "include": "...", "exclude": "...", "json": false}`
- **search**: `{"dedup": "heuristic", "synthesize": false, "limit": 10, "text": false, "type": "auto", "highlights": false, "maxChars": 5000, "noAutoprompt": false, "domains": [], "excludeDomains": [], "dateStart": "", "dateEnd": "", "additionalQueries": [], "systemPrompt": "", "noModeration": false, "json": false, "noSynthesize": false, "synthesisPrompt": ""}`
- **agent**: `{"maxSteps": 3, "resultsPerStep": 5, "fetchText": false, "depth": "medium", "json": false, "output": "markdown"}`

The `providers` array lists all configured providers for the invocation. The `failures`
map (optional) records provider names that returned errors, with the error message as
the value. Providers that succeeded are omitted from `failures`.

`caller` is the value from the `--caller` flag (default `"human"`). This enables
filtering and attribution when persisted results are later indexed and queried.

`llmProfile` and `llmModel` record the resolved LLM configuration used for any
synthesis/fusion operations, captured from the runtime config. These are omitted
if the command did not use an LLM (e.g., `web crawl`).

The `command` field stores the canonical cobra command name (`"search"`, `"fetch"`,
`"crawl"`, `"agent"`), not any aliases. All metadata.json files share a common subset
of fields (`timestamp`, `caller`, `command`, `query` where applicable) plus
command-specific `flags`.

### raw/<provider>.json schema

Each file under `raw/` is a JSON-marshaled `[]web.SearchResult` from a single provider,
captured before dedup/merge. These preserve the exact provider response including any
provider-specific fields in `Extra`. The `_provider` key injected by `MultiSearch` for
internal tracking is stripped before writing — the filename already identifies the provider.

### query.txt

Plain text file containing the exact original query string, copied verbatim before
any fusion expansions or autoprompt modifications.

### content.md / pages/*.md / results/*.md schema

- **fetch/content.md**: Raw content from `GetContentResult.Content` with a frontmatter header:
  ```markdown
  ---
  url: https://example.com/article
  title: Example Article
  fetched: 2026-06-10T15:30:00Z
  provider: cloudflare
  ---
  ```
- **crawl/pages/*.md**: Each crawled page as a markdown file with similar frontmatter.
  Files are zero-padded 6-digit indexed: `000001-index.md`, `000002-page.md`, etc.
- **search/results/*.md**: Each fused/deduped search result as markdown with frontmatter
  (URL, title, score, provider). Files use 4-digit zero-padded prefix:
  `0001-result-title.md`, `0002-another-title.md`.
- **agent/sources/*.md**: Each agent source as markdown with frontmatter
  (URL, title, step, result). Files use a two-level zero-padded prefix:
  `<step>-<result>-slug.md` (e.g. `0001-0002-title.md` for step 1, result 2).
  This avoids collisions when multiple steps return results with the same title slug.

## Edge Cases and Error Handling

| Case | Behavior |
|---|---|---|
| Persist directory unwritable | Print warning to stderr, do not fail the command |
| Empty content/results | Still persist metadata JSON, skip empty content files |
| Nil result pointer (provider call failed) | Still persist metadata JSON with error info, skip content files |
| Very long slugs (>100 chars) | Truncate to 100 chars |
| Special chars in URL slugs | Slugify: replace non-alphanumeric with dashes |
| Two gmd processes persist simultaneously | Nanosecond-precision timestamps make collisions extremely unlikely in practice. In the rare event of a collision `os.Mkdir` would return EEXIST; `os.MkdirAll` silently overwrites, so the second writer's files take precedence. |
| `--no-persist` + `--output file` | File output still works; persistence is skipped. Both paths are independent: `--output file` writes to the user-specified `--outdir`, persistence writes to the persistence directory. Running both simultaneously would write content to two locations, which is allowed but noisy. |
| Persistence disabled in config | Skip silently, no warning (user's explicit choice) |
| Project root not found | Use `os.UserCacheDir()/gmd/web/` as global fallback; do not scatter across CWD |
| Agent runs with 0 search steps (immediate DONE) | Still persist metadata.json and empty steps/ directory |
| Agent step search returns 0 results | Persist empty `000N-search.json` with zero results |
| Raw provider result has `_provider` in Extra | Strip `_provider` key before marshaling to raw/<provider>.json (filename identifies provider) |
| Raw provider result is nil (provider failed) | Write `null` to raw/<provider>.json, note failure in metadata |
| Partial provider failure (some succeed, some fail) | Omit failed providers from `rawResults`; record failures in `metadata.json.failures` map |
| No `--caller` flag provided | Default to `"human"` |
| `--caller=""` explicitly passed | Treat as `"human"` (empty string is not a meaningful caller) |
| `--caller` contains special chars | Slugify caller name for directory safety, store raw value in metadata.json |
| Provider returns success with zero results | Write empty array `[]` to raw/<provider>.json; this is valid, not a failure |
| Agent step provider call fails (network error) | Record error in `metadata.json.failures["step-N"]`; omit `000N-search.json` for that step |

## Upstream API Changes Required

Persistence requires the fusion and agent packages to expose data that they currently
keep internal. These changes are planned in Phase 3 of implementation.

### fusion.Result additions (`pkg/web/fusion/fusion.go`)

Two new fields on `Result`:

```go
type Result struct {
    Answer   string                         // existing
    Results  []web.SearchResult             // existing
    Costs    []web.CostSummary              // existing
    Raw      map[string][]web.SearchResult  // NEW: per-provider raw results (before dedup)
    Failures map[string]string              // NEW: provider name → error string
}
```

`MultiSearch()` already collects per-provider errors internally (in a local `errs` slice
that is currently discarded). It will be modified to return both the flat result slice
AND a failures map. `Run()` aggregates both into the `Result` struct.

The `_provider` key injected by `MultiSearch` into `Extra` remains on the main `Results`
slice (needed for synthesis display). It is stripped from `Raw` entries before they are
marshaled to `raw/<provider>.json` by the persist layer.

### AgentResult additions (`pkg/web/agent.go`)

One new field on `AgentResult`:

```go
type AgentResult struct {
    Answer  string          // existing: synthesized final answer
    Sources []AgentSource   // existing: deduplicated sources
    Steps   []json.RawMessage // NEW: per-step provider search responses
}
```

`Agent.Run()` currently makes EXA search calls in a loop and accumulates results
internally. It will be modified to capture the raw search response from each step
(serialized as `json.RawMessage`) and populate the `Steps` field before returning.

## Design Trade-offs

### Append-only vs indexed
Append-only is simpler and sufficient for the initial implementation. An index file
(`.gmd/web/index.json`) mapping URLs/timestamps to directories could be added later if
there's demand for querying or deduplicating persisted results. Not included in v1.

### Per-invocation directory vs flat files
One directory per invocation is slightly more overhead but gives cleaner isolation, easier
cleanup (delete one dir), and natural grouping when looking at results manually. Flat files
would need UUIDs or compound names to avoid collisions.

### Reusing LocalConfig.cache_dir vs new config
`LocalConfig.cache_dir` is semantically for browser-level content caching within the local
provider. Persistence serves a different purpose (user-oriented result archival across all
providers). A dedicated `persistence.dir` is cleaner and avoids coupling to an unimplemented
provider.

### Content deduplication
Persisting the same URL twice (e.g., fetching the same URL on different days) will create
separate timestamped directories. This is intentional: each invocation is a separate snapshot.
Dedup-aware persistence (overwriting previous results for the same URL) could be added as a
`persistence.dedup` option later.

### Raw provider results vs fused-only
Saving both fused and raw results roughly doubles the storage per search invocation for the
raw JSON (the fused `results/*.md` files are rendered markdown, not duplicated from raw).
The benefit — full audit trail, per-provider debugging, and ability to re-process — outweighs
the storage cost. Raw results significantly aid debugging when provider responses differ in
quality or structure.

### Agent step persistence granularity
Each agent step saves the full provider search response as JSON (coarse-grained, entire
API response) rather than extracting individual result fields. This preserves everything
for future re-processing but makes the steps directory opaque without reading JSON. A
future iteration could also render step results as markdown files.

### metadata.json vs embedding in result.json
Storing invocation metadata in a separate `metadata.json` rather than embedding it in
`result.json` keeps the result file as a pure serialization of the domain type. This avoids
modifying the `fusion.Result` or `AgentResult` structs just for persistence concerns. The
metadata file is small and cheap to read alongside the result.

### Interaction with `gmd doctor`

There are two doctor commands:

- **`gmd doctor`** (top-level, `cmd/gmd/doctor.go`): Reports on config loading, project root
  detection, Typesense connectivity, all collections+wikis (per-source chunk counts and
  schema validation), and LLM endpoint reachability.
- **`gmd wiki doctor <name>`** (`cmd/gmd/wiki_doctor.go`): Wiki-specific diagnostics —
  config, filesystem structure, Typesense sync, agent compatibility. Supports `--fix` for
  auto-repair and auto-launches the agent harness.

Persisted web results (`.gmd/web/`) are not indexed in Typesense, so neither doctor command
currently reports on them. If a future phase indexes `.gmd/web/` as a collection or adds it
as a wiki source reference, both `gmd doctor` (which already iterates collections+wikis) and
`gmd wiki doctor` would naturally pick it up. No changes to doctor are needed for v1.

## Implementation Plan

### Phase 1: Core persistence package (no config changes)
1. Move `slugify()` from `cmd/gmd/web.go` to new `pkg/web/slug.go`, increase limit from 60 to 100 chars
2. Update all callers of `slugify()` in `cmd/gmd/web.go` to use the moved function
3. Create `pkg/web/persist/persist.go` with four Save functions and helpers
4. Unit tests for slugify, urlSlug, directory creation, file writing
5. Integration test: create temp dir, persist each result type, verify file contents

### Phase 2: Config + schema
1. Add `WebPersistenceConfig` to `pkg/config/embeds/types.cue`
2. Add `persistence` field to `WebConfig` in the CUE schema
3. Add Go struct and JSON tag in `pkg/config/config.go`
4. Test: parse config with persistence block

### Phase 3: Fusion + agent API changes
1. Modify `fusion.MultiSearch()` to return per-provider failure info alongside results
   (add a `[]ProviderError` or equivalent to the return signature, or add a
   `Failures map[string]string` field to `fusion.Result`)
2. Modify `web.Agent.Run()` to expose intermediate step data. Add a `Steps` field to
   `AgentResult` containing the raw search responses from each step, serialized as
   `[]json.RawMessage`. The agent internally captures provider responses during its
   multi-step loop and returns them.

### Phase 4: CLI integration
1. Add `--no-persist`, `--persist-dir`, and `--caller` persistent flags on `webCmd` in `cmd/gmd/web.go`
2. Add `resolvePersistDir()` helper in `cmd/gmd/web.go` (with global cache fallback)
3. Hook into `web_fetch.go`, `web_crawl.go`, `web_search.go`, `web_agent.go` RunE functions
   — persistence happens BEFORE printing results (safer: results are on disk before stdout)
4. In `web_search.go`: use the enriched `fusion.Result` (with per-provider raw results and
   failures from Phase 3) to feed `PersistSearchResult()`
5. In `web_agent.go`: use the enriched `AgentResult` (with `Steps` from Phase 3) to feed
   `PersistAgentResult()`
6. In `web_fetch.go`: handle nil `*GetContentResult` by persisting metadata with error info

### Phase 5: Docs
1. Update `docs/web-providers.md` with persistence section (if it exists)
2. Update `AGENTS.md` if needed

## Tests

The `pkg/web/persist/` package has no external API dependencies (pure filesystem I/O),
so it does not need the full three-layer test pattern (replay/integration tapes). It
follows the project's standard table-driven unit test convention as seen in
`pkg/web/errors_test.go`.

| Test | Type | Location |
|---|---|---|
| `TestSlugify` | Unit | `pkg/web/persist/persist_test.go` |
| `TestURLSlug` | Unit | `pkg/web/persist/persist_test.go` |
| `TestTimestamp` | Unit | `pkg/web/persist/persist_test.go` |
| `TestPersistFetchResult` | Unit | `pkg/web/persist/persist_test.go` |
| `TestPersistCrawlResult` | Unit | `pkg/web/persist/persist_test.go` |
| `TestPersistSearchResult` | Unit | `pkg/web/persist/persist_test.go` |
| `TestPersistAgentResult` | Unit | `pkg/web/persist/persist_test.go` |
| `TestPersistEmptyContent` | Unit | `pkg/web/persist/persist_test.go` |
| `TestPersistTimestampCollision` | Unit | `pkg/web/persist/persist_test.go` |
| `TestPersistDirUnwritable` | Unit | `pkg/web/persist/persist_test.go` |
| `TestPersistWithCaller` | Unit | `pkg/web/persist/persist_test.go` |
| `TestPersistRawProviderResults` | Unit | `pkg/web/persist/persist_test.go` |
| `TestWebPersistenceConfig` | Unit | `pkg/config/config_test.go` (new file) |

All persist tests use `t.TempDir()` for isolated filesystem test directories and
follow the table-driven sub-test pattern (`t.Run(name, func(t *testing.T) {...})`).
Each test validates both file existence/contents after the persist function returns
and verifies error conditions produce expected behavior (nil vs. non-nil errors).

## Open Questions

1. **Should `web agent` persistence be part of this effort?**
   **Resolved: Yes.** Agent persistence is included in v1. The agent performs multi-step
   searches that are valuable to capture. The existing `--save`/`--wiki` flags target wiki
   integration (a separate feature) and are unaffected. Agent runs use caller `"gmd-agent"`
   by default.

2. **Should persisted results be searchable via `gmd query`?**
   **Resolved: Yes, as a future phase.** This would require indexing `.gmd/web/` as a
   collection or wiki. Out of scope for v1 but a natural next step — add a `.gmd/web/`
   wiki source reference to any wiki, or register it as a special collection. The
   `metadata.json` files (with `caller`, `timestamp`, `query`) provide the data needed
   for filtering and attribution when indexed.

3. **Should `.gmd/web/` be added to `.gitignore`?**
   **Resolved: No.** Persistence is not about ignoring — it's about saving. The persistence
   directory is not a cache or temporary artifact; it's user data that should remain
   visible on the filesystem. Users who want to exclude it from version control can add
   it to their `.gitignore` themselves. `gmd init` will not touch `.gitignore`.

4. **Should fetch multi-URL invocations batch into one directory?**
   **Resolved: No.** Each URL gets its own timestamped directory, matching the behavior
   of invoking `gmd web fetch <url>` one-by-one from bash. This keeps results self-contained
   and avoids ambiguity about which file corresponds to which URL.

5. **Retention/cleanup strategy?**
   **Resolved: Beyond scope for v1.** No automatic cleanup. A `gmd web cleanup` command or
   `persistence.max_age`/`persistence.max_size` config could be added in a future phase.
   The immediate goal is reliable persistence; disk usage is not a concern for alpha.

6. **How should `--caller` interact with internal gmd agents?**
   Internal components (e.g. `gmd wiki ingest`, `gmd web agent`) set caller programmatically
   (e.g. `"gmd-agent"`, `"gmd-wiki-ingest"`) and should ignore/override any user-supplied
   `--caller` flag value. The CLI `--caller` flag is for external users and their agents.

7. **Should raw provider results also be saved as markdown files?**
   Not in v1. Raw results are saved as JSON only under `raw/<provider>.json`. Rendering
   them as markdown files (similar to `results/*.md`) would add significant storage for
   marginal benefit — users who want readable per-provider results can use the fused
   `results/*.md` files. Raw JSON is for programmatic access and debugging.
