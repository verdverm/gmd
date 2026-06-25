# GMD — Markdown Search Engine (Go port of QMD)

**gmd** indexes collections of local markdown files and lets you search them with
full-text, vector, or hybrid search - backed by [Typesense](https://typesense.org)
and any OpenAI-compatible LLM. Build compounding LLM wikis that ingest source
documents, extract knowledge, and link pages via standard markdown links
([OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md) compatible). Run web searches,
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
- **Agent harness launcher** - launch external AI agent harnesses (OpenCode, Claude Code, Codex, or
  generic) with tmux sessions and git worktree isolation. Profiles define launch presets.

## How it works

**Search** (`gmd query`) expands your query via LLM, searches Typesense with text and vector
similarity, fuses results with RRF, optionally reranks with an LLM, and blends by chunk position.
You get the most relevant chunks ranked intelligently - no manual query tuning needed.
See [docs/search-pipeline.md](docs/search-pipeline.md) for the full pipeline diagram.

**Wiki** (`gmd wiki`) maintains a compounding knowledge base of interlinked markdown pages. The
built-in LLM agent reads source documents, extracts entities and claims, writes or updates wiki
pages, and links them via standard markdown links. Querying the wiki uses the same Typesense-backed search pipeline: retrieve relevant pages,
synthesize an answer with inline citations.

**Web** (`gmd web`) lets you search the web, fetch clean content from URLs, crawl sites, or run a
multi-step LLM-orchestrated research agent that searches, reads, and synthesizes across multiple
rounds. Backed by a multi-provider architecture (EXA, Cloudflare, Tavily, SearXNG) with configurable
provider groups. `gmd web search` fans out queries to all configured search providers in parallel,
merges and deduplicates results, and optionally synthesizes a unified cited answer via LLM.

**Agent** (`gmd agent`) launches external AI agent harnesses (OpenCode, Claude Code, Codex, or
generic) with optional tmux sessions and git worktree workspaces. Profiles define harness-specific
flags, messages, and launch behavior. Sessions are tracked via tmux + git worktrees for review,
merge (`gmd agent session merge`), and teardown (`gmd agent session kill`).
`gmd wiki doctor --fix` auto-launches the agent after applying fixes.

**Deploy** (`gmd serve` / `gmd mcp`) exposes gmd over HTTP and/or MCP so AI coding assistants and
other tools can search your docs, query wikis, and browse indexed content.

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
make gofmt                              # gofmt -s formatting check (Go Report Card style)
make lint-all                           # golangci-lint (comprehensive: errcheck, gosec, gocyclo, revive, staticcheck, ...)
make vulncheck                          # govulncheck (OSV vulnerability scanner)
make nilaway                            # nilaway (nil pointer analysis)
make check                              # Full pre-commit: tidy → gofmt → license → lint → lint-all → vulncheck → test
make tidy                               # go mod tidy
make tools-install                      # Install/update pinned tools (golangci-lint, govulncheck, nilaway)
make tools-update                       # Re-run tools-install
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
gmd ls [source]                        # List indexed documents
gmd collection list                    # List collections + wikis with "referenced by"
gmd collection create <name> --path --pattern
gmd collection show <name>             # Shows collections or wikis + chunk count
gmd collection remove <name>           # Remove collection or wiki
gmd collection rename <old> <new>      # Rename collection or wiki
gmd collection include <name> <patterns...>
gmd collection exclude <name> <patterns...>
gmd context add <source> "text"
gmd context list
gmd context rm <source>
gmd doctor                             # Diagnostics (collections + wikis)
gmd env                                # Print resolved config with secrets masked
gmd cleanup                            # Remove stale chunks for deleted files
gmd web search <query>                  # Tier 1: Multi-provider web search with LLM synthesis
gmd web fetch <url> [url2 ...]           # Tier 1: Fetch clean content from URLs
gmd web crawl <url> [--depth] [--max-pages] # Tier 1: Crawl from seed URL
gmd web agent <query> [--steps] [--save] # Tier 2: LLM-orchestrated research agent
gmd web research <query> [--depth]       # Tier 3: Deep structured research (stub)
gmd serve [--port] [--host]              # REST API server (stub, Phase 5)
gmd mcp [--http]                       # MCP server (stub, Phase 6)
gmd agentsmd [oneline|summary|detailed|full]  # Output AGENTS.md content for users
gmd llm status                          # Health check all LLM providers and roles
gmd llm providers                       # List configured LLM providers
gmd llm profiles                        # List configured LLM profiles
gmd llm show <name>                     # Show role->provider mappings for a profile
gmd llm test <provider>                 # Quick chat test against a provider
gmd agent [task-name] [message]          # Launch external AI agent harness
gmd agent list                           # List configured harnesses + profiles
gmd agent profile list                   # List profiles
gmd agent profile show <profile>         # Show resolved config for a profile
gmd agent session list                   # List active sessions + workspaces
gmd agent session kill <name>            # Kill tmux session + remove workspace
gmd agent session merge <name>           # Merge workspace into current branch
gmd wiki export <name> [--output <dir>]  # Export wiki as a self-contained directory
gmd wiki create <name> [--path] [--wiki-dir] [--raw-dir] [--skills]
gmd wiki list                          # List all wikis
gmd wiki show <name>                   # Wiki config details + chunk count
gmd wiki remove <name>                 # Remove wiki + Typesense chunks
gmd wiki rename <old> <new>            # Rename wiki in config
gmd wiki include <name> <patterns...>   # Add file patterns (proxy)
gmd wiki exclude <name> <patterns...>   # Add ignore patterns (proxy)
gmd wiki context add <name> "text"     # Set wiki context (proxy)
gmd wiki context list                  # List wiki contexts (proxy)
gmd wiki context rm <name>             # Remove wiki context (proxy)
gmd wiki ref add <name> <source>       # Add source reference (validated)
gmd wiki ref rm <name> <source>        # Remove source reference
gmd wiki ref list <name>               # List source references
gmd wiki ingest <name> <src> [--batch] # LLM agent reads source → writes wiki pages
gmd wiki query <name> "..." [--save]   # Search wiki → LLM synthesis with citations
gmd wiki graph <name> [--format]       # Export wikilink graph (dot/mermaid/json)
gmd wiki lint <name>                   # Structure + content health checks
gmd wiki doctor <name> [--fix]         # Diagnostics + auto-configure agents
gmd wiki skills [list|show|write]      # Manage embedded agent skill templates
```

## Architecture

```
cmd/gmd/          CLI entry point (cobra commands)
pkg/config/       CUE config loading, validation, project detection, CUE AST editing
                  Schema in schema/: #Source, CollectionConfig, WikiConfig, ProjectConfig
pkg/chunking/     Markdown chunker (heading-aware breakpoints)
pkg/indexer/      File scanning + SHA-256 dedup + chunk → embed → upsert pipeline
pkg/search/       Search pipeline: signal detection, expansion, RRF fusion, rerank, blend
pkg/ts/           Typesense client wrapper (chunks collection, hybrid/text search, CRUD)
pkg/llm/          OpenAI-compatible API client (embeddings, chat, rerank)
pkg/llm/auth/     Auth methods: none, apikey, service-account (GCP)
pkg/llm/builder.go Client builder for multi-provider structured config
pkg/output/       Result formatting (CLI, JSON)
pkg/runtime/      Runtime struct — owns Typesense client lifecycle
pkg/agentsmd/     Embedded AGENTS.md content (oneline/summary/detailed/full)
pkg/agent/        Agent harness launcher: config resolution, tmux, workspace, session management
pkg/wiki/         LLM Wiki: scaffold, built-in agent, graph, lint, skills
pkg/web/          Web providers: shared interfaces, registry, agent, prompts, fusion
pkg/web/providers/ Provider implementations: exa, cloudflare, local, tavily, searxng
pkg/web/fusion/    Multi-provider parallel search, dedup, LLM synthesis
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
- **LLM providers and profiles.** Providers are named endpoints with auth config (none, apikey,
  service-account). Profiles map roles (embedding, expansion, etc.) to provider+model pairs.
  Active profile selected via `llm.profile`.
- **Chunks as Typesense documents.** `group_by=collection,path` collapses to document level.
- **Wikis are first-class entities** parallel to collections, with a shared `#Source` indexing
  model and `SourceConfig` Go struct. Both use the same Typesense `chunks` collection.
- **Wikis aggregate via sourceRefs.** A wiki can reference collections or other wikis to
  aggregate their searchable content. Collections and wikis share a namespace (name uniqueness).

## Config

CUE schema lives in `pkg/config/schema/` (embedded via `//go:embed`).
Global: `<UserConfigDir>/gmd/config.cue`. Project: `<root>/.gmd/config.cue`.
Both are optional; defaults come from the embedded schema + `pipeline.cue`.
Project root detected by walking up from CWD looking for `.gmd/` sentinel.

## Agent Config

```cue
agent: {
  defaultHarness: "opencode"
  harnesses: {
    opencode: { bin: "opencode", flagStyle: "double-dash", env: { "KEY": "value" } }
    claude:   { bin: "claude",   flagStyle: "double-dash" }
    codex:    { bin: "codex",    flagStyle: "double-dash" }
    generic:  { bin: "myharness", flagStyle: "single-dash" }
  }
  profiles: {
    wiki: {
      harness:    "opencode"
      configFile: "opencode.json"
      workspace:  true
      message:    "Work on the gmd wiki. Run /help for tools."
    }
    dev: {
      harness:   "opencode"
      tmux:      true
      workspace: true
    }
  }
}
```

Harness types: `opencode` (uses `run` subcommand), all others are `generic` (uses `--message` flag
by default). Generic harnesses support `flagStyle: "single-dash"` for tools using `-message` style.

## Dependencies

| Module | Purpose |
|---|---|
| `github.com/typesense/typesense-go/v4` | Typesense client |
| `github.com/openai/openai-go/v3` | OpenAI-compatible API client |
| `cuelang.org/go` | CUE config loading, validation, CUE AST editing |
| `github.com/spf13/cobra` | CLI framework |
| `github.com/bmatcuk/doublestar/v4` | Glob pattern matching for file scanning |

## Tooling

| Tool | Purpose |
|---|---|
| `github.com/golangci/golangci-lint` | Meta-linter: errcheck, gosec, revive, staticcheck, etc. |
| `golang.org/x/vuln/cmd/govulncheck` | OSV vulnerability scanner for Go stdlib + deps |
| `go.uber.org/nilaway/cmd/nilaway` | Nil pointer analysis |

### golangci-lint enabled linters

Includes all Go Report Card checks (gofmt excluded, handled natively):
`bodyclose`, `copyloopvar`, `durationcheck`, `errcheck`, `gocyclo`,
`gosec`, `ineffassign`, `misspell`, `noctx`, `prealloc`, `revive`,
`staticcheck`, `unconvert`, `unparam`, `usetesting`, `wastedassign`

## Testing

### Unit Tests (`make test`)

Unit tests run offline with no external dependencies. Most packages that touch Typesense or LLM
APIs now use HTTP tape recording/replay via `pkg/testutil/tape.go`:

- **Tape files** are committed JSON arrays of request/response exchanges in per-package `testdata/`
  directories. They are generated by integration test runs against real APIs.
- **Replay tests** (`*_replay_test.go`) load tapes via `testutil.NewReplayTape()`, inject the tape
  transport via `cfg.HTTPClient`, and assert structural validity of responses. They never call real
  APIs.
- **When adding/modifying code that calls external APIs**, write replay tests using existing tapes.
  If the API interaction changes, new tapes must be generated (see below).
- **Tape recording** is on by default during integration tests. Set `GMD_NORECORD=1` to opt out.

### Integration Tests (`make test.integration`)

Integration tests use `//go:build integration` and require Docker + LLM API keys.
They take ~2-3min depending on external service availability (the wiki package starts a Typesense
Docker container). Use `make test.integration` but note that the tool timeout must be set high
enough (e.g. 10min = 600000ms). Do NOT use Go's `-timeout` flag — set the timeout on the
bash/tool invocation instead.

When integration tests run with recording enabled, they regenerate `testdata/*.json` tape files.
Always commit updated tapes alongside the code changes that produced them.

### Test Patterns

```
pkg/<name>/
  client.go                           # Source code
  client_test.go                      # Pure unit tests (no tape, no external deps)
  client_integration_test.go          # +build integration — wired with tape recording
  client_replay_test.go               # Unit tests replaying committed tapes
  testdata/                           # Committed JSON tape files
    001_foo.json
    002_bar.json
```

See `.design/test-data-capture.md` for the full design.

### Test Naming Convention

```
Test<Subsystem>_<Variant>           # Unit test (no build tag)
TestIntegration<Subsystem>_<Variant> # Integration test (//go:build integration)
```

The `_` separates the subsystem/operation from the test variant. Only one `_` allowed.
Names must be fully descriptive — include the subsystem name even in its own package.
No "Real" or ambiguous qualifiers in any test or tape file name.

Examples:
```
TestWikiDoctor_FormatResultFullyConnected
TestIntegrationWikiDoctor_WithTSAndLLM
```

Tape files follow the same naming pattern:
```
testdata/WikiDoctor_WithTSAndLLM.json
testdata/WikiDoctor_WithTSNilLLM.json
```

## Rules

- Never run `gmd update`, `gmd embed`, `gmd collection create`, or `gmd wiki create` automatically.
  Write the command for the user to run.
- Never modify CUE config files or the Typesense index directly without being asked.
- Always run `make lint` after code changes. Run `make lint-all` for comprehensive linting
  before committing.
- **Never run `make test.integration` unless the user explicitly asks you to.** Integration tests
  require Docker and LLM API keys. All API-dependent code should be tested via tape replay instead.
- Keep `CGO_ENABLED=0` — never introduce CGO dependencies.
- New CLI commands go in `cmd/gmd/<name>.go` and register in `main.go` init().
- New library packages go under `pkg/<name>/`.
- Tests live alongside source files (`*_test.go`).
- Integration tests requiring external systems (Typesense, LLMs) use `//go:build integration`
  build tag and are excluded from `make test`. Run `make test.integration` to include them.
- The `gmd agentsmd` command outputs embedded content from `pkg/agentsmd/content/`. Those files are
  user-facing (for end users and AI agents consuming gmd), not developer-facing. Update them
  when CLI commands or architecture change, but keep content focused on usage, not development.
- Never commit `bin/` or `qmd/` (both in .gitignore).
- Always include `.sessions/` when making commits.
- The project is still in alpha / just-me state, we do not need to worry about backwards compatibility.
