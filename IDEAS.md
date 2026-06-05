# IDEAS - Feature & Expansion Ideas for GMD

This document catalogs feature requests, bug reports, and expansion ideas drawn from:

1. Issues and PRs against the upstream [qmd](https://github.com/tobi/qmd) project
2. [Typesense](https://typesense.org) capabilities that go beyond what the local-first qmd could support

---

## Pipeline & Search Features

- **Boost title matches in hybrid search** ([qmd#669](https://github.com/tobi/qmd/pull/669)) - bump chunk weight when the query matches the document title.
- **Expose configurable chunk size / overlap** ([qmd#641](https://github.com/tobi/qmd/issues/641)) - already in schema design; ensure CLI flags exist for override.
- **Configurable FTS tokenizer** ([qmd#623](https://github.com/tobi/qmd/pull/623)) - allow setting `unicode61`, `cjk`, or custom tokenizer per collection.
- **`--no-expand` / `--no-hyde` flag** ([qmd#683](https://github.com/tobi/qmd/issues/683), [qmd#622](https://github.com/tobi/qmd/pull/622)) - disable LLM query expansion / HyDE for fast-path queries.
- **Opt-out of query expansion per collection** ([qmd#627](https://github.com/tobi/qmd/issues/627)) - useful for reference-doc corpora where term precision matters.
- **Multiple results per file** ([qmd#685](https://github.com/tobi/qmd/issues/685)) - configurable `group_limit` > 1 so top-N chunks per doc are returned.
- **Compare two texts without indexing** ([qmd#631](https://github.com/tobi/qmd/issues/631)) - ad-hoc semantic comparison utility.
- **`--context` flag to query expansion** ([qmd#628](https://github.com/tobi/qmd/issues/628)) - pass an extra context descriptor to the expansion prompt.
- **AST-aware chunking** ([qmd#676](https://github.com/tobi/qmd/pull/676), [qmd#630](https://github.com/tobi/qmd/issues/630)) - language-aware parsing for Java, JS, etc. Requires tree-sitter (Go tree-sitter, no CGO).
- **Path filtering + explain metadata in search APIs** ([qmd#665](https://github.com/tobi/qmd/pull/665)) - filter by path prefix/glob and return match-debug info.
- **Report ignored documents distinctly** ([qmd#667](https://github.com/tobi/qmd/pull/667)) - show skipped files separately in status output.
- **Strong signal detection** - already in gmd; could expose thresholds per-collection in CUE config.
- **`--low-vram` mode** ([qmd#662](https://github.com/tobi/qmd/pull/662)) - for users sharing GPU with other models. LLM-level concern but could hint at batch sizes.

---

## Indexing & Embedding

- **`collection create` with no args** ([qmd#684](https://github.com/tobi/qmd/issues/684)) - auto-create a collection from CWD without editing config.
- **`--watch` mode for periodic re-indexing** ([qmd#646](https://github.com/tobi/qmd/pull/646)) - filesystem watcher that triggers update on change.
- **Scope `update` to a single collection** ([qmd#671](https://github.com/tobi/qmd/issues/671)) - `gmd update -c <name>` to avoid re-scanning everything.
- **Exclude patterns for nested collections** ([qmd#645](https://github.com/tobi/qmd/issues/645)) - prevent double-indexing when collection paths overlap.
- **`--glob` as alias for `--mask`** ([qmd#664](https://github.com/tobi/qmd/pull/664)) - friendlier CLI UX.
- **Remote embedding / reranking / expansion backends** ([qmd#629](https://github.com/tobi/qmd/pull/629), [qmd#689](https://github.com/tobi/qmd/pull/689)) - first-class OpenAI-compatible support (already in gmd design, but worth tracking).
- **Configurable embedding session timeout** ([qmd#609](https://github.com/tobi/qmd/issues/609), [qmd#673](https://github.com/tobi/qmd/issues/673)) - expose max duration -- already in gmd as CUE config knob.
- **Keep partial embeddings pending on timeout** ([qmd#654](https://github.com/tobi/qmd/pull/654)) - avoid losing progress on large corpora.
- **Incremental indexing** - track mtime or filesystem events; skip unchanged files entirely.
- **Index snapshots / aliases** ([qmd#561](https://github.com/tobi/qmd/issues/561), [qmd#560](https://github.com/tobi/qmd/issues/560)) - useful for A/B testing config or rollback.
- **Configurable overlap as fraction or tokens** - CUE schema supports this; ensure CLI + API expose it.

---

## MCP Server

- **Add `update` and `embed` MCP tools** ([qmd#587](https://github.com/tobi/qmd/issues/587), [qmd#632](https://github.com/tobi/qmd/issues/632), [qmd#632](https://github.com/tobi/qmd/pull/632)) - let agents trigger re-indexing.
- **`--host` flag for MCP HTTP server** ([qmd#677](https://github.com/tobi/qmd/pull/677)) - bind to non-localhost for remote clients.
- **Stateless MCP HTTP transport** ([qmd#607](https://github.com/tobi/qmd/issues/607), [qmd#624](https://github.com/tobi/qmd/pull/624)) - avoid reuse-port issues.
- **MCP instructions: singular vs plural `collection`** ([qmd#606](https://github.com/tobi/qmd/pull/606)) - fix schema naming mismatch.
- **Add lifecycle commands to skill** ([qmd#605](https://github.com/tobi/qmd/pull/605)) - expose init/update/embed via MCP skill interface.
- **plugin.json for Claude Desktop** ([qmd#633](https://github.com/tobi/qmd/issues/633)) - auto-detect gmd as a third-party MCP server.
- **ServerInstructions opt-in** ([qmd#647](https://github.com/tobi/qmd/issues/647)) - make full catalogue injection optional (clutter reduction).

---

## REST API Server

- **`gmd serve` shared model server** ([qmd#663](https://github.com/tobi/qmd/pull/663)) - run LLM models as a sidecar service alongside the API.
- **Daemon-aware CLI fast-path** ([qmd#608](https://github.com/tobi/qmd/pull/608)) - ~4x speedup by reusing a running daemon's model process.
- **API key authentication** - add optional auth to the REST server for shared deployments.
- **CORS configuration** - expose via CUE config for web UIs.
- **Rate limiting** - configurable per-endpoint rate limits.

---

## Typesense-Native Features (Beyond qmd's Capabilities)

These are features Typesense offers that a local SQLite-first approach (qmd) could not support. GMD can leverage them because Typesense is the backing store.

### Typo-Tolerant / Fuzzy Search
Typesense has built-in, configurable typo tolerance (`num_typos`, `typo_tokens_threshold`). qmd had no fuzziness beyond FTS5 prefix wildcards. GMD could offer a `--fuzzy` flag or per-collection `typo_tolerance` CUE setting.

### Faceted Search & Filtering by Fields
Return aggregated counts for any facet field and filter results with `filter_by` (e.g., `language:=go`, `stars:>100`). Useful once gmd indexes project metadata alongside doc content - filter by file extension, directory, git commit, tag, etc.

### Sorting by Any Field
Typesense blends BM25 relevance and structured sort (e.g., `sort_by=modified_at:desc`). qmd could only sort by BM25 + custom post-processing. Enable e.g. `gmd query "deploy" --sort-by modified_at:desc`.

### Grouping / Distinct (Already Used in GMD)
`group_by=collection,path` is already in gmd. Could extend to `--group-by author` or `--group-by directory` for richer result organization.

### Synonym Groups
Define synonym sets server-side (e.g., `k8s` <-> `kubernetes` <-> `kube`). No reindexing needed. CUE config could carry per-collection synonym tables that get pushed to Typesense.

### Curated Rankings / Overrides
Pin specific docs to positions or boost/bury them conditionally (e.g., always return `README.md` first when query contains "intro"). In qmd this required client-side hacks.

### Real-Time Indexing
Typesense is built for concurrent writes + reads at OLTP speeds. gmd already upserts chunks in batch; could offer a `gmd watch` mode that streams file changes as they happen.

### Scoped API Keys / Multi-Tenancy
Generate API keys restricted to specific collections, fields, or filter conditions. Relevant if gmd serves multiple teams or projects from one Typesense cluster.

### Federated / Multi-Collection Search
Search across entirely separate collections in one HTTP request. GMD could let users query across unrelated codebases with a single command.

### Search Analytics
Typesense tracks popular searches, no-result queries, click-through. Useful for understanding what users look for - expose via `gmd analytics` or the REST API.

### Autocomplete / Query Suggestions
Dedicated endpoint for prefix-based query completion with frequency boosting. `gmd suggest "dep"` returning "deploy, dependency, deprecated".

### Raft Clustering / High Availability
Typesense runs as a multi-node Raft cluster with zero-downtime rolling upgrades. Relevant only if gmd is deployed as a shared service, but a differentiator vs the single-node qmd.

### Geolocation Search
Built-in geo-radius / bounding-box filtering. Niche for gmd but could be useful for indexing location-aware content (events, meetups, docs about places).

### Overrides / Merchandizing
Pin, boost, or bury results based on query conditions. E.g., "pin the getting-started guide when query contains 'beginner'". Configurable via CUE.

### Query Suggestions
Typesense can auto-complete queries from indexed data with frequency boosts. Could offer `gmd suggest` command.

---

## Deploy & Operations

- **Ship Docker container image** ([qmd#670](https://github.com/tobi/qmd/issues/670)) - multi-arch image for easy deployment.
- **`gmd doctor` vector diagnostics** ([qmd#659](https://github.com/tobi/qmd/pull/659)) - already in gmd; extend to check Typesense health + LLM endpoints.
- **Project-local indexes** ([qmd#655](https://github.com/tobi/qmd/pull/655)) - already in gmd design (`.gmd/` sentinel).
- **QMD migration tool** - already in gmd design as `gmd import-qmd`.
- **Share pre-built indexes across team** ([qmd#642](https://github.com/tobi/qmd/issues/642)) - Typesense Cloud or shared cluster as index distribution mechanism.
- **Graceful Typesense degradation** - when Typesense is down, fall back to LLM-only or print clear error guidance.

---

## Integrations & Backends

- **OpenAI-compatible backend support** ([qmd#620](https://github.com/tobi/qmd/issues/620), [qmd#619](https://github.com/tobi/qmd/pull/619)) - already core to gmd's design. Track provider-specific quirks (Ollama, vLLM, OpenAI, Groq, etc.).
- **Support local GGUF models via llama.cpp** - optional fallback for air-gapped / offline use. OpenAI-compatible proxy like `llama.cpp --server` already covers this, but a native path could be explored.
- **CJK support with n-gram FTS** ([qmd#616](https://github.com/tobi/qmd/pull/616)) - already handled by Typesense's built-in tokenizer, but worth verifying for CJK content.
- **Jina Embeddings v4 / GGUF prompt formatting** ([qmd#614](https://github.com/tobi/qmd/issues/614)) - ensure embedding models with special prompt templates are supported.
- **Apple MLX support** ([qmd#649](https://github.com/tobi/qmd/issues/649)) - macOS users may want local inference via MLX instead of the OpenAI API.

---

## CLI & UX

- **`--glob` as alias for `--mask`** ([qmd#664](https://github.com/tobi/qmd/pull/664)) - friendlier UX.
- **`status --json` emitting actual JSON** ([qmd#594](https://github.com/tobi/qmd/issues/594)) - already supported in gmd; verify CLI consistency.
- **Support `--index` flag on MCP subcommand** ([qmd#691](https://github.com/tobi/qmd/issues/691)) - pass typesense index config to MCP mode.
- **Better error messages on missing config / Typesense down** - already planned in Phase 7.

---

## Markdown Frontmatter - Extraction & Faceted Search

GMD currently does **not** parse frontmatter. Markdown files with YAML/TOML/JSON frontmatter (delimited by `---`) are treated as plain content -- the frontmatter block is chunked and indexed alongside body text, making it searchable but unstructured.

Typesense's faceted search makes this a high-value opportunity.

### Frontmatter Extraction Pipeline

At indexing time, parse the frontmatter block, strip it from chunk content, and store selected fields as typed Typesense document fields:

```
Raw file                  Typesense document
----------                ------------------
---                        {
  title: "My Doc"            collection: "docs"
  tags: [go, search]         path: "my-doc.md"
  author: alice              title: "My Doc"
  status: published          content: "..."         <- no frontmatter
  difficulty: 3              hash: "abc123"
---                          tags: ["go", "search"]    <- string[], facet: true
# My Doc                     author: "alice"           <- string, facet: true
This is the...               status: "published"       <- string, facet: true
                              difficulty: 3             <- int32, sortable
```

### Per-Collection CUE Configuration

```cue
collections: myapp: {
  path:    "docs"
  pattern: "**/*.md"
  context: "MyApp user documentation"
  frontmatter: {
    fields: {
      tags:       { type: "string[]", facet: true }
      author:     { type: "string",   facet: true }
      status:     { type: "string",   facet: true }
      difficulty: { type: "int32",    sort: true }
      version:    { type: "string" }
    }
  }
}
```

The CUE schema would validate known field types and map them to the Typesense schema at index time. Unknown fields in the YAML frontmatter could be dropped or stored in a generic `string[] all_tags` field.

### What This Enables

| Capability | Example |
|-|-|
| Filter by tag | `gmd query "deploy" --filter "tags:=go"` |
| Filter by status | `gmd query "search" --filter "status:=published"` |
| Filter by author | `gmd query "auth" --filter "author:=alice"` |
| Sort by difficulty | `gmd query "setup" --sort-by difficulty:asc` |
| Sort by version | `gmd query "api" --sort-by version:desc` |
| Multi-filter | `--filter "tags:=go && difficulty:>2"` |
| Facet counts | Results include `{"tags": {"go": 42, "python": 15}}` |
| Facet drill-down | `--facet tags` to show tag distribution |
| Group by status | `--group-by status` to cluster results |

### Tags / Labels - Specific Ideas

- **`tags`** - general-purpose categorization surfaced as a faceted `string[]` field
- **`labels`** - could alias `tags` or be a separate field for more structured taxonomies
- **Auto-tagging from directory** - if no tags in frontmatter, infer from parent directory name (e.g., `docs/api/` -> tag `api`)
- **Collection-level tag taxonomies** - define allowed tags in CUE config for validation and autocomplete hints
- **Weighted tags** - tags with numeric values (e.g., `tags: [go:5, search:3]`) that influence ranking (overrides + custom scoring)
- **Tag-based overrides / boosting** - Typesense overrides to pin or boost docs with specific tags for given queries

### Frontmatter Fields as Omitted Content

The frontmatter block itself should be **removed from the chunk content** before embedding and indexing, so search results don't return `---\ntitle:...\n---` noise. The extracted fields serve as structured metadata on the Typesense document. A `frontmatter_raw` text field could optionally preserve the full block for LLM context.

### Frontmatter-Aware Chunking

The chunking boundary logic should skip lines between the opening `---` and closing `---` (or `...`) delimiters. These lines should not count toward token budgets or produce segments.

---

## Cross-Cutting Concerns

- **LLM response caching** ([qmd#578](https://github.com/tobi/qmd/issues/578)) - cache expansion + rerank responses to reduce latency/cost on repeated queries.
- **Benchmark harness** ([qmd#621](https://github.com/tobi/qmd/issues/621)) - port from qmd's `src/bench/`; compare recall/precision/latency across configs.
- **Transaction safety for partial failures** - all-or-nothing per-file upsert; retry + backoff for LLM API errors.
- **Multi-get max bytes** ([qmd#666](https://github.com/tobi/qmd/pull/666)) - expose Typesense `multi_get` max response size.
- **Concurrent writer `busy_timeout`** ([qmd#686](https://github.com/tobi/qmd/pull/686)) - Typesense handles this natively but worth noting for embedding pipeline concurrency.
- **Model config documentation** ([qmd#678](https://github.com/tobi/qmd/issues/678)) - document how to configure embedding/expansion/rerank models in CUE.

---

## LLM Wiki Integration (Karpathy Pattern)

Andrej Karpathy's [LLM Wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) (April 2026) proposes a pattern where an LLM incrementally builds and maintains a persistent, interlinked wiki from raw sources — shifting from query-time RAG retrieval to a compounding knowledge artifact. Karpathy explicitly calls out **qmd** as the recommended search engine for wikis that outgrow a single `index.md`. This is a natural integration point for GMD.

### The Pattern in Brief

Three layers:
- **Raw sources** — immutable originals (articles, papers, transcripts)
- **The wiki** — LLM-generated markdown (entities, concepts, comparisons, synthesis) with `[[wikilinks]]`
- **The schema** — instructions (CLAUDE.md/AGENTS.md) governing ingest/query/lint workflows

Two special files: `index.md` (content catalog) and `log.md` (chronological record). At small scale the LLM navigates via the index; beyond ~100 sources a search engine becomes necessary.

### Where GMD Fits

| Role | Tool |
|---|---|
| Wiki search engine | GMD (via CLI and MCP) |
| Wiki content indexing | `gmd collection create wiki/` |
| Agent integration | `gmd mcp` exposes search as native tool |
| Embedding + rerank | GMD's LLM pipeline for hybrid search |

A typical LLM wiki agent workflow:
1. Schema instructs agent to use GMD's MCP tools for `search`, `multi-get`
2. On query, agent calls GMD search instead of reading `index.md` linearly
3. GMD returns ranked chunks with paths → agent reads full pages
4. On ingest, agent triggers `gmd update wiki/` to keep index current

### Integration Ideas

#### Auto-index on ingest
- Schema instructs LLM to run `gmd update wiki/` after every ingest
- Optionally scoped: `gmd update -c wiki-<project>`
- Keeps the search index synchronized without manual steps

#### Query pipeline via GMD MCP
- Agent calls `gmd mcp` search tool with the user's question
- GMD returns ranked chunks with paths
- Agent reads top-N full pages and synthesizes answer
- Answer can be filed back as a new wiki page (compounding)

#### Wiki-specific GMD commands
- `gmd wiki init` — bootstrap a Karpathy-style wiki directory (raw/, wiki/, CLAUDE.md skeleton)
- `gmd wiki lint` — run GMD search + LLM to surface contradictions, orphans, stale claims
- `gmd wiki ingest <path|url>` — drop source, trigger re-index
- `gmd wiki status` — page count, source count, embedding coverage

#### Structured schema fields
Wiki pages carry YAML frontmatter (type, tags, source refs). GMD's frontmatter extraction (see section above) would index these as typed, facetable fields — enabling filter-by-type, tag drill-down, sort-by-date.

#### MCP tool additions
Beyond the current `search`/`get`/`multi-get` tools:
- `search-wiki` — scoped to wiki collections, returns page paths with summaries
- `wiki-query` — hybrid search + LLM rerank tailored for wiki navigation
- `graph-neighbors` — given a page, return linked pages (via [[wikilinks]]) for graph traversal

#### Use case: research deep-dive
A user reading papers on a topic over weeks:
1. Drops PDF summary into `raw/` → tells agent to ingest
2. Agent reads, writes concept page, updates entities, cross-references
3. Agent runs `gmd update wiki/` to refresh index
4. Next session: user asks a cross-cutting question
5. Agent searches via GMD MCP, finds relevant pages across 5 sources
6. Synthesizes answer, files as new synthesis page
7. Lint pass catches contradiction between two sources → flags for user review

#### Related ecosystem
The LLM Wiki idea has spawned dozens of implementations. Notable for GMD alignment:
- **llm-wiki-compiler** (atomicstrata/llmwiki) — 1.3k★, TS compiler + MCP server, explicit GMD/qmd mention
- **WikiMind** (HAL-9909) — BM25-only, no embeddings, uses qmd as sole search backend
- **claude-obsidian** (AgriciDaniel) — Claude Code plugin, git-backed, pure markdown
- **OmegaWiki** (skyllwt) — 740★, 26 Claude Code skills for paper lifecycle
- **llm-wiki-manager** (sametbrr) — 8 operating modes, Python bookkeeping scripts
- **Synthadoc** (axoviq-ai) — adversarial review + claim-level provenance
- Various CLAUDE.md template repos — single-file bootstrap, zero dependencies

### Priorities

1. **MCP integration first** — GMD's existing MCP server is already usable; document the LLM wiki use case in `gmd mcp` help and examples
2. **Auto-index on ingest** — lightweight script or MCP tool that runs `gmd update` after wiki content changes
3. **`gmd wiki` subcommands** — if the pattern gains traction, dedicated init/lint/ingest commands
4. **Frontmatter-aware wiki indexing** — combine with the frontmatter extraction section above for facetable wiki metadata
