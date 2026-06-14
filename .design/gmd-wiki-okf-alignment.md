# GMD Wiki — Open Knowledge Format (OKF) Alignment

**Date:** 2026-06-13 (created), 2026-06-13 (updated — v5)
**Phase:** Design — updated per feedback
**OKF Version:** 0.1 (Draft)
**OKF SPEC:** https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md

---

## 1. Context & Goal

OKF defines an open, human- and agent-friendly format for knowledge bundles:
directories of markdown files with YAML frontmatter, cross-linked via standard
markdown links. It is intentionally minimal — no schema registry, no central
authority, no required tooling.

GMD already implements a Karpathy-style LLM Wiki with many OKF-adjacent
conventions: markdown + YAML frontmatter, [[wikilinks]], index files, log files,
cross-link graphs, and linking constraints. The two systems overlap substantially
but diverge on specific choices that prevent interoperability.

**Goal:** Align GMD's wiki output and exchange interfaces with the OKF v0.1
specification so that:

1. **GMD wikis can be exported as valid OKF bundles** (a directory of markdown
   files consumable by any OKF-compatible system).
2. **GMD's internal wiki conventions are a strict superset of OKF** —
   additional features (status lifecycle, [[wikilinks]] for convenience)
   are layered on top without breaking conformance.
3. **GMD can serve as an OKF enrichment agent** — reading source documents,
   synthesizing new concepts, and writing back conformant updates.

---

## 2. Gap Analysis — GMD Wiki vs OKF v0.1

| Concern | OKF v0.1 | GMD Wiki (current) | Gap |
|---|---|---|---|
| **Required frontmatter** | `type` (free-form string) | `type` is present in wiki_schema.md but not enforced in code | `type` is a free-form string. GMD must validate non-empty `type` on every page. |
| **Recommended frontmatter** | `title`, `description`, `resource`, `tags`, `timestamp` | `title` (often H1-derived), `tags`, `status`, `sources`, `source_url` | Missing `description` (auto-generated via summarizer LLM), `resource` (canonical URI), `timestamp` (deterministic — set by tool on write, ISO 8601). GMD extras (`status`) are valid OKF extensions. |
| **Reserved filenames** | `index.md`, `log.md` | `_index.md`, `_log.md` (underscore-prefixed) | OKF explicitly reserves `index.md` and `log.md` without underscores. GMD prefix convention must be reconciled. |
| **Cross-linking** | Standard markdown links: `[text](/path/to/file.md)` (bundle-relative, recommended) or `[text](./other.md)` (relative) | `[[Page Name]]` wikilinks with optional alias/heading | **Largest structural difference.** OKF uses filesystem paths with `.md` extension; GMD uses page names. Must bridge both. |
| **Concept ID** | File path minus `.md` suffix (e.g., `tables/users`) | Page name (slugified title, e.g., `customer-orders`) | GMD page names are broader — they are independent of filesystem location. For OKF conformance, the file path IS the concept ID. |
| **Citations** | `# Citations` body section with numbered references | Query responses use `[[citations]]`; pages don't currently generate `# Citations` | Ingest agent should produce `# Citations` sections per OKF convention. |
| **Index format** | `index.md` — sections with `* [Title](url) - description` entries | `_index.md` — sections with `[[Page Name]] - description` entries | Change to `index.md` with standard markdown links. |
| **Log format** | `log.md` — `## YYYY-MM-DD` date headings, bullet list of **bold-word** entries | `_log.md` — `## [YYYY-MM-DD HH:MM] action | source` headings, bullet detail | GMD keeps the richer heading format (more information is better). Not a conformance concern — consumers tolerate extra heading content. |
| **Bundle structure** | Arbitrary directory hierarchy; producers organize however makes sense | Wiki directory is flat/free-form | No prescribed subdirectory layout. Pages live wherever the user or agent places them. |
| **OKF version declaration** | `okf_version: "0.1"` in bundle-root `index.md` frontmatter | Not present | Trivial to add. |
| **Body structure** | No prescribed body sections; spec examples use `# Schema`, `# Examples`, `# Citations` as illustrations only | No conventional body sections enforced | OKF does not mandate body section headings. Body structure is concept-type-dependent. Ingest prompts provide varied examples per concept kind rather than universal headings. |
| **Forbidden types** | No fixed taxonomy; consumers tolerate unknown types | No internal type taxonomy | Compatible — `type` is free-form. Any string is valid. |
| **Conformance validation** | 1) Every .md has YAML frontmatter, 2) Every frontmatter has non-empty `type`, 3) Reserved files follow structure | No formal conformance check; `lint` checks orphans, broken links, stale entries | Need `gmd wiki okf-validate` or extend `lint` with OKF conformance rules. |

