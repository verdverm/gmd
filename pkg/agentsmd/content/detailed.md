# GMD — Markdown Search Engine (Detailed Reference)

GMD indexes local markdown files and provides full-text, vector, and hybrid search. Built in Go, backed by Typesense, powered by any OpenAI-compatible LLM API.

## Requirements

- **Typesense** — must be running (Docker, Kubernetes, or cloud), default `http://localhost:8108`
- **Three LLM models** served by an OpenAI-compatible API:

  | Model | Purpose | Default |
  |---|---|---|
  | embedding | Converts document chunks into vector embeddings | `google/embeddinggemma-300m` |
  | expansion | Generates query variants (lexical, vector, HyDE) | `Qwen/Qwen3-1.7B` |
  | rerank | Re-scores search results for relevance | `Qwen/Qwen3-Reranker-0.6B` |

- API keys: `OPENAI_API_KEY` for LLM endpoints (per-role overrides: `GMD_EMBEDDING_API_KEY`, `GMD_EXPANSION_API_KEY`, `GMD_RERANK_API_KEY`, `GMD_SUMMARIZING_API_KEY`, `GMD_GENERAL_BIG_API_KEY`, `GMD_GENERAL_MID_API_KEY`, `GMD_GENERAL_SMALL_API_KEY`), `GMD_TYPESENSE_API_KEY` for Typesense (both read from environment)

## Quick Start

```bash
# Install
go install github.com/verdverm/gmd/cmd/gmd@latest

# Start Typesense (Docker)
docker run -p 8108:8108 -e TYPESENSE_API_KEY=xyz -e TYPESENSE_DATA_DIR=/data typesense/typesense:30.2

# Start LLM models (vLLM example)
# See models/vllm/ and models/systemd/ for serve scripts

# Initialize a project
gmd init     # creates .gmd/config.cue

# Edit .gmd/config.cue to set your endpoints, then:
gmd update   # index all collections
gmd query "your question"   # search
```

## Configuration

Config is in CUE format. Three layers, merged at load time (later layers override earlier):

1. **Embedded schema** in the binary — provides all defaults
2. **Global config** at `~/.config/gmd/config.cue` — shared across projects
3. **Project config** at `<project-root>/.gmd/config.cue` — project-specific settings

Project root is detected by walking up from CWD looking for a `.gmd/` directory.

### Example `.gmd/config.cue`

```cue
package gmd

Config: {
  project:  "myapp"
  llm: {
    embedding_base_url:  "http://localhost:8001/v1"
    expansion_base_url:  "http://localhost:8002/v1"
    rerank_base_url:     "http://localhost:8003/v1"
    embedding_model:     "google/embeddinggemma-300m"
    expansion_model:     "Qwen/Qwen3-1.7B"
    rerank_model:        "Qwen/Qwen3-Reranker-0.6B"
  }
  typesense: {
    host:    "http://localhost:8108"
  }
  collections: docs: {
    path:    "."
    pattern: "**/*.md"
    context: "Project documentation"
  }
}
```

### Pipeline Settings (with defaults)

```cue
pipeline: {
  chunk: { targetTokens: 900, overlap: 0.15 }
  strongSignal: { minScore: 0.85, minGap: 0.15 }
  rrf: { k: 60, originalWeight: 2.0, expansionWeight: 1.0 }
  rerank: { candidateLimit: 40, contextSize: 4096 }
  blending: { thresholds: { top: 3, middle: 10 }, weights: { top: 0.75, middle: 0.60, bottom: 0.40 } }
  output: { defaultFormat: "cli", maxResults: 5 }
}
```

## Complete Command Reference

### Indexing & Status

| Command | Description |
|---|---|
| `gmd init` | Create `.gmd/config.cue` in current directory |
| `gmd update` | Scan, chunk, embed, and index all collections |
| `gmd embed` | Re-embed all documents (use when embedding model changes) |
| `gmd status` | Show index health and per-collection document counts |
| `gmd cleanup` | Remove stale entries for files that have been deleted (both chunks and documents) |

### Search

| Command | Description |
|---|---|
| `gmd query <query>` | Full pipeline: expansion → hybrid → RRF → rerank → blend |
| `gmd search <query>` | Text-only keyword search (no LLM) |
| `gmd vsearch <query>` | Vector similarity search (no LLM) |

Search flags:
- `-c, --collection` — collection(s) to search (default: auto-detect from CWD)
- `-n, --limit` — max results (default: 5)
- `-f, --format` — output format: `cli` (default) or `json`

### Document Retrieval

| Command | Description |
|---|---|
| `gmd get <path>` | Get full document content by relative path |
| `gmd multi-get <pattern>` | Batch fetch documents matching a glob pattern |
| `gmd ls [collection]` | List all indexed documents (paths and chunk counts) |

### Collections

| Command | Description |
|---|---|
| `gmd collection list` | List all configured collections |
| `gmd collection show <name>` | Show collection details and chunk count |
| `gmd collection create <name> --path <dir> --pattern <glob>` | Create a new collection in config |
| `gmd collection remove <name>` | Remove a collection from config |
| `gmd collection rename <old> <new>` | Rename a collection |
| `gmd collection include <name> <patterns...>` | Add file-matching patterns (append, or --replace-all) |
| `gmd collection exclude <name> <patterns...>` | Add ignore patterns (append, or --replace-all) |

### Context Documents

| Command | Description |
|---|---|
| `gmd context add <collection> "text"` | Add a context document to a collection |
| `gmd context list` | List all context documents |
| `gmd context rm <collection>` | Remove context documents for a collection |

