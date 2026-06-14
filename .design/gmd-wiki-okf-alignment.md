# GMD Wiki — Open Knowledge Format (OKF) Alignment

**Date:** 2026-06-13 (created), 2026-06-13 (reviewed — v2)
**Phase:** Design — reviewed, ready for implementation planning
**OKF Version:** 0.1 (Draft)
**OKF SPEC:** https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md

---

## 1. Context & Goal

OKF defines an open, human- and agent-friendly format for knowledge bundles:
directories of markdown files with YAML frontmatter, cross-linked via standard
markdown links. It is intentionally minimal — no schema registry, no central
authority, no required tooling.

GMD already implements a Karpathy-style LLM Wiki with many OKF-adjacent
conventions: markdown + YAML frontmatter, hierarchical directories,
[[wikilinks]], index files, log files, cross-link graphs, and linking
constraints. The two systems overlap substantially but diverge on specific
choices that prevent interoperability.

**Goal:** Align GMD's wiki output, import, and exchange interfaces with the
OKF v0.1 specification so that:

1. **GMD wikis can be exported as valid OKF bundles** (a directory of markdown
   files consumable by any OKF-compatible system).
2. **GMD can import OKF bundles as wiki source material** (feeding the ingest
   agent from an external OKF bundle — local directory or git repo).
3. **GMD's internal wiki conventions are a strict superset of OKF** —
   additional features (status lifecycle, difficulty ratings, [[wikilinks]]
   for internal use) are layered on top without breaking conformance.
4. **GMD can serve as an OKF enrichment agent** — reading OKF bundles,
   synthesizing new concepts, and writing back conformant updates.

---

## 2. Gap Analysis — GMD Wiki vs OKF v0.1

| Concern | OKF v0.1 | GMD Wiki (current) | Gap |
|---|---|---|---|
| **Required frontmatter** | `type` (free-form string) | `type` is present in wiki_schema.md but not enforced in code; uses enum-like values (`entity`, `concept`, `comparison`, `source`, `synthesis`) | GMD `type` values are a subset of OKF's free-form model. OKF producers can use any string; GMD agents should output specific values that are valid OKF types. OKF conformance: validate non-empty `type`. |
| **Recommended frontmatter** | `title`, `description`, `resource`, `tags`, `timestamp` | `title` (often H1-derived), `tags`, `status`, `sources`, `difficulty`, `source_url` | Missing `description` (explicit), `resource` (canonical URI), `timestamp` (ISO 8601 last-modified). GMD extras (`status`, `difficulty`) are valid OKF extensions. |
| **Reserved filenames** | `index.md`, `log.md` | `_index.md`, `_log.md` (underscore-prefixed) | OKF explicitly reserves `index.md` and `log.md` without underscores. GMD prefix convention must be reconciled. |
| **Cross-linking** | Standard markdown links: `[text](/path/to/file.md)` (bundle-relative, recommended) or `[text](./other.md)` (relative) | `[[Page Name]]` wikilinks with optional alias/heading | **Largest structural difference.** OKF uses filesystem paths with `.md` extension; GMD uses page names. Must bridge both. |
| **Concept ID** | File path minus `.md` suffix (e.g., `tables/users`) | Page name (slugified title, e.g., `customer-orders`) | GMD page names are broader — they are independent of filesystem location. For OKF conformance, the file path IS the concept ID. |
| **Citations** | `# Citations` body section with numbered references | Query responses use `[[citations]]`; pages don't currently generate `# Citations` | Ingest agent should produce `# Citations` sections per OKF convention. |
| **Index format** | `index.md` — sections with `* [Title](url) - description` entries | `_index.md` — sections with `[[Page Name]] - description` entries | Same structure, different link syntax. |
| **Log format** | `log.md` — `## YYYY-MM-DD` date headings, bullet list of **bold-word** entries | `_log.md` — `## [YYYY-MM-DD HH:MM] action | source` headings, bullet detail | Slightly different heading format. Trivial to align. |
| **Bundle structure** | Arbitrary directory hierarchy; producers organize however makes sense | Prescribed layout: `entities/`, `concepts/`, `comparisons/`, `synthesis/`, `sources/` | GMD layout is a valid OKF bundle structure. Additional subdirectories are OKF-compatible. |
| **OKF version declaration** | `okf_version: "0.1"` in bundle-root `index.md` frontmatter | Not present | Trivial to add. |
| **Body convention sections** | `# Schema`, `# Examples`, `# Citations` have conventional meaning | No conventional body sections enforced in ingest prompts | Ingest prompt should be updated to use OKF conventional headings. |
| **Forbidden types** | No fixed taxonomy; consumers tolerate unknown types | GMD defines fixed types (entity, concept, comparison, source, synthesis) | Compatible — GMD's types are valid OKF type values. But GMD lint/doc should clarify that arbitrary type values are valid. |
| **Conformance validation** | 1) Every .md has YAML frontmatter, 2) Every frontmatter has non-empty `type`, 3) Reserved files follow structure | No formal conformance check; `lint` checks orphans, broken links, stale entries | Need `gmd wiki okf-validate` or extend `lint` with OKF conformance rules. |

