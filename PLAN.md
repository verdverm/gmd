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
- [x] `gmd agents` — embedded AGENTS.md content for AI coding assistants (oneline/summary/detailed/full)

### Phase 5: REST API Server — Stub Exists
- [ ] `gmd serve` — HTTP handler code not implemented; CLI stub with `--host`/`--port` flags in `cmd/gmd/serve.go`

### Phase 6: MCP Server — Stub Exists
- [ ] `gmd mcp` — MCP protocol code not implemented; CLI stub with `--http` flag in `cmd/gmd/mcp.go`

### Phase 7: Polish — Not Started
- [ ] LLM cache integration
- [ ] Benchmark harness (port from `./qmd/src/bench/`)
- [ ] Error handling, edge cases (empty collections, missing config, Typesense down)
- [ ] CI/CD with `CGO_ENABLED=0` check


## 1. Typesense ↔ QMD Overlap Analysis

| QMD Custom Code | Typesense Replaces It? | GMD Approach |
|---|---|---|
| **FTS5 BM25 search** (`searchFTS`) | Typesense full-text search | Typesense handles it |
| **sqlite-vec vector search** (`searchVec`) | Typesense vector search | Typesense handles it |
| **RRF fusion between FTS + vector** | Typesense hybrid search (built-in Rank Fusion) | Typesense handles per-variant fusion |
| **Manual dedup by filepath** | `group_by=collection,path` collapses chunk results | Typesense handles grouping |
| **Query expansion** (LLM lex/vec/hyde) | Synonyms are complementary but don't replace LLM | LLM expansion kept; synonyms optionally layered on |
| **RRF fusion across expansion variants** | Typesense operates on a single query | Custom Go code (RRF across variant result sets) |
| **LLM reranking** | Typesense has no rerank | Custom Go code (LLM API rerank endpoint) |
| **Position-aware blending** (RRF + reranker) | Application-side logic | Custom Go code |
| **Chunking** (markdown headings + AST) | Typesense indexes whole documents or existing chunks | Custom Go code (port from QMD) |
| **Content-addressable dedup** (SHA-256) | `hash` field on Typesense chunk documents | Filter by `path` + compare hash |

### Simplified Search Pipeline

Before (QMD):
```
Per variant: FTS search → ranked list + Vector search → ranked list → manual RRF fusion
                                                                         ↓
                    All variants fused via RRF → rerank → position-blend
```

After (GMD with Typesense):
```
Per variant: Typesense hybrid search (text + vector fused internally, grouped by doc)
                                                                         ↓
                    All variants fused via RRF → rerank → position-blend
```

Each variant goes from **2 queries + custom fusion** (QMD) to **1 query** (GMD).


## 2. Storage Architecture

There is no operational database. Typesense is the sole data store. CUE config is the
source of truth for collection definitions; the filesystem is the source of truth for
document content.

### Typesense — Search Index + Change Detection

Chunks are indexed as individual Typesense documents with `group_by` for document-level
collapse and a `hash` field for content-based change detection:

```json
{
  "name": "chunks",
  "fields": [
    {"name": "collection", "type": "string", "facet": true},
    {"name": "path",       "type": "string", "facet": true},
    {"name": "title",      "type": "string"},
    {"name": "content",    "type": "string"},
    {"name": "hash",       "type": "string"},            ← SHA-256 of source file
    {"name": "chunk_seq",  "type": "int32"},
    {"name": "total_chunks","type": "int32"},
    {"name": "embedding",  "type": "float[]", "num_dim": 768}
  ]
}
```

Indexing uses filter-by-path + hash comparison to skip unchanged files without
re-chunking or re-embedding. Search uses `group_by=collection,path` with
`group_limit=1` to return one result per document (best chunk).

### Embedding Strategy: External (Go → OpenAI API)

Typesense supports both auto-embedding (server-side) and external embeddings. GMD uses
**external embeddings**:

| Step | What Happens |
|---|---|
| **Index time** | Go chunks documents → calls OpenAI-compatible API for embeddings → upserts `{ content, embedding, ... }` with `hash` to Typesense |
| **Search time** | Go embeds query text via API → sends `vector_query` param to Typesense hybrid search |
| **Why external?** | User controls the embedding model in GMD config (not locked to Typesense-supported models). Consistent with "OpenAI-compatible module" requirement. |


## 3. Data Flow

### Indexing Pipeline

```
CUE config loaded (global + project-local unified)
  ↓
Project root detected (walk up from CWD)
  ↓
For each collection defined in merged config:
  ↓
Filesystem scan (filepath.Walk + glob pattern matching)
  ↓
SHA-256 hash each file → query Typesense for matching path + hash
  ├─ Chunk exists with same hash → skip (no re-chunking or re-embedding)
  ├─ Chunk exists with different hash → delete stale chunks, re-index
  └─ No chunks for path → index from scratch
  ↓
Chunking (heading-aware breakpoints, target tokens + overlap from CUE config)
  ↓
Batch embedding via OpenAI-compatible API
  ↓
Upsert chunks into Typesense (hash included on every chunk document)
```

### Search Pipeline

```
User runs `gmd query "..."` in a directory
  ↓
Project auto-detected (walk up from CWD)
  ↓
Collections auto-selected based on CWD
  ↓
CUE config loaded and merged (global + project-local)
  ↓
LLM query expansion (OpenAI chat completion, prompt from CUE config if overridden)
  → generates {type, query}[] where type ∈ {lex, vec, hyde}
  ↓
Strong Signal Check (thresholds from CUE config)
  └─ If strong, skip expansion and use original query directly
  ↓
For each variant (original ×2 weight, expansions ×1 weight):
  → Embed query via OpenAI API (for vec/hyde variants)
  → Typesense Hybrid Search with CUE-configured parameters
       q=<text> + query_by=content + vector_query=<embedding>
       group_by=collection,path + group_limit=1
  → Returns ranked results with Typesense's built-in fusion score
  ↓
RRF Fusion across all variant result sets
  → RRF_score(d) = Σ(w_i / (k + rank_i(d))) + topRankBonus    (k from CUE config)
  → Take top N candidates (N from CUE config)
  ↓
Best-chunk selection per candidate
  → Apply intent-weighted keyword scoring to find best chunk per doc
  ↓
LLM Reranking (LLM API rerank endpoint)
  → Score each (query, chunk) pair for relevance
  ↓
Position-Aware Blending (thresholds + weights from CUE config)
  → Rank 1-3:     topWeight × RRF + (1-topWeight) × Reranker
  → Rank 4-10:    middleWeight × RRF + (1-middleWeight) × Reranker
  → Rank 11+:     bottomWeight × RRF + (1-bottomWeight) × Reranker
  ↓
Dedup, filter by minScore, slice to limit
  ↓
Return final ranked results
```

