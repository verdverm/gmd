# Lint & Security Audit Report — 2026-06-06 (final)

## Environment

- Go 1.26.4
- golang.org/x/net v0.55.0

## Final Tool Results

| Tool | Result |
|---|---|
| `go vet` | **PASS** |
| `govulncheck` | **PASS** — 0 vulnerabilities |
| `nilaway` | **PASS** — 0 nil deref issues |
| `golangci-lint` (non-gocyclo) | **PASS** — 0 issues in all categories |
| `golangci-lint` (gocyclo only) | 21 functions > 15 complexity (tech debt) |

---

## All Issues Resolved (80+ fixed)

### CRITICAL → 0
- **15 vulnerabilities** — resolved via Go 1.26.4 + golang.org/x/net v0.55.0

### HIGH → 0
- **14 nil pointer derefs** — nil guards added in LLM client (6 sites), HTTP clients (5 files), runtime+formatter+searxng+tavily receivers. Nilaway clean.
- **3 staticcheck** — nil context, empty branch, HasPrefix side-effect
- **9 dead code items** — removed boolField, indexFile, fileStatus/fileInfo, testMultiConfig, boolPtr, wikiName, all unused helpers
- **2 redundant error checks** — inlined to `return writeConfigFile(...)`

### MEDIUM → 0
- **11 WriteFile permission sites** — all changed from 0644 to 0600 (config files, wiki pages, test fixtures)
- **18 unchecked errors in production** — fixed: ast.Walk, filepath.Walk (3x), cmd.Help (5x), type assertion
- **10 test-file errcheck** — suppressed with `_ =` for json.Encode/Decode/Write in test handlers
- **4 gosec (testserver)** — nolint comments for docker commands + http.Get replaced with NewRequestWithContext
- **1 naming (IncludeHtmlTags)** — renamed to IncludeHTMLTags

### LOW → 0
- **34 usetesting** — all `context.Background()` → `t.Context()` via golangci-lint --fix
- **6 unparam** — removed unused parameters/returns: sourceMapField middle return, makeResult collection param, loadIndexContext ctx+error, searchOverlap error, checkWiki ctx
- **3 unnecessary conversions** — removed EmbeddingModel casts, int64() cast; nolint'd shared.ChatModel (false positive)
- **12 prealloc** — all slices pre-allocated with `make([]T, 0, N)`
- **3 revive stutter naming** — search.SearchMode→Mode, search.SearchParams→Params, wiki.WikiGraph→Graph
- **1 noctx** — http.Get replaced with http.NewRequestWithContext

### GOCYCLO — 21 remaining (tech debt)

Functions still above the 15 complexity threshold. These are structural refactoring tasks:

| Complexity | Function | File | Strategy |
|---|---|---|---|
| 28 | `TestParseFrontmatter` | `pkg/wiki/wiki_test.go` | Already uses t.Run; split into top-level test funcs |
| 26 | `Load` | `pkg/config/config.go` | Extract schema loading, wiki backfill, env-var blocks |
| 26 | `Search` | `pkg/web/providers/exa/search.go` | Extract request builder + result converter |
| 25 | `TestGenerateVariantsParsing` | `pkg/search/pipeline_test.go` | Already uses t.Run; split into top-level test funcs |
| 24 | `lintStructure` | `pkg/wiki/lint.go` | Extract graph walker + 3 analysis passes |
| 24 | `updateCollection` | `pkg/indexer/indexer.go` | Extract inner file-processing loop |
| 21 | `RemoveSourceRef` | `pkg/config/edit.go` | Share AST-traversal helper with other edit functions |
| 20 | `Run` | `pkg/web/agent.go` | Extract fetch + dedup phases |
| 20 | `TestScanFilesFS` | `pkg/indexer/indexer_test.go` | Already uses t.Run; split into top-level test funcs |
| 18 | `AddSourceRef` | `pkg/config/edit.go` | Same AST helper as RemoveSourceRef |
| 18 | `ValidateFrontmatter` | `pkg/wiki/frontmatter.go` | Split validation by field type |
| 17 | `Crawl` | `pkg/web/providers/cloudflare/client.go` | Extract URL processing loop |
| 17 | `FrontmatterToFilter` | `pkg/wiki/frontmatter.go` | Extract tag handling |
| 17 | `TestIgnorePatterns` | `pkg/config/edit_test.go` | Already uses t.Run; split into top-level |
| 17 | `TestValidateFrontmatter` | `pkg/wiki/wiki_test.go` | Already uses t.Run; split into top-level |
| 16 | `searchDocsByPattern` | `pkg/ts/client.go` | Break loop body into steps |
| 16 | `AddCollectionPatterns` | `pkg/config/edit.go` | Share AST helper |
| 16 | `AddIgnorePatterns` | `pkg/config/edit.go` | Share AST helper |
| 16 | `RemoveIgnorePattern` | `pkg/config/edit.go` | Share AST helper |
| 16 | `Ingest` | `pkg/wiki/agent.go` | Already well-factored at method level |
| 16 | `TestSearchAdapter_ExtraMapping` | `pkg/web/providers/exa/search_test.go` | Already uses t.Run |

**Already reduced from original 23:**
- `mergeConfigs` (was 35) → reduced via `mergeStringField` helper, collapsed 30+ identical branches
- `NewWikiTools` (was 25) → reduced via promotion of 7 closures to methods on WikiTools

---

## Summary of Changes

```
Files modified: ~30
Dead code removed: 9 items (6 functions/types)
Nil guards added: 14 sites
Permission fixes: 11 sites (0644→0600)
Slice preallocation: 12 sites
Type renames: 3 (SearchMode→Mode, SearchParams→Params, WikiGraph→Graph)
Context fixes: 34 sites (Background→t.Context)
Unused params removed: 6
Vulnerabilities: 15 → 0
```