---

## 3. Design Decisions

### 3.1 File Naming: `index.md` / `log.md` vs `_index.md` / `_log.md`

**Decision:** Adopt OKF names as the canonical names. Make the underscore-prefixed
variants deprecated aliases.

- **Config schema**: Change `WikiConfig.indexFile` default from `"_index.md"` to `"index.md"`, and `logFile` from `"_log.md"` to `"log.md"`.
- **New wikis**: `gmd wiki create` scaffolds `index.md` and `log.md` (without underscores).
  **Also**: update the hardcoded `Ignore` patterns in `cmd/gmd/wiki_create.go:82-98`
  that currently ignore `_index.md` and `_log.md` to use the config values instead.
- **Existing wikis**: Read either name. If both exist, prefer the non-underscore version.
  A `gmd wiki doctor` check reports wikis still using underscore-prefixed files and
  offers rename via `--fix`.
- **All meta-file skip sites**: The codebase has several places that hardcode
  `strings.HasPrefix(base, "_")` to skip meta files. These must be updated to check
  against `WikiConfig.IndexFile`/`WikiConfig.LogFile` instead:
  - `pkg/wiki/graph.go:41` (BuildGraph skip)
  - `pkg/wiki/lint.go:152` (lintContent skip)
  - `pkg/wiki/watch.go:87` (checkWiki skip)
  - Note: `pkg/wiki/lint.go:67` (lintStructure) already checks config names correctly.
- **Dual-file race condition**: When both `_index.md` and `index.md` exist, the agent
  must not create a second conflicting index. Every read site must use the same
  fallback logic (try canonical name, fallback to underscored). Read sites:
  `loadIndexContext()` (agent.go:400), `appendLogFile()` (agent.go:272),
  `updateIndexFile()` (agent.go:218), `lintStructure()` (lint.go:67),
  `lintContent()` (lint.go:152), `lintGaps()` (lint.go:190). Write sites
  (`updateIndexFile`, `appendLogFile`) always write to the configured name.
- **gitignore**: Ensure `.gmd/` patterns ignore `log.md` and `index.md` are handled
  (these are human-editable, not generated-only). The files are part of the bundle
  (checked into source control), not stored in `.gmd/`.

### 3.2 Cross-linking: Dual [[wikilinks]] + Markdown Links

**Decision:** Support both formats in parallel. Write in OKF format by default.

**Parsing (read path):**
- Extend `pkg/chunking/markdown.go` `ExtractWikilinks()` to also extract
  standard markdown links that reference `.md` files (bundle-relative or relative).
- Both link types populate the `Links` field in `Chunk` structs and
  `ChunkDocument.Links` in Typesense.
- Graph building (`pkg/wiki/graph.go`) reads both link types.

**Writing (output path):**
- Ingest agent (`pkg/wiki/agent.go`) writes standard markdown links for
  cross-references: `[Page Name](/path/to/page.md)` (bundle-relative, OKF
  preferred form).
- `[[wikilinks]]` are still accepted on input. The ingest prompt tells the
  LLM to output OKF-standard markdown links.
- The query agent synthesizes answers with OKF standard links in `[[citations]]`.

**Link resolution (for graph, neighbors, lint):**
- From `[[Page Name]]`: look up `Page Name` in wiki directory → resolve to file path.
  This requires building a page-name → file-path registry by scanning the wiki
  directory and parsing each page's H1 or frontmatter `title`.
- From `[text](/path/to/file.md)`: strip leading `/` and `.md` suffix → concept ID.
- From `[text](./file.md)` or `[text](../other/file.md)`: resolve relative to
  source page's directory → absolute bundle-relative path → concept ID.
  This requires the link extractor to carry the source file's directory context.
  The current `BuildGraph()` delegates to `chunking.ExtractWikilinks()` which
  has no directory context — the API must be extended to pass the source directory.