### Servers & Diagnostics

| Command | Description |
|---|---|
| `gmd serve [--port] [--host]` | Start REST API server (default: `:8181`) |
| `gmd mcp [--http]` | Start MCP server for AI agent integration |
| `gmd doctor` | Run system diagnostics |
| `gmd agentsmd [oneline|summary|detailed|full]` | Output AGENTS.md content for AI coding assistants |

### Web Search

Three-tier spectrum for searching the live web:

| Tier | Command | Description |
|---|---|---|
| 1 | `gmd web search <query>` | Traditional web search (no LLM) |
| 1 | `gmd web fetch <url> [url2 ...]` | Clean content extraction from URLs |
| 1 | `gmd web crawl <url>` | Discover + fetch linked pages (stub) |
| 2 | `gmd web agent <query>` | Multi-step LLM-orchestrated research agent |
| 3 | `gmd web research <query>` | Deep structured research pipeline (stub) |

Requires `EXA_API_KEY` (or other provider credentials). See `gmd web --help`.

### LLM Wiki

| Command | Description |
|---|---|
| `gmd wiki init [--name] [--path]` | Scaffold wiki directory structure + CUE config entry |
| `gmd wiki ingest <source>` | LLM reads source, extracts entities/concepts/claims, writes wiki pages |
| `gmd wiki query "<question>" [--save]` | RAG search over wiki → LLM synthesis with [[page]] citations |
| `gmd wiki graph [--format]` | Export wikilink graph as dot, mermaid, or JSON |
| `gmd wiki lint` | Structure checks (orphans, broken links) + LLM content analysis |
| `gmd wiki skills [list|show|write]` | Manage embedded skill templates for AI agents |
| `gmd wiki doctor [--fix]` | Diagnostics + auto-configure MCP servers for detected agents |

### REST API Endpoints (`gmd serve`)

| Endpoint | Method | Description |
|---|---|---|
| `/health` | GET | Liveness check |
| `/status` | GET | Index and collection health |
| `/search` | POST | Full-text search |
| `/vsearch` | POST | Vector search |
| `/query` | POST | Full hybrid pipeline |
| `/documents/{path}` | GET | Get document by path |
| `/collections` | GET | List collections |
| `/update` | POST | Trigger reindex |

## Search Pipeline Detail

```
gmd query "deploy config"
       │
       ▼
Strong signal check ──── if score ≥ 0.85 and gap ≥ 0.15 ────► use query directly
       │
       ▼
LLM query expansion ──── generates lex / vec / hyde variants
       │
       ▼
For each variant ──────── embed → Typesense hybrid search (text + vector, grouped by doc)
       │
       ▼
RRF fusion ────────────── Σ(w / (k + rank)) across all variants
       │
       ▼
LLM reranking ─────────── /v1/rerank endpoint (skipped if unsupported)
       │
       ▼
Position blending ─────── top/middle/bottom tiers with configurable weights
       │
       ▼
Results
```

## Architecture

```
cmd/gmd/          CLI entry point (cobra commands)
pkg/config/       CUE config loading, validation, project detection, CUE AST editing
pkg/chunking/     Markdown chunker (heading-aware breakpoints)
pkg/indexer/      File scanning + SHA-256 dedup + chunk → embed → upsert + full doc storage
pkg/search/       Search pipeline: signal detection, expansion, RRF fusion, rerank, blend
pkg/ts/           Typesense client wrapper (chunks + documents collections, search, CRUD)
pkg/llm/          OpenAI-compatible API client (embeddings, chat, rerank)
pkg/output/       Result formatting (CLI, JSON)
pkg/runtime/      Runtime struct — owns Typesense client lifecycle
pkg/wiki/         LLM Wiki: scaffold, built-in agent, graph, lint, skills
pkg/mcp/          MCP server tools (wiki-aware tools)
pkg/web/          Web providers: shared interfaces, registry, agent, prompts
pkg/web/providers/ Provider implementations: exa, cloudflare, local, tavily, searxng
```

## Key Design Decisions

- **No operational DB.** Typesense is the sole data store. Filesystem is source of truth.
- **Content-addressable dedup.** SHA-256 hash stored on every chunk and full document; unchanged files skip re-processing.
- **External embeddings.** Embeddings computed in Go via API, stored in Typesense.
- **No CGO.** `CGO_ENABLED=0` enforced. No tree-sitter, no sqlite.
- **CUE config only.** No YAML. Global + project-local CUE files unified at load time.
- **OpenAI-compatible, not OpenAI-specific.** Any provider via `base_url`. API keys via env vars.
- **Two Typesense collections.** `chunks` for vector/hybrid search (with embeddings), `documents` for full-content retrieval (`gmd get`). `group_by=collection,path` on chunks collapses to document level.

## Important Rules for AI Agents Using GMD

- **Never run `gmd update`, `gmd embed`, or `gmd collection create` automatically.** Write the command for the user to run.
- **Never modify CUE config files or the Typesense index directly** without being asked.
- Use `gmd query` for general questions — it provides the best results through the full pipeline.
- Use `gmd search` when you need fast keyword results without LLM overhead.
- Use `gmd get <path>` to retrieve full document content after finding relevant files.
- Use `gmd ls` to see what documents are currently indexed.
- If `gmd query` returns no results, check `gmd status` to verify the index is populated.
- The `-f json` flag is useful for parsing results programmatically.
- Collection auto-detection works by matching CWD against collection paths — run `gmd query` from within a collection's directory.