### Search Modes

| CLI Command | Pipeline | LLM Needed? |
|---|---|---|
| `gmd search` | Text-only Typesense search (no vector, no expansion) | No |
| `gmd vsearch` | Vector-only Typesense search (no text) | Embedding model |
| `gmd query` | Full pipeline: expansion → hybrid → RRF → rerank → blend | All 3 models |


## 4. API Server

`gmd serve` starts an HTTP server exposing all GMD operations as REST endpoints. Shares the same `Runtime` backend as the CLI and MCP server.

| Endpoint | Method | Description | Analogous CLI |
|---|---|---|---|
| `/health` | GET | Liveness check | — |
| `/status` | GET | Index and collection health | `gmd status` |
| `/search` | POST | Full-text keyword search | `gmd search` |
| `/vsearch` | POST | Vector similarity search | `gmd vsearch` |
| `/query` | POST | Full hybrid pipeline (expansion → rerank → blend) | `gmd query` |
| `/documents/{path}` | GET | Get document content by path | `gmd get` |
| `/documents/multi-get` | POST | Batch fetch by path pattern | `gmd multi-get` |
| `/collections` | GET | List collections | `gmd collection list` |
| `/update` | POST | Trigger reindex | `gmd update` |
| `/embed` | POST | Trigger embedding | `gmd embed` |

**Request/response format**: JSON for all endpoints. Search endpoints accept the same parameters as CLI flags (`collection`, `limit`, `min_score`, `format`, etc.) as JSON body fields. The `query` endpoint supports both simple query strings and pre-expanded structured queries (`lex`/`vec`/`hyde`).

**Configuration**: The `serve` subcommand accepts `--port` (default 8181), `--host` (default localhost), and reads all other settings (CORS origins, TLS, rate limiting) from the CUE config.

**Implementation**: Uses Go 1.22+ `net/http` with enhanced `ServeMux` for routing (no external router dependency). Standard middleware: request logging, recovery, CORS, optional API key authentication.


## 5. Key Dependencies (Go)

| Module | Purpose |
|---|---|
| `github.com/typesense/typesense-go` | Typesense client |
| `github.com/openai/openai-go` | OpenAI-compatible API client |
| `cuelang.org/go` | CUE config loading, validation, unification |
| `github.com/spf13/cobra` | CLI framework |
| `golang.org/x/sync` | errgroup, semaphore for parallel work |


## 6. Implementation Phases

### Phase 1: Scaffold + Config + Data Layer
- Define CUE schema (types.cue, pipeline.cue, config.cue) with all pipeline knobs + defaults
- Embed schema via `//go:embed`
- Implement Go config loader: load global CUE → detect project root → load project-local CUE → unify → validate → export to Go struct
- Implement project root detection (walk up from CWD looking for `.gmd/`)
- Implement Typesense client wrapper (schema creation, document CRUD, hybrid search)
- `Runtime` struct with `Open()` / `Close()` lifecycle
- `gmd init` command: creates `.gmd/config.cue` in project root

### Phase 2: Indexing
- File scanning with glob matching (respecting ignore patterns)
- Content-addressed dedup via Typesense hash field (filter by path → compare hash → skip or re-index)
- Markdown chunking (heading-aware breakpoints, parameters from CUE config)
- Batch embedding pipeline (OpenAI-compatible API, retry logic, progress)
- Typesense upsert (delete stale chunks for path, insert new set with hash)
- Progress reporting (CLI output)

### Phase 3: Search Pipeline
- Strong signal detection (BM25 probe via Typesense text-only search)
- LLM query expansion (chat completion with grammar-like constraint for lex/vec/hyde)
- Typesense hybrid search wrapper (`q` + `vector_query` + `group_by`)
- RRF fusion across expansion variants (k, weights, bonuses from CUE config)
- LLM reranking (LLM API rerank endpoint; skip if unsupported)
- Position-aware blending (thresholds + weights from CUE config)
- Result formatting with snippets

### Phase 4: CLI — Commands Registered, Stubs Implemented
All QMD commands, ported:
`status` `update` `embed` `search` `vsearch` `query` `get` `multi-get`
`collection [add|list|remove|rename|show|include|exclude]`
`context [add|list|rm]` `ls` `init` `doctor` `cleanup` `mcp` `serve`
`import-qmd` (migration helper: reads QMD SQLite DB → Typesense + CUE config)

Auto-detection integration: `status` shows project root + matched collections, `query`/`search`/`vsearch` auto-select collections from CWD.

### Phase 5: REST API Server — Stub Exists
- HTTP server setup (Go 1.22+ `net/http` ServeMux, middleware stack)
- Endpoint handlers for all 10 routes (health, status, search, vsearch, query, documents, multi-get, collections, update, embed)
- Request validation, JSON response formatting, error handling
- CORS, rate limiting, optional API key auth via CUE config
- `gmd serve` command with `--port` and `--host` flags

### Phase 6: MCP Server — Stub Exists
- MCP tools: `query`, `get`, `multi_get`, `status`
- MCP resource: `gmd://{+path}`
- Transports: stdio and Streamable HTTP
- Daemon mode (PID file, signal handling)

### Phase 7: Polish
- LLM cache integration
- Benchmark harness (port from `./qmd/src/bench/`)
- Error handling, edge cases (empty collections, missing config, Typesense down)
- Documentation, CI/CD with `CGO_ENABLED=0` check


## 7. Key Design Decisions

### 7a. Typesense handles per-query fusion; Go handles cross-variant fusion
Typesense's built-in hybrid search fuses text + vector rankings for a single query. But GMD generates multiple expansion variants (lex/vec/hyde) and needs RRF fusion across them. That cross-variant fusion stays in Go.

### 7b. Chunks as Typesense documents with grouping
Each chunk is a separate Typesense document. The `group_by=collection,path` parameter collapses chunk results to document level.

### 7c. External embeddings (not Typesense auto-embedding)
Embeddings computed in Go via OpenAI-compatible API, stored in Typesense's `float[]` field. Gives model flexibility.

### 7d. LLM reranking via the `/v1/rerank` endpoint
Reranking uses the LLM API's `/v1/rerank` endpoint (same base URL, same API key as embeddings and chat). This is the Jina/Cohere-compatible cross-encoder rerank format supported natively by vLLM and other OpenAI-compatible providers. It mirrors the original QMD approach of using a dedicated reranker model via `context.rankAll()`. If the provider does not support `/v1/rerank`, reranking is skipped.

### 7e. Content-addressable dedup via Typesense hash field
SHA-256 hash stored on every chunk document. On re-index, filter by `path`, compare hash — if
unchanged, skip the file entirely (no re-chunking, no re-embedding). Typesense doubles as both
search index and change-detection source of truth.