- **Unified node identity**: Both page names and concept IDs must canonicalize to
  the same namespace. When a page contains both `[[Transformer]]` and
  `[Transformer](/concepts/transformer.md)`, these must produce ONE link, not two.
  Deduplication by resolved concept ID (file path minus `.md`) happens at the
  graph/link level, not at the raw extraction level.
- Broken link detection handles both formats, reporting the original link text
  for actionable diagnostics.

### 3.3 Frontmatter: OKF Required + Recommended Fields

**Decision:** GMD's ingest agent outputs the OKF required field plus recommended
fields. GMD wiki pages are a superset of OKF.

**Ingest agent output:**
```yaml
---
type: entity              # REQUIRED by OKF; GMD uses entity|concept|comparison|source|synthesis
title: Transformer        # OKF recommended (derived from H1 if absent)
description: The transformer architecture uses self-attention...  # OKF recommended
resource: https://arxiv.org/abs/1706.03762   # OKF recommended (canonical URI)
tags: [ai, transformer, architecture]  # OKF recommended
timestamp: 2026-06-13T14:30:00Z  # OKF recommended (ISO 8601)
# GMD extensions (OKF permits arbitrary keys):
status: draft
difficulty: 3
sources: [source-page.md]
---
```

**Config schema additions** (`WikiConfig.FrontmatterFields`):
- `fields` are the OKF-recommended set: `type` (required), `title`, `description`, `resource`, `tags`, `timestamp`.
- GMD's existing fields (`status`, `sources`, `difficulty`, `source_url`) remain as extensions.
- `okf_version` is a bundle-level field on `index.md` frontmatter only.

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
- GMD detects `okf_version` on import and warns when a bundle targets a
  version the tool doesn't fully support.
- Missing `okf_version` means OKF v0.0 (pre-spec) — consume best-effort.

### 3.7 Body Convention Sections

**Decision:** Update ingest prompts to produce OKF conventional body sections.

- `# Schema` — for structured descriptions of an asset's columns/fields
- `# Examples` — for concrete usage examples
- `# Citations` — for external sources (see §3.4)
- Existing GMD conventions (e.g., `# See Also`) are retained as additional sections.

---

## 4. Conformance — GMD as OKF Consumer & Producer

### 4.1 GMD as OKF Producer (Export)

When `gmd wiki export <name>` is run (new command), or when wiki pages are
written by the ingest agent:

1. Every `.md` file has YAML frontmatter with a non-empty `type` field.
2. Reserved files (`index.md`, `log.md`) follow OKF structure (§6, §7).
3. Cross-links use standard markdown bundle-relative paths.
4. `index.md` has `okf_version: "0.1"` in frontmatter.
5. All `.md` files are UTF-8 encoded.

### 4.2 GMD as OKF Consumer (Import)

When `gmd wiki import <bundle-path>` is run (new command), or when
`gmd wiki ingest` processes an OKF bundle:

1. **Tolerate unknown `type` values** — treat them as generic concepts (OKF §4.1).
2. **Tolerate missing optional fields** — `title`, `description`, `resource`, etc. (OKF §9).
3. **Tolerate broken links** — link targets that don't exist are not errors (OKF §5.3).
4. **Tolerate missing `index.md`** — synthesize one if absent (OKF §6).
5. **Tolerate unknown `okf_version`** — attempt best-effort consumption (OKF §11).
6. **Preserve unknown frontmatter keys** when round-tripping (OKF §4.1).

### 4.3 GMD as OKF Enrichment Agent

