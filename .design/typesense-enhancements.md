# typesense-enhancements

created: 2026-06-14 | phase: brainstorming (0 — ideation)

## Context

gmd sits on top of Typesense (v26-era APIs in the active codebase, v30 docs now analyzed). Typesense v30 ships
substantial new surface area: conversational RAG, natural language search, JOINs, curation sets, synonym sets,
stemming dictionaries, analytics, federation enhancements, MMR diversity, and more. Meanwhile IDEAS.md catalogs
60+ feature ideas drawn from upstream qmd issues/PRs and Typesense-native capabilities, and SCRATCH.md surfaces
additional developer notes on agent tooling, SQLite, and operational gaps.

**Sources analyzed:**
- `docs/typesense/` — full API reference (v30.2) + usage guides (106 files)
- `IDEAS.md` — feature/enhancement catalog (318 lines)
- `SCRATCH.md` — developer scratch notes (94 lines)

**Goal of this document:** Brainstorm what Typesense enhancements gmd could adopt, in what areas, and with what
trade-offs. This is the divergence phase — we are generating options, not narrowing them. Subsequent `.design/`
documents will drill into specific enhancements with implementation detail.

---

## 1. Search Quality & Relevance

### 1.1 Synonym Sets

Typesense v30 moved synonyms to global `synonym_sets` resources, linkable to multiple collections. Supports
one-way (canonical root) and multi-way (interchangeable) synonym groups, locale-specific synonyms, and
per-search dynamic overrides.

**Options:**
- **A — Built-in gmd synonyms.** Ship a curated set of technical synonym groups (k8s<->kubernetes,
  LLM<->large language model, RAG<->retrieval augmented generation, etc.) as an embedded defaults file
  that `gmd init` optionally applies.
- **B — Per-collection/per-wiki CUE config.** Expose `synonyms: { items: [...] }` in CUE under
  `collections.<name>` and `wikis.<name>`, synchronized to Typesense synonym_sets on `gmd update`.
- **C — User-managed synonym files.** Point at external JSONL synonym dictionaries (e.g. from Algolia's
  public repo) and push them at init time.
- **D — LLM-generated synonym suggestions.** "gmd doctor" or a new `gmd suggest-synonyms` command that
  uses an LLM to scan collection content and propose synonym groups.

**Open questions:**
- Should synonyms be global (shared across all collections) or per-collection? Typesense supports both.
- One-way vs multi-way: for technical docs, which direction makes sense? (e.g. should "k8s" expand to
  "kubernetes" but not the reverse?)
- How do we handle synonym conflicts between collections that share a Typesense synonym_set?

### 1.2 Curation / Overrides

v30 renamed overrides → curation_sets (global resources). Rules can pin/hide documents, apply dynamic
filters, dynamic sorting, replace queries, remove matched tokens, and diversify via MMR. Supports
time-bounded rules (`effective_from_ts`/`effective_to_ts`), rule chaining (`stop_processing`), tagging,
and `metadata` passthrough.

**Options:**
- **A — Pinned key docs.** Pin AGENTS.md, index pages, or wiki overview pages for certain query
  patterns (e.g. "getting started" → pin the project README).
- **B — Dynamic collection filtering.** Curation rule `{collection} search` → auto-filters to that
  collection's content, removing "collection" from the query tokens.
- **C — MMR diversity across collections.** When searching multiple collections/wikis simultaneously,
  use MMR to ensure results aren't all from one source (diversity over collection name, source path).
- **D — Time-bounded promotions.** Promote sprint docs, release notes, or temporary guides for a
  limited time window, then auto-unpin.
- **E — CUE-driven curation config.** `curation` block in CUE config, synced to Typesense on each
  `gmd update`.

**Open questions:**
- How much curation belongs in Typesense vs in gmd's own search pipeline (position blending, LLM rerank)?
- MMR diversity: should it be on by default for multi-collection queries? What fields make sense for
  similarity computation (collection, source path, tags)?
- Dynamic filtering with `{placeholder}` tokens: could this replace gmd's current query expansion +
  filter pipeline in some cases?

### 1.3 Stemming

Two approaches: field-level Snowball stemming (locale-aware) and custom stemming dictionaries (JSONL).

**Options:**
- **A — Enable Snowball stemming by default.** Simple quality win for English-language markdown.
- **B — Ship a pre-made plurals dictionary.** Typesense publishes a free English plurals dictionary;
  apply it on `gmd init` for all collections.
- **C — Domain-specific stemming dictionary.** Custom dictionary for gmd-specific terms
  (indexed→index, chunked→chunk, embedded→embed, reindexing→index).
- **D — Per-collection stemming config in CUE.** `stem: true | false`, `stem_dictionary: "name"`.

**Open questions:**
- Does stemming interact poorly with code blocks in markdown? (e.g. "indexed" in code vs prose)
- Locale: should we auto-detect language per-collection or assume English by default?

### 1.4 Typo Tolerance & Drop Tokens

Typesense's typo tolerance and `drop_tokens_threshold`/`drop_tokens_mode` are already active in gmd's
search path but not configurable.

**Options:**
- **A — Expose per-collection typo config in CUE.** `typo_tolerance: { num_typos: 2, min_len_1typo: 4 }`
- **B — `--fuzzy` / `--strict` search flags.** CLI shortcuts for user-adjustable fuzziness.
- **C — `drop_tokens_mode` tuning.** Configure how Typesense handles no-results fallback (drop
  individual tokens vs all at once).

### 1.5 Ranking & Relevance Tuning

Typesense supports: `text_match_type` (max_score/max_weight/sum_score), popularity-recency bucketing,
`_eval()` conditional scoring, `pinned_hits`/`hidden_hits`, `prioritize_exact_match`.

