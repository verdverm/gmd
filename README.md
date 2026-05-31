# gmd — markdown search engine

**gmd** indexes local markdown files and lets you search them with full-text, vector, or hybrid search. Built in Go, backed by [Typesense](https://typesense.org), powered by any OpenAI-compatible LLM.

```
gmd update                     # index your markdown files
gmd query "how do I deploy?"   # full hybrid search
gmd search "error X"           # fast text-only search
gmd status                     # see what's indexed
```

## Requirements

- **Typesense** — must be running (Docker, Kubernetes, or cloud)
- **Three LLM models** — must be served by an OpenAI-compatible API (vLLM, Ollama, OpenAI, etc.):

  | Model | Purpose | Default |
  |---|---|---|
  | embedding | Converts document chunks into vector embeddings for similarity search | `google/embeddinggemma-300m` |
  | expansion | Generates query variants (lexical, vector, HyDE) to improve recall | `Qwen/Qwen3-1.7B` |
  | rerank | Re-scores search results for relevance | `Qwen/Qwen3-Reranker-0.6B` |

  See [`models/`](models/) for vLLM serve scripts and systemd service files.

- **Go 1.25+** — to build from source

## Quick start

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
  collections: myapp: {
    path:    "docs"
    pattern: "**/*.{md,mdx}"
    context: "MyApp user documentation"
  }
  llm: {
    embedding_base_url:  "http://localhost:8001/v1"
    expansion_base_url:  "http://localhost:8002/v1"
    rerank_base_url:     "http://localhost:8003/v1"
    api_key:             ""
    embedding_model:     "google/embeddinggemma-300m"
    expansion_model:     "Qwen/Qwen3-1.7B"
    rerank_model:        "Qwen/Qwen3-Reranker-0.6B"
  }
  typesense: {
    host:    "http://localhost:8108"
    api_key: "xyz"
  }
}
```

If `api_key` is empty, gmd reads `OPENAI_API_KEY` from the environment.

For shared settings across projects, create `~/.config/gmd/config.cue` with the same structure — project and global configs are merged automatically, with project values taking precedence.

See [docs/configuration.md](docs/configuration.md) for the full config reference.

### 4. Index and search

```bash
gmd update              # scan, chunk, embed, index
gmd status              # verify docs are indexed
gmd query "your question here"    # full hybrid search
```

Run `gmd query` from within `myproject/docs/` and the `myapp` collection is selected automatically.

## Commands

| Command | Description |
|---|---|
| `gmd update` | Index or re-index all collections (scan, chunk, embed, upsert) |
| `gmd embed` | Re-embed all documents (when the embedding model changes) |
| `gmd status` | Show index health and per-collection counts |
| `gmd search <query>` | Text-only keyword search |
| `gmd vsearch <query>` | Vector similarity search |
| `gmd query <query>` | Full pipeline: expansion → hybrid → RRF → rerank → blend |
| `gmd get <path>` | Get document content by path |
| `gmd multi-get <pattern>` | Batch fetch documents |
| `gmd collection list` | List collections |
| `gmd init` | Create `.gmd/config.cue` in the current directory |
| `gmd serve` | Start REST API server |
| `gmd mcp` | Start MCP server (for AI agent integration) |
| `gmd doctor` | Run diagnostics |
| `gmd cleanup` | Remove stale chunks for deleted files |

Search flags:

```
--collection, -c     collection(s) to search (default: auto-detect from CWD)
--limit, -n          max results (default: 5)
--format, -f         output format: cli, json (default: cli)
```

## How it works

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

### Key details

- **Chunking:** heading-aware breakpoints with configurable token target and overlap
- **Dedup:** SHA-256 hash stored on each chunk; unchanged files skip re-indexing entirely
- **Changes:** when a file changes, old chunks are deleted and new ones are indexed
- **No operational DB:** Typesense is the sole data store — filesystem is source of truth
- **Content-addressable:** changes detected by querying Typesense hash field, no local state needed

## REST API

`gmd serve` starts an HTTP server on `:8181`.

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