GMD's ingest agent is an OKF enrichment agent: it reads OKF concepts, extracts
knowledge, and writes new/updated concepts back into the bundle. This is core
to the Karpathy compounding wiki pattern and aligns with OKF's stated goal of
defining a format that enrichment agents can write into (OKF §1, Goal 1).

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
    frontmatter?: {
        fields: [string]: FrontmatterField
    }
}
```

### 5.2 Config defaults (`pkg/config/config.go`)

Apply runtime defaults:
- `indexFile` → `"index.md"` if empty (was `"_index.md"`)
- `logFile` → `"log.md"` if empty (was `"_log.md"`)
- `okfVersion` → `"0.1"` if empty

### 5.3 Wiki Agent defaults

`pkg/wiki/wiki.go` `Init()`:
- Scaffold `index.md` (not `_index.md`), with `okf_version: "0.1"` frontmatter.
- Scaffold `log.md` (not `_log.md`).
- For backwards compat: `readIndexFile()` and `readLogFile()` try the canonical
  name first, fall back to underscored name.

---

## 6. New / Changed Commands

| Command | Status | Description |
|---|---|---|
| `gmd wiki lint <name> [--okf]` | Modified | `--okf` flag adds OKF conformance checks (every .md has frontmatter with `type`, reserved files follow structure). Calls the shared `ValidateOKF()` library function. Broken markdown links reported when detected (tolerant by default). `--strict` upgrades to non-zero exit on broken links. |
| `gmd wiki okf export <name> [--output <dir>]` | New | Export wiki as a standalone OKF bundle directory. Converts [[wikilinks]] → markdown links (requires page-name → file-path registry), ensures frontmatter compliance. |
| `gmd wiki okf import <bundle-path> [--as <wiki-name>]` | New | Import an external OKF bundle as a new wiki (or add to existing). Copies to `raw/` for ingest or directly scaffolds wiki pages. |
| `gmd wiki ingest <name> [source]` | Modified | Accepts OKF bundles as sources. When source is an OKF bundle directory, map concepts to wiki pages with OKF linking conventions. |
| `gmd wiki doctor <name>` | Modified | Add check: underscores-prefixed index/log files → offer rename. Add check: missing `okf_version` in index.md → offer to add. Add check: pages missing `type` frontmatter. Add check: pages with stale `timestamp` vs file mtime. |
| `gmd wiki create <name>` | Modified | Scaffold `index.md`/`log.md` (no underscores). Write `okf_version: "0.1"` in `index.md` frontmatter. Update hardcoded ignore patterns to use config values. |

---

## 7. Code Changes Required

### 7.1 Link Parsing (`pkg/chunking/markdown.go`)

**Current:** `ExtractWikilinks()` parses only `[[...]]` syntax.
**Change:** Add `ExtractMarkdownLinks()` that captures standard markdown links
targeting `.md` files. Both populate the `Links` field. The extractor should
normalize to concept IDs (strip leading `/`, strip `.md` suffix).
Relative links require source-directory context passed by the caller.

```go
// New regex for standard markdown links pointing to .md files
var mdLinkRe = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+\.md)\)`)

// ExtractMarkdownLinks extracts all markdown links targeting .md files.
// sourceDir is the directory of the source file for resolving relative links.
func ExtractMarkdownLinks(content string, sourceDir string) []string {
    // "/path/to/file.md" → "path/to/file"
    // "./file.md" → "sourceDir/file" (resolved against sourceDir)
    // "../other/file.md" → resolved against sourceDir
    // Deduplicated by resolved concept ID
}

// NormalizeConceptID converts a link target to a unified concept ID
func NormalizeConceptID(linkTarget string) string {
    // Strip leading "/" and trailing ".md"
}
```

### 7.2 Link Writing (`pkg/wiki/agent.go`)

**Change ingest prompts** (`embeds/ingest_system.md`):
- Instruct LLM to output standard markdown links: `[Page Title](/path/to/page.md)`
  not `[[Page Title]]`.
- Add `# Citations` section generation.
- Add OKF conventional body sections (`# Schema`, `# Examples`).
- Add `description` field to frontmatter (distinct from the index summary —
  the index entry should reuse the frontmatter `description` to avoid
  LLM duplication).

**Change query prompts** (`embeds/query_system.md`):
- Instruct LLM to use standard markdown links in inline citations.

**Update `updateIndexFile()` (agent.go:218):**
- Write OKF-format index entries: `* [Title](relative/url.md) - description`
  instead of `- [[Page Name]] — description`.

**Update `saveQueryResult()` (agent.go:371):**
- Replace `## Sources` section (with `[[wikilinks]]`) with `# Citations`
  section (numbered references with standard markdown links).
- Add `description`, `timestamp` to saved page frontmatter.

