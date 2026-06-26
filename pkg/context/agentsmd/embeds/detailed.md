# GMD — Markdown Search Engine (Detailed Reference)

GMD indexes local markdown files and provides full-text, vector, and hybrid search. Built in Go, backed by Typesense, powered by any OpenAI-compatible LLM API.

For setup requirements and quick start instructions, see [setup](setup.md).

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

### LLM Providers & Profiles

| Command | Description |
|---|---|
| `gmd llm status` | Health check all LLM providers and roles |
| `gmd llm providers` | List configured LLM providers |
| `gmd llm profiles` | List configured LLM profiles |
| `gmd llm show <name>` | Show role→provider mappings for a profile |
| `gmd llm test <provider>` | Quick chat test against a provider |

### Servers & Diagnostics

| Command | Description |
|---|---|
| `gmd serve [--port] [--host]` | Start REST API server (default: `:8181`) |
| `gmd mcp [--http]` | Start MCP server for AI agent integration |
| `gmd doctor` | Run system diagnostics |
| `gmd env` | Print resolved config with secrets masked |
| `gmd context agentsmd show [oneline|summary|detailed|full]` | Output AGENTS.md content for AI coding assistants |

### Agent Harness

| Command | Description |
|---|---|
| `gmd agent [task] [message]` | Launch external AI agent harness |
| `gmd agent list` | List configured agent harnesses and profiles |
| `gmd agent profile list` | List configured agent profiles |
| `gmd agent profile show <name>` | Show resolved configuration for a profile |
| `gmd agent session list` | List active tmux sessions and git workspaces |
| `gmd agent session kill <name>` | Kill tmux session and remove its workspace |
| `gmd agent session merge <name>` | Merge a workspace into the current branch |

Launch flags:
- `-p, --profile <name>` — profile or harness name to launch
- `-m, --message <text>` — message/prompt for the agent
- `--config <path>` — path to harness-specific config file
- `--cwd <path>` — working directory
- `--tmux` — launch inside a named tmux session
- `--tmux-conf <path>` — path to tmux config file
- `--workspace` — create a git worktree before launching
- `--workspace-base <ref>` — git ref for worktree (default: current branch)
- `--async` — don't block; return after launching
- `--dry-run` — print resolved command without executing
- `--flag KEY=VAL` — extra flag for the harness (repeatable)
- `--env KEY=VAL` — extra env var (repeatable)
- `--args VAL` — extra positional args (repeatable)

### Web Search

Three-tier spectrum for searching the live web via multiple providers (EXA, Cloudflare,
Tavily, SearXNG). Configure provider groups in CUE config or override per-command:

| Tier | Command | Description |
|---|---|---|
| 1 | `gmd web search <query>` | Multi-provider web search: parallel fan-out → merge → dedup → LLM synthesis |
| 1 | `gmd web fetch <url> [url2 ...]` | Clean content extraction from URLs |
| 1 | `gmd web crawl <url>` | Crawl a site from seed URL (Cloudflare or local) |
| 2 | `gmd web agent <query>` | Multi-step LLM-orchestrated research agent |
| 3 | `gmd web research <query>` | Deep structured research pipeline (stub) |

Tier 1 search runs all configured providers in parallel, merges results, deduplicates (URL-based
or LLM), and synthesizes a cited answer via the summarizer LLM (enabled by default, disable with
`--no-synthesize`).

**Provider groups** use list syntax for search: `search: ["exa", "tavily"]`. All listed
providers are queried in parallel. Configure dedup/synthesis defaults via `web.search`.

**Credentials** (set via env vars or env files, never in CUE config): `EXA_API_KEY`,
`TAVILY_API_KEY`, `CLOUDFLARE_API_KEY` + `CLOUDFLARE_ACCOUNT_ID`, `SEARXNG_BASE_URL`.
SearXNG is self-hosted (no API key). Use `gmd env` to verify your resolved config.

**CLI flags:** `--provider-group <name>` (select a named group), `--search-provider a,b`
(comma-separated providers override), `--browser-provider <name>` (override browser role),
`--dedup heuristic|llm|none`, `--synthesize` / `--no-synthesize`, `--synthesis-prompt <path>`.

### LLM Wiki

| Command | Description |
|---|---|
| `gmd wiki create <name> [--path] [--wiki-dir] [--raw-dir] [--skills]` | Scaffold wiki directory structure + CUE config entry |
| `gmd wiki ingest <name> <src>` | LLM reads source, extracts entities/concepts/claims, writes wiki pages |
| `gmd wiki query <name> "<question>" [--save]` | RAG search over wiki → LLM synthesis with citations |
| `gmd wiki graph <name> [--format]` | Export link graph as dot, mermaid, or JSON |
| `gmd wiki lint <name>` | Structure checks (orphans, broken links) + OKF conformance + LLM content analysis |
| `gmd wiki export <name> [--output <dir>]` | Export wiki as a self-contained directory |
| `gmd wiki skills [list|show|write]` | Manage embedded skill templates for AI agents |
| `gmd wiki doctor <name> [--fix]` | Diagnostics + auto-configure MCP servers for detected agents |

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
pkg/agent/        Agent harness launcher: config resolution, tmux, workspace, session management
pkg/wiki/         LLM Wiki: scaffold, built-in agent, graph, lint, skills
pkg/mcp/          MCP server tools (wiki-aware tools)
pkg/web/          Web providers: shared interfaces, registry, agent, prompts, fusion
pkg/web/providers/ Provider implementations: exa, cloudflare, local, tavily, searxng
pkg/web/fusion/    Multi-provider parallel search, dedup, LLM synthesis
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