### 7f. CUE as the sole config language
No YAML fallback. CUE handles global + project-local config with structural sharing and validation. The config loader:
1. Embeds built-in schema
2. Loads global `~/.config/gmd/config.cue` (optional)
3. Detects project root by walking up from CWD
4. Loads `<project-root>/.gmd/config.cue` (optional)
5. Unifies: `built-in & global & project-local`
6. Validates against schema
7. Exports validated Go struct

### 7g. Project auto-detection by sentinel walk
Walk up from CWD checking for `.gmd/` dir. Once found, that's the project root. Collections have paths relative to project root. CWD-based collection matching uses path prefix comparison.

### 7h. No CGO
No CGO dependencies. CI enforces `CGO_ENABLED=0`.

### 7i. OpenAI-compatible, not OpenAI-specific
The `llm.Client` abstraction wraps any OpenAI-compatible provider via `base_url` + `api_key`.

### 7j. REST API as a first-class interface alongside CLI and MCP
`gmd serve` provides a full REST API sharing the same `Runtime` backend. Three interfaces (CLI, REST, MCP) serve different use cases: interactive use, programmatic/scripting, and AI agent integration. The API uses stdlib `net/http` (Go 1.22+ enhanced ServeMux) to avoid external HTTP router dependencies.


## 8. Configuration (CUE)

### Global config: `~/.config/gmd/config.cue`

```cue
package gmd

Config: {
	// LLM provider (OpenAI-compatible)
	llm: {
		base_url:           "http://localhost:11434/v1"
		api_key:            ""   // fallback: OPENAI_API_KEY env
		embedding_model:    "google/embeddinggemma-300m"
		expansion_model:    "Qwen/Qwen3-1.7B"
		rerank_model:       "Qwen/Qwen3-Reranker-0.6B"
	}

	// Search engine (Typesense is the sole data store)
	typesense: {
		host:    "http://localhost:8108"
		api_key: "xyz"
	}

	// Pipeline overrides (optional — all fields have defaults)
	pipeline: chunk: targetTokens: 1024

	// Global collections
	collections: docs: {
		path:    "~/documents"
		pattern: "**/*.md"
		ignore:  ["node_modules/**"]
		context: "Technical documentation"
	}
}
```

### Project-local config: `<project-root>/.gmd/config.cue`

```cue
package gmd

Config: {
	collections: myapp: {
		path:    "docs"
		pattern: "**/*.{md,mdx}"
		context: "MyApp user documentation"
	}

	pipeline: {
		rrf: k: 80
		rerank: candidateLimit: 20
	}
}
```

### Exposed pipeline parameters (all with defaults):

| Parameter | CUE Path | Default | Description |
|---|---|---|---|
| Chunk target tokens | `pipeline.chunk.targetTokens` | 900 | Target tokens per chunk |
| Chunk overlap | `pipeline.chunk.overlap` | 0.15 | Fraction overlap between chunks |
| Heading weight H1 | `pipeline.chunk.headingWeights.h1` | 100 | Breakpoint score for H1 headings |
| Heading weight H6 | `pipeline.chunk.headingWeights.h6` | 50 | Breakpoint score for H6 headings |
| Code fence weight | `pipeline.chunk.codeFenceWeight` | 10 | Breakpoint score for code fences |
| Newline weight | `pipeline.chunk.newlineWeight` | 1 | Breakpoint score for newlines |
| Strong signal min score | `pipeline.strongSignal.minScore` | 0.85 | BM25 score threshold |
| Strong signal min gap | `pipeline.strongSignal.minGap` | 0.15 | Gap between top 2 scores |
| RRF k constant | `pipeline.rrf.k` | 60 | RRF rank scaling |
| Original query weight | `pipeline.rrf.originalWeight` | 2.0 | RRF weight for original query |
| Expansion weight | `pipeline.rrf.expansionWeight` | 1.0 | RRF weight for variants |
| Rerank candidate limit | `pipeline.rerank.candidateLimit` | 40 | Max docs to rerank |
| Rerank context size | `pipeline.rerank.contextSize` | 4096 | Token budget per doc |
| Blend top threshold | `pipeline.blending.thresholds.top` | 3 | Rank threshold for top tier |
| Blend middle threshold | `pipeline.blending.thresholds.middle` | 10 | Rank threshold for middle tier |
| Blend top weight | `pipeline.blending.weights.top` | 0.75 | RRF weight in top tier |
| Blend middle weight | `pipeline.blending.weights.middle` | 0.60 | RRF weight in middle tier |
| Blend bottom weight | `pipeline.blending.weights.bottom` | 0.40 | RRF weight in bottom tier |
| Default output format | `pipeline.output.defaultFormat` | "cli" | CLI output format |
| Max results | `pipeline.output.maxResults` | 5 | Default result count |


## 9. Migration from QMD

| Concern | Approach |
|---|---|
| Existing QMD SQLite DB | Optional `gmd import-qmd` command: read QMD's `collection`/`content`/`documents` tables, migrate to Typesense + CUE config |
| Typesense server setup | Must be running (Docker compose provided, docs for self-host/cloud) |
| API key for LLM | Default to `OPENAI_API_KEY` env var; docs for Ollama/local setups (no key needed) |
| CGO-free CI | `CGO_ENABLED=0 go build ./...` in CI pipeline |
| Converting QMD YAML to CUE | `gmd import-qmd` generates `~/.config/gmd/config.cue` from existing YAML config |


## 10. What Stays the Same (from QMD)

- Query expansion prompt format and grammar (`lex`/`vec`/`hyde` lines)
- RRF fusion formula with weights and top-rank bonuses
- Position-aware blending thresholds (top/middle/bottom tiers)
- Strong signal detection heuristic (score + gap thresholds)
- Chunking strategy: heading-aware breakpoints with configurable token target
- Output formatters (CLI, JSON, CSV, MD, XML, files)
- MCP server tools and resources
- Core CLI commands (search, vsearch, query, update, embed, status, etc.)

## 11. What Changes