**Options:**
- **A — Popularity-weighted ranking.** Use analytics `counter` rules (see section 3) to increment a
  `popularity` field on frequently-accessed chunks, then sort by `_text_match(buckets):desc,popularity:desc`.
- **B — Recency boosting.** If frontmatter carries `date`/`updated` fields, boost newer content.
- **C — `pinned_hits` as lightweight curation.** Simpler than full curation_sets for one-off pinning.
- **D — Conditional ranking via `_eval()`.** Boost chunks from specific directories or document types
  (e.g. API docs > blog posts for "how to deploy" queries).

**Open questions:**
- How does gmd's LLM rerank step interact with Typesense-level ranking? Should LLM rerank be the
  final pass after Typesense ranking, or should we lean more on Typesense-native ranking?
- The `_text_match(buckets)` approach: does this complement or conflict with gmd's position blending?

---

### 1.6 Text Processing & Tokenization

Beyond stemming, Typesense offers token-level control that matters for markdown indexing, especially
code-heavy or multilingual content.

**Options:**
- **A — Stopwords.** Typesense supports global stopword sets (linkable to collections). For gmd,
  stopwords could reduce noise from high-frequency terms like "the", "how", "what", "is", "a"
  that dominate markdown text but add little relevance signal. Pre-made stopword lists exist for 50+
  languages. Expose via `stopwords` in CUE collection config.
- **B — Token separators.** By default Typesense splits on whitespace/punctuation. Code-heavy docs
  need `symbols_to_index: ["+", "#", "_"]` to make terms like `C++`, `foo_bar`, `#tag` searchable.
- **C — Infix search.** `infix: true` enables substring matching anywhere in tokens (e.g. "config"
  matches "reconfigure"). Useful for finding partial identifiers in code docs. Trade-off: larger
  index size, slower indexing. Per-field opt-in.
- **D — CJK support.** Typesense's built-in tokenizer handles CJK natively. `pre_segmented_query`
  accepts pre-tokenized queries for languages without explicit word boundaries.
- **E — Per-collection tokenizer config in CUE.** `tokenizer: { separators: [...], symbols_to_index: [...], infix: true }`

**Open questions:**
- Should stopwords be applied globally (all collections) or per-collection? Technical docs might
  need different stopwords than general prose.
- Infix search for code: does it meaningfully improve search recall, or is prefix search (default)
  sufficient?
- How does `symbols_to_index` interact with markdown syntax characters like `#` (headings), `_`
  (emphasis), `*` (lists)? Stripping markdown formatting before indexing may be necessary.

---

## 2. Structured Metadata & Faceting

### 2.1 Frontmatter Extraction

IDEAS.md has a comprehensive section on extracting YAML/TOML/JSON frontmatter into typed Typesense
fields. This is the highest-value structural enhancement because it unlocks faceting, filtering,
sorting, and grouping.

**Options:**
- **A — Full frontmatter pipeline.** Parse frontmatter at index time, strip from chunk content,
  store as typed fields (`string`, `string[]`, `int32`, `float`, `bool`), configure via CUE.
- **B — Simple key-value passthrough.** Extract all frontmatter keys into a `metadata` object field
  without per-field typing. Less powerful but zero config.
- **C — Selective field extraction.** User configures which frontmatter keys to extract; unknown
  keys are dropped or stored in a generic `all_tags` field.
- **D — Auto-detect from content.** First `gmd update` scans frontmatter across all files and
  proposes a schema (field names + types) for user review. Alternatively, use Typesense's `auto`
  schema type to let Typesense detect field types from document content — a middle ground
  between fully typed and unstructured.

**Open questions:**
- Should frontmatter extraction be opt-in (per-collection) or automatic?
- What happens when frontmatter schema changes? Typesense v30 supports `PATCH /collections` to
  add new `optional` fields but not remove or retype existing ones. gmd may need a migration path.
- Nested frontmatter (YAML objects): flatten or store as `object`? Typesense supports `object`
  and `object[]` field types with nested field search/filter support.
