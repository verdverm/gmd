# gmd — markdown search engine

**gmd** indexes local markdown files and lets you search them with full-text, vector, or hybrid
search. Built in Go, backed by [Typesense](https://typesense.org), powered by any
OpenAI-compatible LLM.

```
gmd init                        # scaffold .gmd/config.cue
gmd update                      # index your markdown files
gmd query "how do I deploy?"    # full hybrid search
gmd search "error X"            # fast text-only search
gmd status                      # see what's indexed
gmd agentsmd summary            # get AGENTS.md for AI assistants
```

## Features

- **Full hybrid search pipeline** — `gmd search` (text), `gmd vsearch` (vector), and
  `gmd query` (expansion + hybrid + RRF fusion + LLM reranking + position blending)
- **Global + project collections** — define collections globally (`~/.config/gmd/`) or per-project
  (`.gmd/config.cue`); merge is automatic, project values take precedence
- **Multi-collection projects** — index separate doc sets (e.g. user guide, dev API) as distinct
  collections within the same project, search across all or target one
- **Web search, fetch, crawl, research** — `gmd web` subcommands powered by the EXA API
- **LLM Wiki** — compounding Karpathy-style knowledge base with built-in agent for ingest,
    search-powered query, graph, and lint
- **MCP + REST API** — wiki-aware MCP tools for AI agents (`gmd mcp`); HTTP endpoints for search,
  status, and indexing (`gmd serve`) — see [docs/rest-api.md](docs/rest-api.md)
- **agentsmd** — output AGENTS.md instructions for AI assistants working with gmd

## How it works

**Search** (`gmd query`) expands your query via LLM, searches Typesense with text and vector
similarity, fuses results with RRF, optionally reranks with an LLM, and blends by chunk position.
You get the most relevant chunks ranked intelligently — no manual query tuning needed.
See [docs/search-pipeline.md](docs/search-pipeline.md) for the full pipeline diagram.

**Wiki** (`gmd wiki`) maintains a compounding knowledge base of interlinked markdown pages. The
built-in LLM agent reads source documents, extracts entities and claims, writes or updates wiki
pages, and links them via `[[wikilinks]]`. Querying the wiki uses the same Typesense-backed search pipeline: retrieve relevant pages,
synthesize an answer with inline `[[citations]]`.

**Web** (`gmd web`) lets you search the web, fetch clean content from URLs, or run a multi-step
LLM-orchestrated research agent that searches, reads, and synthesizes across multiple rounds — all
powered by the EXA API.

**Deploy** (`gmd serve` / `gmd mcp`) exposes gmd over HTTP and/or MCP so AI coding assistants and
other tools can search your docs, query wikis, and browse indexed content.

## Index and search

```bash
gmd init                        # scaffold .gmd/config.cue
gmd collection add docs --path ./guide --patterns "**/*.md"
gmd collection include docs "**/*.md" "**/*.mdx"
gmd collection exclude docs "node_modules/**"
gmd collection list             # see what's configured
gmd collection show docs        # details + chunk count
gmd collection remove docs      # delete collection + indexed chunks
gmd collection rename docs api  # rename a collection

gmd update                      # scan, chunk, embed, index
gmd status                      # verify docs are indexed
gmd ls docs                     # list indexed documents

gmd search "keyword"            # text-only keyword search
gmd vsearch "concept"           # vector similarity search
gmd query "your question"       # full hybrid search pipeline

gmd get docs/guide.md           # fetch document by path
gmd cleanup                     # remove stale chunks for deleted files
gmd doctor                      # diagnostics
```

Run `gmd query` from within a project directory and the matching collection is selected
automatically. Use `-c` to target specific collections.

## Web search, fetch, crawl, research

Powered by the [EXA](https://exa.ai) API (web search, content fetch, crawling). Requires `EXA_API_KEY`.

```bash
# Semantic web search
gmd web search "transformer architecture"
gmd web search "golang generics" --type deep --limit 5 --text
gmd web search "kubernetes" --domain kubernetes.io --date-start 2026-01-01

# Fetch clean content from URLs
gmd web fetch https://example.com/article
gmd web fetch https://a.com https://b.com --max-chars 2000
gmd web fetch https://example.com --output file -o ./fetched/

# Multi-step LLM-orchestrated research agent
gmd web agent "compare Nuxt 4 vs Next.js 16" --depth deep --save
gmd web agent "latest Go 1.24 developments" --steps 5 --text
```

The agent mode chains multiple searches: the LLM analyzes results, decides what to search next, and
synthesizes a final answer. Use `--save` to persist results to a wiki.

## LLM Wiki

gmd can maintain a Karpathy-style compounding knowledge base. The built-in LLM agent
reads sources, extracts entities/concepts/claims, writes interlinked wiki pages, and keeps
everything searchable.

```bash
gmd wiki init --name myresearch      # scaffold wiki directory + config
gmd wiki ingest paper.md             # LLM reads paper, creates/updates wiki pages
gmd wiki query "what is..."          # search → LLM synthesis with [[citations]]
gmd wiki lint                        # health checks (orphans, broken links)
gmd wiki doctor                      # diagnostics + auto-configure agent MCP
gmd wiki graph                       # export wikilink graph (dot/mermaid/json)
gmd wiki skills write --target all   # install skill templates for AI agents
```

See wiki skill templates with `gmd wiki skills list` and `WIKI_SCHEMA.md` for conventions.

## Requirements

- **Typesense** — must be running (Docker, Kubernetes, or cloud)
- **OpenAI-compatible LLM API** — vLLM, Ollama, or any provider via `base_url`. Seven model roles
  (embedding, expansion, rerank, summarizing, general-big/mid/small). See
  [`models/`](models/) for vLLM serve scripts.
- **EXA API key** — required only for `gmd web` commands
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
  project: "myapp"
  collections: {
    userdocs: {
      path:    "docs/user-guide"
      pattern: "**/*.md"
      context: "MyApp user documentation"
    }
    devdocs: {
      path:    "docs/dev-api"
      pattern: "**/*.{md,mdx}"
      ignore:  ["_drafts/**"]
      context: "MyApp developer API reference"
    }
  }
  llm: {
    embedding_model:      "google/embeddinggemma-300m"
    embedding_base_url:   "http://localhost:8001/v1"
    expansion_model:      "Qwen/Qwen3-1.7B"
    expansion_base_url:   "http://localhost:8002/v1"
    rerank_model:         "Qwen/Qwen3-Reranker-0.6B"
    rerank_base_url:      "http://localhost:8003/v1"
    summarizing_model:    "Qwen/Qwen3.6-27B-FP8"
    summarizing_base_url: "http://localhost:8000/v1"
  }
  typesense: {
    host: "http://localhost:8108"
  }
}
```

**API keys.** Set `OPENAI_API_KEY` (default for all LLM roles) and
`GMD_TYPESENSE_API_KEY` for Typesense. Per-role overrides exist if needed
(`GMD_EMBEDDING_API_KEY`, `GMD_EXPANSION_API_KEY`, etc.) — see
[docs/configuration.md](docs/configuration.md) for the full reference.

**Global config.** Put shared LLM and Typesense settings in `~/.config/gmd/config.cue`.
Project and global configs merge automatically, with project values taking precedence.

## Commands

| Command | Description |
|---|---|
| `gmd init` | Create `.gmd/config.cue` in the current directory |
| `gmd agentsmd [name]` | Output AGENTS.md content for AI coding assistants |
| `gmd update` | Index or re-index all collections (scan, chunk, embed, upsert) |
| `gmd embed` | Re-embed all documents (when the embedding model changes) |
| `gmd status` | Show index health and per-collection counts |
| `gmd search <query>` | Text-only keyword search |
| `gmd vsearch <query>` | Vector similarity search |
| `gmd query <query>` | Full pipeline: expansion → hybrid → RRF → rerank → blend |
| `gmd get <path>` | Get document content by path |
| `gmd ls [collection]` | List indexed documents |
| `gmd collection list` | List collections |
| `gmd collection add <name>` | Add a collection with --path and --pattern |
| `gmd collection show <name>` | Collection details + chunk count |
| `gmd collection include <name> <patterns>` | Set file-matching patterns for a collection |
| `gmd collection exclude <name> <pattern>` | Add an ignore pattern to a collection |
| `gmd collection rename <old> <new>` | Rename a collection |
| `gmd collection remove <name>` | Remove a collection |
| `gmd doctor` | Run diagnostics |
| `gmd cleanup` | Remove stale chunks for deleted files |
| `gmd context add <collection> "text"` | Set context for a collection |
| `gmd context list` | List all collection contexts |
| `gmd context rm <collection>` | Remove context from a collection |
| `gmd serve` | Start REST API server |
| `gmd mcp` | Start MCP server (for AI agent integration) |
| `gmd web search <query>` | Semantic web search via EXA |
| `gmd web fetch <url>` | Fetch clean content from URLs via EXA |
| `gmd web agent <query>` | Multi-step LLM-orchestrated web research agent |
| `gmd wiki init` | Scaffold wiki directory + CUE config |
| `gmd wiki ingest <src>` | Ingest a source into the wiki using built-in LLM agent |
| `gmd wiki query "..."` | Query the wiki with search + LLM synthesis |
| `gmd wiki graph` | Export wikilink graph (dot/mermaid/json) |
| `gmd wiki lint` | Health checks (orphans, broken links, contradictions) |
| `gmd wiki doctor` | Diagnostics + auto-configure agent MCP |
| `gmd wiki skills` | Manage embedded agent skill templates |

## Development

See [docs/development.md](docs/development.md) for build, test, and contribution instructions.

This project is developed with OpenCode + DeepSeek-v4-flash/pro via [OpenCode Go](https://opencode.ai/go?ref=73R104W9KX).
