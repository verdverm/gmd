# Configuration

gmd uses [CUE](https://cuelang.org) for configuration. There is no YAML fallback.

## Config loading order

Configuration is loaded in three layers and unified at runtime (later layers override earlier):

1. **Embedded schema** — built into the binary (`pkg/config/schema/*.cue`), provides all defaults
2. **Global** — `~/.config/gmd/config.cue` (optional), shared across all projects
3. **Project** — `<project-root>/.gmd/config.cue` (optional), project-specific overrides

The project root is detected by walking up from the current working directory looking for a `.gmd/` directory. Create one with `gmd init`.

## Global config

```cue
package gmd

Config: {
  project:  "my-project"         # prefix for collection keys (auto-detected by gmd init)
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
    path:    "~/documents"
    pattern: "**/*.md"
    ignore:  ["node_modules/**"]   # glob patterns to exclude
    context: "Technical documentation"
  }
}
```

API keys are read from environment variables:
- `OPENAI_API_KEY` — API key for all LLM endpoints (embedding, expansion, rerank)
- `GMD_TYPESENSE_API_KEY` — API key for typesense

If not set, gmd will fail.

## Project key

The `project` field acts as a namespace prefix for collection keys in Typesense.
A collection named `docs` in project `myapp` is stored as `myapp-docs`. This
prevents name collisions when multiple projects share a Typesense instance.

`gmd init` auto-detects the project name from the git remote URL (falling back
to the directory name). If unspecified, it defaults to the project root directory
name. The prefix is applied transparently — all CLI commands accept the original
collection name and translate it internally.

## Project config

The project-local config at `.gmd/config.cue` only needs to specify what differs
from the global config and embedded defaults. A minimal project config:

```cue
package gmd

Config: {
  collections: myapp: {
    path:    "docs"
    pattern: "**/*.md"
    context: "MyApp user documentation"
  }
}
```

## Collection fields

| Field | Type | Description |
|---|---|---|
| `path` | string | Directory path relative to project root (required) |
| `pattern` | string | Glob pattern for matching files (supports `doublestar`) |
| `ignore` | `[...string]` | Glob patterns for files to skip during indexing |
| `context` | string | Description used in query expansion prompts |
| `includeByDefault` | bool | Whether collection is searched by default (default: true) |
| `wiki` | struct | Wiki-specific settings (optional, activates wiki mode) |

### Wiki configuration

When `wiki` is present on a collection, gmd applies wiki-aware behavior:

```cue
collections: myresearch: {
  path:    "wiki"
  pattern: "wiki/**/*.md"
  ignore:  ["wiki/_index.md", "wiki/_log.md"]
  context: "AI research knowledge base"
  wiki: {
    enabled:    true
    indexFile:  "_index.md"
    logFile:    "_log.md"
    graphLinks: true
    frontmatter: {
      fields: {
        type:   { type: "string",  facet: true }
        tags:   { type: "string[]", facet: true }
        status: { type: "string",  facet: true }
      }
    }
  }
}
```

| Field | Type | Default | Description |
|---|---|---|---|
| `wiki.enabled` | bool | true | Enable wiki-aware behavior |
| `wiki.indexFile` | string | `_index.md` | Path to wiki catalog file (skipped during indexing) |
| `wiki.logFile` | string | `_log.md` | Path to chronological log file (skipped during indexing) |
| `wiki.graphLinks` | bool | true | Parse `[[wikilinks]]` for graph edges |
| `wiki.frontmatter.fields` | map | — | YAML frontmatter fields to extract as typed fields in Typesense |

## Pipeline reference

All parameters have sensible defaults — you only need to set what you want to override.

| Parameter | Default | Description |
|---|---|---|
| `llm.embedding_base_url` | — | Endpoint for embedding model (required) |
| `llm.expansion_base_url` | — | Endpoint for expansion model (required) |
| `llm.rerank_base_url` | — | Endpoint for rerank model (required) |
| `llm.embedding_model` | `google/embeddinggemma-300m` | Model for embeddings |
| `llm.expansion_model` | `Qwen/Qwen3-1.7B` | Model for query expansion |
| `llm.rerank_model` | `Qwen/Qwen3-Reranker-0.6B` | Model for reranking |
| `typesense.host` | — | Typesense server URL |
| `pipeline.chunk.targetTokens` | 900 | Target tokens per chunk |
| `pipeline.chunk.overlap` | 0.15 | Fraction overlap between chunks |
| `pipeline.strongSignal.minScore` | 0.85 | BM25 score threshold for strong signal |
| `pipeline.strongSignal.minGap` | 0.15 | Min gap between top 2 scores |
| `pipeline.rrf.k` | 60 | RRF rank scaling constant |
| `pipeline.rrf.originalWeight` | 2.0 | RRF weight for original query |
| `pipeline.rrf.expansionWeight` | 1.0 | RRF weight for expansion variants |
| `pipeline.rerank.candidateLimit` | 40 | Max docs to rerank |
| `pipeline.rerank.contextSize` | 4096 | Token budget per doc for reranking |
| `pipeline.blending.thresholds.top` | 3 | Rank cutoff for top tier |
| `pipeline.blending.thresholds.middle` | 10 | Rank cutoff for middle tier |
| `pipeline.blending.weights.top` | 0.75 | RRF weight in top tier |
| `pipeline.blending.weights.middle` | 0.60 | RRF weight in middle tier |
| `pipeline.blending.weights.bottom` | 0.40 | RRF weight in bottom tier |
| `pipeline.output.defaultFormat` | `cli` | Output format |
| `pipeline.output.maxResults` | 5 | Default result count |
