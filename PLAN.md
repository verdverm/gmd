# GMD — Port of QMD to Go

**Goal**: Rewrite [`./qmd/`](./qmd/) (TypeScript local search engine for markdown) in Go with five key architectural differences:

| QMD (TypeScript) | GMD (Go) |
|---|---|
| `node-llama-cpp` (local GGUF models) | OpenAI-compatible API (any provider) |
| `better-sqlite3` + `sqlite-vec` | [Typesense](https://typesense.org) — no operational DB |
| SQLite FTS5 + sqlite-vec for retrieval | Typesense full-text + vector hybrid search (single query) |
| YAML config | [CUE](https://cuelang.org) config (global + project-local) |
| Manual `--collection` required | Auto-detected from project root + CWD |
| CLI only | CLI + REST API + MCP server |

## Design Documents

| Document | Content |
|---|---|
| [`.design/architecture.md`](.design/architecture.md) | Storage architecture, data flow, API server, design decisions, CUE config, migration, K8s |
| [`.design/llm-wiki.md`](.design/llm-wiki.md) | LLM Wiki integration (Karpathy pattern) — architecture, built-in agent, skills, MCP tools, file layout |
| [`.design/websearch.md`](.design/websearch.md) | Web search & research (EXA) — fetch, search, agent, deep research pipeline design |

## Implementation Status

### Phase 1: Scaffold + Config + Data Layer — Done
- [x] CUE schema (`pkg/config/schema/*.cue`) with pipeline defaults
- [x] Config loader (global + project-local unification via CUE)
- [x] Project root detection (walk up from CWD)
- [x] Typesense v4 client wrapper (schema mgmt, hybrid/text search, CRUD)
- [x] `Runtime` struct with Typesense lifecycle (no operational DB)
- [x] Hash field in Typesense chunks schema for content-based change detection
- [x] `gmd init` command
- [x] Makefile with CGO-free build targets
- [x] Library packages under `pkg/` — all importable

### Phase 2: Indexing — Done
- [x] LLM client (`pkg/llm/client.go`): embeddings, chat, rerank
- [x] Markdown chunker (heading-aware breakpoints)
- [x] File scanner + SHA-256 dedup via Typesense hash field
- [x] Batch embedding + Typesense upsert
- [x] CLI commands: `update`, `embed`

### Phase 3: Search Pipeline — Done
- [x] Strong signal detection (BM25 probe via Typesense text-only search, score + gap thresholds)
- [x] LLM query expansion (chat completion generating lex/vec/hyde variants)
- [x] Typesense hybrid search wrapper (`pkg/ts/client.go` — wired into pipeline)
- [x] RRF fusion across expansion variants (k, weights, top-rank bonus from CUE config)
- [x] LLM reranking (via `/v1/rerank` endpoint; gracefully skipped if unsupported)
- [x] Position-aware blending (top/middle/bottom tiers with configurable weights)
- [x] Result formatting (CLI text + JSON output with snippets)

### Phase 4: CLI Commands — Done

All commands backed by `pkg/` code:
- [x] `gmd search` — text-only search via `pkg/search`
- [x] `gmd vsearch` — vector search via `pkg/search`
- [x] `gmd query` — full hybrid pipeline via `pkg/search`
- [x] `gmd update` — index/re-index via `pkg/indexer`
- [x] `gmd embed` — re-embed via `pkg/indexer`
- [x] `gmd status` — index health via `pkg/runtime` + `pkg/ts`
- [x] `gmd init` — creates `.gmd/config.cue`
- [x] `gmd collection list` — lists collections from `pkg/config`
- [x] `gmd collection show` — collection details + chunk count via `pkg/ts`
- [x] `gmd collection add` — config file editing (CUE AST) + `--path`/`--pattern` flags
- [x] `gmd collection remove` — config file editing + chunk deletion via `pkg/ts`
- [x] `gmd collection rename` — config file editing (CUE AST label rename)
- [x] `gmd collection include` — config file editing (sets pattern field)
- [x] `gmd collection exclude` — config file editing (adds to ignore list)
- [x] `gmd context add` — config file editing (sets context field on collection)
- [x] `gmd context list` — lists context docs from `pkg/config`
- [x] `gmd context rm` — config file editing (removes context field)
- [x] `gmd get` — fetch document content via `pkg/ts` (path-filtered search)
- [x] `gmd multi-get` — batch fetch via `pkg/ts` path filter search
- [x] `gmd ls` — list indexed documents via `pkg/ts`
- [x] `gmd doctor` — diagnostics via `pkg/config` + `pkg/runtime` + `pkg/ts`
- [x] `gmd cleanup` — stale chunk detection via `pkg/indexer` + `pkg/ts`
- [x] `gmd agentsmd` — embedded AGENTS.md content for AI coding assistants

### Phase 5: REST API Server — Stub Exists
- [ ] `gmd serve` — HTTP handler code not implemented; CLI stub with `--host`/`--port` flags in `cmd/gmd/serve.go`

### Phase 6: MCP Server — Stub Exists
- [ ] `gmd mcp` — MCP protocol code not implemented; CLI stub with `--http` flag in `cmd/gmd/mcp.go`

### Phase 7: Polish — Not Started
- [ ] LLM cache integration
- [ ] Benchmark harness (port from `./qmd/src/bench/`)
- [ ] Error handling, edge cases (empty collections, missing config, Typesense down)
- [ ] CI/CD with `CGO_ENABLED=0` check

### Phase 14: LLM Wiki Integration (Karpathy Pattern) — Done
- [x] `gmd wiki init` — scaffolds wiki directory structure, CUE config, WIKI_SCHEMA.md
- [x] `gmd wiki ingest` — built-in LLM agent: reads source → extracts entities/concepts/claims → writes wiki pages
- [x] `gmd wiki query` — searches wiki → LLM synthesizes answer with citations
- [x] `gmd wiki graph` — wikilink graph export (dot/mermaid/json)
- [x] `gmd wiki lint` — structure lint (Go) + content lint (LLM) health checks
- [x] `gmd wiki skills [list|show|write]` — embedded agent skill templates
- [x] `gmd wiki doctor` — diagnostics + agent auto-configuration
- [x] `gmd wiki watch` — fsnotify watcher for auto-ingest + auto-index
- [x] `pkg/wiki/` — agent, graph, lint, watch, frontmatter, skills, doctor packages
- [x] Embedded skill templates (`pkg/wiki/skills/`) for Claude Code, Codex, OpenCode

### Phase 15: Web Search & Research (EXA) — Designed
- [ ] `pkg/exa/client.go` — thin HTTP wrapper over EXA REST API
- [ ] `gmd web fetch <url>` — fetch clean markdown from URLs
- [ ] `gmd web search <query>` — neural web search with type/domain/date filters
- [ ] `gmd web agent <query>` — multistep searching agent with LLM orchestration
- [ ] `gmd web research <query>` — deep research pipeline (decompose → explore → cross-ref → validate → synthesize)

## Dependencies

| Module | Purpose |
|---|---|
| `github.com/typesense/typesense-go/v4` | Typesense client |
| `github.com/openai/openai-go/v3` | OpenAI-compatible API client |
| `cuelang.org/go` | CUE config loading, validation, CUE AST editing |
| `github.com/spf13/cobra` | CLI framework |
| `github.com/bmatcuk/doublestar/v4` | Glob pattern matching for file scanning |
| `gopkg.in/yaml.v3` | YAML frontmatter parsing (wiki) |
| `github.com/fsnotify/fsnotify` | File watcher (wiki watch) |

## TODO

- **Tests** — some coverage exists; need integration tests across all packages
- **Partial failure handling** — indexer needs transactional upsert (all-or-nothing per file) and retry/backoff for LLM API errors
- **Typesense resilience** — no health check, retry, or timeout logic; needs graceful degradation when Typesense is down
