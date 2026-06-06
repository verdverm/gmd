# Configuration

gmd uses [CUE](https://cuelang.org) for configuration. There is no YAML fallback.

## Config loading order

Configuration is loaded in three layers and unified at runtime (later layers override earlier):

1. **Embedded schema** — built into the binary (`pkg/config/schema/*.cue`), provides all defaults
2. **Global** — `$UserConfigDir/gmd/config.cue` (optional), shared across all projects
3. **Project** — `<project-root>/.gmd/config.cue` (optional), project-specific overrides

The project root is detected by walking up from the current working directory looking for a `.gmd/` directory. Create one with `gmd init`.

## Environment files

In addition to CUE config, gmd loads environment variables from `.env` files on every
invocation. This lets you keep credentials out of the CUE config tree.

### File locations and precedence

Files are loaded in this order (later files overwrite earlier keys):

| Precedence | File | Git-safe? |
|---|---|---|
| 1 (lowest) | `<UserConfigDir>/gmd/default.env` | yes (global) |
| 2 | `<UserConfigDir>/gmd/secret.env` | yes (global) |
| 3 | `<project>/.gmd/default.env` | yes |
| 4 | `<project>/.gmd/secret.env` | git-ignored by default |
| 5 | `--env VAR=VAL` flag | CLI only |
| 6 (highest) | `--secret VAR=VAL` flag | CLI only |

Missing files are silently skipped — none are required.

### File format

Standard `KEY=VALUE` format, one per line. Blank lines and lines starting with `#` are ignored.

```bash
# <UserConfigDir>/gmd/default.env — non-sensitive defaults (can be committed)
TYPESENSE_HOST=http://localhost:8108
# <UserConfigDir>/gmd/secret.env — shared secrets (never committed)
OPENAI_API_KEY=sk-...
GMD_TYPESENSE_API_KEY=ts-...
EXA_API_KEY=exa-...
TAVILY_API_KEY=tvly-...
CLOUDFLARE_API_KEY=cf-...
CLOUDFLARE_ACCOUNT_ID=abc123...
```

```bash
# .gmd/default.env — project-specific defaults (can be committed)
# .gmd/secret.env — project-specific overrides (git-ignored)
```

### CLI flags

```bash
# Inline overrides (highest precedence)
gmd update --env GMD_TYPESENSE_API_KEY=ts-temp --secret EXA_API_KEY=exa-rotated

# Both flags are repeatable
gmd search "query" --env FOO=bar --env BAZ=qux
```

### When it runs

Env file loading happens in `PersistentPreRunE` on the root command — before `config.Load`
reads OS environment variables. This means values from env files behave identically to
exported shell variables.

## Global config

```cue
package gmd

Config: {
  project:  "my-project"         # prefix for collection keys (auto-detected by gmd init)
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
        auth:     "none"          # local server, no API key
        features: { embed: false, chat: false, rerank: true }
      }
    }
    profiles: {
      default: {
        embedding:   { provider: "embedder", model: "google/embeddinggemma-300m" }
        expansion:   { provider: "small",    model: "Qwen/Qwen3-1.7B" }
        rerank:      { provider: "local",    model: "Qwen/Qwen3-Reranker-0.6B" }
        summarizing: { provider: "small" }
      }
    }
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

API keys are resolved per provider based on the `provider` type field:
- `openai` — reads `OPENAI_API_KEY`
- `anthropic` — reads `ANTHROPIC_API_KEY`
- `opencode` — reads `OPENCODE_API_KEY`
- `vertex` — uses GCP service-account (needs `project_id` + `location`, optional `credentials_file`)
- `custom` (or any other string) — reads `GMD_LLM_API_KEY`
- `none` auth — no API key required (local servers like vLLM)

Use `gmd env` to verify resolved config and `gmd llm status` to test connectivity.

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
| `llm.providers.<name>.provider` | — | Provider type: openai, anthropic, vertex, opencode, or custom |
| `llm.providers.<name>.base_url` | — | Endpoint URL for the provider |
| `llm.providers.<name>.auth` | `apikey` | Auth method: none, apikey, service-account |
| `llm.providers.<name>.features` | — | Feature flags: { embed, chat, rerank } |
| `llm.profile` | `default` | Active profile name |
| `llm.profiles.<name>.<role>.provider` | — | Which provider handles this role |
| `llm.profiles.<name>.<role>.model` | — | Model name for this role |
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

## Web Search Configuration

`gmd web` commands use a multi-provider architecture. Credentials come from environment
variables (env files or exported shell vars); provider selection and non-secret settings
live in CUE config. Search supports multiple providers in parallel with automatic dedup
and optional LLM synthesis.

### Credentials

| Provider | Env Vars | Purpose |
|---|---|---|
| **EXA** | `EXA_API_KEY` | Web search + cached content retrieval |
| **Tavily** | `TAVILY_API_KEY` | Web search |
| **SearXNG** | `SEARXNG_BASE_URL` | Self-hosted web search |
| **Cloudflare** | `CLOUDFLARE_API_KEY`, `CLOUDFLARE_ACCOUNT_ID` | Browser rendering + crawl |
| **Local** | none | Local browser-based fetch/crawl |

API keys (`EXA_API_KEY`, `TAVILY_API_KEY`, `CLOUDFLARE_API_KEY`) are **never** stored in CUE files —
they come from environment variables only. Non-secret values (`SEARXNG_BASE_URL`, `CLOUDFLARE_ACCOUNT_ID`)
can be set in either CUE config or environment variables.

Example `secret.env` with all provider credentials:

```bash
# <UserConfigDir>/gmd/secret.env

# Search providers
EXA_API_KEY=exa-...
TAVILY_API_KEY=tvly-...

# SearXNG — self-hosted, URL to your or a public instance
SEARXNG_BASE_URL=https://searx.example.com

# Cloudflare browser rendering
CLOUDFLARE_API_KEY=cf-...
CLOUDFLARE_ACCOUNT_ID=abc123...
```

### Provider Groups

Provider groups map a preset name to `{search, browser}` role selections. The `search` field
accepts a list of provider names — all are queried in parallel. The `group` field sets the
active group; `--provider-group` overrides per-command.

```cue
web: {
  group: "default"              // active provider group

  groups: {
    default:    { search: ["exa", "tavily"],        browser: "exa" }
    full:       { search: ["exa", "tavily", "searxng"], browser: "cloudflare" }
    custom:     { search: ["tavily"],               browser: "cloudflare" }
    offline:    { search: ["searxng"],              browser: "local" }
  }
```

### Search Behavior

Multi-provider search runs all configured providers in parallel, merges results, deduplicates
(by URL or LLM), and optionally synthesizes a unified cited answer via the summarizer LLM.
Configure defaults in `web.search`:

```cue
  search: {
    dedup:      "heuristic"     // "heuristic" (URL-based), "llm", or "none"
    synthesize: true            // synthesize results via LLM (uses summarizer model)
    synthesis_prompt: ""        // path to custom system prompt file
  }
```

CLI flags override config defaults:
- `--dedup heuristic|llm|none` — dedup strategy
- `--synthesize` / `--no-synthesize` — enable/disable LLM synthesis
- `--synthesis-prompt <path>` — custom system prompt file
- `--search-provider exa,tavily` — comma-separated provider list (overrides group)

  // Provider-specific config (optional — only needed to override defaults)
  //
  // Keys (api_key, account_id) are env-var-only and cannot be set here.
  // Non-secret fields (base_url, engines, etc.) can be set here or via env var.

  // EXA — api_key from EXA_API_KEY env var only
  exa: {}

  // Tavily — api_key from TAVILY_API_KEY env var only
  tavily: {}

  // SearXNG — base URL to your instance (or public one like https://searx.tuxcloud.net)
  // Can be set here OR via SEARXNG_BASE_URL env var.
  searxng: {
    base_url: ""                 // optional: "https://searx.example.com"
    engines:  ""                 // optional: "google,bing" (specific engines)
  }

  // Cloudflare — api_key + account_id from env vars only
  cloudflare: {}

  // Local browser — Phase 4 (not yet implemented)
  local: {
    no_browser: false            // true = only raw HTTP fetch, no headless browser
    chromium_path: ""            // custom Chromium binary path
    crawl_delay_ms: 1000         // delay between page fetches (ms)
    cache_enabled: true          // cache fetched pages to disk
    cache_dir: ""                // defaults to ~/Library/Caches/gmd/web
    cache_ttl: "24h"             // cache entry lifetime
  }
}
```

### Provider Roles

| Role | Interface | Providers |
|---|---|---|
| **search** | `SearchProvider` — query web indexes, return ranked results | `exa`, `tavily`, `searxng` |
| **browser** | `BrowserProvider` — retrieve/render content, crawl, scrape | `exa`, `cloudflare`, `local` |

EXA is the only provider registered in both roles: it implements `SearchProvider` via its
`/search` endpoint and `BrowserProvider.GetContent` via its `/contents` endpoint.

### CLI Flags

| Flag | Scope | Description |
|---|---|---|
| `--provider-group <name>` | Persistent | Override the configured active group for this command |
| `--search-provider <name,...>` | Persistent | Override search providers in the active group (comma-separated) |
| `--browser-provider <name>` | Persistent | Override only the browser role within the active group |

Priority order: individual role override → `--provider-group` → configured `group` → `"default"`.

### Examples

```bash
# Search with default provider group
gmd web search "transformer architecture"

# Search across multiple providers
gmd web search "golang generics" --search-provider exa,tavily

# Search with LLM dedup, no synthesis
gmd web search "rust features" --dedup llm --no-synthesize

# Custom synthesis prompt
gmd web search "compare frameworks" --synthesis-prompt ./my-prompt.txt

# Fetch a page (uses browser provider from active group)
gmd web fetch https://example.com

# Search via self-hosted SearXNG
gmd web search "kubernetes pods" --provider-group offline

# Override inline with env/secrets
gmd web search "AI trends" --secret TAVILY_API_KEY=tvly-temp
```

For per-provider API details and tuning, see [docs/web-providers.md](web-providers.md).

### Verifying your config

Run `gmd env` to print the fully resolved configuration (global + project CUE +
env vars) with all API keys masked as `*****`. This is useful for verifying that
credentials and provider settings are being loaded correctly.