- Frontmatter in wikis vs collections: same pipeline, different configured fields?
- How does `dirty_values` (Typesense's tolerance for schema mismatches) interact with
  frontmatter extraction when field types change?

### 2.2 Faceting

Once frontmatter fields are indexed, Typesense's `facet_by` returns aggregated counts for faceted
navigation. `facet_query` supports drill-down, `facet_sample_threshold`/`facet_sample_slope`
controls accuracy vs speed.

**Options:**
- **A — `--facet tags` CLI flag.** Return tag counts alongside search results.
- **B — Faceted sidebar in `gmd serve` UI.** Collection, source, author, status facets in a future
  web interface.
- **C — Multi-facet queries.** `gmd query "deploy" --facet tags,status,author`
- **D — Facet-driven content discovery.** "What topics are covered in this wiki?" via tag faceting.

**Open questions:**
- Faceted search is more UI-centric. Does it make sense for a CLI-first tool?
- Facet sampling: for large collections, approximate counts are fine. What's the right default?

### 2.3 Filtering by Structured Fields

`filter_by` supports exact match (`:=`), contains (`:`), negation (`:!=`, `:!`), range (`:[min..max]`),
and boolean operators (`&&`, `||`).

**Options:**
- **A — `--filter` flag on search/query/vsearch.** `gmd query "deploy" --filter "tags:=go && difficulty:>2"`
- **B — CUE-configured default filters per collection.** Always exclude draft/unpublished docs.
- **C — `--filter` with shorthand.** `--filter-tag go` expands to `--filter "tags:=go"`.
- **D — Filter by path/glob.** Already in IDEAS.md from qmd#665.

**Open questions:**
- Filter syntax: mirror Typesense's native `filter_by` syntax or provide a friendlier abstraction?
- Do we need to escape special characters in filter values? Typesense uses backtick escaping.

### 2.4 Grouping Beyond Document-Level

gmd already uses `group_by=collection,path` to collapse chunks into documents. Typesense supports
`group_limit > 1` (multiple results per group) and grouping by any field.

**Options:**
- **A — Multiple chunks per document.** `--group-limit 3` returns top 3 chunks per file.
- **B — Group by author, directory, status.** `--group-by author` clusters results.
- **C — Group by wiki source.** For wiki queries, group results by which source document they came from.

**Open questions:**
- Multiple chunks per document: how does this interact with LLM rerank context windows?
- Does grouping by arbitrary fields require those fields to be indexed as `facet: true`?

---

## 3. Search Intelligence & Analytics

### 3.1 Analytics — Popular Queries

Typesense's built-in analytics (`--enable-search-analytics`) can aggregate popular queries,
no-hit queries, and user interaction events (clicks, conversions, visits). Requires server-side
flags and `--analytics-dir`.

**Options:**
- **A — `gmd analytics` command.** Show popular queries, no-hit queries, trending topics.
- **B — Auto-enable analytics on `gmd init`.** Create analytics rules and destination collections
  automatically for all gmd collections.
- **C — No-hit queries → wiki ingest pipeline.** Surface content gaps detected by `nohits_queries`
  as candidates for new wiki pages or source documents.
- **D — Meta-field analytics.** Track `filter_by` and `analytics_tag` to segment query patterns
  by collection, wiki, or user context.

**Open questions:**
- Analytics requires Typesense server flags. Should gmd `serve` or `init` configure these?
- Privacy: does analytics tracking conflict with single-user, local-first use cases?
- The 4-second pause heuristic for typeahead queries: does gmd need this, or are queries discrete?

### 3.2 Counter Events & Popularity Ranking

`counter` analytics rules increment numeric fields on documents based on click/conversion/visit
events, enabling popularity-weighted ranking.

**Options:**
- **A — Click-through popularity.** Track which search results users `gmd get` after a query,
  increment a `popularity` field, use for ranking.
- **B — Weighted events.** `gmd get` = weight 1, `gmd multi-get` = weight 2 (more intentional).
- **C — Time-decayed popularity.** Periodic job that decays older popularity scores to favor recency.

**Open questions:**
- For a CLI tool, what constitutes a "click"? `gmd get` after a search? Opening a file?
- Is popularity ranking useful for personal knowledge bases, or does it require multi-user scale?

### 3.3 Query Suggestions / Autocomplete

Typesense's `prefix` search (already default) enables search-as-you-type. The analytics
`popular_queries` collection enables query suggestions based on history.

**Options:**
- **A — `gmd suggest <prefix>` command.** Query the popular_queries collection for completions.
- **B — API endpoint for autocomplete.** `GET /api/suggest?q=dep` returns ["deploy", "dependency"].
- **C — MCP suggest tool.** AI agents get query suggestions alongside search results.

**Open questions:**
- Is autocomplete valuable for a CLI tool, or is this purely for web UI / MCP use cases?

### 3.4 A/B Testing

Typesense doesn't provide built-in A/B testing, but the guide describes methodology: vary search
parameters (query_by weights, alpha, curation rules, synonym sets) per bucket.

**Options:**
- **A — LLM profile A/B testing.** Compare ranking quality across different embedding models
  or LLM providers.
- **B — Curation rule A/B testing.** Test whether curated results improve user satisfaction.
- **C — Benchmark harness.** Port qmd's `src/bench/` infrastructure (IDEAS.md references this)
  to compare recall/precision/latency across configurations.

---

## 4. Data Modeling & Schema Architecture

### 4.1 JOINs / References

Typesense JOINs allow modeling relationships between collections: `reference` fields link documents,
supporting one-to-one, one-to-many, many-to-many, nested joins, sorting on joined fields, and
asynchronous references.

**Options:**
- **A — Chunk → Document explicit reference.** Instead of the current implicit group_by path,
  create a `documents` collection and a `chunks` collection with a `reference: documents.id`.
  JOIN at query time to fetch parent document metadata.
- **B — Wiki → Source reference.** Model wiki pages as referencing their source collections,
  enabling JOIN-based provenance queries.
- **C — Nested JOIN for wiki graph.** `wiki_page → $source_chunks($parent_documents(*))` to
  traverse the full content lineage.
- **D — `related_docs_count`.** Show how many chunks a document has, or how many wiki pages
  reference a source.

**Open questions:**
- Does introducing JOINs simplify gmd's Go-side merge logic, or add complexity?
- The current single-collection design is simple. Is the added data modeling power worth the
  migration cost?
- `async_reference`: if references are eventually consistent, does that break search consistency?

### 4.2 Collection Aliases

Aliases are virtual names that point to real collections, enabling zero-downtime reindexing
(background index into `foo_v2`, atomically swap alias `foo` → `foo_v2`).

**Options:**
- **A — `gmd update` with alias swap.** On reindex, create a temporary collection, index into
  it, then swap alias for atomic cutover.
- **B — `gmd collection alias` subcommand.** Manual alias management for power users.
- **C — Blue/green wiki deployments.** Each wiki has a "live" alias and a "staging" collection.

**Open questions:**
- How does the current gmd collection name → Typesense collection name mapping work? Does it
  already support aliases or does it directly reference collection names?
- Alias swap during `gmd update`: would this double storage during reindex (old + new collection)?

### 4.3 Schema Evolution & Partial Updates

v30 supports `PATCH /collections` to add new `optional` fields and modify certain properties.
Frontmatter extraction would benefit from this. Additionally, Typesense supports partial document
updates (`action=update`) — field-level changes without replacing the full document.

**Options:**
- **A — Auto-detect new frontmatter fields on `gmd update`.** Patch collection schema to add
  newly discovered fields.
- **B — `gmd collection migrate` command.** Explicit schema migration with validation.
- **C — Schema versioning.** Track collection schema versions in CUE config, enabling rollback.
- **D — Partial document updates for watch mode.** When a file changes (IDEAS.md #646),
  update only the affected chunks rather than re-indexing the entire file.
- **E — `dirty_values` configuration.** Control how Typesense handles schema mismatches
  during indexing (coerce, drop, reject).

**Open questions:**
- Typesense can add optional fields but can't remove them or change types. How does gmd handle
  frontmatter schema drift?
- If a field is removed from CUE config, do we leave the Typesense field orphaned or warn the user?
- Partial updates: does this conflict with gmd's content-addressable dedup strategy (SHA-256
  hash per chunk)?

---

## 5. Search APIs — Deep Integration

### 5.1 Conversational Search (RAG) — Built into Typesense

Typesense v30 has built-in RAG: create a conversation history collection, configure a conversation
model (OpenAI/Azure/Google/Cloudflare/vLLM), then search with `conversation=true`. Typesense handles
standalone question generation, context management, multi-turn conversation history, and SSE streaming.

**Options:**
- **A — Replace gmd's LLM synthesis with Typesense RAG.** Use Typesense's `conversation=true` for
  `gmd query --rag` mode, removing the Go-side synthesis LLM call.
- **B — Complement, not replace.** Keep gmd's expansion + rerank pipeline, use Typesense RAG only
  for the final answer generation step.
- **C — Multi-turn wiki Q&A.** `gmd wiki chat <name>` — interactive session using Typesense
  conversational search with conversation history.
- **D — Streaming output.** `gmd query --stream` for real-time token-by-token output via SSE.

**Open questions:**
- Does Typesense RAG handle prompt construction (system prompt + retrieved chunks) the way gmd wants?
  gmd's current prompt templates may differ from what Typesense generates.
- Conversation history: Typesense manages this server-side. Is that better or worse than gmd
  managing it client-side (no server state)?
- Model support: does Typesense RAG support the same provider set as gmd's LLM profiles? (GCP
  service accounts, etc.)