---

## 3. Design Decisions

### 3.1 File Naming: `index.md` / `log.md` vs `_index.md` / `_log.md`

**Decision:** Use OKF names (`index.md`, `log.md`) as the canonical names. GMD is alpha software — no migration, no fallbacks for legacy names.

- **Config schema**: `WikiConfig.indexFile` default is `"index.md"`, `logFile` default is `"log.md"`.
- **All meta-file skip sites**: Replace `strings.HasPrefix(base, "_")` with name-based checks against the configured `IndexFile`/`LogFile` values. Only files matching those names are skipped. This is a behavioral change: previously ALL `_`-prefixed `.md` files were skipped; after the change, only files named `index.md` and `log.md` are reserved (per OKF spec).
- **Sites needing update**:
  - `pkg/wiki/graph.go:41` (BuildGraph skip)
  - `pkg/wiki/lint.go:152` (lintContent skip)
  - `pkg/wiki/watch.go:88` (checkWiki skip)
  - `pkg/wiki/lint.go:97-110` (broken-link source walk — currently has NO skip guard, must be added)
  - Note: `pkg/wiki/lint.go:67` (lintStructure first walk) already checks config names correctly.

### 3.2 Cross-linking: Standard Markdown Links

**Decision:** Use OKF standard markdown links for all cross-references. `[[wikilinks]]` are removed from agent output; input parsing retains `[[wikilinks]]` as a convenience for human-authored pages but they are syntactic sugar that resolves the same way.

**Parsing (read path):**
- Extend `pkg/chunking/markdown.go` `ExtractWikilinks()` to also extract
  standard markdown links that reference `.md` files (bundle-relative or relative).
- Both link types populate the `Links` field in `Chunk` structs and
  `ChunkDocument.Links` in Typesense.
- Graph building (`pkg/wiki/graph.go`) reads both link types.

**Writing (output path):**
- Ingest agent writes standard markdown links for cross-references:
  `[Page Name](/path/to/page.md)` (bundle-relative, OKF preferred form).
- Query agent synthesizes answers with standard markdown links in `[[citations]]`.
- `updateIndexFile()` writes `* [Title](relative/url.md) - description`.

**Link resolution (for graph, neighbors, lint):**
- From `[[Page Name]]`: look up `Page Name` in wiki directory → resolve to file path.
  Requires a page-name → file-path registry built by scanning the wiki
  directory and parsing each page's H1 or frontmatter `title`.
- From `[text](/path/to/file.md)`: strip leading `/` and `.md` suffix → concept ID.
- From `[text](./file.md)` or `[text](../other/file.md)`: resolve relative to
  source page's directory → absolute bundle-relative path → concept ID.
  Requires the link extractor to carry source file directory context.
- **Unified node identity**: Both page names and concept IDs must canonicalize to
  the same namespace. When a page contains both `[[Transformer]]` and
  `[Transformer](/concepts/transformer.md)`, these produce ONE link, not two.
  Deduplication by resolved concept ID (file path minus `.md`) happens at the
  graph/link level.

### 3.3 Frontmatter: OKF Required + Recommended Fields

**Decision:** GMD's ingest agent outputs the OKF required field plus recommended
fields. GMD wiki pages are a superset of OKF.

**Ingest agent output:**
```yaml
---
type: entity              # REQUIRED by OKF; free-form string
title: Transformer        # OKF recommended (derived from H1 if absent)
description: The transformer architecture uses self-attention...  # Auto-generated via summarizer LLM
resource: https://arxiv.org/abs/1706.03762   # OKF recommended (canonical URI)
tags: [ai, transformer, architecture]  # OKF recommended
timestamp: 2026-06-13T14:30:00Z  # OKF recommended (ISO 8601) — deterministic, set by tool on write
# GMD extensions (OKF permits arbitrary keys):
status: draft
sources: [source-page.md]
---
```

**Key decisions:**
- `type` is free-form. No GMD-internal type taxonomy or mapping. Any string is valid.
- `difficulty` is removed from frontmatter entirely.
- `description` is auto-generated by the summarizer LLM after the page is written
  (not produced by the ingest agent). The tool calls the summarizer on the full page
  content and writes the result into frontmatter.