**Update `appendLogFile()` (agent.go:272):**
- Align heading format with OKF: `## YYYY-MM-DD` (date only) in the OKF canonical
  format output. GMD keeps `## [YYYY-MM-DD HH:MM] action | source` as the internal
  format for `log.md` entries (valid OKF extension since the body format is
  convention, not requirement in OKF §7).

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
  For each page: check frontmatter exists, `type` is non-empty, reserved files
  follow structure. Report violations. This walk must also skip configured meta
  files (`IndexFile`/`LogFile`) — there is a second walk in `lintStructure()`
  (lint.go:97-110) for broken-link source collection that currently has no
  meta-file skip guard and would include `index.md`/`log.md` entries incorrectly.
  Both walks need the guard.
- `lintBrokenMarkdownLinks()`: Extend existing `lintStructure()` broken-link
  detection to cover markdown `.md` links. The current broken-link check
  (`lint.go:91-112`) only tests `[[wikilinks]]`. Refactor to a single pass
  that checks both link types.
- `lintMissingDescriptions()`: Flag pages without a `description` frontmatter
  field (soft warning, not error).
- `lintStaleTimestamp()`: Flag pages whose file mtime is newer than frontmatter
  `timestamp` (soft warning).

### 7.5 Doctor (`pkg/wiki/doctor.go`)

**Add checks:**
- **Underscore-prefixed files**: Detect `_index.md` or `_log.md` in wiki dir.
  Report: "wiki uses legacy underscored meta files. Run --fix to rename to
  index.md / log.md." Fix: rename files, update any ignore patterns.
- **Missing `okf_version`**: Check `index.md` frontmatter for `okf_version`.
  Fix: write `okf_version: "0.1"` into frontmatter.
- **Missing `type` frontmatter**: Walk `.md` files, check for `type` field.
  No auto-fix — requires agent or human intervention. Report count.
- **Connectivity**: Existing Typesense and LLM checks (already implemented).
- **Wiki file stats**: New: count pages, index entries, log entries, track
  last ingest timestamp from log.

### 7.6 Embedded Templates (`pkg/wiki/embeds/`)

**Update `wiki_schema.md`:**
- Replace `_index.md` with `index.md`, `_log.md` with `log.md` throughout.
- Replace `[[wikilinks]]` in examples with standard markdown links.
- Add OKF cross-reference: mention that the wiki follows OKF v0.1 conventions.
- Add `description`, `resource`, `timestamp` to the frontmatter schema example.

**Update `ingest_system.md`:**
- Use standard markdown links in template examples.
- Add `# Citations` section generation instructions.
- Add `# Schema`, `# Examples` conventional section instructions.
- Add `description` and `timestamp` to required frontmatter output.

**Update `query_system.md`:**
- Use standard markdown links in citation examples.

### 7.7 Wiki Creation (`cmd/gmd/wiki_create.go`)

**Changes:**
- Lines 82-83: Replace hardcoded `"_index.md"`, `"_log.md"` with config values
  (`cfg.WikiConfig.IndexFile`, `cfg.WikiConfig.LogFile`) or use the newly
  defaulted names.
- Line 98: Replace hardcoded `AddIgnorePatterns` call similarly.
- The scaffolded `index.md` content must include frontmatter with `okf_version`.

### 7.8 Meta-File Skip Guards (ultiple files)

Update all `strings.HasPrefix(filepath.Base(path), "_")` guards to check
against `WikiConfig.IndexFile`/`WikiConfig.LogFile`:
- `pkg/wiki/graph.go:41` — `BuildGraph()` skip
- `pkg/wiki/lint.go:152` — `lintContent()` skip
- `pkg/wiki/watch.go:87` — `checkWiki()` skip

### 7.9 CUE Schema (`pkg/config/embeds/types.cue`)

Add `okfVersion` and `layout` fields to `WikiConfig`:

```cue
WikiConfig: Source & {
    // ... existing fields ...
    okfVersion:  string | *"0.1"            // NEW: declared OKF version
    layout:      string | *"categorized"     // NEW: "categorized" | "flat"
    // ...
}
```

### 7.10 Go Config Struct (`pkg/config/config.go`)

Add corresponding fields to `WikiConfig` Go struct (currently at line 550):
```go
type WikiConfig struct {
    SourceConfig
    // ... existing fields ...
    OkfVersion string `json:"okfVersion"` // default "0.1"
    Layout     string `json:"layout"`     // default "categorized"
}
```

### 7.11 Wiki Init (`pkg/wiki/wiki.go`)

- `Init()`: Scaffold `index.md` (with `okf_version` frontmatter) and `log.md`.
- `readIndexFile()` / `readLogFile()`: Fallback logic — try canonical name,
  fall back to underscored name.
