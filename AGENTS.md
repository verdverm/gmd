# GMD — Markdown Search Engine (Go port of QMD)

Go rewrite of [qmd](./qmd/) with Typesense as the backing search engine and an
OpenAI-compatible LLM API for embeddings, query expansion, and reranking.

## Commands

```sh
go build -o bin/gmd ./cmd/gmd          # Build
make build                              # Same, with CGO_ENABLED=0
make test                               # Unit tests (no external deps)
make test.integration                   # All tests including integration
make cover                              # Unit test coverage
make cover.integration                  # All test coverage
make cover.detailed                     # Unit test coverage (profile + HTML + func)
make cover.detailed.integration         # All test coverage (profile + HTML + func)
make lint                               # go vet ./...
make tidy                               # go mod tidy
```

## CLI

```sh
gmd init                               # Create .gmd/config.cue in current dir
gmd update                             # Index/re-index all collections
gmd embed                              # Re-embed all documents
gmd status                             # Index health and per-collection counts
gmd search <query>                     # Text-only keyword search
gmd vsearch <query>                    # Vector similarity search
gmd query <query>                      # Full pipeline: expansion → hybrid → RRF → rerank → blend
gmd get <path>                         # Get document content by path
gmd multi-get <pattern>                # Batch fetch documents
gmd ls [collection]                    # List indexed documents
gmd collection list                    # List collections
gmd collection add <name> --path --pattern
gmd collection show <name>             # Collection details + chunk count
gmd collection remove <name>
gmd collection rename <old> <new>
gmd collection include <name> --pattern
gmd collection exclude <name> --pattern
gmd context add <collection> "text"
gmd context list
gmd context rm <collection>
gmd doctor                             # Diagnostics
gmd cleanup                            # Remove stale chunks for deleted files
gmd serve [--port] [--host]            # REST API server (stub, Phase 5)
gmd mcp [--http]                       # MCP server (stub, Phase 6)
gmd agents [oneline|summary|detailed|full]  # Output AGENTS.md content for users
gmd wiki init [--name] [--path]        # Create wiki scaffold + CUE config entry
gmd wiki ingest <src> [--name]         # LLM agent reads source → writes wiki pages
gmd wiki query "..." [--name] [--save] # Search wiki → LLM synthesis with citations
gmd wiki graph [--name] [--format]     # Export wikilink graph (dot/mermaid/json)
gmd wiki lint [--name]                 # Structure + content health checks
gmd wiki doctor [--name] [--fix]       # Diagnostics + auto-configure agents
gmd wiki skills [list|show|write]      # Manage embedded agent skill templates
```

## Architecture

```
cmd/gmd/          CLI entry point (cobra commands)
pkg/config/       CUE config loading, validation, project detection, CUE AST editing
pkg/chunking/     Markdown chunker (heading-aware breakpoints)
pkg/indexer/      File scanning + SHA-256 dedup + chunk → embed → upsert pipeline
pkg/search/       Search pipeline: signal detection, expansion, RRF fusion, rerank, blend
pkg/ts/           Typesense client wrapper (chunks collection, hybrid/text search, CRUD)
pkg/llm/          OpenAI-compatible API client (embeddings, chat, rerank)
pkg/output/       Result formatting (CLI, JSON)
pkg/runtime/      Runtime struct — owns Typesense client lifecycle
pkg/agents/       Embedded AGENTS.md content (oneline/summary/detailed/full)
pkg/wiki/         LLM Wiki: scaffold, built-in agent, graph, lint, skills
pkg/mcp/          MCP server tools (wiki-aware tools)
models/           vLLM serve scripts + systemd units for 3 LLM models
k8s/              Typesense Kubernetes manifest
docs/             Configuration reference
api/              Reserved for REST API (empty)
```

## Key Design Decisions

- **No operational DB.** Typesense is the sole data store. Filesystem is source of truth.
- **Content-addressable dedup.** SHA-256 hash stored on every chunk; unchanged files skip
  re-chunking and re-embedding.
- **External embeddings.** Embeddings computed in Go via API, stored in Typesense.
- **No CGO.** `CGO_ENABLED=0` enforced. No tree-sitter, no sqlite.
- **CUE config only.** No YAML. Global + project-local CUE files unified at load time.
- **OpenAI-compatible, not OpenAI-specific.** Any provider via `base_url`. API keys via env vars.
- **Chunks as Typesense documents.** `group_by=collection,path` collapses to document level.

## Config

CUE schema lives in `pkg/config/schema/` (embedded via `//go:embed`).
Global: `~/.config/gmd/config.cue`. Project: `<root>/.gmd/config.cue`.
Both are optional; defaults come from the embedded schema + `pipeline.cue`.
Project root detected by walking up from CWD looking for `.gmd/` sentinel.

## Dependencies

| Module | Purpose |
|---|---|
| `github.com/typesense/typesense-go/v4` | Typesense client |
| `github.com/openai/openai-go/v3` | OpenAI-compatible API client |
| `cuelang.org/go` | CUE config loading, validation, CUE AST editing |
| `github.com/spf13/cobra` | CLI framework |
| `github.com/bmatcuk/doublestar/v4` | Glob pattern matching for file scanning |

## Rules

- Never run `gmd update`, `gmd embed`, or `gmd collection add` automatically. Write the command
  for the user to run.
- Never modify CUE config files or the Typesense index directly without being asked.
- Always run `make lint` after code changes.
- Keep `CGO_ENABLED=0` — never introduce CGO dependencies.
- New CLI commands go in `cmd/gmd/<name>.go` and register in `main.go` init().
- New library packages go under `pkg/<name>/`.
- Tests live alongside source files (`*_test.go`).
- Integration tests requiring external systems (Typesense, LLMs) use `//go:build integration`
  build tag and are excluded from `make test`. Run `make test.integration` to include them.
- The `gmd agents` command outputs embedded content from `pkg/agents/content/`. Those files are
  user-facing (for end users and AI agents consuming gmd), not developer-facing. Update them
  when CLI commands or architecture change, but keep content focused on usage, not development.
- Never commit `bin/` or `qmd/` (both in .gitignore).
- Always include `.sessions/` when making commits.