- `timestamp` is deterministic — the tool sets it to the current time on every
  write, not produced by the LLM agent. The agent is not asked to include it.
- `status` and `sources` remain as GMD extensions.

**Config schema:** `WikiConfig.frontmatter` is a simple CUE struct. Required fields use
CUE's `!` marker (e.g., `type!: string`). Only `type` is required. All other fields
are optional. Unknown fields pass through via `...` per OKF §4.1.

### 3.4 Citations Section

**Decision:** Generate `# Citations` sections per OKF convention.

- Ingest agent: When ingesting a source, the agent generates a `# Citations`
  section on source pages listing the original source and any referenced
  external material.
- Query agent: When saving synthesis pages via `saveQueryResult()` (agent.go:371),
  it generates a `# Citations` section with numbered references to wiki pages that
  support the answer, replacing the current `## Sources` section that uses
  `[[wikilinks]]`.
- The existing `[[citations]]` syntax in CLI output remains for interactive
  consumption; the saved page uses OKF's `# Citations` format.

### 3.5 Concept ID = File Path

**Decision:** Concept ID is the file path minus `.md` suffix (OKF §2). Page name
is the display title (from frontmatter `title` or derived).

This is already how GMD works internally — the `Path` field in Typesense
`ChunkDocument` is the relative file path. The `Title` field is the display
title. No code changes required, but the terminology should align with OKF.

### 3.6 OKF Version Declaration

**Decision:** `index.md` frontmatter includes `okf_version: "0.1"` (OKF §11).

- `gmd wiki create` writes this into the initial `index.md`.
- GMD detects `okf_version` when loading a wiki and validates it against
  supported versions.
- Missing `okf_version` means pre-spec — the wiki was created before OKF
  alignment and needs the declaration added (doctor check, §7.5).

### 3.7 Body Structure by Concept Type

**Decision:** Body structure varies by concept type/knowledge kind. Ingest prompts
provide varied examples per type rather than prescribing universal body headings
(e.g., `# Schema`, `# Examples`, `# Citations`). These spec headings are examples
only, not requirements.

Example body structures by concept type (illustrative, not exhaustive):

- **Data schema / table definition:** `# Schema` (structured field listing), `# Examples` (concrete usage), `# Constraints` (validation rules)
- **Algorithm / process:** `# Overview`, `# Steps / Pseudocode`, `# Complexity` (time/space), `# See Also`
- **Concept / idea:** `# Overview`, `# Key Properties`, `# Relationship to X` (how it relates to other concepts), `# See Also`
- **Comparison / trade-off:** `# Context`, `# Options` (table), `# Trade-offs`, `# Recommendation`
- **Tutorial / guide:** `# Prerequisites`, `# Steps`, `# Examples`, `# Common Issues`
- **Reference / glossary:** `# Definition`, `# Examples`, `# See Also`
- **Source page (ingested raw):** `# Citations` (links to original sources), `# Key Points`, free-form body

The ingest prompt includes a `concept_kind` field in the JSON response to allow
the LLM to choose the appropriate body structure. The `packageDoc()` function
validates that the chosen kind is recognized and maps it to the corresponding
body section template for structural consistency.

`# Citations` is a standard OKF convention for source pages, retained as the
one universal body heading for that concept type only.

---

## 4. Conformance — GMD as OKF Producer

### 4.1 GMD as OKF Producer (Export)

When `gmd wiki okf export <name>` is run (new command), or when wiki pages are
written by the ingest agent:

1. Every `.md` file has YAML frontmatter with a non-empty `type` field.
2. Reserved files (`index.md`, `log.md`) follow OKF structure (§6, §7).
3. Cross-links use standard markdown bundle-relative paths.
4. `index.md` has `okf_version: "0.1"` in frontmatter.
5. All `.md` files are UTF-8 encoded.

### 4.2 GMD as Enrichment Agent

GMD's ingest agent is an OKF enrichment agent: it reads source documents (from
`raw/`), extracts knowledge, and writes new/updated pages back into the wiki.
This is core to the Karpathy compounding wiki pattern and aligns with OKF's
stated goal of defining a format that enrichment agents can write into (OKF §1,
Goal 1). The ingest agent reads raw sources, not OKF bundles — import of
external OKF bundles is out of scope.

---

## 5. Configuration Changes

### 5.1 CUE Schema (`pkg/config/embeds/types.cue`)

