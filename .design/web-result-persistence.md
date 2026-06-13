# Web Result Persistence

| Field | Value |
|---|---|
| Created | 2026-06-10 |
| Updated | 2026-06-10 |
| Phase | Design |
| Status | Reviewed |

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
and `web search` invocation, after the command completes its main work, serialize results into a
timestamped subdirectory under the persistence root. Each persisted result includes structured JSON
metadata alongside human-readable content files.

## Key Decisions

1. **On by default.** Persistence is opt-out (`--no-persist`), matching the user's "automatically
   (in the background)" intent. Config has `persistence.enabled: *true`.

2. **Separate from LocalConfig cache.** The existing `local.cache_*` fields are about HTTP-level
   caching for content freshness within a local browser provider. Persistence is different: it
   saves final processed results for all providers (EXA, Cloudflare, Tavily, etc.) and across sessions.

3. **Project-relative default.** Defaults to `.gmd/web/` resolved against `cfg.ProjectRoot` (set
   by `config.Load()`). If no project root is found, falls back to CWD. Uses `filepath.Join`
   for cross-platform path construction.

4. **Timestamped subdirectories.** Each invocation creates `fetch/<iso8601>-<slug>/`,
   `crawl/<iso8601>-<slug>/`, or `search/<iso8601>-<slug>/`. Avoids filename collisions and
   provides natural chronology.

5. **JSON + content dual format.** Each result saves a `result.json` (full structured data) plus
   extracted content files (`.md`). The JSON preserves everything for programmatic access; the
   content files enable direct reading and markdown tooling.

6. **No cross-command dedup or cache index.** Persistence is append-only with no shared index.
   An index/manifest (`.gmd/web/index.json`) could be added later if querying persisted results
   is desired.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ CLI commands (cmd/gmd/web_*.go)                             │
│                                                             │
│  1. Execute main operation (GetContent / Crawl / fusion.Run)│
│  2. Print results to stdout (existing behavior)             │
│  3. If persistence enabled:                                 │
│      call pkg/web/persist.Save*(...)                        │
│                                                             │
│  --no-persist flag skips step 3                             │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ pkg/web/persist/                                            │
│                                                             │
│  PersistFetchResult(dir, url, *GetContentResult) error      │
│  PersistCrawlResult(dir, url, []Page) error                  │
│  PersistSearchResult(dir, query, *fusion.Result) error      │
│                                                             │
│  Each function:                                             │
│    - Creates <dir>/<type>/<timestamp>-<slug>/               │
│    - Writes result.json                                     │
│    - Writes content*.md files                               │
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
.gmd/web/
  fetch/
    2026-06-10T15_30_00_123456789Z-example-com-article/
      result.json      # Full web.GetContentResult marshaled
      content.md       # markdown content (or .txt for text format)
    ...
  crawl/
    2026-06-10T15_35_00_987654321Z-docs-example-com/
      result.json      # []web.Page marshaled
      pages/
        001-index.md
        002-getting-started.md
        ...
    ...
  search/
    2026-06-10T15_40_00_456789123Z-search-query-text/
      result.json      # fusion.Result marshaled
      answer.md        # Synthesized answer, if present
      results/
        001-result-title.md   # Per-result content with metadata header
        002-another-title.md
        ...
    ...

```

## Configuration

### CUE schema additions (`pkg/config/embeds/types.cue`)

Add after `WebSearchConfig` (line 212) and before `WebConfig` (line 214):

```cue
// WebPersistenceConfig controls automatic persistence of web results to disk.
WebPersistenceConfig: {
  enabled:  bool   | *true
  dir:      string | *".gmd/web"
}
```

Then add the field to `WebConfig` (after `search` on line 223):

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

### Shared flag (`cmd/gmd/web.go`)

Add a persistent `--no-persist` flag on `webCmd`:

```go
var webNoPersist bool