- If Typesense does RAG, what's the role of gmd's own LLM client? Does it become purely for
  embeddings and query expansion?

### 5.2 Natural Language Search

Typesense's `nl_query=true` uses an LLM to parse free-form queries into structured search params
(filters, sorts, query text). The LLM receives the collection schema as context.

**Options:**
- **A — `gmd nlquery "powerful docs about search"` command.** Wraps `nl_query`, returns
  structured parameters and results.
- **B — Automatic NL mode.** `gmd query` detects natural language patterns and auto-applies
  `nl_query=true`.
- **C — Schema-aware NL prompts.** Since gmd knows the collection schema (fields, facets),
  feed it to the NL model for better parsing accuracy.

**Open questions:**
- NL search requires an LLM call per query. Is the latency acceptable for CLI use?
- Schema prompt caching (configurable TTL): how do we keep the cached schema in sync with
  collection changes?
- How does NL search interact with gmd's own query expansion? Two LLM calls per query may be
  excessive.

### 5.3 Federated Search & Union

Typesense's `multi_search` supports `union: true` to merge results from multiple collections into
a single ranked list. v30 adds `remove_duplicates`, `pinned_hits` in union, and `group_by`.

**Options:**
- **A — Replace gmd's RRF fusion with Typesense union.** Instead of Go-side Reciprocal Rank Fusion,
  use `union: true` + `remove_duplicates: true` and let Typesense merge across collections.
- **B — Union with per-collection weighting.** Adjust per-collection result weights via `query_by`
  weights or curation rules.
- **C — Union over wikis.** Search multiple wikis simultaneously with dedup.

**Open questions:**
- Typesense union sorting requires uniform field types across collections. Do gmd's collections
  have consistent field schemas?
- Is Typesense's union ranking (per-collection relevance scores) comparable in quality to gmd's
  RRF + LLM rerank?

### 5.4 Vector Auto-Embedding

Typesense can auto-generate embeddings internally using ONNX models (`ts/all-MiniLM-L12-v2`,
`ts/e5-small`, CLIP), OpenAI, Azure, Google PaLM, GCP Vertex AI, or any OpenAI-compatible API.

**Options:**
- **A — Offload embedding to Typesense entirely.** Remove gmd's Go-side embedding pipeline.
  Configure embedding model at collection creation time.