Changes to `WikiConfig`:

```cue
WikiConfig: Source & {
    wikiDir:     string | *"wiki"
    rawDir:      string | *"raw"
    indexFile:   string | *"index.md"       // was "_index.md"
    logFile:     string | *"log.md"         // was "_log.md"
    okfVersion:  string | *"0.1"            // NEW: declared OKF version for this wiki
    graphLinks:  bool | *true
    excludeFromDefault?: bool | *false
    sourceRefs?: [...string]
    frontmatter?: {                        // NEW: OKF frontmatter schema
        type!:       string               // REQUIRED by OKF — any non-empty string
        title?:      string
        description?: string
        resource?:   string
        tags?:       [...string]
        timestamp?:  string
        status?:     string               // free-form, no fixed enum
        sources?:    [...string]
        ...                               // unknown fields pass through (OKF §4.1)
    }
}
```

- `type!` is the only required field. CUE's `!` marker enforces it.
- No regexp constraints on string lengths. CUE's `string` type is sufficient.
- `status` is a plain `string` — no fixed enum, no known value set.
- `...` at the end allows any additional frontmatter keys through without validation.
- The whole `frontmatter?` block is optional — if omitted, no frontmatter schema
  is applied (beyond what `ValidateOKF()` checks for `type` presence).

### 5.2 Config defaults (`pkg/config/config.go`)

Apply runtime defaults:
- `indexFile` → `"index.md"` if empty (was `"_index.md"`)
- `logFile` → `"log.md"` if empty (was `"_log.md"`)
- `okfVersion` → `"0.1"` if empty

### 5.3 Wiki Init defaults (`pkg/wiki/wiki.go`)

`Init()`:
- Scaffold `index.md` (with `okf_version: "0.1"` frontmatter) and `log.md`.
- No prescribed subdirectory structure — wiki starts empty, pages go wherever
  the user or agent places them.

---

## 6. New / Changed Commands

| Command | Status | Description |
|---|---|---|
| `gmd wiki lint <name> [--okf] [--strict]` | Modified | `--okf` flag adds OKF conformance checks (every .md has frontmatter with `type`, reserved files follow structure, non-root `index.md` has no frontmatter). Calls `ValidateOKF()`. `--strict` makes violations non-zero exit. |
| `gmd wiki okf export <name> [--output <dir>]` | New | Export wiki as a standalone OKF bundle directory. Converts [[wikilinks]] → markdown links, ensures frontmatter compliance. |
| `gmd wiki doctor <name>` | Modified | Add check: missing `okf_version` in index.md → offer to add. Add check: pages missing `type` frontmatter. Add check: pages with stale `timestamp` vs file mtime. |
| `gmd wiki create <name>` | Modified | Scaffold `index.md`/`log.md`. Write `okf_version: "0.1"` in `index.md` frontmatter. Update hardcoded ignore patterns to use config values. No subdirectory scaffold. |

**Removed:**
- `gmd wiki okf import` — importing external OKF bundles is out of scope.
- `gmd wiki ingest` OKF bundle source detection — ingest reads from `raw/` only.

OKF validation lives under `gmd wiki lint --okf`, not a separate command.
The `okf` subcommand group under `wiki` exists only for export.

## 7. Code Changes Required

### 7.1 Link Parsing (`pkg/chunking/markdown.go`)

**Current:** `ExtractWikilinks()` parses only `[[...]]` syntax.
**Change:** Add `ExtractMarkdownLinks()` that captures standard markdown links
targeting `.md` files. Both extractors return raw link targets (page names for
wikilinks, file paths for markdown links). Normalization to concept IDs
(stripping `/`, `.md`, resolving relatives) and deduplication across both link
types happens at the resolution layer (see §7.3). Relative link resolution
requires source-directory context passed by the caller.

```go
// New regex for standard markdown links pointing to .md files
var mdLinkRe = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+\.md)\)`)

// ExtractMarkdownLinks extracts all markdown links targeting .md files.
// sourceDir is the directory of the source file for resolving relative links.
// Returns raw link targets (may include leading "/" or "./" prefixes).
func ExtractMarkdownLinks(content string) []string {
    // "/path/to/file.md", "./other.md", "../nested/concept.md"
    // Returns deduplicated list of raw link targets.
}