- `createWikiPage()` (agent.go:174): Include `description`, `timestamp`,
  `resource` in frontmatter when applicable.

### 7.12 New Library (`pkg/wiki/okf.go`)

- `ValidateOKF(wiki *Wiki)` — conformance checking (shared by lint and export).
  Returns `OKFReport` with violations, warnings, passthrough count.
- `ExportOKF(wiki *Wiki, outputDir string)` — convert wiki to OKF bundle.
  Builds page-name → file-path registry. Converts `[[wikilinks]]` → markdown links.
  Writes conformant `index.md` / `log.md`. Copies all `.md` files.
- `ImportOKF(bundlePath string, opts)` — map OKF bundle concepts to wiki pages.
  Walks bundle directory, reads frontmatter from each concept, maps to wiki
  subdirectory, inserts into `raw/` for ingest.

### 7.13 New CLI Commands (`cmd/gmd/`)

- `wiki_okf_export.go` — `gmd wiki okf export <name> [--output <dir>]`
- `wiki_okf_import.go` — `gmd wiki okf import <bundle-path> [--as <wiki-name>]`
- OKF validation lives under `gmd wiki lint --okf`, not a separate command.
- **CLI pattern note**: Export/import use a subcommand group (`gmd wiki okf ...`)
  while validation uses a flag (`gmd wiki lint --okf`). This is deliberate:
  export/import are standalone operations; validation is an additional check
  mode on the existing lint command that shares lint's output format,
  `--fix` semantics, and report structure.

### 7.14 MCP Tools (`pkg/mcp/wiki_tools.go`)

After the library functions exist (P3), expose OKF operations as MCP tools:
- `okf_validate` — validate wiki bundle conformance
- `okf_export` — export wiki as OKF bundle
- `okf_import` — import external OKF bundle

This enables Path B (external agent) OKF workflows.

### 7.15 Tests

| Area | Test File | What |
|---|---|---|
| Link parsing | `pkg/chunking/markdown_test.go` | Test `ExtractMarkdownLinks()`, normalization, both link types in same file, relative link resolution. |
| Link resolution | `pkg/wiki/graph_test.go` | Test graph building with markdown links, relative link resolution, mixed link types, dedup of same-target links. |
| Frontmatter | `pkg/wiki/frontmatter_test.go` | Test OKF required+recommended fields, unknown field preservation. |
| OKF conformance | `pkg/wiki/okf_test.go` (new) | Test validating a bundle: pass, fail on missing type, fail on reserved file misuse. |
| OKF import | `pkg/wiki/okf_integration_test.go` (new) | Import a minimal OKF bundle, verify concepts mapped to wiki pages. |
| OKF export | `pkg/wiki/okf_integration_test.go` (new) | Export a wiki, verify output is OKF-conformant, verify wikilink conversion. |
| Ingest prompts | `pkg/wiki/agent_prompts_test.go` | Verify prompts reference OKF conventions, standard markdown links. |
| Lint | `pkg/wiki/lint_test.go` | Test OKF conformance lint checks, broken markdown links, missing description/timestamp. |
| Doctor | `pkg/wiki/doctor_test.go` | Test underscore-prefix detection, okf_version check, type-frontmatter check. |
| Index/Log update | `pkg/wiki/agent_test.go` | Test `updateIndexFile()` writes OKF-format entries, `saveQueryResult()` uses `# Citations`. |

---

## 8. Migration Path for Existing Wikis

### Phase 1: Soft migration (backward-compatible)
- Existing `_index.md` and `_log.md` continue to work.
- Agent reads from both underscored and non-underscored filenames.
- New content written to `index.md` / `log.md`.
- `gmd wiki doctor` detects legacy filenames and suggests migration.
- `[[wikilinks]]` continue to work for existing pages; new pages written
  by ingest agent use markdown links.

### Phase 2: Hard migration (opt-in)
- `gmd wiki doctor --fix` renames `_index.md` → `index.md`,
  `_log.md` → `log.md`.
- `gmd wiki okf export` converts `[[wikilinks]]` → markdown links in output.
- Existing `[[wikilinks]]` in wiki pages not rewritten in-place (this is
  a human review step). The export command handles conversion for exchange.

### Phase 3: Full OKF conformance (future)
- All new wiki pages written in OKF format exclusively.
- `[[wikilinks]]` support maintained indefinitely for reading (backward-compat),
  with a deprecation notice in docs. No removal planned — the dual-format
  support is a feature, not a transition state.
