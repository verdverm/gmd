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
```

## Setup

1. **Typesense must be running** (default: `http://localhost:8108`). Set `GMD_TYPESENSE_API_KEY` if auth is enabled.
2. **Three LLM endpoints are needed** (embedding, expansion, rerank). Set `OPENAI_API_KEY` env var. Any OpenAI-compatible API works (vLLM, Ollama, etc.).
3. **Run `gmd init`** in your project root to create `.gmd/config.cue`. Edit it to set your LLM endpoints and configure which files to index.
4. **Run `gmd update`** to scan, chunk, embed, and index all configured collections.

## Key Behavior

- **Auto-detect collection:** `gmd query` run from inside a collection's path automatically selects that collection. Use `-c` to specify explicitly.
- **Content-addressable dedup:** Unchanged files skip re-chunking and re-embedding on subsequent `gmd update` runs. Only modified files are re-processed.
- **Config:** CUE format. Global config at `~/.config/gmd/config.cue`, project config at `<root>/.gmd/config.cue`. Both are optional.
- **Search pipeline** (`gmd query`): Detects strong signals (scores ≥0.85 with gap ≥0.15) and uses query directly if found; otherwise expands the query via LLM, runs hybrid search for each variant, fuses results with RRF, reranks, and blends by position tier.
- **Output formats:** `cli` (default, human-readable) or `json`. Use `-f json` for machine-readable output.
- **Results limit:** Default 5. Use `-n` to change (e.g., `-n 10`).

## Important Rules for AI Agents Using GMD

- **Never run `gmd update`, `gmd embed`, or `gmd collection add` automatically.** Always write the command for the user to run.
- **Never modify CUE config files or the Typesense index directly** without being asked.
- Use `gmd query` for general questions — it provides the best results through the full pipeline.
- Use `gmd search` when you need fast keyword results without LLM overhead.
- Use `gmd get <path>` to retrieve full document content after finding relevant files.
- Use `gmd ls` to see what documents are currently indexed.
- If `gmd query` returns no results, check `gmd status` to verify the index is populated.