| QMD | GMD | Why |
|---|---|---|
| `better-sqlite3` + `sqlite-vec` | Typesense | No CGO, single data store for search + metadata |
| `node-llama-cpp` (local GGUF) | OpenAI-compatible API | User's requirement |
| Two searches per variant (FTS + vec) | One hybrid search per variant | Typesense does both + fusion |
| Manual chunk dedup | `group_by=collection,path` | Typesense built-in |
| Raw BM25 + cosine scores | Typesense `_text_match` fusion score | Typesense abstraction |
| Three local GGUF models | One API client for all LLM tasks | Unified interface |
| Tree-sitter AST chunking for code | Dropped — text-only focus, no CGO | Tree-sitter requires CGO; not needed for markdown |
| YAML config file | CUE config (`.cue` files) | Validation + constraints + merging |
| Global config only | Global + project-local with CUE unification | Project awareness |
| Manual `--collection` flag | Auto-detected from CWD + project root | Zero-config in project dirs |
| Fixed pipeline parameters | All pipeline knobs exposed in CUE schema | Power-user customization |
| CLI only | CLI + REST API + MCP server | `gmd serve`, `gmd mcp`, and `gmd <subcommand>` |
| No agent docs | `gmd agents` embedded content (oneline/summary/detailed/full) | AI agent onboarding without external docs |


## 12. K8s Infrastructure (gmd namespace)

These resources already exist and are managed via `kubectl apply -f k8s/`. See
[k8s/README.md](k8s/README.md) for operational details.

All resources are in the `gmd` namespace, pinned to node `nitrogen` via `nodeSelector`.
Typesense is deployed as a `TypesenseCluster` CRD and exposed via NodePort services.

### Files

```
k8s/
└── typesense.yaml   # TypesenseCluster + NodePort Service
```


## 13. TODO (Next)

- **Tests** — some coverage exists; need integration tests across all packages
- **Partial failure handling** — indexer needs transactional upsert (all-or-nothing per file) and retry/backoff for LLM API errors
- **Typesense resilience** — no health check, retry, or timeout logic; needs graceful degradation when Typesense is down


## 14. LLM Wiki Integration (Karpathy Pattern)

Andrej Karpathy's [LLM Wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) (April 2026) proposes a pattern where an LLM agent incrementally builds and maintains a persistent, interlinked markdown wiki from raw sources. It shifts from query-time RAG retrieval to a compounding knowledge artifact. Karpathy explicitly names **qmd** as the recommended search backend for wikis that outgrow simple `index.md` navigation — making GMD (qmd's Go successor) the natural search infrastructure for this ecosystem.

### 14a. Architecture Overview

GMD is both the **search infrastructure** AND the **built-in wiki operator**. Users have two paths:

**Path A — Built-in agent (CLI-only):** GMD itself acts as the LLM agent, using its own `pkg/llm` client to read sources, generate wiki pages, answer queries, and run lint passes. No external agent needed. `gmd wiki ingest paper.md` → wiki pages appear.

**Path B — MCP + Skills (external agent):** GMD runs as an MCP server providing search/indexing tools, and the user's preferred agent (Claude Code, Codex, OpenCode) consumes them. GMD ships embedded skill files that get written to the agent's discovery path so any agent can be a capable wiki operator.

| Layer | Owned By | What It Does |
|---|---|---|
| Raw sources (`raw/`) | User | Immutable originals, not indexed by GMD |
| The wiki (`wiki/`) | GMD agent OR external agent | LLM-authored markdown with `[[wikilinks]]` |
| Schema (WIKI_SCHEMA.md) | GMD (embedded template) | Conventions, workflows, page formats |
| **Search index** | **GMD** | Typesense index of wiki pages with hybrid search |
| **Built-in agent** | **GMD** | CLI-driven ingest/query/lint/watch using its LLM client |
| **MCP tools** | **GMD** | `search`, `get`, `neighbors`, `update` exposed to external agents |
| **Skills** | **GMD** | Embedded skill files written to agent discovery paths |
| **Lint infrastructure** | **GMD + LLM** | Structure checks (Go) + content analysis (LLM) |

The key insight: GMD indexes the *wiki pages* (LLM output), not the raw sources. GMD provides search over the compiled, cross-referenced, curated knowledge. Whether the wiki pages were authored by GMD's built-in agent or an external agent is transparent to the search layer.

### 14b. Data Flow — Two Paths

**Path A: Built-in agent (CLI-only, self-contained)**

```
User runs: gmd wiki ingest paper.md
        │
        ▼
GMD built-in agent reads paper.md (raw source)
        │
        ▼
GMD calls LLM chat completion with ingest prompt + wiki schema
   → LLM returns structured response: entities, concepts, claims, summary
        │
        ▼
GMD writes/updates wiki pages to wiki/
   (summary page, entity pages, concept pages, updates _index.md, _log.md)
        │
        ▼
GMD auto-triggers re-index: updates Typesense with new chunks
        │
        ▼
Wiki is queryable via: gmd wiki query "what is..."
   → GMD's search pipeline finds relevant pages → LLM synthesizes answer
   → Optionally files answer as new wiki page
```