- `gmd wiki lint --strict` flags `[[wikilinks]]` as non-OKF (soft warning).

---

## 9. Implementation Plan

| Phase | Scope | Key Files | Dependencies |
|---|---|---|---|
| **P0: Foundation** | 1. Rename defaults: `indexFile` → `"index.md"`, `logFile` → `"log.md"` in config + CUE<br>2. Add `okfVersion` to CUE schema + config defaults<br>3. Update `embeds/wiki_schema.md` (OKF names, markdown links, new fields)<br>4. Update `wiki create` scaffolding + hardcoded ignore patterns<br>5. Update all `_`-prefix skip guards in graph.go, lint.go, watch.go<br>6. Add fallback reads: `readIndexFile()`, `readLogFile()` try canonical → underscored<br>7. `Init()` writes `okf_version` in `index.md` frontmatter | `types.cue`, `config.go`, `wiki_create.go`, `wiki.go`, `wiki_schema.md`, `graph.go:41`, `lint.go:152`, `watch.go:87` | None |
| **P1: Links** | 1. Implement `ExtractMarkdownLinks()` in `chunking/markdown.go`<br>2. Update `BuildGraph()` for dual link types + relative resolution<br>3. Update broken-link detection in `lintStructure()` for markdown links<br>4. Update `ingest_system.md` prompt: markdown links, `# Citations`, `# Schema`, `# Examples`, `description`<br>5. Update `query_system.md` prompt: markdown link citations<br>6. Update `updateIndexFile()`: OKF index entry format<br>7. Update `saveQueryResult()`: `# Citations` instead of `## Sources`, add `description`/`timestamp`<br>8. Update `createWikiPage()`: include `description`, `timestamp`, `resource` | `markdown.go`, `graph.go`, `lint.go`, `ingest_system.md`, `query_system.md`, `agent.go` | P0 |
| **P2: Conformance** | 1. Implement `pkg/wiki/okf.go`: `ValidateOKF()`, `ExportOKF()`, `ImportOKF()`<br>2. Add `--okf` flag to `gmd wiki lint` (calls `ValidateOKF()`)<br>3. Add doctor checks: underscore files, missing `okf_version`, missing `type`, stale `timestamp`<br>4. Add logic for page-name → file-path registry (needed by both ExportOKF and graph resolution)<br>5. Update `appendLogFile()` format alignment | `okf.go` (new), `lint.go`, `doctor.go`, `agent.go` | P1 |
| **P3: Exchange** | 1. Add `gmd wiki okf export` CLI + `ExportOKF()` implementation<br>2. Add `gmd wiki okf import` CLI + `ImportOKF()` implementation<br>3. `[[wikilinks]]` → markdown link conversion in export (uses page registry)<br>4. OKF bundle → wiki page mapping in import<br>5. Add MCP tools: `okf_validate`, `okf_export`, `okf_import` | `wiki_okf_export.go` (new), `wiki_okf_import.go` (new), `okf.go`, `wiki_tools.go` | P2 |
| **P4: Hardening** | 1. Full test coverage (unit + integration with minimal OKF bundle fixture)<br>2. `gmd wiki lint --strict` — non-zero exit on OKF violations<br>3. Docs update (agentsmd content, CLI help text)<br>4. `index.md` synthesis for OKF bundles without one | Tests, `agentsmd/`, CLI help | P3 |

---

## 10. Open Questions

1. **Should `[[wikilinks]]` be fully deprecated in favor of markdown links?**
   Pro: cleaner alignment, one link syntax. Con: [[wikilinks]] are more concise
   and visually distinct in wiki pages intended for human reading. **Lean:** Keep
   both, default to markdown links in agent output. Revisit when OKF
   community norms settle. No removal planned.

2. **Should the fixed wiki directory structure (entities/, concepts/, etc.)
   become configurable/suppressed for OKF import?**
   OKF allows arbitrary hierarchy. **Lean:** Make the wiki layout configurable
   via `WikiConfig.layout` (e.g., `"flat"` vs `"categorized"`). Keep
   `"categorized"` as default for GMD internal wikis. Use `"flat"` when importing
   an OKF bundle to preserve its original structure.

