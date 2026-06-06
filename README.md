# gmd - a markdown knowledge base

**gmd** indexes collections of local markdown files and lets you search them with
full-text, vector, or hybrid search - backed by [Typesense](https://typesense.org)
and any OpenAI-compatible LLM. Build compounding LLM wikis that ingest source
documents, extract knowledge, and link pages via `[[wikilinks]]`. Run web searches,
fetch clean content from URLs, or crawl sites through a multi-provider architecture
(EXA, Tavily, SearXNG, Cloudflare). Agent-orchestrated workflows, instructions,
skills, and harness configurations are exportable for use in your tools.

```
gmd init                        # scaffold .gmd/config.cue
gmd agentsmd summary            # get AGENTS.md for AI assistants

gmd collection create docs --path ./guide --pattern "**/*.md"
gmd update                      # index your markdown files

gmd query "how do I deploy?"    # full hybrid search
gmd search "error X"            # fast text-only search
gmd status                      # see what's indexed

gmd web search "topic"          # multi-provider web search
gmd wiki create <name> --from docs
gmd wiki ingest                 # ingest source into wiki
gmd wiki lint                   # health checks
```

## Features

- **Full hybrid search pipeline** - `gmd search` (text), `gmd vsearch` (vector), and
  `gmd query` (expansion + hybrid + RRF fusion + LLM reranking + position blending)
- **Global + project collections** - define collections globally (OS config dir) or per-project
  (`.gmd/config.cue`); merge is automatic, project values take precedence
- **Multi-collection projects** - index separate doc sets (e.g. user guide, dev API) as distinct
  collections within the same project, search across all or target one
- **LLM Wiki** - compounding Karpathy-style knowledge base with built-in agent for ingest,
    search-powered query, graph, and lint
- **Web search, fetch, crawl, research** - `gmd web` subcommands with multi-provider support (EXA, Cloudflare, Tavily, SearXNG). Parallel fan-out across providers with dedup and LLM synthesis.
- **MCP + REST API** - wiki-aware MCP tools for AI agents (`gmd mcp`); HTTP endpoints for search,
  status, and indexing (`gmd serve`) - see [docs/rest-api.md](docs/rest-api.md)
- **agentsmd** - output AGENTS.md instructions for AI assistants working with gmd

## How it works

**Search** (`gmd query`) expands your query via LLM, searches Typesense with text and vector
similarity, fuses results with RRF, optionally reranks with an LLM, and blends by chunk position.
You get the most relevant chunks ranked intelligently - no manual query tuning needed.
See [docs/search-pipeline.md](docs/search-pipeline.md) for the full pipeline diagram.

**Wiki** (`gmd wiki`) maintains a compounding knowledge base of interlinked markdown pages. The
built-in LLM agent reads source documents, extracts entities and claims, writes or updates wiki
pages, and links them via `[[wikilinks]]`. Querying the wiki uses the same Typesense-backed search pipeline: retrieve relevant pages,
synthesize an answer with inline `[[citations]]`.

**Web** (`gmd web`) lets you search the web, fetch clean content from URLs, crawl sites, or run a
multi-step LLM-orchestrated research agent that searches, reads, and synthesizes across multiple
rounds. Backed by a multi-provider architecture (EXA, Cloudflare, Tavily, SearXNG) with configurable
provider groups. `gmd web search` fans out queries to all configured search providers in parallel,
merges and deduplicates results, and optionally synthesizes a unified cited answer via LLM.

**Deploy** (`gmd serve` / `gmd mcp`) exposes gmd over HTTP and/or MCP so AI coding assistants and
other tools can search your docs, query wikis, and browse indexed content.

## Index and search

```cue
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
```

See the [Configure](#3-configure) section for `typesense` and `llm` model settings.

```bash
gmd init                        # scaffold .gmd/config.cue
gmd collection create docs --path ./guide --patterns "**/*.md"
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

## LLM Wiki

gmd can maintain a Karpathy-style compounding knowledge base. The built-in LLM agent
reads sources, extracts entities/concepts/claims, writes interlinked wiki pages, and keeps
everything searchable.

```cue
wikis: {
  research: {
    path:    "wiki/research"
    pattern: "**/*.md"
    context: "Compounding research wiki - agent-generated notes with citations"
    sourceRefs: ["devdocs"]
  }
}
```

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

## Web search, fetch, crawl, research

Multi-provider web access with EXA, Tavily, SearXNG, and Cloudflare. Credentials come from
environment variables (env files or exported shell vars). Select which providers handle each
role (search, browser) via provider groups in CUE config. Search supports multiple providers
in parallel with automatic dedup and optional LLM synthesis.

```cue
web: {
  group: "default"
  groups: {
    default:  { search: ["searxng", "exa"], browser: "cloudflare" }
    full:     { search: ["searxng", "exa", "tavily"], browser: "cloudflare" }
    custom:   { search: ["tavily"],  browser: "cloudflare" }
    offline:  { search: ["searxng"], browser: "local" }
  }
  search: {
    dedup:      "heuristic"   // "heuristic", "llm", or "none"
    synthesize: true          // synthesize results via LLM (summarizer model)
  }
  searxng: { base_url: "http://localhost:8080" }
  local:   { no_browser: false, cache_enabled: true }
}
```

```bash
# Multi-provider web search (parallel → merge → dedup → optional LLM synthesis)
gmd web search "transformer architecture"
gmd web search "golang generics" --search-provider tavily,exa --limit 5
gmd web search "kubernetes" --domain kubernetes.io --date-start 2026-01-01
gmd web search "ai safety" --dedup llm --no-synthesize
gmd web search "climate" --synthesis-prompt ./my-prompt.txt

# Fetch clean content from URLs (--browser-provider overrides the configured default)
gmd web fetch https://example.com/article
gmd web fetch https://a.com https://b.com --max-chars 2000
gmd web fetch https://example.com --output file -o ./fetched/

# Crawl a site (Cloudflare or Local browser provider required)
gmd web crawl https://example.com/docs --depth 2 --max-pages 20
gmd web crawl https://blog.example.com --browser-provider cloudflare

# Multi-step LLM-orchestrated research agent
gmd web agent "compare Nuxt 4 vs Next.js 16" --depth deep --save
gmd web agent "latest Go 1.24 developments" --steps 5 --text
```

The agent mode chains multiple searches: the LLM analyzes results, decides what to search next, and
synthesizes a final answer. Use `--save` to persist results to a wiki.

## Requirements

- **Docker** - helpful for running Typesense and SearXNG locally and the easiest way to get started
- **Typesense** - must be running (Docker, Kubernetes, or cloud)
- **OpenAI-compatible LLM API** - vLLM, Ollama, or any provider via `base_url`. Named providers
  are mapped to roles through profiles (embedding, expansion, rerank, summarizing,
  general-big/mid/small). Supports openai, anthropic, vertex, opencode, and custom providers.
  See [`models/`](models/) for vLLM serve scripts.
- **Web provider credentials** - EXA (`EXA_API_KEY`), Tavily (`TAVILY_API_KEY`),
  Cloudflare (`CLOUDFLARE_API_KEY` + `CLOUDFLARE_ACCOUNT_ID`), or
  SearXNG (`SEARXNG_BASE_URL`, self-hosted or public instance).
  Only needed for the providers you configure.
- **Go 1.25+** - to build from source

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

### 2. Start containers

**Typesense (Docker):**

```bash
docker run -p 8108:8108 \
  -e TYPESENSE_API_KEY=xyz \
  -e TYPESENSE_DATA_DIR=/data \
  typesense/typesense:30.2
```

**SearXNG (Docker):**

```bash
docker run -p 8080:8080 \
  searxng/searxng
```

Set `SEARXNG_BASE_URL` in env or config.

**Kubernetes:** apply the manifest in `k8s/typesense.yaml`.

**Typesense Cloud:** sign up at [cloud.typesense.org](https://cloud.typesense.org).

### 3. Configure

Create a `.gmd/config.cue` in your project root (`gmd init` does this automatically):

```cue
package gmd

Config: {
  // Project name used for Typesense collection prefix
  project: "myapp"

  // Indexed markdown collections (chunked, embedded, searchable)
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

  // LLM wikis - compounding knowledge bases with agent-driven content
  wikis: {
    research: {
      path:    "wiki/research"
      pattern: "**/*.md"
      context: "Compounding research wiki - agent-generated notes with citations"
      sourceRefs: ["devdocs"]
    }
  }

  // Web search & content retrieval - multi-provider with configurable groups
  web: {
    group: "default"
    groups: {
      default:  { search: ["searxng", "exa"], browser: "cloudflare" }
      full:     { search: ["searxng", "exa", "tavily"], browser: "cloudflare" }
      custom:   { search: ["tavily"],  browser: "cloudflare" }
      offline:  { search: ["searxng"], browser: "local" }
    }
    search: {
      dedup:      "heuristic"   // "heuristic", "llm", or "none"
      synthesize: true          // synthesize results via LLM
    }
    // API keys (exa, tavily, cloudflare) are env-var-only - never in CUE
    // Non-secret settings can go here:
    searxng: { base_url: "http://localhost:8080" }
    //   local:   { no_browser: false, cache_enabled: true }
  }

  // Typesense search backend connection
  typesense: {
    host: "http://localhost:8108"
  }

  // LLM providers (named endpoints) and profiles (role→provider+model mappings)
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
      reranker: {
        provider: "openai"
        base_url: "http://localhost:8003/v1"
        auth:     "apikey"
        features: { embed: false, chat: false, rerank: true }
      }
      default: {
        provider: "openai"
        base_url: "http://localhost:8000/v1"
        auth:     "apikey"
        features: { embed: false, chat: true, rerank: false }
      }
    }
    profiles: {
      default: {
        embedding:    { provider: "embedder", model: "google/embeddinggemma-300m" }
        expansion:    { provider: "small",    model: "Qwen/Qwen3-1.7B" }
        rerank:       { provider: "reranker", model: "Qwen/Qwen3-Reranker-0.6B" }
        summarizing:  { provider: "default" }
        general_big:  { provider: "default" }
        general_mid:  { provider: "default" }
        general_small:{ provider: "default" }
      }
    }
  }
}
```

**API keys.** LLM provider auth is resolved by provider type: `OPENAI_API_KEY` (openai),
`ANTHROPIC_API_KEY` (anthropic), `OPENCODE_API_KEY` (opencode), or `GMD_LLM_API_KEY`
(custom providers). Set `GMD_TYPESENSE_API_KEY` for Typesense. Providers can use `auth: "none"`
for local servers with no key. See [docs/configuration.md](docs/configuration.md) for the
full reference.
For `gmd web` commands, set `EXA_API_KEY`, `TAVILY_API_KEY`, `CLOUDFLARE_API_KEY`,
`CLOUDFLARE_ACCOUNT_ID`, and/or `SEARXNG_BASE_URL` depending on which providers you use
(see [docs/web-providers.md](docs/web-providers.md)).

**Env files.** Place `default.env` / `secret.env` in the global config dir (`<UserConfigDir>/gmd/`) and/or
`.gmd/` (project-local) to auto-load credentials. `default.env` is for non-sensitive defaults (can be committed),
`secret.env` is for API keys and secrets (never committed). Use `--env VAR=VAL` and `--secret VAR=VAL`
flags for inline overrides (highest precedence). Project `.gmd/secret.env` is git-ignored. See
[docs/configuration.md#environment-files](docs/configuration.md#environment-files) for full precedence
and file format.

**Global config.** Put shared LLM and Typesense settings in the global config file
(`<UserConfigDir>/gmd/config.cue` - `~/.config/gmd/config.cue` on Linux, `~/Library/Application Support/gmd/config.cue` on macOS).
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
| `gmd collection create <name>` | Create a collection with --path and --pattern |
| `gmd collection show <name>` | Collection details + chunk count |
| `gmd collection include <name> <patterns...>` | Add file-matching patterns to a collection (append, or --replace-all) |
| `gmd collection exclude <name> <patterns...>` | Add ignore patterns to a collection (append, or --replace-all) |
| `gmd collection rename <old> <new>` | Rename a collection |
| `gmd collection remove <name>` | Remove a collection |
| `gmd doctor` | Run diagnostics |
| `gmd env` | Print resolved config with secrets masked |
| `gmd cleanup` | Remove stale chunks for deleted files |
| `gmd llm status` | Health check all LLM providers and roles |
| `gmd llm providers` | List configured LLM providers |
| `gmd llm profiles` | List configured LLM profiles |
| `gmd llm show <name>` | Show role→provider mappings for a profile |
| `gmd llm test <provider>` | Quick chat test against a provider |
| `gmd context add <collection> "text"` | Set context for a collection |
| `gmd context list` | List all collection contexts |
| `gmd context rm <collection>` | Remove context from a collection |
| `gmd serve` | Start REST API server |
| `gmd mcp` | Start MCP server (for AI agent integration) |
| `gmd web search <query>` | Multi-provider web search with parallel fan-out, dedup, and optional LLM synthesis |
| `gmd web fetch <url>` | Fetch clean content from URLs via configured browser provider |
| `gmd web crawl <url>` | Crawl a site from seed URL via configured browser provider |
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