**Path B: MCP + Skills (external agent, user's preferred tool)**

```
gmd mcp --wiki myresearch starts MCP server with wiki tools
        │
User's agent (Claude Code, Codex, etc.) loads GMD skill from discovery path
        │
User: "ingest https://example.com/article"
        │
Agent reads article, calls GMD MCP: gmd_wiki_search (check for existing)
        │
Agent writes wiki pages (using its own file tools)
        │
Agent calls GMD MCP: gmd_wiki_update → Typesense re-indexed
        │
User: "what does the wiki say about X?"
        │
Agent calls GMD MCP: gmd_wiki_search("X") → ranked pages + snippets
        │
Agent reads top pages, synthesizes answer
```

### 14c. Wiki as a GMD Collection

A wiki is a standard GMD collection with additional conventions. No new data model needed — the existing `CollectionConfig` with `path`, `pattern`, `ignore`, `context` covers it:

```cue
collections: myresearch: {
    path:    "wiki"
    pattern: "**/*.md"
    ignore:  ["_index.md", "_log.md"]   // meta-files, not content
    context: "AI research knowledge base"
    wiki: {                              // NEW: wiki-specific settings
        enabled:      true
        indexFile:    "_index.md"        // path to catalog file
        logFile:      "_log.md"          // path to chrono log
        graphLinks:   true               // parse [[wikilinks]] for graph edges
        frontmatter: {                   // per IDEAS.md frontmatter extraction
            fields: {
                type:   { type: "string",  facet: true }
                tags:   { type: "string[]", facet: true }
                sources:{ type: "string[]" }
                status: { type: "string",  facet: true }
            }
        }
    }
}
```

The `wiki` field in `CollectionConfig` is optional — when present, GMD applies wiki-aware behavior:
- Skips `indexFile` and `logFile` from chunking (they're navigation, not content)
- Parses `[[wikilinks]]` during chunking to extract outgoing link edges
- Populates Typesense `links` field (string array) for each chunk
- Extracts YAML frontmatter into typed fields enabling `--filter "type:=concept"`, `--facet tags`, etc.

### 14d. New CUE Schema Fields

Add to `types.cue`:

```cue
CollectionConfig: {
    // ... existing fields ...
    wiki?: WikiConfig | *null   // optional, activates wiki mode
}

WikiConfig: {
    enabled:        bool | *true
    indexFile:      string | *"_index.md"
    logFile:        string | *"_log.md"
    graphLinks:     bool | *true
    frontmatter?: {                         // optional per-wiki frontmatter config
        fields: [string]: FrontmatterField
    }
}

FrontmatterField: {
    type:  "string" | "string[]" | "int32" | "float64" | "bool"
    facet?: bool | *false
    sort?:  bool | *false
}
```

Typesense schema additions for wiki mode:
```
links:       string[]    // outgoing [[wikilinks]] from this page/chunk
// plus dynamic fields from frontmatter extraction
```

### 14e. Implementation Phases

#### Phase W1: Wiki Scaffolding — `gmd wiki init`

New `cmd/gmd/wiki.go` with a parent command and subcommands. `wiki init` creates the directory structure, CUE config entry, and writes embedded skill files:

```
gmd wiki init [--name <name>] [--path <path>]
    Creates:
    ├── raw/                  (immutable sources, not indexed)
    ├── wiki/
    │   ├── entities/         (people, orgs, products, technologies)
    │   ├── concepts/         (methodologies, architectures, theories)
    │   ├── comparisons/      (X vs Y analyses)
    │   ├── synthesis/        (cross-source analysis, saved answers)
    │   ├── sources/          (summaries of ingested content)
    │   ├── _index.md         (content catalog, LLM-maintained)
    │   └── _log.md           (chronological record, LLM-maintained)
    ├── WIKI_SCHEMA.md        (embedded template — conventions + workflows)
    └── .gmd/config.cue       (updated with wiki collection + WikiConfig)
```

`wiki init` also offers to write skill files to agent discovery paths via `--skills` flag (see Phase W7).

#### Phase W2: Built-in Agent — `gmd wiki ingest`, `gmd wiki query`

GMD's built-in agent uses `pkg/llm` (chat completion) to operate the wiki without any external agent. This is the simplest user experience: install GMD, run `gmd wiki ingest paper.md`, done.

**`gmd wiki ingest <source> [--name <name>] [--batch]`**
```
1. Read source file (markdown, text, or URL via HTTP fetch)
2. Search wiki for existing pages that overlap (via gmd search)
3. Call LLM chat completion with structured ingest prompt:
   - System: WIKI_SCHEMA.md content + existing page summaries
   - User: source content + "Extract entities, concepts, claims. Generate wiki pages."
4. LLM returns structured JSON response:
   {
     "title": "Source summary title",
     "summary": "...",
     "entities": [{"name": "...", "page": "..."}, ...],
     "concepts": [{"name": "...", "page": "..."}, ...],
     "claims": [{"text": "...", "contradicts": "page-name", ...}, ...],
     "comparisons": [{"a": "...", "b": "...", "analysis": "..."}, ...],
     "index_updates": [...],
     "log_entry": "..."
   }
5. GMD writes/updates files in wiki/ (create new or merge with existing)
6. Updates _index.md with new/updated page entries
7. Appends entry to _log.md
8. Auto-runs gmd update wiki/ to re-index
9. Reports: "Ingested paper.md → created 3 pages, updated 2, flagged 1 contradiction"
```

**`gmd wiki query "<question>" [--name <name>] [--save] [--limit N]`**
```
1. Calls GMD's own search pipeline (hybrid search over wiki collection)
2. Reads top-N matching wiki pages in full
3. Calls LLM chat completion with RAG-style prompt:
   - System: WIKI_SCHEMA.md content
   - User: question + page contents with [[wikilink]] citations
4. LLM returns answer with citations (inline [[page]] references)
5. If --save: files answer as new page in wiki/synthesis/
6. Outputs answer + sources to CLI
```

**`gmd wiki watch [--name <name>]`**
```
Runs indefinitely:
1. Watches raw/ for new files (fsnotify)
2. On new file: auto-runs gmd wiki ingest
3. Watches wiki/ for changes
4. On change: auto-runs incremental re-index (single file)
User drops a paper into raw/ → wiki auto-updates, stays indexed.
```

This is the "set and forget" mode. Combine with `gmd wiki lint --watch` for periodic health checks.

#### Phase W3: Wikilink Parsing & Graph Indexing

Modify `pkg/chunking/markdown.go` to extract `[[wikilinks]]` during chunking:

```go
// New fields on Chunk:
type Chunk struct {
    Content     string
    Title       string
    ChunkSeq    int32
    TotalChunks int32
    Links       []string                   // NEW: outgoing [[wikilinks]]
    Frontmatter map[string]interface{}     // NEW: extracted frontmatter fields
}
```

Wikilink regex: `\[\[([^\]|#]+)(?:[|#][^\]]+)?\]\]` captures the target page name.

For inbound link resolution, query Typesense at read time: `filter_by=links:=[pageName]`. This avoids needing to update all chunks when a new page links to an existing one.

**`gmd wiki graph [--name <name>] [--format dot|mermaid|json]`**
Reads all chunks with non-empty `links` field, constructs adjacency list, outputs graph. Useful for visualization (Obsidian graph view, Gephi, Graphviz).

#### Phase W4: Frontmatter Extraction

Parse the leading `---` delimited YAML block before chunking, extract configured fields, strip the block from chunk content. Combined with wiki mode, this enables facet filtering and sort:

```
gmd wiki query "deploy" --filter "type:=concept && tags:=kubernetes"
gmd wiki query "architecture" --facet type,status
gmd wiki query "deploy" --sort-by "difficulty:asc"
```

See IDEAS.md "Markdown Frontmatter" section for full design. Enables the wiki frontmatter conventions:
```yaml
---
type: concept
tags: [kubernetes, deployment]
status: reviewed
sources: [paper-2024.md]
difficulty: 3
---
```

#### Phase W5: Wiki Lint (LLM-Powered Health Check)

**`gmd wiki lint [--name <name>] [--watch]`**

Multi-step analysis, results saved to `_lint.md`:

1. **Structure lint** — pure Go, no LLM needed:
   - Orphan pages: zero inbound `[[wikilinks]]`
   - Missing pages: wikilink targets with no matching page
   - Broken links: wikilink targets with no matching file
   - Stale entries in `_index.md` referencing deleted pages

2. **Content lint** — LLM-powered:
   - Contradiction detection: send pairs of pages to LLM, flag opposing claims
   - Staleness: pages not updated despite newer related sources existing
   - Source coverage: concepts with only one source vs. multiple corroborating

3. **Gap analysis** — LLM-powered:
   - Read `_index.md` + `overview.md`, ask LLM to identify missing topics
   - Generate suggested web search queries for filling gaps

With `--watch`, runs lint on a schedule (e.g., hourly or daily), appending new findings to `_lint.md`.

#### Phase W6: MCP Tools (Wiki-Aware)

Add to the MCP server (Phase 6) a set of wiki-scoped tools alongside the general ones:

| MCP Tool | Params | Description |
|---|---|---|
| `gmd_wiki_search` | `query`, `wiki`, `filter?`, `limit?` | Hybrid search scoped to wiki collection, with optional frontmatter filter |
| `gmd_wiki_get` | `path`, `wiki` | Read a specific wiki page by path |
| `gmd_wiki_neighbors` | `path`, `wiki`, `direction?` | Return pages linked to/from this page (inbound + outbound [[wikilinks]]) |
| `gmd_wiki_status` | `wiki` | Wiki health: page counts by type, orphan count, last update |
| `gmd_wiki_suggest` | `prefix`, `wiki` | Autocomplete page titles for [[wikilink]] target suggestions |
| `gmd_wiki_update` | `wiki` | Trigger re-index of wiki collection |
| `gmd_wiki_ingest` | `source`, `wiki` | Run built-in ingest agent on a raw source (uses Path B agent) |

General MCP tools (`gmd_query`, `gmd_get`, `gmd_multi_get`) remain available for any collection. Wiki tools add link traversal, page-type filtering, and auto-complete.

#### Phase W7: Embedded Skills & Agent Discovery

**`gmd wiki skills [write|list|show] [--target <agent>]`**

GMD ships with embedded skill templates (via `//go:embed` in `pkg/wiki/skills/`). These are the instructions that tell any LLM agent how to be a disciplined wiki maintainer — the equivalent of Karpathy's "schema" layer, but packaged as installable skills for specific agent platforms.

```
pkg/wiki/skills/              # Embedded skill files
├── AGENTS.md                 # Universal agent instructions (ingest/query/lint workflows)
├── WIKI_SCHEMA.md            # Wiki conventions, directory structure, page formats
├── claude-code.md            # Claude Code-specific skill (tool mappings)
├── codex-cli.md              # Codex CLI-specific skill
├── opencode.md               # OpenCode-specific skill
└── generic.md                # Fallback for any agent that reads AGENTS.md
```

**`gmd wiki skills list`** — lists available skill templates (name, target agent, description)

**`gmd wiki skills show [name]`** — prints a skill template to stdout

**`gmd wiki skills write [--target claude|codex|opencode|all] [--wiki <name>]`**

Writes skill files to the agent's discovery path so the user's preferred agent automatically picks them up:

| `--target` | Writes to | Agent auto-discovers? |
|---|---|---|
| `claude` | `~/.claude/skills/gmd-wiki.md` | Yes (Claude Code skills directory) |
| `codex` | `.agents/skills/gmd-wiki/` (project-local) | Yes (Codex skill discovery) |
| `opencode` | `~/.config/opencode/skills/gmd-wiki.md` | Yes (OpenCode skill discovery) |
| `all` | All of the above | — |

The skill files include:
- Tool mapping (which GMD MCP tools to use for each workflow step)
- Ingest workflow (read raw → search existing → generate pages → update index/log)
- Query workflow (search wiki → read pages → synthesize → optionally save)
- Lint workflow (structure checks → LLM content review → fix suggestions)
- Page templates (entity page, concept page, comparison page, source summary)
- Naming conventions for wiki pages and YAML frontmatter fields

**`gmd wiki init --skills [--target all]`** writes skills during initialization.

#### Phase W8: Agent Discovery Auto-Configuration

`gmd wiki doctor` checks whether the wiki's MCP server is configured for the user's detected agents:

```
$ gmd wiki doctor
  Wiki: myresearch (12 pages, 3 sources)
  Typesense: ✓ connected (8108)
  LLM: ✓ embedding ✓ expansion ✓ rerank
  Agent discovery:
    Claude Code: ✓ skill installed (~/.claude/skills/gmd-wiki.md)
                 ✗ MCP not configured → run: gmd wiki doctor --fix
    Codex CLI:   ✓ skill installed (.agents/skills/gmd-wiki/)
                 ✓ MCP configured
    OpenCode:    - not detected
```

With `--fix`, GMD writes the MCP configuration snippets to the appropriate agent config files (e.g., `.claude/settings.json`, `opencode.jsonc`).

#### Phase W9: Advanced Agent Features

**Multi-source batch ingest:**
`gmd wiki ingest raw/*.md --batch` — ingest many sources in one run. GMD reads all, sends to LLM in a single prompt with all sources, receives consolidated output. Faster and cheaper than sequential single-source ingest.

**Interactive ingest mode:**
`gmd wiki ingest paper.md --interactive` — after LLM generates proposed wiki changes, GMD shows a diff summary and prompts user to accept/reject/edit before writing files. Crucial for high-stakes wikis where accuracy matters.

**Cross-wiki query:**
`gmd wiki query "compare approaches" --wikis research,engineering` — search across multiple wikis, synthesize answer drawing from both knowledge bases.

**Wiki export:**
`gmd wiki export [--name <name>] [--format llms.txt|jsonld|html|pdf]`
- `llms.txt` / `llms-full.txt` — for other AI agents to consume
- `jsonld` — linked data graph export
- `html` — static site generation from wiki pages
- `pdf` — via Marp or equivalent markdown-to-PDF pipeline

### 14f. File Layout

```
pkg/wiki/                       # New package
├── wiki.go                     # Wiki type, init logic, directory scaffolding
├── agent.go                    # Built-in agent: ingest, query, watch orchestrator
├── agent_prompts.go            # Embedded LLM prompts for ingest/query/lint
├── graph.go                    # Wikilink parsing, graph construction, export
├── lint.go                     # Structure lint (Go) + content lint (LLM) orchestrator
├── watch.go                    # fsnotify watcher for raw/ + wiki/ auto-indexing
├── frontmatter.go              # YAML frontmatter parser + field extraction
├── skills.go                   # Embedded skill loader + agent discovery writer
├── skills/                     # Embedded skill templates (//go:embed)
│   ├── AGENTS.md               # Universal agent instructions
│   ├── WIKI_SCHEMA.md          # Wiki conventions + page templates
│   ├── claude-code.md          # Claude Code-specific skill
│   ├── codex-cli.md            # Codex CLI-specific skill
│   ├── opencode.md             # OpenCode-specific skill
│   └── generic.md              # Fallback for any AGENTS.md-reading agent
├── doctor.go                   # Wiki health diagnostics + auto-config
└── wiki_test.go                # Tests

cmd/gmd/wiki.go                 # CLI commands (wiki init, ingest, query, lint, graph, watch, skills, export, doctor)

pkg/mcp/wiki_tools.go           # Wiki-specific MCP tool implementations (Phase 6+)

pkg/config/schema/types.cue     # Add WikiConfig + FrontmatterField types
```

### 14g. Integration with Existing Infrastructure

| Existing Component | How Wiki Uses It |
|---|---|
| `pkg/config` CUE loader | Wiki config is a `CollectionConfig` with `wiki: {...}` field |
| `pkg/config/edit.go` | `wiki init` adds collection via existing AST edit functions |
| `pkg/runtime` | Shared Runtime provides TS client + config to wiki commands |
| `pkg/ts/client.go` | Existing `HybridSearch`, `TextSearch`, `SearchChunksByPath` — wiki MCP tools are thin wrappers |
| `pkg/search/pipeline.go` | Wiki queries use the same pipeline; wiki-lint uses LLM for contradiction analysis |
| `pkg/chunking` | Extended with wikilink extraction and frontmatter parsing |
| `pkg/indexer` | Reused for wiki indexing; `wiki watch` calls single-file incremental index |
| `pkg/llm/client.go` | Wiki lint uses chat completion for contradiction detection |
| `pkg/output` | Wiki results formatted identically to regular search results |

### 14h. Agent Integration Contract

The "contract" between GMD and the LLM agent is defined by what the wiki MCP tools expose. The agent's CLAUDE.md or AGENTS.md should include instructions like:

```markdown
## Wiki Search (via GMD MCP)

Use `gmd_wiki_search` to find relevant wiki pages before answering.
- Pass the user's question as `query`
- Use `filter` for type-specific searches: `"type:=concept"`, `"type:=entity"`
- Read the most relevant pages with `gmd_wiki_get` before synthesizing

## After Ingest

After writing or updating any wiki pages:
1. Run `gmd_wiki_update` to re-index (or rely on watch mode if enabled)
2. Consider running `gmd_wiki_lint` periodically to surface issues

## Wikilink Suggestions

When writing new pages, use `gmd_wiki_suggest` to find pages to link to.
Search for existing entities/concepts before creating duplicate pages.
```

### 14i. Priorities & Dependencies

| Priority | Task | Depends On |
|---|---|---|
| 1 | `gmd wiki init` CLI command | Nothing — filesystem + CUE AST, standalone |
| 2 | Embed skill templates (`pkg/wiki/skills/`) | Nothing — `//go:embed` markdown files |
| 3 | `gmd wiki skills write` CLI | Skill templates + filesystem paths |
| 4 | Wikilink extraction in chunker | Nothing — pure Go regex in `pkg/chunking` |
| 5 | Built-in agent: `gmd wiki ingest` | LLM client (already exists), wikilink extraction, chunker |
| 6 | Built-in agent: `gmd wiki query` | Search pipeline (already exists), LLM client |
| 7 | `gmd wiki graph` CLI command | Wikilink extraction (link edges in Typesense) |
| 8 | `gmd wiki watch` (auto-ingest + auto-index) | Built-in agent ingest + `fsnotify` |
| 9 | MCP tools (`gmd_wiki_*`) | Phase 6 MCP server + wiki agent internals |
| 10 | Frontmatter extraction (chunker) | CUE schema extension for `frontmatter` config |
| 11 | `gmd wiki lint` (full LLM health check) | Wikilink extraction + LLM client |
| 12 | `gmd wiki doctor` (agent config check) | Skill templates + agent discovery path detection |
| 13 | `gmd wiki export` | Frontmatter + graph + MCP tools |
| 14 | Multi-source batch ingest / interactive mode | Built-in agent core |

**Quick wins (no dependencies):** `wiki init`, embedded skill templates, `wiki skills write`, wikilink extraction. These four items can be implemented immediately and together provide a complete bootstrap experience (scaffold wiki + install skills + enable search).

**Next wave:** Built-in agent (`ingest`/`query`) makes GMD a self-contained wiki operator using its existing LLM client. Watch mode ties it together for hands-off operation.

**External agent support:** MCP tools unlock Path B (external agents), and the skill templates ensure any agent can be a capable wiki maintainer out of the box.

### 14j. Built-in Agent Design

The built-in agent (`pkg/wiki/agent.go`) is the core of Path A — GMD operating the wiki directly using its own LLM client. It is NOT a generic agent framework; it is purpose-built for the three wiki operations.

#### Agent Type

```go
type Agent struct {
    wikiName    string
    wikiPath    string          // path to wiki/ directory
    rawPath     string          // path to raw/ directory
    cfg         *config.Config
    tsClient    *ts.Client
    llmClient   *llm.Client
    schema      string          // WIKI_SCHEMA.md content (loaded at init)
    indexCache  map[string]string // page path → one-line summary
}
```

#### Ingest Pipeline

```
func (a *Agent) Ingest(sourcePath string, opts IngestOpts) (*IngestReport, error)
    opts: Batch bool, Interactive bool
```

1. **Read source** — read file from `rawPath / sourcePath` (markdown, text). If URL, HTTP fetch first, save to raw/.
2. **Load context** — read `_index.md` for existing page catalog, get one-line summaries
3. **Search for overlap** — search wiki for existing pages matching source title/key entities
4. **Build prompt** — system: WIKI_SCHEMA.md, context: index summaries + existing pages, user: source content
5. **Call LLM** — chat completion with structured output format (JSON)
6. **Parse response** — unmarshal LLM's JSON output into ingest actions
7. **Execute actions** — write new pages, merge/update existing pages, update _index.md, append _log.md
8. **Re-index** — call indexer to update Typesense with new/changed files
9. **Return report** — summary of what was created/updated/flagged

#### LLM Output Contract (Structured JSON)

The ingest prompt instructs the LLM to return a JSON object (not free text). This is parsed by GMD and translated into filesystem operations:

```json
{
  "source_summary": {
    "title": "Attention Is All You Need",
    "page": "sources/2026-04-attention-is-all-you-need.md",
    "frontmatter": {"type": "source", "tags": ["transformer", "nlp"], "source_url": "..."}
  },
  "entities": [
    {
      "name": "Transformer Architecture",
      "page": "entities/transformer-architecture.md",
      "action": "create",
      "content": "# Transformer Architecture\n\n...",
      "frontmatter": {"type": "entity", "tags": ["deep-learning", "architecture"]},
      "links_to": ["Self-Attention", "Multi-Head Attention"],
      "claims": ["Transformers process entire sequences in parallel"]
    },
    {
      "name": "Self-Attention",
      "page": "entities/self-attention.md",
      "action": "update",
      "merge_section": "## Scaling Properties",
      "append_content": "The original paper reports...",
      "frontmatter": {"type": "entity", "tags": ["attention"]}
    }
  ],
  "concepts": [
    {
      "name": "Scaled Dot-Product Attention",
      "page": "concepts/scaled-dot-product-attention.md",
      "action": "create",
      "content": "...",
      "links_to": ["Self-Attention", "Softmax"]
    }
  ],
  "comparisons": [
    {
      "a": "Transformer",
      "b": "RNN",
      "page": "comparisons/transformer-vs-rnn.md",
      "action": "update",
      "new_dimension": "Training efficiency: Transformers enable..."
    }
  ],
  "contradictions": [
    {
      "claim": "Transformers require more data than RNNs",
      "source_page": "sources/2026-04-attention-is-all-you-need.md",
      "contradicts_page": "concepts/transformer-data-efficiency.md",
      "existing_claim": "Transformers are more data-efficient than RNNs",
      "resolution_hint": "Check dataset sizes — paper uses large corpus, existing claim may reference small-data regime"
    }
  ],
  "index_updates": [
    {"page": "entities/transformer-architecture.md", "summary": "Core architecture replacing recurrence with self-attention", "category": "entities"},
    {"page": "concepts/scaled-dot-product-attention.md", "summary": "Attention score computation: Q·K^T / √d_k · V", "category": "concepts"}
  ],
  "log_entry": "## [2026-05-31] ingest | Attention Is All You Need\n- Created: entities/transformer-architecture.md, concepts/scaled-dot-product-attention.md\n- Updated: entities/self-attention.md, comparisons/transformer-vs-rnn.md\n- Flagged contradiction: transformer data efficiency vs existing claim"
}
```

The structured output ensures GMD is doing deterministic filesystem operations, not executing arbitrary LLM-generated code. The LLM provides content; GMD handles all I/O.

#### Query Pipeline

```
func (a *Agent) Query(question string, opts QueryOpts) (*QueryResult, error)
    opts: Save bool, Limit int
```

1. **Search** — call GMD's hybrid search pipeline over wiki collection, return top-N pages
2. **Read pages** — read full content of top-N wiki pages
3. **Build prompt** — system: WIKI_SCHEMA.md, user: question + page contents with `[[page]]` citations
4. **Call LLM** — chat completion
5. **Optionally save** — if `--save`, write answer to `wiki/synthesis/YYYY-MM-DD-question-slug.md`
6. **Return** — answer text + list of cited source pages

#### Watch Loop

```
func (a *Agent) Watch() error
```

Runs indefinitely:
1. Add fsnotify watcher on `raw/` directory
2. On new file created in raw/: debounce 500ms, then call `a.Ingest(filename, IngestOpts{Batch: false})`
3. Add fsnotify watcher on `wiki/` directory
4. On file change in wiki/: debounce 500ms, then call indexer for single-file incremental index
5. On file delete in wiki/: delete from Typesense
6. Print status updates: "Ingested paper.md → +3 pages, ~2", "Re-indexed wiki/concepts/foo.md"

### 14k. Embedded Skills & Agent Discovery Design

GMD ships with embedded skill templates that teach any LLM agent how to operate the wiki. These are the "schema" layer from Karpathy's pattern, but packaged as installable skill files for specific agent platforms.

#### Skill Template Structure

Each skill file is a markdown document following the conventions of the target agent platform, plus GMD-specific wiki instructions:

```markdown
# GMD Wiki Operator

## Description
Operate a Karpathy-style LLM Wiki using GMD's search and indexing infrastructure.
Maintains a compounding knowledge base: ingest sources, query the wiki,
lint for health, and export results.

## Required Tools
- MCP: gmd_wiki_search, gmd_wiki_get, gmd_wiki_neighbors, gmd_wiki_update,
       gmd_wiki_ingest, gmd_wiki_suggest, gmd_wiki_status
- Filesystem: read_file, write_file (for wiki page authoring)

## Ingest Workflow
When user provides a source to ingest:
1. Read the source file (or fetch URL, save to raw/)
2. Call gmd_wiki_search to find existing pages that overlap
3. Read related wiki pages for context
4. Extract entities, concepts, claims, contradictions
5. Write/update wiki pages (entities/, concepts/, comparisons/, sources/)
6. Update _index.md with new/updated page entries
7. Append entry to _log.md
8. Call gmd_wiki_update to re-index
9. Report summary to user

## Query Workflow
When user asks a question:
1. Call gmd_wiki_search with the question
2. Read top matching wiki pages with gmd_wiki_get
3. Synthesize answer with citations ([[page]] links)
4. Offer to save answer to wiki/synthesis/

## Page Templates
... (entity page, concept page, comparison page, source summary formats)

## Frontmatter Conventions
... (type, tags, sources, status, difficulty fields)

## Lint & Maintenance
... (when to run gmd_wiki_lint, what to check, how to fix issues)
```

#### Agent Discovery Path Mapping

```go
var agentPaths = map[string]string{
    "claude":   filepath.Join(home, ".claude", "skills", "gmd-wiki.md"),
    "codex":    filepath.Join(cwd, ".agents", "skills", "gmd-wiki"),
    "opencode": filepath.Join(home, ".config", "opencode", "skills", "gmd-wiki.md"),
    "cursor":   filepath.Join(cwd, ".cursor", "skills", "gmd-wiki.md"),
}
```

`gmd wiki skills write --target all` writes to all detected agent paths. `gmd wiki skills write --target claude` writes only the Claude Code skill. The skill file for each target is customized with platform-specific tool names and conventions.

#### Auto-Discovery in `wiki doctor`

`gmd wiki doctor` also detects which agents are installed and whether their skills/MCP configs are set up. It reports missing pieces and can auto-configure with `--fix`:

```
$ gmd wiki doctor
  Wiki: myresearch (12 pages, 3 sources)
  Typesense: ✓
  LLM: ✓ embedding ✓ expansion ✓ rerank
  Agents detected:
    Claude Code:  ✓ installed, ✓ skill, ✗ MCP → gmd wiki doctor --fix
    OpenCode:     ✓ installed, ✗ skill, ✗ MCP → gmd wiki doctor --fix
  Run: gmd wiki doctor --fix to auto-configure all agents
```

#### MCP Auto-Configuration

When `--fix` writes MCP config, it generates the appropriate JSON/YAML snippet for each agent:

**Claude Code** (`.claude/settings.json`):
```json
{
  "mcpServers": {
    "gmd-wiki": {
      "type": "local",
      "command": ["gmd", "mcp", "--wiki", "myresearch"],
      "enabled": true
    }
  }
}
```

**OpenCode** (`opencode.jsonc`):
```jsonc
{
  "mcp": {
    "gmd-wiki": {
      "type": "local",
      "command": ["gmd", "mcp", "--wiki", "myresearch"],
      "enabled": true
    }
  }
}
```