// NormalizeConceptID converts a raw link target to a unified concept ID.
// Used by the resolution layer (not the extractor).
// "/path/to/file.md" → "path/to/file"
// "./file.md" → resolved relative to sourceDir → "sourceDir/file"
func NormalizeConceptID(linkTarget string, sourceDir string) string {
    // Strip leading "/" and trailing ".md", resolve "./" and "../" relatives.
}
```

### 7.2 Link Writing (`pkg/wiki/agent.go`)

**Change ingest prompts** (`embeds/ingest_system.md`):
- Instruct LLM to output standard markdown links: `[Page Title](/path/to/page.md)`
  not `[[Page Title]]`.
- Add `# Citations` section generation for source pages.
- Add `concept_kind` field to the JSON response so the LLM can select the
  appropriate body structure (see §3.7).
- Body sections are concept-kind-dependent, not universal `# Schema` / `# Examples`.
- Do NOT ask the LLM to produce `description` or `timestamp` — these are handled
  by tool-level post-processing (see §7.15c).

**Change query prompts** (`embeds/query_system.md`):
- Instruct LLM to use standard markdown links in inline citations.

**Update `updateIndexFile()` (agent.go:218):**
- Write OKF-format index entries: `* [Title](relative/url.md) - description`
  instead of `- [[Page Name]] — description`.

**Update `saveQueryResult()` (agent.go:371):**
- Replace `## Sources` section (with `[[wikilinks]]`) with `# Citations`
  section (numbered references with standard markdown links).
- Tool sets `timestamp` deterministically on write (not from LLM).

**Update `appendLogFile()` (agent.go:272):**
- Keep the richer `## [YYYY-MM-DD HH:MM] action | source` heading format.
  More information is better. OKF consumers tolerate extra heading content.

**Update `packageDoc()` (agent.go):**
- Use standard markdown links (not `[[wikilinks]]`) in body text.
- Map `concept_kind` from LLM response to appropriate body section template.
- Do NOT include `timestamp` from LLM — set deterministically by tool.

### 7.3 Link Resolution (`pkg/wiki/graph.go`)

**Current:** `BuildGraph()` walks files, extracts `[[wikilinks]]` via
`chunking.ExtractWikilinks()`.
**Change:** Extract both `[[wikilinks]]` AND markdown `.md` links. Pass source
directory to the markdown link extractor for relative link resolution.
Both link types populate the same adjacency graph after normalization.

**Link resolution pipeline:**
1. For each page, extract raw `[[wikilinks]]` targets (page names).
2. Extract raw markdown link targets (file paths with `.md`).
3. Resolve page names → concept IDs via registry (page-name → file-path map).
4. Normalize markdown links → concept IDs (strip `/`, `.md`, resolve relatives).
5. Deduplicate links by normalized concept ID.
6. Build graph edges from the unified link set.

**`pageName()` update:** The existing `pageName()` (graph.go:180) returns the
wiki-relative file-path-minus-md as the node identifier. Markdown links resolve
to the same format, so existing graph algorithms work unchanged once links are
normalized.

### 7.4 Lint (`pkg/wiki/lint.go`)

**Add OKF conformance check functions (called from `Lint()` with new `LintOKFOpts`):**
- `lintOKFConformance()`: Walk all `.md` files in wiki directory (single pass).
  For each page: check frontmatter exists, `type` is non-empty.
  Check reserved files (`index.md`, `log.md`) follow structure.
  **Check subdirectory `index.md` files have NO frontmatter** (OKF §6 prohibits
  frontmatter on non-root index files; only bundle-root `index.md` may have
  `okf_version` per OKF §11). Report violations.
  This walk must skip configured meta files (`IndexFile`/`LogFile`).
- `lintBrokenMarkdownLinks()`: Extend existing `lintStructure()` broken-link
  detection to cover markdown `.md` links. The current broken-link check
  (`lint.go:91-112`) only tests `[[wikilinks]]`. Refactor to a single pass
  that checks both link types. The second walk (`lint.go:97-110`) currently has
  no meta-file skip guard — this must be added (see §7.8).

### 7.5 Doctor (`pkg/wiki/doctor.go`)

**Add checks:**
- **Missing `okf_version`**: Check `index.md` frontmatter for `okf_version`.
  Fix: write `okf_version: "0.1"` into frontmatter.
- **Missing `type` frontmatter**: Walk `.md` files, check for `type` field.
  No auto-fix — requires agent or human intervention. Report count.
- **Connectivity**: Existing Typesense and LLM checks (already implemented).
- **Wiki file stats**: New: count pages, index entries, log entries, track
  last ingest timestamp from log.