3. **Should `gmd wiki create` accept an `--okf` flag to scaffold a bundle that
   is a more literal OKF bundle (less GMD convention, more generic structure)?**
   **Lean:** No separate flag. The default template already aligns with OKF.
   Advanced customization via config fields.

4. **How should `type` values be handled for wikis that are imported, not
   generated by GMD's agent?**
   The agent maps ingested type values to GMD's internal categories if possible,
   but preserves the original value. Unknown types are treated as generic
   concepts — which is explicitly OKF-conformant behavior.

5. **Should GMD enforce `timestamp` auto-update on wiki page modification?**
   OKF spec says `timestamp` is "last meaningful change" (not file stat).
   **Lean:** The ingest agent sets `timestamp` when it creates/updates a page.
   Manual edits by humans don't auto-update. Lint can flag pages where the file
   modification time is newer than the frontmatter `timestamp` as a "needs timestamp
   update" warning.

6. **What about the `description` field — should GMD auto-generate it?**
   Currently, the ingest agent doesn't produce a `description` field. The one-line
   summary in `_index.md`/`index.md` entries serves the same purpose.
   **Lean:** The ingest prompt should instruct the LLM to include a `description`
   in frontmatter. The `index.md` entry should reuse the frontmatter `description`
   as its summary text rather than having the LLM produce the summary twice
   (once in `description`, once in `index_updates.summary`). The ingest JSON
   contract's `IndexUpdates[].Summary` field becomes optional — if empty, the
   agent copies from the corresponding page's `description`.

7. **How does this interact with MCP tools and external agent paths (Path B)?**
   MCP tools should expose OKF operations: `okf_validate`, `okf_export`,
   `okf_import`. External agents using GMD as MCP can then read/write OKF
   bundles through the same interface. Implementation tracked in P3.

8. **Should `log.md` heading format strictly match OKF (`## YYYY-MM-DD` date-only)
   or keep GMD's richer `## [YYYY-MM-DD HH:MM] action | source` format?**
   OKF §7 says date headings MUST use `YYYY-MM-DD` form but the entry format
   (bold words, detail text) is convention, not requirement. GMD's extended
   heading is a valid structural extension. **Lean:** Keep GMD's format in
   `log.md` (more useful for agent operations). The heading includes the OKF
   date prefix in valid ISO 8601 form. If exporting a bundle and strict OKF
   is requested, the export step strips the time and action prefix.

---

## Appendix A — Example: OKF-Conformant Wiki Page (GMD-Generated)

```markdown
---
type: concept
title: Transformer Architecture
description: The transformer architecture uses self-attention mechanisms to process
  sequential data without recurrence, enabling parallel computation and capturing
  long-range dependencies.
resource: https://arxiv.org/abs/1706.03762
tags: [ai, machine-learning, transformer, architecture, attention]
timestamp: 2026-06-13T14:30:00Z
status: draft
difficulty: 3
sources: [2026-06-13-attention-is-all-you-need.md]
---

# Transformer Architecture

The transformer architecture, introduced in "Attention Is All You Need" (Vaswani
et al., 2017), replaces recurrent layers with [self-attention](/concepts/scaled-dot-product-attention.md)
mechanisms.

## Schema

| Component | Input | Output | Description |
|---|---|---|---|
| Multi-Head Attention | Query, Key, Value | Weighted sum | Parallel attention heads |
| Feed-Forward Network | Attention output | Transformed vectors | Position-wise FFN |
| Layer Normalization | Residual + sublayer output | Normalized vectors | Stabilizes training |

## Examples

```python
# Simplified transformer encoder block
def transformer_block(x):
    attn = multi_head_attention(x, x, x)
    x = layer_norm(x + attn)
    ff = feed_forward(x)
    return layer_norm(x + ff)
```

## Citations

[1] Vaswani et al., "Attention Is All You Need", arXiv:1706.03762, 2017.
[2] [The Illustrated Transformer](/concepts/illustrated-transformer.md) by Jay Alammar.

## See Also

- [Scaled Dot-Product Attention](/concepts/scaled-dot-product-attention.md)
- [BERT](/entities/bert.md)
- [GPT](/entities/gpt.md)
```

## Appendix B — Example: Minimal OKF Bundle for Import Testing

```
testdata/okf-minimal/
├── index.md
├── datasets/
│   ├── index.md
│   └── sales.md
└── tables/
    ├── index.md
    ├── orders.md
    └── customers.md
```

This matches the OKF SPEC Appendix A example and serves as an integration test fixture.