// in init():
webCmd.PersistentFlags().BoolVar(&webNoPersist, "no-persist", false, "Skip persisting results to disk")
```

Add a persistent `--persist-dir` flag if per-invocation override is desired (lower priority):

```go
var webPersistDir string
webCmd.PersistentFlags().StringVar(&webPersistDir, "persist-dir", "", "Override persistence directory")
```

### Per-command persistence calls

**web_fetch.go** — Inside the `for _, urlStr := range args` loop, after each `GetContent` call:
```go
if !webNoPersist && cfg.Web.Persistence.Enabled {
    persistDir := resolvePersistDir(cmd, cfg)
    if err := persist.PersistFetchResult(persistDir, urlStr, result); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: persist failed: %v\n", err)
    }
}
```
NOTE: Each URL gets its own timestamped directory. For multi-URL fetch invocations
this creates N directories. Batching into a single directory could be added later.

**web_crawl.go** — After the `bp.Crawl()` call and before printing results:
```go
if !webNoPersist && cfg.Web.Persistence.Enabled {
    persistDir := resolvePersistDir(cmd, cfg)
    if err := persist.PersistCrawlResult(persistDir, args[0], pages); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: persist failed: %v\n", err)
    }
}
```

**web_search.go** — After `fusion.Run()` returns and before printing:
```go
if !webNoPersist && cfg.Web.Persistence.Enabled {
    persistDir := resolvePersistDir(cmd, cfg)
    if err := persist.PersistSearchResult(persistDir, args[0], result); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: persist failed: %v\n", err)
    }
}
```

**web_agent.go** — The existing `--save` and `--wiki` flags are about saving results into a wiki
(a separate feature from web persistence). Web persistence for agent commands is deferred to a
future phase; the agent uses a different data model (`AgentResult` / `AgentSource`) and is
currently hardcoded to EXA only.

### Helper function

```go
// resolvePersistDir returns the absolute persistence directory.
// Precedence: CLI --persist-dir > config web.persistence.dir > default ".gmd/web"
// Relative paths are resolved against cfg.ProjectRoot if available, else CWD.
func resolvePersistDir(cmd *cobra.Command, cfg *config.Config) string {
    dir := cfg.Web.Persistence.Dir
    if cmd.Flags().Changed("persist-dir") {
        dir = webPersistDir
    }
    if !filepath.IsAbs(dir) {
        if cfg.ProjectRoot != "" {
            dir = filepath.Join(cfg.ProjectRoot, dir)
        } else {
            dir = filepath.Join(getCWD(), dir)
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
  persist.go      # PersistFetchResult, PersistCrawlResult, PersistSearchResult
  persist_test.go # Unit tests
```

### Functions

```go
package persist

// PersistFetchResult saves content from a single URL fetch.
// Creates: <dir>/fetch/<timestamp>-<slug>/result.json and content.<ext>
func PersistFetchResult(dir string, url string, result *web.GetContentResult) error

// PersistCrawlResult saves all pages from a crawl.
// Creates: <dir>/crawl/<timestamp>-<slug>/result.json and pages/*.md
func PersistCrawlResult(dir string, startURL string, pages []web.Page) error

// PersistSearchResult saves fused search results.
// Creates: <dir>/search/<timestamp>-<slug>/result.json, answer.md, results/*.md
func PersistSearchResult(dir string, query string, result *fusion.Result) error
```

### Internal helpers

```go
// timestamp generates an ISO8601-like timestamp suitable for filenames.
// Uses nanoseconds to avoid collisions between concurrent gmd processes.
func timestamp() string // "2006-01-02T15_04_05_000000000Z" (colons/periods -> underscores)

// urlSlug extracts a short slug from a URL (host + path tail).
func urlSlug(rawURL string) string

// slugify trims, lowercases, replaces spaces with dashes, limits length.
func slugify(s string) string
```

### result.json schema

Each `result.json` is the JSON-marshaled struct from the provider/fusion layer, so the
exact fields vary by command:

- **fetch**: `web.GetContentResult` — `{"content": "...", "cost": {...}, "extra": {...}}`
- **crawl**: `[]web.Page` — array of pages, each with URL, title, content, depth, etc.
- **search**: `fusion.Result` — `{"answer": "...", "results": [...], "costs": [...]}`

No custom schema needed; reusing existing types ensures forward compatibility.

### content.md / pages/*.md schema

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
- **search/results/*.md**: Each search result as markdown with frontmatter (URL, title, score, provider).

## Edge Cases and Error Handling

| Case | Behavior |
|---|---|---|
| Persist directory unwritable | Print warning to stderr, do not fail the command |
| Empty content/results | Still persist metadata JSON, skip empty content files |
| Very long slugs (>100 chars) | Truncate to 100 chars |
| Special chars in URL slugs | Slugify: replace non-alphanumeric with dashes |
| Timestamp collision (same second) | Append `-2`, `-3`, etc. to directory name |
| Two gmd processes persist simultaneously | Timestamp+nanosecond or PID suffix avoids collisions; `os.MkdirAll` handles EEXIST gracefully |
| `--no-persist` + `--output file` | File output still works; persistence is skipped |
| Persistence disabled in config | Skip silently, no warning (user's explicit choice) |
| Project root not found | Resolve relative dir against CWD |

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

### Interaction with `gmd doctor`
The `gmd doctor` command currently reports on collections and wikis. Persisted web results
are not indexed and have no documents in Typesense, so they do not appear in doctor output.
If a future version indexes `.gmd/web/` as a collection/wiki, doctor would naturally pick it up.
No changes to doctor are needed for v1.

## Implementation Plan

### Phase 1: Core persistence package (no config changes)
1. Create `pkg/web/persist/persist.go` with three Save functions and helpers
2. Unit tests for slugify, urlSlug, directory creation, file writing
3. Integration test: create temp dir, persist each result type, verify file contents

### Phase 2: Config + schema
1. Add `WebPersistenceConfig` to `pkg/config/embeds/types.cue`
2. Add `persistence` field to `WebConfig` in the CUE schema
3. Add Go struct and JSON tag in `pkg/config/config.go`
4. Add `.gmd/web/` to the project's `.gitignore` during `gmd init`
5. Test: parse config with persistence block

### Phase 3: CLI integration
1. Add `--no-persist` persistent flag on `webCmd` in `cmd/gmd/web.go`
2. Add `--persist-dir` persistent flag on `webCmd`
3. Add `resolvePersistDir()` helper in `cmd/gmd/web.go`
4. Hook into `web_fetch.go`, `web_crawl.go`, `web_search.go` RunE functions

### Phase 4: Docs
1. Update `docs/web-providers.md` with persistence section
2. Update `AGENTS.md` if needed

## Tests

| Test | Type | Location |
|---|---|---|
| `TestSlugify` | Unit | `pkg/web/persist/persist_test.go` |
| `TestURLSlug` | Unit | `pkg/web/persist/persist_test.go` |
| `TestPersistFetchResult` | Unit | `pkg/web/persist/persist_test.go` |
| `TestPersistCrawlResult` | Unit | `pkg/web/persist/persist_test.go` |
| `TestPersistSearchResult` | Unit | `pkg/web/persist/persist_test.go` |
| `TestPersistEmptyContent` | Unit | `pkg/web/persist/persist_test.go` |
| `TestPersistTimestampCollision` | Unit | `pkg/web/persist/persist_test.go` |
| `TestPersistDirUnwritable` | Unit | `pkg/web/persist/persist_test.go` |
| `TestWebPersistenceConfig` | Unit | `pkg/config/config_test.go` |

## Open Questions

1. **Should `web agent` persistence be part of this effort?** The agent has `--save` and
   `--wiki` flags parsed but unimplemented. Those flags target wiki integration (saving agent
   output as wiki pages), which is a separate feature from web result persistence. Adding
   `PersistAgentResult()` behind the existing `--no-persist` gate is straightforward but the
   agent's output model (`AgentResult` with `Answer` + `[]AgentSource`) differs from the
   fetch/crawl/search models. Defer to a follow-up phase unless user requests it now.

2. **Should persisted results be searchable via `gmd query`?** This would require indexing
   `.gmd/web/` as a collection or wiki. Out of scope for this design but a natural next
   step — add a `.gmd/web/` wiki source reference to any wiki.

3. **Should `.gmd/web/` be added to `.gitignore`?** Yes. The `.gmd/` directory contains both
   config (tracked) and transient data (persisted results, which should not be tracked). Recommendation:
   add `.gmd/web/` to the project's `.gitignore` during `gmd init`. If the user prefers global
   persistence outside the project tree, they can set `persistence.dir` to an absolute path
   such as `~/.cache/gmd/web/`.

4. **Should fetch multi-URL invocations batch into one directory?** Single-URL-per-dir is
   simpler and avoids ambiguity. Could add a `--batch` flag later if needed.

5. **Retention/cleanup strategy?** No automatic cleanup in v1. Could add a `gmd web cleanup`
   command or `persistence.max_age` / `persistence.max_size` config later.
