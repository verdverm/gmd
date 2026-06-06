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
gmd agentsmd [name]          # Output AGENTS.md content for AI assistants (oneline, summary, detailed, full)
gmd wiki init [--name]        # Create a Karpathy-style LLM Wiki
gmd wiki ingest <src>         # LLM agent reads source, creates/updates wiki pages
gmd wiki query "<q>"          # RAG search → LLM synthesis with [[page]] citations
```

## Setup

1. **Typesense must be running** (default: `http://localhost:8108`). Set `GMD_TYPESENSE_API_KEY` if auth is enabled.
2. **Three LLM endpoints are needed** (embedding, expansion, rerank). Set `OPENAI_API_KEY` env var (or per-role overrides: `GMD_EMBEDDING_API_KEY`, `GMD_EXPANSION_API_KEY`, `GMD_RERANK_API_KEY`, `GMD_SUMMARIZING_API_KEY`, `GMD_GENERAL_BIG_API_KEY`, `GMD_GENERAL_MID_API_KEY`, `GMD_GENERAL_SMALL_API_KEY`). Any OpenAI-compatible API works (vLLM, Ollama, etc.).
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

GMD can search the live web via multiple providers (EXA, Tavily, SearXNG, Cloudflare):

```sh
gmd web search "<query>"       # Web search via configured search provider
gmd web fetch <url>            # Clean content extraction from URLs
gmd web crawl <url>            # Crawl a site from seed URL (Cloudflare)
gmd web agent "<question>"     # Multi-step LLM-orchestrated research
gmd web research "<topic>"     # Deep structured research pipeline (stub)
```

Commands fall on a three-tier spectrum: deterministic (search/fetch/crawl) → conversational agent → deep research. Each tier builds on the prior.

Select providers via named groups in config, or override per-command with `--search-provider` / `--browser-provider`. Set credentials via env vars or env files: `EXA_API_KEY`, `TAVILY_API_KEY`, `CLOUDFLARE_API_KEY` + `CLOUDFLARE_ACCOUNT_ID`, `SEARXNG_BASE_URL`. SearXNG is self-hosted (no API key). Use `gmd env` to verify your resolved config. See [docs/web-providers.md](docs/web-providers.md) for details.

## LLM Wiki

GMD includes a built-in agent for Karpathy-style compounding knowledge bases:

- `gmd wiki init --name myresearch` — scaffold wiki directory structure + CUE config
- `gmd wiki ingest paper.md` — LLM reads source, extracts entities/concepts/claims, writes/updates interlinked wiki pages
- `gmd wiki query "what is..."` — RAG search over wiki → LLM synthesizes answer with [[page]] citations
- `gmd wiki skills write --target all` — install skill templates for AI agents (Claude Code, Codex, OpenCode)
- `gmd wiki lint` — check for orphan pages, broken wikilinks, contradictions, knowledge gaps
- `gmd wiki doctor --fix` — diagnostics + auto-configure MCP servers for detected agents
