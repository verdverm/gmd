# Web Providers — Configuration Guide

`gmd web` commands support multiple providers for search and content retrieval. This guide
covers credential setup, provider group configuration, multi-provider parallel search,
dedup/synthesis, and provider-specific behavior.

## Quick Start

```bash
# Set credentials (only for providers you intend to use)
# Option A: export in shell
export EXA_API_KEY="exa-..."
export TAVILY_API_KEY="tvly-..."
export CLOUDFLARE_API_KEY="cf-..."
export CLOUDFLARE_ACCOUNT_ID="abc123..."
export SEARXNG_BASE_URL="https://searx.example.com"

# Option B: store in <UserConfigDir>/gmd/secret.env (auto-loaded)
#   EXA_API_KEY=exa-...
#   TAVILY_API_KEY=tvly-...
#   CLOUDFLARE_API_KEY=cf-...
#   CLOUDFLARE_ACCOUNT_ID=abc123...
#   SEARXNG_BASE_URL=https://searx.example.com

# EXA everything (default group)
gmd web search "transformer architecture"

# Override browser provider to Cloudflare
gmd web fetch https://example.com --browser-provider cloudflare

# Use the "offline" provider group (SearXNG search + local fetch)
gmd web search "self-hosted search" --provider-group offline
```

## Credentials

