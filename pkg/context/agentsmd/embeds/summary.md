# GMD — Markdown Search Engine

GMD indexes local markdown files and provides full-text, vector, and hybrid search via the `gmd` CLI. It is backed by Typesense for search and uses OpenAI-compatible LLM APIs for embeddings, query expansion, and reranking.

## Essential Commands

```sh
gmd init                     # Create .gmd/config.cue in the current directory
gmd update                   # Index/re-index all collections
gmd status                   # Show index health and per-collection document counts
gmd query "<question>"       # Full pipeline: expansion → hybrid search → RRF fusion → rerank → blend
gmd search "<terms>"         # Text-only keyword search (fast, no LLM)
gmd get <path>               # Retrieve document content by file path
gmd ls [collection]          # List indexed documents
gmd collection list          # List all collections
gmd context agentsmd show [name]          # Output AGENTS.md content for AI assistants (oneline, summary, detailed, full)
gmd wiki create <name>         # Create a Karpathy-style LLM Wiki
gmd wiki ingest <name> <src>    # LLM agent reads source, creates/updates wiki pages
gmd wiki query <name> "<q>"     # RAG search → LLM synthesis with citations
gmd llm status                # Health check all LLM providers and roles
gmd agent list                # List configured agent harnesses and profiles
gmd agent mytask --profile wiki  # Launch external AI agent harness
```

## Setup

1. **Typesense must be running** (default: `http://localhost:8108`). Set `GMD_TYPESENSE_API_KEY` if auth is enabled.
2. **Configure LLM providers and profiles** in CUE (see `gmd init` scaffold). Named providers with provider type (openai, anthropic, vertex, opencode, custom), base_url, and auth method (none, apikey, service-account). Profiles map roles (embedding, expansion, rerank, summarizing, general_big/mid/small) to provider+model pairs. API keys are resolved from env vars by provider type: `OPENAI_API_KEY` (openai), `ANTHROPIC_API_KEY` (anthropic), `OPENCODE_API_KEY` (opencode), `GMD_LLM_API_KEY` (custom). Use `gmd llm status` to test connectivity.
3. **Run `gmd init`** in your project root to create `.gmd/config.cue`. Edit it to set your LLM endpoints and configure which files to index.
4. **Run `gmd update`** to scan, chunk, embed, and index all configured collections.

## Key Behavior

- **Auto-detect collection:** `gmd query` run from inside a collection's path automatically selects that collection. Use `-c` to specify explicitly.
- **Content-addressable dedup:** Unchanged files skip re-chunking and re-embedding on subsequent `gmd update` runs. Only modified files are re-processed.
- **Config:** CUE format. Global config at `<UserConfigDir>/gmd/config.cue`, project config at `<root>/.gmd/config.cue`. Both are optional.
- **Search pipeline** (`gmd query`): Detects strong signals (scores ≥0.85 with gap ≥0.15) and uses query directly if found; otherwise expands the query via LLM, runs hybrid search for each variant, fuses results with RRF, reranks, and blends by position tier.
- **Output formats:** `cli` (default, human-readable) or `json`. Use `-f json` for machine-readable output.
- **Results limit:** Default 5. Use `-n` to change (e.g., `-n 10`).

## Important Rules for AI Agents Using GMD

- **Never run `gmd update`, `gmd embed`, or `gmd collection create` automatically.** Always write the command for the user to run.
- **Never modify CUE config files or the Typesense index directly** without being asked.
- Use `gmd query` for general questions — it provides the best results through the full pipeline.
- Use `gmd search` when you need fast keyword results without LLM overhead.
- Use `gmd get <path>` to retrieve full document content after finding relevant files.
- Use `gmd ls` to see what documents are currently indexed.
- If `gmd query` returns no results, check `gmd status` to verify the index is populated.

## Web Search

GMD can search the live web via multiple providers (EXA, Tavily, SearXNG, Cloudflare)
with parallel fan-out, automatic dedup, and optional LLM synthesis:

```sh
gmd web search "<query>"       # Multi-provider web search → merge → dedup → synthesis
gmd web fetch <url>            # Clean content extraction from URLs
gmd web crawl <url>            # Crawl a site from seed URL (Cloudflare)
gmd web agent "<question>"     # Multi-step LLM-orchestrated research
gmd web research "<topic>"     # Deep structured research pipeline (stub)
```

Search runs all configured providers in parallel, merges results, deduplicates (by URL or
LLM), and optionally synthesizes a unified cited answer via the summarizer LLM.
Flags: `--search-provider exa,tavily` (comma-separated list), `--dedup heuristic|llm|none`,
`--synthesize` / `--no-synthesize`, `--synthesis-prompt <path>`.

Select providers via named groups in config (`web.groups` with `search: [...]` lists), or
override per-command with `--search-provider` / `--browser-provider`. Set credentials via
env vars or env files. Use `gmd env` to verify your resolved config.

## LLM Wiki

GMD includes a built-in agent for Karpathy-style compounding knowledge bases:

- `gmd wiki create myresearch` — scaffold wiki directory structure + CUE config
- `gmd wiki ingest myresearch paper.md` — LLM reads source, extracts entities/concepts/claims, writes/updates interlinked wiki pages
- `gmd wiki query myresearch "what is..."` — RAG search over wiki → LLM synthesizes answer with citations
- `gmd wiki skills write --target all` — install skill templates for AI agents (Claude Code, Codex, OpenCode)
- `gmd wiki lint myresearch` — check for orphan pages, broken links, OKF conformance, contradictions, knowledge gaps
- `gmd wiki export myresearch` — export wiki as self-contained directory
- `gmd wiki doctor myresearch --fix` — diagnostics + auto-configure MCP servers for detected agents

## Agent Harness

GMD can launch external AI agent harnesses (OpenCode, Claude Code, Codex, or generic) with optional
tmux session management and git worktree isolation. Configure harnesses and profiles in the `agent`
section of your CUE config. `gmd agent` resolves profiles, builds the harness command, and can
launch in tmux with `--tmux` or in an isolated git worktree with `--workspace`.

```sh
gmd agent mytask "fix the bug"        # Launch with default harness
gmd agent mytask --profile wiki       # Launch with a specific profile
gmd agent mytask --tmux --workspace   # Launch in tmux + isolated worktree
gmd agent list                        # List configured harnesses + profiles
gmd agent profile show wiki           # Show resolved config for a profile
gmd agent session list                # List active sessions
gmd agent session kill mytask         # Kill session + remove workspace
gmd agent session merge mytask        # Merge workspace into current branch
```