### 7.6 Embedded Templates

**`pkg/wiki/embeds/wiki_schema.md`:**
- Replace `_index.md` with `index.md`, `_log.md` with `log.md` throughout.
- Replace `[[wikilinks]]` in examples with standard markdown links.
- Add OKF cross-reference: mention that the wiki follows OKF v0.1 conventions.
- Add `description`, `resource`, `timestamp` to the frontmatter schema example.
- Remove `difficulty` from frontmatter schema example.

**`pkg/wiki/embeds/ingest_system.md`:**
- Use standard markdown links in template examples.
- Add `# Citations` section generation for source pages.
- Add `concept_kind` field to JSON response schema. Provide varied body structure
  examples per concept kind (see §3.7).
- Do NOT ask LLM for `description` (auto-generated post-write) or `timestamp`
  (deterministic, set by tool).

**`pkg/wiki/embeds/query_system.md`:**
- Use standard markdown links in citation examples.

**`pkg/context/skills/embeds/gmd-wiki/SKILL.md`:**
- Replace `_index.md` → `index.md`, `_log.md` → `log.md` in all references.
- Update wikilink examples to use standard markdown links.

**`pkg/context/agentsmd/embeds/full.md`:**
- Replace `_index.md` → `index.md`, `_log.md` → `log.md` in documentation.

### 7.7 Wiki Creation (`cmd/gmd/wiki_create.go`)

**Changes:**
- Lines 82-83: Replace hardcoded `"_index.md"`, `"_log.md"` with config values
  (`cfg.WikiConfig.IndexFile`, `cfg.WikiConfig.LogFile`) or use the newly
  defaulted names.
- Line 98: Replace hardcoded `AddIgnorePatterns` call similarly.
- The scaffolded `index.md` content must include frontmatter with `okf_version`.

### 7.8 Meta-File Skip Guards (multiple files)

Replace all `strings.HasPrefix(filepath.Base(path), "_")` guards with
name-based checks against `WikiConfig.IndexFile`/`WikiConfig.LogFile`.
This is a behavioral change: only files matching the configured reserved
names are skipped; all other `_`-prefixed `.md` files become regular concept
files (correct per OKF — only `index.md` and `log.md` are reserved).

Sites needing update:
- `pkg/wiki/graph.go:41` — `BuildGraph()` skip
- `pkg/wiki/lint.go:152` — `lintContent()` skip
- `pkg/wiki/watch.go:88` — `checkWiki()` skip
- `pkg/wiki/lint.go:97-110` — broken-link source walk in `lintStructure()`
  (currently has **no** skip guard — must be added)

### 7.9 CUE Schema (`pkg/config/embeds/types.cue`)

Add `okfVersion` and update `frontmatter` in `WikiConfig`:

```cue
WikiConfig: Source & {
    // ... existing fields ...
    okfVersion:  string | *"0.1"            // NEW: declared OKF version
    frontmatter?: {                          // OKF frontmatter schema
        type!:       string                  // REQUIRED by OKF
        title?:      string
        description?: string
        resource?:   string
        tags?:       [...string]
        timestamp?:  string
        status?:     string                  // free-form, no fixed enum
        sources?:    [...string]
        ...                                  // unknown fields pass through
    }
    // ...
}
```

### 7.10 Go Config Struct (`pkg/config/config.go`)

Add corresponding fields to `WikiConfig` Go struct (currently at line 550):
```go
type WikiConfig struct {
    SourceConfig
    // ... existing fields ...
    OkfVersion  string            `json:"okfVersion"`  // default "0.1"
    Frontmatter *WikiFrontmatter   `json:"frontmatter"` // NEW — nil if not configured
}

// WikiFrontmatter holds OKF frontmatter fields for wiki page validation.
// Only Type is required. Unknown fields in page frontmatter are preserved
// (ParseFrontmatter() returns map[string]interface{} which naturally
// captures all keys). The Extra map holds any frontmatter keys defined
// in CUE config beyond the well-known set below.
type WikiFrontmatter struct {
    Type        string                 `json:"type"`
    Title       string                 `json:"title,omitempty"`
    Description string                 `json:"description,omitempty"`
    Resource    string                 `json:"resource,omitempty"`
    Tags        []string               `json:"tags,omitempty"`
    Timestamp   string                 `json:"timestamp,omitempty"`
    Status      string                 `json:"status,omitempty"`
    Sources     []string               `json:"sources,omitempty"`
    Extra       map[string]interface{} `json:"-"` // unknown keys, populated via custom UnmarshalJSON
}
```

