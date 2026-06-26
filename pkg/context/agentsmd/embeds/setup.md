# GMD — Setup Guide

## Requirements

- **Typesense** — must be running (Docker, Kubernetes, or cloud), default `http://localhost:8108`
- **Three LLM models** served by an OpenAI-compatible API (vLLM, Ollama, OpenAI, etc.):

  | Model | Purpose | Default |
  |---|---|---|
  | embedding | Converts document chunks into vector embeddings for similarity search | `google/embeddinggemma-300m` |
  | expansion | Generates query variants (lexical, vector, HyDE) to improve recall | `Qwen/Qwen3-1.7B` |
  | rerank | Re-scores search results for relevance | `Qwen/Qwen3-Reranker-0.6B` |

- **Go 1.25+** — to build from source
- API keys: LLM provider auth resolved by provider type — `OPENAI_API_KEY` (openai), `ANTHROPIC_API_KEY` (anthropic), `OPENCODE_API_KEY` (opencode), `GMD_LLM_API_KEY` (custom). Use `auth: "none"` for local servers with no key. `GMD_TYPESENSE_API_KEY` for Typesense (read from environment)

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
    providers: {
      embedder: {
        provider: "openai"
        base_url: "http://localhost:8001/v1"
        auth:     "apikey"
        features: { embed: true, chat: false, rerank: false }
      }
      small: {
        provider: "openai"
        base_url: "http://localhost:8002/v1"
        auth:     "apikey"
        features: { embed: false, chat: true, rerank: false }
      }
      local: {
        provider: "openai"
        base_url: "http://localhost:8003/v1"
        auth:     "none"
        features: { embed: false, chat: false, rerank: true }
      }
    }
    profiles: {
      default: {
        embedding:   { provider: "embedder", model: "google/embeddinggemma-300m" }
        expansion:   { provider: "small",    model: "Qwen/Qwen3-1.7B" }
        rerank:      { provider: "local",    model: "Qwen/Qwen3-Reranker-0.6B" }
      }
    }
  }
  typesense: {
    host:    "http://localhost:8108"
  }
}
```

For shared settings across projects, create `<UserConfigDir>/gmd/config.cue` with the same structure — project and global configs are merged automatically, with project values taking precedence.

### 4. Index and search

```bash
gmd update              # scan, chunk, embed, index
gmd status              # verify docs are indexed
gmd query "your question here"    # full hybrid search
```

Run `gmd query` from within `myproject/docs/` and the `myapp` collection is selected automatically.