| Provider | Env Vars | Account / Setup |
|---|---|---|
| **EXA** | `EXA_API_KEY` | [exa.ai](https://exa.ai) — free tier: 1000 queries/mo |
| **Tavily** | `TAVILY_API_KEY` | [tavily.com](https://tavily.com) — pay-per-query |
| **SearXNG** | `SEARXNG_BASE_URL` | self-hosted or public instance — no API key needed. See [SearXNG Docker](#searxng-docker) below for quick local setup. |
| **Cloudflare Browser Run** | `CLOUDFLARE_API_KEY`, `CLOUDFLARE_ACCOUNT_ID` | [dash.cloudflare.com](https://dash.cloudflare.com) → Workers & Pages → Browser Rendering. Workers Paid $5/mo, 10 hrs free. |
| **Local** | none | no credentials needed (Phase 4) |

Credentials are read from environment variables at startup. They are **never** stored in CUE
config files. Omitting a provider block from config means "don't use this provider."

Use `gmd env` to verify your resolved config — it prints all settings with API keys masked.

## Provider Groups

Provider groups map a preset name to `{search, browser}` role selections — similar to
`searchDefaults` for collections and wikis. The `web.group` field selects the active group.
The `search` field accepts a list of provider names; all are queried in parallel.

```cue
web: {
  group: "default"
  groups: {
    default:  { search: ["exa", "tavily"],        browser: "exa" }
    full:     { search: ["exa", "tavily", "searxng"],  browser: "cloudflare" }
    offline:  { search: ["searxng"],              browser: "local" }
    research: { search: ["tavily", "exa"],         browser: "cloudflare" }
    quick:    { search: ["exa"],                   browser: "exa" }
  }
}
```

### Common Presets

| Group | Search | Browser | Use Case |
|---|---|---|---|
| `default` | exa, tavily | exa | Multi-provider breadth |
| `full` | exa, tavily, searxng | cloudflare | Maximum coverage + live rendering |
| `offline` | searxng | local | Self-hosted, no cloud dependencies |
| `research` | tavily, exa | cloudflare | Broad index coverage for deep research |

### Search Behavior

When a group lists multiple search providers, `gmd web search` fans out the query to all of
them in parallel. Results are then:

1. **Merged** — collected from all providers, tagged with provider name
2. **Deduplicated** — by URL (heuristic, default) or via LLM (`--dedup llm`)
3. **Sorted** — by relevance score
4. **Synthesized** — optional LLM-summarized answer with citations (`--synthesize`, default: on)

Configure defaults:

```cue
web: {
  search: {
    dedup:      "heuristic"     // "heuristic", "llm", or "none"
    synthesize: true            // synthesize results via LLM (summarizer model)
    synthesis_prompt: ""        // path to custom system prompt file
  }
}
```

CLI overrides:
- `--dedup heuristic|llm|none` — dedup strategy
- `--synthesize` / `--no-synthesize` — enable/disable LLM synthesis
- `--synthesis-prompt <path>` — custom system prompt file
- `--search-provider a,b,c` — comma-separated providers (overrides group)

### CLI Overrides

| Flag | Effect |
|---|---|
| `--provider-group <name>` | Use a different provider group for this command |
| `--search-provider <a,b,...>` | Override search providers in the active group (comma-separated) |
| `--browser-provider <name>` | Override only the browser role within the active group |
| `--dedup heuristic\|llm\|none` | Dedup strategy (default: config or "heuristic") |
| `--synthesize` / `--no-synthesize` | Enable/disable LLM synthesis (default: config or true) |
| `--synthesis-prompt <path>` | Custom system prompt for synthesis |

Priority: individual role override → `--provider-group` → configured `group` → `"default"`.

## Providers

### EXA

Implements `SearchProvider` (web index search) and `BrowserProvider.GetContent` (cached
page retrieval via `/contents`). Does **not** support `Crawl` or `Scrape`.

```cue
exa: {
  // api_key from EXA_API_KEY env var — never here
}
```

### Cloudflare Browser Run

Implements `BrowserProvider`. Supports `GetContent` (live rendering via `/markdown` and
`/content`) and `Crawl` (link-following via repeated rendering). Does **not** implement
`SearchProvider`.

```cue
cloudflare: {
  // api_key    from CLOUDFLARE_API_KEY env var — never here
  // account_id from CLOUDFLARE_ACCOUNT_ID env var — never here
}
```

### Tavily

Implements `SearchProvider`. Web search with optional raw content extraction.

```cue
tavily: {
  // api_key from TAVILY_API_KEY env var — never here
}
```

Provider-specific `SearchOptions.Extra` keys:
- `search_depth` — `"basic"` (default) or `"advanced"`
- `include_answer` — `true` to get LLM-generated answer
- `include_raw_content` — `true` to get raw page HTML

### SearXNG

Implements `SearchProvider`. Self-hosted metasearch engine — no API key required.
Set `base_url` via CUE config or `SEARXNG_BASE_URL` env var (env var wins if both set).

```cue
searxng: {
  base_url: "https://searx.example.com"   // or via SEARXNG_BASE_URL env var
  engines:  ""                            // optional: "google,bing" for specific engines
}
```

Provider-specific `SearchOptions.Extra` keys:
- `categories` — comma-separated category filter
- `engines` — comma-separated engine filter (also settable in CUE config)
- `language` — language code (e.g., `"en"`)

#### SearXNG Docker

Public SearXNG instances aggressively rate-limit or block automated API access.
Running your own instance via Docker is recommended for reliable use:

```bash
# Start SearXNG
docker run --rm -d --name searxng -p 8080:8080 searxng/searxng

# Write settings enabling JSON API format
cat > /tmp/searxng-settings.yml << 'EOF'
use_default_settings: true
search:
  formats:
    - html
    - json
server:
  secret_key: "replace-with-random-string"
  limiter: false
EOF

# Apply settings and restart
docker cp /tmp/searxng-settings.yml searxng:/etc/searxng/settings.yml
docker restart searxng

# Verify
curl "http://localhost:8080/search?format=json&q=test"
```

**Key points:**
- `use_default_settings: true` is required — without it many defaults are missing
- `search.formats` must include `json` — only `html` is enabled by default
- `server.secret_key` can be any random string (Flask session key)
- `server.limiter: false` disables rate limiting (convenient for local use)

Then configure: `searxng: base_url: "http://localhost:8080"` in your CUE config.

Official docs: [SearXNG Installation](https://docs.searxng.org/admin/installation-docker.html),
[SearXNG Search API](https://docs.searxng.org/dev/search_api.html).

### Local

Implements `BrowserProvider`. Static HTTP fetch, headless browser render, crawl, and scrape
without any cloud dependency (Phase 4).

```cue
local: {
  no_browser:            false   // true = only raw HTTP fetch, no headless browser
  chromium_path:         ""      // custom Chromium binary path
  crawl_delay_ms:        1000    // delay between page fetches (ms)
  max_concurrent_domains: 2      // concurrent crawl domain limit
  max_pages_per_domain:  200     // per-domain page limit during crawl
  cache_enabled:         false   // cache fetched pages to disk
  cache_dir:             ""      // defaults to ~/Library/Caches/gmd/web
  cache_max_size:        536870912  // 512MB max disk cache
  cache_ttl:             "24h"   // cache entry lifetime
}
```

## Role Matrix

| Provider | SearchProvider | BrowserProvider.GetContent | BrowserProvider.Crawl | BrowserProvider.Scrape |
|---|---|---|---|---|
| EXA | yes | yes (cached) | no | no |
| Cloudflare | no | yes (live) | yes | no |
| Tavily | yes | no | no | no |
| SearXNG | yes | no | no | no |
| Local | no | yes (static HTTP) | yes | yes |

EXA is the only provider registered in both roles. Cloudflare provides live browser rendering
but cannot search a web index. Local provides fetch and crawl without any cloud dependency
(Phase 4).

## Command Mapping

| Command | Tier | Interface Used | Valid Providers |
|---|---|---|---|
| `gmd web search` | 1 | `SearchProvider` (multi) | exa, tavily, searxng (parallel) |
| `gmd web fetch` | 1 | `BrowserProvider.GetContent` | exa, cloudflare, local |
| `gmd web crawl` | 1 | `BrowserProvider.Crawl` | cloudflare, local |
| `gmd web agent` | 2 | EXA (hardcoded) | exa |
| `gmd web research` | 3 | `SearchProvider` + `BrowserProvider` | any (stub) |

Tier 1 search runs all configured search providers in parallel via the fusion engine
(`pkg/web/fusion`), then deduplicates and optionally synthesizes via LLM.
The agent (Tier 2) remains EXA-specific. It will adopt the provider interfaces when a second
search provider is fully integrated.

## Cost Display

Providers return a `CostSummary` alongside results. The CLI renders cost information
generically:

```
Cost: $0.001530 query (exa)
```

Different billing models (per-query, per-minute, credit-based) are distinguished by the
`unit` field.