### 7.11 Wiki Init and File Access (`pkg/wiki/wiki.go`)

- `Init()`: Scaffold `index.md` (with `okf_version` frontmatter) and `log.md`.
  Currently creates `# Wiki Index` and `# Wiki Log` content; update both for
  OKF conventions.
- No prescribed subdirectory scaffold — wiki starts empty. Pages go wherever
  the user or agent places them.
- `createWikiPage()` (agent.go:174): Include `description` (auto-generated),
  `timestamp` (deterministic), `resource` in frontmatter when applicable.
  Do NOT include `difficulty`.

### 7.12 New Library (`pkg/wiki/okf.go`)

- `ValidateOKF(wiki *Wiki)` — conformance checking (shared by lint and export).
  Returns `OKFReport` with violations, warnings, passthrough count.
- `ExportOKF(wiki *Wiki, outputDir string)` — convert wiki to OKF bundle.
  Builds page-name → file-path registry. Converts `[[wikilinks]]` → markdown links.
  Writes conformant `index.md` / `log.md`. Copies all `.md` files.

### 7.13 New CLI Commands (`cmd/gmd/`)

- `wiki_okf_export.go` — `gmd wiki okf export <name> [--output <dir>]`
- Import is out of scope.
- The `okf` subcommand group under `wiki` exists only for export.
- OKF validation lives under `gmd wiki lint --okf`.

### 7.14 Tool-Level Post-Processing (`pkg/wiki/`)

New functions that run after the ingest agent writes a page:

- `generateDescription(pagePath string)`: Calls the summarizer LLM on the full
  page content, writes the result into frontmatter `description`.
- `setTimestamp(pagePath string)`: Writes current ISO 8601 time into frontmatter
  `timestamp`.
- Both are called after `createWikiPage()` and `saveQueryResult()` write the
  initial file content. The agent does not produce these fields.

### 7.15 Tests

| Area | Test File | What |
|---|---|---|
| Link parsing | `pkg/chunking/markdown_test.go` | Test `ExtractMarkdownLinks()`, normalization, both link types in same file, relative link resolution. |
| Link resolution | `pkg/wiki/graph_test.go` | Test graph building with markdown links, relative link resolution, mixed link types, dedup of same-target links. |
| Frontmatter | `pkg/wiki/frontmatter_test.go` | Test OKF required+recommended fields, unknown field preservation. |
| OKF conformance | `pkg/wiki/okf_test.go` (new) | Test validating a bundle: pass, fail on missing type, fail on reserved file misuse. |
| OKF export | `pkg/wiki/okf_integration_test.go` (new) | Export a wiki, verify output is OKF-conformant, verify wikilink conversion. |
| Ingest prompts | `pkg/wiki/agent_prompts_test.go` | Verify prompts use standard markdown links, concept_kind field, no timestamp/description in LLM output. |
| Lint | `pkg/wiki/lint_test.go` | Test OKF conformance lint checks, broken markdown links. |
| Doctor | `pkg/wiki/doctor_test.go` | Test okf_version check, type-frontmatter check. |
| Index/Log update | `pkg/wiki/agent_test.go` | Test `updateIndexFile()` writes OKF-format entries, `saveQueryResult()` uses `# Citations`. |
| Post-processing | `pkg/wiki/postprocess_test.go` (new) | Test `generateDescription()` via summarizer LLM, `setTimestamp()` deterministic behavior, both called after page write.

---

## 8. Implementation Plan