- **B — Hybrid approach.** Keep Go-side embeddings for custom pipeline control, use Typesense
  auto-embedding for offline/local mode (ONNX models don't need an API key).
- **C — Model flexibility.** Allow per-collection embedding model selection (e.g. code-specific
  models for source repos, general models for prose docs).

**Open questions:**
- Typesense's auto-embedding uses its own ONNX runtime. Does this require a different Typesense
  binary (GPU vs CPU)?
- `matryoshka` dimension reduction: should gmd use this for storage efficiency?
- How does auto-embedding interact with gmd's content-addressable dedup? Typesense re-embeds
  on every upsert, but gmd only upserts changed files.
- If Typesense handles embeddings, gmd's embedding LLM provider becomes optional. Is that
  desirable (fewer dependencies) or limiting (less control)?

### 5.5 Query Performance Tuning

Typesense offers parameters to control the speed vs precision trade-off. For a CLI tool where
perceived latency matters, these are directly relevant.

**Options:**
- **A — `search_cutoff_ms`.** Hard timeout for search; Typesense returns best results found so far.
  Could be exposed as `--timeout-ms 200` for fast-first-byte queries.
- **B — `exhaustive_search`.** Trades speed for accuracy. Default is false (approximate). Expose
  as `--exhaustive` flag for when precision matters more than speed.
- **C — `max_candidates`.** Cap number of candidate documents considered. Combined with gmd's
  `per_page`, could bound search cost.
- **D — `remote_wait_ms`.** Time to wait for slow replicas in HA setups. Less relevant for
  single-node but important for clustered deployments.
- **E — `use_cache`.** Enable server-side query cache for repeated searches. High value for
  shared gmd deployments where multiple users search similar topics.
- **F — `drop_tokens_mode`.** Configure no-results fallback behavior (drop individual tokens
  vs drop all at once). Already surfaced in 1.4 but the mode selection is a tuning decision.

**Open questions:**
- What's an acceptable default `search_cutoff_ms` for CLI use? 200ms? 500ms?
- Does `use_cache` cause staleness issues after `gmd update`? Typesense should invalidate on
  document changes, but worth verifying.

### 5.6 Search Output & UX

How Typesense returns results affects the gmd CLI, REST API, and MCP server display quality.

**Options:**
- **A — Highlighted snippets.** Typesense `highlight_fields` and `highlight_full_fields` return
  matched text with HTML `<mark>` tags. gmd could convert these to ANSI bold/color for CLI output
  or render properly in REST responses. `highlight_affix_num_tokens` controls context around matches.
- **B — Snippet threshold.** `snippet_threshold` controls when snippets are returned vs full text.
  gmd already blends chunks; this would let Typesense return only the most relevant paragraph.
- **C — Configurable `per_page` / pagination.** Currently hardcoded; expose as `--limit`/`--page`
  on search commands.
- **D — Include/exclude fields in results.** `include_fields`/`exclude_fields` already used by gmd
  but could expose `--fields` to let users control output format (e.g. hide hash, show embedding dims).
- **E — Max hits / exhaustiveness.** Typesense `max_hits` caps total retrievable results.
- **F — Trigger `--conversation` output mode.** If Typesense conversational RAG is enabled, format
  the answer prominently alongside search results.

**Open questions:**
- Highlighted snippets: how much markdown formatting should survive in snippet output? Raw markdown
  snippets may be ugly in CLI.
- Pagination: useful for CLI (`gmd query | less`) or mostly an API concern?

---

## 6. Operational & Infrastructure

### 6.1 Backups & Restore

Typesense snapshot API (`POST /operations/snapshot`) creates point-in-time backups.

**Options:**
- **A — `gmd backup` command.** Snapshot → tar → local/remote path.
- **B — `gmd restore` command.** Stop Typesense, extract snapshot, restart.
- **C — CUE-configured backup schedule.** Periodic snapshot via cron/timer.
- **D — Pre-upgrade snapshot.** Auto-snapshot before `gmd` version upgrades that change schema.

### 6.2 Scoped API Keys & Multi-Tenancy

Typesense supports fine-grained scoped API keys with per-action, per-collection (regex),
embedded `filter_by`, `limit_hits`, and `expires_at`.

**Options:**
- **A — Multi-user gmd deployments.** Each user gets a scoped key limited to their collections.
- **B — Read-only search keys for MCP/API consumers.** Clients get keys with only `documents:search`
  and `documents:get`, scoped to specific wikis.
- **C — `gmd serve` authentication.** Require API keys for REST API access, with configurable scopes.
- **D — Key rotation.** `gmd keys rotate` command with `expires_at`-based lifecycle.

**Open questions:**
- Is multi-tenancy a realistic use case for gmd, or is it primarily single-user?
- Scoped keys embedded with `filter_by`: could this enable row-level security (e.g. user A can
  only search documents tagged "team-a")?

### 6.3 Cluster Operations & Monitoring

Typesense exposes `/health`, `/metrics.json`, `/stats.json`, `/debug`, and `/operations/*`.

**Options:**
- **A — `gmd status` cluster health.** Extend `gmd status` to include Typesense health, disk usage,
  memory pressure.
- **B — `gmd doctor` Typesense diagnostics.** Check server reachability, version compatibility,
  collection health alongside existing checks.
- **C — Metrics export for monitoring.** Expose Typesense `/metrics.json` through `gmd serve`
  for Prometheus scraping.
- **D — Slow query logging.** Toggle `--log-slow-requests-time-ms` and surface in `gmd status`.

### 6.4 High Availability & Clustering

Typesense runs as a Raft cluster (3+ nodes) with automatic replication and failover.

**Options:**
- **A — Multi-node gmd deployment docs.** Document how to set up gmd + Typesense cluster for
  team use.
- **B — `gmd serve` multi-node.** Configure gmd to connect to a Typesense cluster with failover.
- **C — gmd Cloud / SaaS.** Long-term vision: hosted gmd backed by Typesense Cloud.

**Open questions:**
- Does HA matter for gmd's primary use case (personal/team knowledge bases)?
- gmd's Typesense client already supports multi-node config. Is this sufficient or do we need
  more cluster-awareness in gmd's logic?

### 6.5 Typesense Cloud Integration

Typesense Cloud provides managed clusters, automatic upgrades, analytics, and search delivery
networks. gmd could position Typesense Cloud as the recommended backend for team deployments.

**Options:**
- **A — First-class Typesense Cloud support.** Document `gmd init` workflow for Typesense Cloud
  (cluster URL + API key input, auto-configure proxy if needed).
- **B — Proxy configuration.** Corporate environments may require an HTTP proxy to reach
  Typesense (SCRATCH.md mention). Expose proxy config in CUE.
- **C — Index sharing via Cloud.** IDEAS.md #642 proposes sharing pre-built indexes across
  teams. Typesense Cloud clusters + snapshot sharing make this feasible.
- **D — Search Delivery Network.** Typesense's SDN distributes indexes globally. Niche but
  relevant for distributed team wikis.

### 6.6 Bulk Import / Export & Data Migration

Typesense supports document import/export in JSONL format, which intersects with gmd's
indexing pipeline and potential migration needs.

**Options:**
- **A — `gmd export` command.** Export a collection or wiki as Typesense-compatible JSONL
  for backup, migration, or sharing.
- **B — `gmd import` command.** Bulk-import from JSONL (useful for migration from qmd or
  other search tools).
- **C — Incremental reindex strategies.** Partial document updates (4.3-D), alias-based
  blue/green deployment (4.2-A), and snapshot-based migration paths.

**Open questions:**
- How does gmd's content-addressable dedup (SHA-256 hash) interact with bulk import? Imported
  documents would need hash recomputation.

---

## 7. Wiki-Specific Enhancements

gmd's wiki is the most differentiated feature relative to qmd and other markdown search tools.
Typesense v30 capabilities can directly strengthen the wiki's ingest → query → maintain cycle.

### 7.1 Conversational Wiki Q&A

Combine Typesense RAG with wiki content for multi-turn conversational Q&A.

**Options:**
- **A — `gmd wiki chat <name>`.** Interactive chat session with a wiki, powered by Typesense
  conversational search with `conversation_id` tracking for follow-ups.
- **B — `gmd wiki query --conversational`.** Single-turn RAG with conversation_id tracking
  for follow-ups.
- **C — Conversation history as wiki content.** Save Q&A sessions as wiki pages for
  compounding knowledge.
- **D — Push vs pull conversation state.** Typesense manages conversation history server-side
  (push — simpler, server-side state) vs gmd manages it client-side (pull — stateless server,
  gmd provides context). Which is more aligned with gmd's architecture?

### 7.2 Graph Traversal via JOINs

Wiki page linking (`[[wikilinks]]`) could be modeled as Typesense references for JOIN-based
graph queries.

**Options:**
- **A — `gmd wiki graph` with JOINs.** `wiki_graph` collection where each page references its
  linked pages, enabling JOIN-based traversal.
- **B — Backlinks via reverse JOIN.** `filter_by: $wiki_pages(links:=<current_page>)` to find
  all pages that link to the current page.
- **C — Nested graph depth.** Fetch N-hop neighbors via nested JOINs.
- **D — Graph as first-class MCP tool.** `graph-neighbors` MCP tool for AI agents to navigate
  wiki link topology.

### 7.3 Wiki Analytics

Track which wiki pages are searched/found, identify content gaps.

**Options:**
- **A — Wiki-specific analytics rules.** Per-wiki `popular_queries` and `nohits_queries`.
- **B — Popularity-ranked wiki pages.** Counter events on wiki page views.
- **C — Content gap detection.** `gmd wiki doctor` auto-detects topics with no results and
  suggests new pages.
- **D — Ingestion pipeline feedback.** No-hit queries feed into wiki ingest: "users searched
  for X but found nothing — should there be a page?"

### 7.4 Natural Language Wiki Queries

NL search tailored to wiki schemas.

**Options:**
- **A — `gmd wiki nlquery <name> "find pages about search from last month"`.** Auto-generates
  filters + sorts.
- **B — Schema-aware NL models per wiki.** Each wiki's schema (frontmatter fields) informs
  the NL model.

### 7.5 Auto-Index on Wiki Ingest

IDEAS.md calls this out explicitly. The ingest → index cycle should be seamless.

**Options:**
- **A — Trigger `gmd update` after every ingest.** Wiki agent or CLI command auto-refreshes
  the Typesense index.
- **B — Scoped update per wiki.** `gmd update -c wiki-<name>` avoids re-scanning all collections.
- **C — Event-driven indexing.** File system watcher (IDEAS.md #646) or webhook-driven reindex
  when wiki content changes.
- **D — Index freshness status.** `gmd wiki status` shows when each wiki was last indexed and
  whether there are unindexed changes.

### 7.6 MCP Tool Extensions

IDEAS.md proposes wiki-specific MCP tools beyond the current `search`/`get`/`multi-get`:

- **`search-wiki`** — scoped to wiki collections, returns page paths with summaries
- **`wiki-query`** — hybrid search + LLM rerank tailored for wiki navigation
- **`graph-neighbors`** — given a page, return linked pages via `[[wikilinks]]` traversal
- **`wiki-ingest`** — trigger ingestion pipeline from MCP
- **`wiki-lint`** — health checks exposed as MCP tools

### 7.7 Wiki Export & Portability

- **Export formats beyond directory copy.** Typesense snapshots could anchor a self-contained
  wiki export (markdown files + index snapshot).
- **Import from other wiki tools.** Typesense as the common index format, with adapters for
  Obsidian, Notion, Logseq, etc.

### 7.8 Duplicate / Stale Page Detection

- **Vector-based dedup.** Use Typesense vector similarity to detect near-duplicate wiki pages.
- **Freshness tracking.** Frontmatter `updated` field + `filter_by` for stale content detection.
- **Orphan page detection.** Pages with zero incoming `[[wikilinks]]` via JOIN or graph query.

**Open questions for wiki section:**
- Should gmd's wiki agent be reconfigured to use Typesense conversational RAG, or should it
  continue using gmd's own query pipeline?
- The wiki ingest pipeline currently uses gmd's LLM client. If gmd goes "thin" and offloads
  RAG/embeddings to Typesense, the agent's tools change significantly. Is this a transitional
  path or a long-term direction?
- Wiki graph queries via JOINs: is this simpler or more complex than gmd's current Go-side
  graph parsing from `[[wikilinks]]`?

---

## 8. Cross-Cutting Architecture Decisions

### 8.1 CUE Config Surface Area (Promoted)

This is arguably the most consequential design decision for gmd, shaping every other
enhancement area.

Most Typesense features are configurable server-side. gmd's CUE config could be:

| Approach | Example | Pros | Cons |
|---|---|---|---|
| **A — Thin passthrough** | `synonyms: { id: "k8s", synonyms: ["kubernetes", "kube"] }` maps 1:1 to TS API | Easy to implement, complete feature coverage | Tightly coupled to TS version, verbose config |
| **B — Semantic abstraction** | `search_intent: "prioritize recent docs over older ones"` → gmd translates to TS params | Portable, user-friendly | High implementation cost, leaky abstraction risk |
| **C — Hybrid** | Simple features thin, complex features semantic | Pragmatic | Inconsistent config idioms |

**Key tensions:**
- CUE schema becomes the user-facing contract. If it mirrors Typesense API surface, users
  need Typesense knowledge. If it abstracts, gmd's CUE becomes a DSL.
- gmd already has a config pipeline (`.gmd/config.cue` with schema validation). Adding
  Typesense-level config increases complexity and test surface.
- When gmd's CUE config grows beyond a certain threshold, does a separate Typesense
  configuration make more sense than embedding everything in gmd's CUE?

### 8.2 How Much Should gmd Push into Typesense vs. Do in Go?

gmd currently handles embeddings, query expansion, RRF fusion, LLM reranking, position blending,
and result formatting in Go. Typesense v30 can absorb many of these:

| Capability | Currently in gmd (Go) | Typesense v30 Can Do |
|---|---|---|
| Embedding generation | Go → LLM API → store in TS | Auto-embedding (ONNX or API) |
| Query expansion | Go → LLM expansion prompt | NL search (partial overlap) |
| Multi-collection merge | Go → RRF fusion | Union search with remove_duplicates |
| LLM answer synthesis | Go → LLM chat completion | Conversational RAG |
| Ranking | Go → LLM rerank + position blend | Curations, pinned_hits, _eval(), popularity bucketing |
| Faceting | Not implemented | facet_by, facet_query |
| Analytics | Not implemented | popular_queries, nohits_queries, counter |

The fundamental question: **do we want gmd to be "thin middleware" that configures Typesense and
adds CLI convenience, or "thick pipeline" that wraps Typesense with significant Go-side intelligence?**

### 8.3 Single Collection vs. Multi-Collection Schema Design

gmd currently uses one Typesense collection per gmd collection/wiki, with a shared `chunks`
schema. Typesense features like JOINs and federated union search enable alternative architectures:

- **Current:** N Typesense collections, Go-side merge for multi-collection queries.
- **Alternative A — Single collection with discriminators:** All chunks in one Typesense collection,
  differentiated by `collection_name` and `wiki_name` fields. Simpler Typesense management, more
  use of `filter_by`.
- **Alternative B — Relational model with JOINs:** Separate `documents`, `chunks`, `wikis`
  collections with references. More powerful queries, more complex setup.

### 8.4 Provider Model for Typesense Features

gmd has a provider model for LLM services (profiles map roles to providers). Should Typesense
features follow a similar model?

- **Synonym providers:** Pre-made dictionaries from URLs, local files, embedded defaults.
- **Curation providers:** Auto-generated rules from LLM analysis, manual CUE config.
- **Analytics providers:** Typesense-native vs external (Amplitude, GA) for richer analytics.

### 8.5 Version Compatibility & Migration

gmd is on Typesense v26-era APIs. v30 introduces breaking changes:
- Synonyms and overrides moved to global synonym_sets / curation_sets
- Analytics rules format changed
- New API key action names (synonym_sets:* vs synonyms:*, curation_sets:* vs overrides:*)

**Questions:**
- Does gmd need to support multiple Typesense versions, or can we require v30+?
- If gmd is deployed with Typesense Cloud, version is managed. Self-hosted users need documentation.
- How do we handle the migration path for existing gmd installations?

---

## 9. Feature Grouping by Impact & Effort (Notional)

Rough clustering of the enhancement areas for prioritization discussion. No decisions here —
just framing for the next phase.

### Foundation (Enables Other Features)
- Frontmatter extraction pipeline (2.1) — unlocks faceting, filtering, sorting, grouping, A/B testing
- Schema evolution support (4.3) — enables iterative schema changes, especially for frontmatter
- Typesense version upgrade path (8.6) — required for v30 features (synonym_sets, curation_sets, RAG, NL search, analytics)
- Text processing config (1.6) — token separators, infix, stopwords set the baseline for code/doc search quality

### High Impact, Moderate Effort
- Synonym sets (1.1)
- Curation / overrides (1.2)
- Stemming (1.3)
- Search output / highlighting (5.6)
- Filtering by structured fields (2.3) — *depends on frontmatter extraction*
- Scoped API keys (6.2) — *gate for multi-user story*

### High Impact, High Effort
- Conversational RAG (5.1) — *depends on "thin vs thick" decision (8.2)*
- Natural Language Search (5.2)
- Analytics: popular queries + counters (3.1, 3.2) — *depends on TS server flags*
- JOINs / relational schema (4.1)
- Vector auto-embedding (5.4) — *if chosen, eliminates gmd's embedding LLM dependency; cascading effect on LLM profiles*
- Faceting (2.2) — *depends on frontmatter extraction*

### Specialized / Niche
- Query suggestions / autocomplete (3.3)
- Personalization (Typesense guide) — *depends on analytics + multi-user context*
- Recommendations (Typesense guide)
- Geo search (not a gmd priority)
- Voice search (not a gmd priority)
- Image search (not a gmd priority)

### Operational
- Backups & restore (6.1)
- Query perf tuning (5.5)
- Cluster monitoring (6.3)
- HA / clustering docs (6.4)
- A/B testing (3.4) — *depends on analytics or structured fields*

### Explicit Dependency Chains

These chains inform ordering if multiple enhancements are pursued:

1. **Typesense v30 upgrade (8.6) →** everything below: synonym_sets, curation_sets, RAG, NL search, analytics, JOINs all require v30 API surface.
2. **Frontmatter extraction (2.1) → Faceting (2.2) + Filtering (2.3) + Grouping by metadata (2.4) + A/B testing (3.4).** Cannot filter/facet/group on fields that aren't indexed.
3. **Analytics server flags (prerequisite) → Analytics rules (3.1) → Counters (3.2) → Popularity ranking (1.5-A).** Analytics infra must exist before any analytics-driven features.
4. **Scoped API keys (6.2) → Multi-user gmd (6.2-A) → Personalization (niche).** Without scoped keys, multi-tenancy is insecure.
5. **"Thin vs thick" decision (8.2) → Conversational RAG (5.1) + Auto-embedding (5.4) + Union fusion (5.3).** If gmd goes thin, these become top priority. If thick, lower priority.
6. **CUE config approach (8.1) → every configurable feature.** The config idiom chosen (thin passthrough vs semantic) affects implementation cost across all enhancement areas.

### Key Synergies

- **Frontmatter + Scoped API keys + Faceting =** multi-user wiki with per-user access control and faceted navigation
- **Analytics (3.1) + Counters (3.2) + Curation (1.2) =** feedback-driven search improvement loop
- **RAG (5.1) + Federation (5.3) =** conversational search across multiple wikis/collections
- **Auto-embedding (5.4) + Synonyms (1.1) + Stemming (1.3) + Stopwords (1.6) =** "zero-config quality" story — users get good results out of the box
- **Curation (1.2) + AI agents (7.6) =** AI-driven curation suggestions via MCP tools

### Architecture Decision Prerequisites

Before detailed designs for individual enhancements, these cross-cutting questions should be settled:

1. **Typesense v30 upgrade path** — gating decision for most new features
2. **"Thin middleware" vs "thick pipeline"** — determines whether gmd invests in or removes Go-side search intelligence
3. **CUE config idiom** — thin passthrough vs semantic abstraction affects every configurable feature's design
4. **Single-user vs multi-user** — determines relevance of analytics, popularity, personalization, scoped keys

---

## Open Questions (Cross-Cutting)

1. **Typesense version target:** Should gmd require v30+ and drop support for earlier versions,
   or maintain backward compatibility?
2. **Server-side features (analytics, RAG, NL search) require Typesense server flags.** Should
   `gmd init` / `gmd serve` configure these, or document them as prerequisites?
3. **API key management:** gmd currently uses a single admin-level API key. If we adopt scoped
   keys, who manages key lifecycle? gmd or the user?
4. **Embedded vs remote embedding:** If Typesense auto-embedding (ONNX models) works offline,
   does gmd still need its own embedding LLM provider? What's the trade-off?
5. **CUE config complexity:** As we expose more Typesense features in CUE, the config schema
   grows. What's the threshold where a separate Typesense config makes more sense than embedding
   everything in gmd's CUE?
6. **CLI vs API vs MCP:** Different features make sense for different interfaces. Should we
   prioritize CLI flags, REST API, or MCP tools for each enhancement?
7. **Testability:** Typesense features like analytics, RAG, and curation are stateful. How do
   we test these? Typesense provides Docker test containers; could tape-replay handle stateful
   features, or do we need full integration tests?
8. **Multi-user story:** Is gmd ultimately a single-user tool or a team tool? Many features
   (analytics, scoped keys, HA, personalization) only make sense at multi-user scale.
9. **Typesense as the long-term store:** SCRATCH.md asks "maybe we end up with an sqlite db
   after all?" — is Typesense the correct sole data store, or should gmd maintain its own
   state (SQLite) for things not suited to a search engine (config history, user prefs, etc.)?
10. **"Thin vs thick" endgame:** If gmd adopts Typesense RAG, auto-embedding, and NL search,
    what's the role of gmd's LLM client? Purely for embeddings (if not auto-embedded)? Purely
    for wiki ingest? Does gmd's LLM profile concept become redundant?

---

## Next Steps

1. **Settle preconditions:** Decide Typesense version target (v30+ or backward compat), "thin vs
   thick" architecture direction, and CUE config idiom. These three decisions shape all subsequent
   detailed designs.
2. **Vote/prioritize:** The team reviews this document, identifies which enhancement clusters feel
   most valuable for the next development phase.
3. **Spike on frontmatter extraction:** This is the highest-leverage foundation. A prototype
   would clarify schema evolution questions, performance impact, and CUE config design —
   regardless of which architecture direction is chosen.
4. **Typesense version audit:** Determine the exact v26→v30 migration impact on current gmd code
   and create an upgrade plan before any v30-dependent features are designed.
5. **Detailed design documents** for each chosen enhancement area, scoped to a specific
   implementation plan with phases, tests, and migration path.
6. **Test infrastructure evaluation:** Determine whether the current tape-replay approach
   suffices for stateful Typesense features (analytics, RAG, curation) or if Docker-based
   integration tests are needed for new feature development.
