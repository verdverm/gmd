# GMD Architecture

This document captures the foundational architectural decisions behind GMD —
why it's built the way it is, what tradeoffs were made, and how the pieces fit
together.

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

`gmd serve` starts an HTTP server exposing all GMD operations as REST endpoints.
Shares the same `Runtime` backend as the CLI and MCP server.

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

**Request/response format**: JSON for all endpoints. Search endpoints accept the same
parameters as CLI flags (`collection`, `limit`, `min_score`, `format`, etc.) as JSON body
fields. The `query` endpoint supports both simple query strings and pre-expanded
structured queries (`lex`/`vec`/`hyde`).

**Configuration**: The `serve` subcommand accepts `--port` (default 8181), `--host`
(default localhost), and reads all other settings (CORS origins, TLS, rate limiting) from
the CUE config.

**Implementation**: Uses Go 1.22+ `net/http` with enhanced `ServeMux` for routing (no
external router dependency). Standard middleware: request logging, recovery, CORS,
optional API key authentication.

## 5. Key Dependencies (Go)

| Module | Purpose |
|---|---|
| `github.com/typesense/typesense-go` | Typesense client |
| `github.com/openai/openai-go` | OpenAI-compatible API client |
| `cuelang.org/go` | CUE config loading, validation, unification |
| `github.com/spf13/cobra` | CLI framework |
| `golang.org/x/sync` | errgroup, semaphore for parallel work |

## 6. Implementation Phases (Original Plan)

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

Auto-detection integration: `status` shows project root + matched collections,
`query`/`search`/`vsearch` auto-select collections from CWD.

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
Typesense's built-in hybrid search fuses text + vector rankings for a single query. But
GMD generates multiple expansion variants (lex/vec/hyde) and needs RRF fusion across
them. That cross-variant fusion stays in Go.

### 7b. Chunks as Typesense documents with grouping
Each chunk is a separate Typesense document. The `group_by=collection,path` parameter
collapses chunk results to document level.

### 7c. External embeddings (not Typesense auto-embedding)
Embeddings computed in Go via OpenAI-compatible API, stored in Typesense's `float[]`
field. Gives model flexibility.

### 7d. LLM reranking via the `/v1/rerank` endpoint
Reranking uses the LLM API's `/v1/rerank` endpoint (same base URL, same API key as
embeddings and chat). This is the Jina/Cohere-compatible cross-encoder rerank format
supported natively by vLLM and other OpenAI-compatible providers. It mirrors the original
QMD approach of using a dedicated reranker model via `context.rankAll()`. If the provider
does not support `/v1/rerank`, reranking is skipped.

### 7e. Content-addressable dedup via Typesense hash field
SHA-256 hash stored on every chunk document. On re-index, filter by `path`, compare
hash — if unchanged, skip the file entirely (no re-chunking, no re-embedding). Typesense
doubles as both search index and change-detection source of truth.

### 7f. CUE as the sole config language
No YAML fallback. CUE handles global + project-local config with structural sharing and
validation. The config loader:
1. Embeds built-in schema
2. Loads global `~/.config/gmd/config.cue` (optional)
3. Detects project root by walking up from CWD
4. Loads `<project-root>/.gmd/config.cue` (optional)
5. Unifies: `built-in & global & project-local`
6. Validates against schema
7. Exports validated Go struct

### 7g. Project auto-detection by sentinel walk
Walk up from CWD checking for `.gmd/` dir. Once found, that's the project root.
Collections have paths relative to project root. CWD-based collection matching uses path
prefix comparison.

### 7h. No CGO
No CGO dependencies. CI enforces `CGO_ENABLED=0`.

### 7i. OpenAI-compatible, not OpenAI-specific
The `llm.Client` abstraction wraps any OpenAI-compatible provider via `base_url` +
`api_key`.

### 7j. REST API as a first-class interface alongside CLI and MCP
`gmd serve` provides a full REST API sharing the same `Runtime` backend. Three interfaces
(CLI, REST, MCP) serve different use cases: interactive use, programmatic/scripting, and
AI agent integration. The API uses stdlib `net/http` (Go 1.22+ enhanced ServeMux) to
avoid external HTTP router dependencies.

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
| No agent docs | `gmd agentsmd` embedded content (oneline/summary/detailed/full) | AI agent onboarding without external docs |

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