| Phase | Scope | Key Files | Dependencies |
|---|---|---|---|
| **P0: Foundation** | 1. Rename defaults: `indexFile` → `"index.md"`, `logFile` → `"log.md"` in CUE + Go config<br>2. Add `okfVersion` + `frontmatter` to CUE schema + Go config defaults (per §5.1)<br>3. Update `embeds/wiki_schema.md` (OKF names, markdown links, remove difficulty)<br>4. Update `wiki create` scaffolding + hardcoded ignore patterns (use config values). No subdirectory scaffold.<br>5. Replace all `_`-prefix skip guards with name-based checks. Add missing guard at `lint.go:97-110`.<br>6. `Init()` writes `okf_version` in `index.md` frontmatter<br>7. Update `SKILL.md` and `agentsmd/full.md` embedded docs<br>8. Remove `difficulty` from all frontmatter references across codebase | `types.cue`, `config.go`, `wiki_create.go`, `wiki.go`, `wiki_schema.md`, `graph.go`, `lint.go`, `watch.go`, `SKILL.md`, `full.md` | None |
| **P1: Links & Resolution** | 1. Implement `ExtractMarkdownLinks()` in `chunking/markdown.go`<br>2. Implement `NormalizeConceptID()` link target resolution<br>3. Build **page-name → file-path registry** (scan wiki dir, parse H1/title per page)<br>4. Update `BuildGraph()` for dual link types: extract wikilinks + markdown links, resolve both to concept IDs, deduplicate<br>5. Update broken-link detection in `lintStructure()` for both link types<br>6. Update `updateIndexFile()`: OKF `* [Title](url) - description` format<br>7. Update `saveQueryResult()`: `# Citations` format, tool sets timestamp<br>8. Update `createWikiPage()` and `packageDoc()`: standard links, no difficulty, `concept_kind` support<br>9. `appendLogFile()` keeps richer heading format<br>10. Update embedded ingest/query prompts: markdown links, `concept_kind` field in JSON schema, no timestamp/description in LLM output | `markdown.go`, `graph.go`, `lint.go`, `agent.go`, `ingest_system.md`, `query_system.md` | P0 |
| **P2: Conformance & Post-Processing** | 1. Implement `pkg/wiki/okf.go`: `ValidateOKF()`<br>2. Add `--okf` flag to `gmd wiki lint` (calls `ValidateOKF()`)<br>3. Add doctor checks: missing `okf_version`, missing `type`, stale `timestamp`<br>4. **Subdirectory index.md no-frontmatter enforcement** (OKF §6)<br>5. Implement `generateDescription()` (summarizer LLM) and `setTimestamp()` (deterministic) in `pkg/wiki/postprocess.go`<br>6. Add frontmatter preservation: export must preserve unknown frontmatter keys | `okf.go` (new), `lint.go`, `doctor.go`, `postprocess.go` (new) | P1 |
| **P3: Exchange** | 1. Add `gmd wiki okf export <name> [--output <dir>]` CLI + `ExportOKF()`<br>2. `[[wikilinks]]` → markdown link conversion in export (uses page registry from P1)<br>3. Wire `okf` subcommand group under `wiki` in cobra | `wiki_okf_export.go` (new), `okf.go`, cmd wiring | P2 |
| **P4: Hardening** | 1. Full test coverage (unit + integration with minimal OKF bundle fixture)<br>2. `gmd wiki lint --okf --strict` — non-zero exit on OKF violations<br>3. Docs update (agentsmd content, CLI help text, `docs/configuration.md`)<br>4. Final review pass: remove any remaining `_index.md`/`_log.md` refs, difficulty refs | Tests, `agentsmd/`, CLI help, `docs/configuration.md` | P3 |

---

## 9. Resolved Decisions

1. **[[wikilinks]] vs markdown links:** No backwards compatibility needed (alpha software). Ingest agent writes standard markdown links. `[[wikilinks]]` support retained on input as a convenience for human-authored pages but is syntactic sugar only — resolved the same way as markdown links. No dual-format bridging layer.

2. **Fixed wiki directory structure:** No fixed subdirectory layout. Wiki pages live wherever the user or agent places them. Any existing prescribed directory structure is removed. No `layout` config field.

3. **`--okf` flag for create:** No separate flag. The default template uses OKF names and conventions.

4. **Type taxonomy / internal categories:** No internal type taxonomy. `type` is a free-form string. No mapping of OKF types to GMD categories. No importing of external OKF bundles — ingest reads from `raw/` only.

5. **Timestamp handling:** Deterministic and enforced by the tool. `setTimestamp()` writes current ISO 8601 time on every page write. The agent is not asked to produce timestamps. Doctor/lint can flag stale timestamps vs file mtime.

6. **Description generation:** Auto-generated by the summarizer LLM via `generateDescription()` after page write. The ingest agent does not produce descriptions. The `index.md` entry reuses the frontmatter `description`.

7. **MCP integration:** Not in scope. MCP tools are unimplemented and deferred. No MCP-related code in this design.

8. **Log heading format:** Keep the richer `## [YYYY-MM-DD HH:MM] action | source` format. More information is better. On export, existing log entries are normalized if needed for strict OKF conformance, but native GMD log writing uses the richer format.
