# GMD — Markdown Search Engine (Full Reference)

GMD is a local search engine for markdown files. It indexes markdown documents and provides full-text, vector, and hybrid search. Built in Go, backed by [Typesense](https://typesense.org), powered by any OpenAI-compatible LLM API.

```
gmd update                     # index your markdown files
gmd query "how do I deploy?"   # full hybrid search
gmd search "error X"           # fast text-only search
gmd status                     # see what's indexed
```

## Requirements

- **Typesense** — must be running (Docker, Kubernetes, or cloud), default `http://localhost:8108`
- **Three LLM models** served by an OpenAI-compatible API (vLLM, Ollama, OpenAI, etc.):

  | Model | Purpose | Default |
  |---|---|---|
  | embedding | Converts document chunks into vector embeddings for similarity search | `google/embeddinggemma-300m` |
  | expansion | Generates query variants (lexical, vector, HyDE) to improve recall | `Qwen/Qwen3-1.7B` |
  | rerank | Re-scores search results for relevance | `Qwen/Qwen3-Reranker-0.6B` |

- **Go 1.25+** — to build from source
- API keys: `OPENAI_API_KEY` for LLM endpoints (per-role overrides: `GMD_EMBEDDING_API_KEY`, `GMD_EXPANSION_API_KEY`, `GMD_RERANK_API_KEY`, `GMD_SUMMARIZING_API_KEY`, `GMD_GENERAL_BIG_API_KEY`, `GMD_GENERAL_MID_API_KEY`, `GMD_GENERAL_SMALL_API_KEY`), `GMD_TYPESENSE_API_KEY` for Typesense (both read from environment)

## Quick Start

### 1. Install

```bash
go install github.com/verdverm/gmd/cmd/gmd@latest
```

Or build from source:

```bash
git clone https://github.com/verdverm/gmd
cd gmd
make build
./bin/gmd
```

### 2. Start Typesense

**Option A — local (Docker):**

```bash
docker run -p 8108:8108 \
  -e TYPESENSE_API_KEY=xyz \
  -e TYPESENSE_DATA_DIR=/data \
  typesense/typesense:30.2
```

**Option B — Kubernetes:** apply the manifest in `k8s/typesense.yaml`.

**Option C — Typesense Cloud:** sign up at [cloud.typesense.org](https://cloud.typesense.org).

### 3. Configure

Create a `.gmd/config.cue` in your project root (`gmd init` does this automatically):

```cue
package gmd

Config: {
  project:  "myapp"                # auto-detected from git remote or dir name
  collections: myapp: {
    path:    "docs"
    pattern: "**/*.{md,mdx}"
    ignore:  ["node_modules/**"]    # skip these patterns
    context: "MyApp user documentation"
  }
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
}
```

For shared settings across projects, create `~/.config/gmd/config.cue` with the same structure — project and global configs are merged automatically, with project values taking precedence.

### 4. Index and search

```bash
gmd update              # scan, chunk, embed, index
gmd status              # verify docs are indexed
gmd query "your question here"    # full hybrid search
```

Run `gmd query` from within `myproject/docs/` and the `myapp` collection is selected automatically.

## Configuration Reference

Config is in CUE format. Three layers, merged at load time (later layers override earlier):

1. **Embedded schema** in the binary (`pkg/config/schema/`) — provides all defaults
2. **Global config** at `~/.config/gmd/config.cue` — shared across projects (optional)
3. **Project config** at `<project-root>/.gmd/config.cue` — project-specific settings (optional)

Project root is detected by walking up from CWD looking for a `.gmd/` directory.

### Config Structure

```cue
Config: {
  project:  string                        # project name (default: directory basename)
  llm:       #LLMConfig                    # LLM endpoint configuration
  typesense: #TypesenseConfig             # Typesense connection settings
  pipeline?: #PipelineConfig              # search pipeline tuning (optional, defaults provided)
  collections: [string]: #CollectionConfig # collection definitions
}
```

### Pipeline Settings (with defaults)

```cue
PipelineConfig: {
  chunk: {
    targetTokens:  900      # target tokens per chunk
    overlap:       0.15     # chunk overlap ratio
    headingWeights: {
      h1: 100, h2: 90, h3: 80, h4: 70, h5: 60, h6: 50
    }
    codeFenceWeight: 10
    newlineWeight:   1
  }
  strongSignal: {
    minScore: 0.85   # minimum score to treat as strong signal
    minGap:   0.15   # minimum gap to next result for strong signal
  }
  rrf: {
    k:               60     # RRF rank smoothing constant
    originalWeight:  2.0   # weight for original query results
    expansionWeight: 1.0   # weight for expansion query results
  }
  rerank: {
    candidateLimit: 40     # max candidates sent to reranker
    contextSize:    4096   # max tokens per rerank input
  }
  blending: {
    thresholds: {
      top:    3            # top N positions
      middle: 10           # top N + middle positions
    }
    weights: {
      top:    0.75         # blend weight for top tier
      middle: 0.60         # blend weight for middle tier
      bottom: 0.40         # blend weight for bottom tier
    }
  }
  output: {
    defaultFormat: "cli"   # "cli" or "json"
    maxResults:    5       # default result limit
  }
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

Wiki layout after `gmd wiki init`:
```
raw/                  Immutable source files (not indexed)
wiki/
  entities/           People, orgs, products, technologies
  concepts/           Methodologies, architectures, theories
  comparisons/        X vs Y analyses
  synthesis/          Cross-source analysis, saved answers
  sources/            Summaries of ingested content
  _index.md           Content catalog (LLM-maintained)
  _log.md             Chronological record (LLM-maintained)
WIKI_SCHEMA.md        Wiki conventions + page templates
```

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

### Pipeline Steps in Detail

1. **Strong Signal Detection:** The query is first sent as-is to Typesense hybrid search. If a result scores ≥0.85 and the gap to the next result is ≥0.15, it's a strong signal — the query is used directly without modification, and results are returned immediately.

2. **Query Expansion:** If no strong signal is detected, the query is sent to the expansion LLM which generates three variants:
   - **lex:** lexical/term-based variant for keyword matching
   - **vec:** a rephrased variant optimized for vector similarity search
   - **hyde:** a Hypothetical Document Embedding — a document-like passage of what the answer might contain

3. **Hybrid Search:** Each variant (including the original) is embedded and sent to Typesense for hybrid search (text + vector, with automatic group_by on `collection,path` to collapse chunks to document level).

4. **RRF Fusion:** Results from all variants are fused using Reciprocal Rank Fusion: `score = Σ(w / (k + rank))` where `w` is the weight and `k=60` is the smoothing constant. Original query results get 2× weight, expansion variants get 1×.

5. **Reranking:** Fused results are sent to the rerank LLM (`/v1/rerank`) which re-scores the top N candidates (default 40) against the original query. This step is skipped if the API doesn't support reranking.

6. **Position Blending:** Final scores are a blend of rerank score and positional weight:
   - Top 3 results: 75% rerank score
   - Positions 4-13 (up to middle): 60% rerank score
   - Remaining: 40% rerank score
   This ensures top positions maintain their advantage while still being influenced by rerank quality.

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
models/           vLLM serve scripts + systemd units for 3 LLM models
k8s/              Typesense Kubernetes manifest
docs/             Configuration reference
api/              Reserved for REST API (empty)
```

## Key Design Decisions

- **No operational DB.** Typesense is the sole data store. Filesystem is source of truth.
- **Content-addressable dedup.** SHA-256 hash stored on every chunk and full document; unchanged files skip re-processing.
- **External embeddings.** Embeddings computed in Go via API, stored in Typesense.
- **No CGO.** `CGO_ENABLED=0` enforced. No tree-sitter, no sqlite.
- **CUE config only.** No YAML. Global + project-local CUE files unified at load time.
- **OpenAI-compatible, not OpenAI-specific.** Any provider via `base_url`. API keys via env vars.
- **Two Typesense collections.** `chunks` for vector/hybrid search (with embeddings), `documents` for full-content retrieval (`gmd get`). `group_by=collection,path` on chunks collapses to document level.

## Markdown Chunking

GMD splits markdown files into chunks using a heading-aware algorithm:

- **Breakpoints** form at heading boundaries based on heading level — lower-level headings (h1, h2) are stronger breakpoints than higher-level ones (h5, h6).
- **Target token count** per chunk (default 900). The chunker accumulates text until approaching the target, then splits at the strongest breakpoint within range.
- **Overlap** (default 15%) ensures continuity between chunks — the last 15% of each chunk's text is prepended to the next chunk.
- **Weights** control breakpoint strength: h1=100, h2=90, h3=80, h4=70, h5=60, h6=50, code fences=10, newlines=1.

## Deduplication & Incremental Updates

Each file's SHA-256 hash is stored on every chunk and on the full document. On `gmd update`:

1. GMD scans all files matching collection patterns
2. For each file, it queries Typesense for the stored hash
3. If the file hasn't changed (hash matches), it's skipped entirely
4. If the file has changed, old chunks and document are deleted; the file is re-chunked, re-embedded, and both chunks and full document are re-indexed
5. After processing all files, stale entries (those for deleted files) are removed from both collections

`gmd get` retrieves the full document directly from the `documents` collection — no chunk reassembly needed.

## Dependencies

| Module | Purpose |
|---|---|
| `github.com/typesense/typesense-go/v4` | Typesense client |
| `github.com/openai/openai-go/v3` | OpenAI-compatible API client |
| `cuelang.org/go` | CUE config loading, validation, CUE AST editing |
| `github.com/spf13/cobra` | CLI framework |
| `github.com/bmatcuk/doublestar/v4` | Glob pattern matching for file scanning |

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
- The `gmd mcp` command provides MCP server integration for AI agents that support the Model Context Protocol.
