# Wiki–Collection Unification

**Date:** 2026-06-04
**Motivation:** Wikis are currently second-class entities nested inside `CollectionConfig`.
This makes configuration, CLI conventions, and operational semantics inconsistent with collections.
Wikis should be first-class entities with the same ergonomics as collections, plus their own
agent-driven operations (ingest, query, lint, graph, doctor).

---

## Design Goals

1. **First-class wikis** — wikis are top-level config entities, parallel to collections.
2. **Zero-or-more aggregation** — a wiki can stand alone (no collection references) or
   aggregate content from one or more collections.
3. **Searchable like a collection** — wiki content lives in the same Typesense `chunks`
   collection as everything else. `gmd query` and `gmd search` work on wikis identically
   via the existing `--collection` flag.
4. **Consistent CLI** — wiki commands use positional args (like collections), not `--name` flags.
5. **Clean separation** — a collection is a file-indexing unit. A wiki is a knowledge base
   with LLM agent operations. They are separate concepts that can interoperate.
6. **Shared indexing model** — collections and wikis share the same indexing fields
   (`path`, `patterns`, `ignore`, `context`, `fields`) via a shared `SourceConfig`.

---

## 1. CUE Schema

### Current (to be replaced)

```cue
// types.cue — current, nested approach
CollectionConfig: {
    path:     string
    patterns: [...string]
    ignore?:  [...string]
    context?: string
    includeByDefault?: bool | *true
    wiki?:    WikiConfig | *null   // ← wiki is optional sub-field of a collection
    fields?:  [string]: FrontmatterField
}

ProjectConfig: {
    project?:    string
    llm:         LLMConfig
    typesense:   TypesenseConfig
    pipeline?:   PipelineConfig
    collections: [string]: CollectionConfig
    // no wikis top-level map
}
```

### New

A private `#Source` constraint captures the shared indexing fields. Both `CollectionConfig`
and `WikiConfig` unify with it, keeping the schema DRY.

```cue
// #Source defines shared file-indexing configuration used by both
// collections and wikis. Both entity types are indexed into the same
// Typesense chunks collection.
#Source: {
    path:     string
    patterns: [...string]
    ignore?:  [...string]
    context?: string
    fields?:  [string]: FrontmatterField
}

// WikiConfig defines an LLM wiki — a compounding knowledge base with
// agent-driven content generation, wikilinks, and optional collection aggregation.
// Collection commands (show, include, exclude, context) accept wiki names
// identically. Wiki CLI commands delegate to the same collection CRUD internals.
WikiConfig: #Source & {
    wikiDir:     string | *"wiki"        // subdirectory for wiki content pages
    rawDir:      string | *"raw"         // subdirectory for raw source material
    indexFile:   string | *"_index.md"
    logFile:     string | *"_log.md"
    graphLinks:  bool | *true
    excludeFromDefault?: bool | *false    // opt-out of default (unscoped) searches

    // Aggregation: when searching this wiki, also search these named sources
    // (collections or other wikis). Each entry must be a key in the top-level
    // collections or wikis map — validation at create/add time rejects
    // unknown names and circular references.
    sourceRefs?: [...string]

    // Wiki-specific frontmatter configuration. Separate from #Source.fields
    // (which controls Typesense field indexing) so wiki frontmatter keys never
    // collide with gmd's own indexing field names.
    frontmatter?: {
        fields: [string]: FrontmatterField
    }
}

// ProjectConfig — updated with wikis top-level map and searchDefaults
ProjectConfig: {
    project?:    string
    llm:         LLMConfig
    typesense:   TypesenseConfig
    pipeline?:   PipelineConfig
    collections: [string]: CollectionConfig
    wikis:       [string]: WikiConfig

    // searchDefaults defines named search presets. Each key is a preset name
    // used with --search, and the value is the list of source names
    // (collections and/or wikis) to search in that preset. When a search uses
    // --search <preset>, only the listed sources are included, overriding
    // the default behavior. When --search is not used, unscoped search
    // includes all sources where excludeFromDefault is false. searchDefaults
    // does NOT intersect with or override excludeFromDefault for unscoped
    // searches — it only takes effect when explicitly invoked via --search.
    searchDefaults?: [string]: [...string]
}

// Usage example:
//   searchDefaults: {
//       "research": ["docs", "papers", "mywiki"]   // named preset
//       "codeonly": ["src"]
//   }

```

### Changes to `CollectionConfig`

Remove the `wiki?: WikiConfig | *null` field. Collections unify with `#Source`.
Both collections and wikis have `excludeFromDefault` (defaulting to `false`, meaning
sources are included in unscoped searches by default).

```cue
CollectionConfig: #Source & {
    excludeFromDefault?: bool | *false
    // wiki field removed
}
```

### Name uniqueness: collections and wikis share a namespace

Collection names and wiki names must be unique across both maps. A collection named `docs` and
a wiki named `docs` would produce the same Typesense `collection` facet value via `CollectionKey()`,
making commands like `gmd collection show docs` and `gmd ls docs` ambiguous.

`gmd wiki create <name>` validates that `<name>` does not already exist in either
`collections` or `wikis`. `gmd collection create <name>` does the same.

### Conflict detection for wiki subdirectories

When `gmd wiki create` runs, the following checks are performed and any conflict is a
hard error (the user must rename or reconfigure):

1. **wikiDir == rawDir** — the two subdirectories must be distinct.
2. **Cross-wiki collision** — a wiki's resolved `wikiDir` and `rawDir` must not overlap
   with `wikiDir`/`rawDir` of any existing wiki at the same `path`.
3. **Wiki vs. collection collision** — resolved `wikiDir` and `rawDir` must not overlap
   with resolved paths of any existing collection.

The same checks run when `gmd collection include` or `gmd wiki include` adds patterns
that would cause a collection's resolved files to overlap with a wiki's `wikiDir`/`rawDir`.
Warn on overlap, allow the user to proceed with confirmation.

The directories `wikiDir` and `rawDir` are relative to `WikiConfig.path`. They are
independent — a user may place them at any non-overlapping locations under `path`, whether
sibling directories or nested in their own preferred structure.

### Circular sourceRefs detection

Circular `sourceRefs` (e.g. Wiki A → Wiki B → Wiki A) cause infinite expansion during
search. `gmd wiki create` validates the sourceRefs graph for cycles. The same check runs
when `gmd wiki ref add` modifies the graph.

```go
// HasSourceRefsCycle performs a global DFS from every wiki to detect any cycle.
func (c *Config) HasSourceRefsCycle() ([]string, bool) {
    // DFS from each wiki; return first cycle found
}

// WouldCreateSourceRefsCycle checks whether adding edge src → target would create a
// cycle. DFS from target to see if src is reachable (i.e. target already
// transitively references src). Used for incremental validation on ref add.
func (c *Config) WouldCreateSourceRefsCycle(src, target string) bool {
    // DFS from target; return true if src is reachable
}
```

---

## 2. Go Types

### `pkg/config/config.go`

A shared `SourceConfig` struct is embedded in both `CollectionConfig` and `WikiConfig`.
CUE `Decode` handles embedded structs transparently (fields are flattened into the JSON output).

```go
// SourceConfig holds file-indexing fields shared by collections and wikis.
type SourceConfig struct {
    Path     string                      `json:"path"`
    Patterns []string                    `json:"patterns"`
    Ignore   []string                    `json:"ignore,omitempty"`
    Context  string                      `json:"context,omitempty"`
    Fields   map[string]FrontmatterField `json:"fields,omitempty"`
}

type Config struct {
    LLM            LLMConfig                   `json:"llm"`
    Typesense      TypesenseConfig             `json:"typesense"`
    Web            WebConfig                   `json:"web,omitempty"`
    Pipeline       PipelineConfig              `json:"pipeline"`
    Collections    map[string]CollectionConfig `json:"collections"`
    Wikis          map[string]WikiConfig       `json:"wikis"`            // ← NEW
    SearchDefaults map[string][]string         `json:"searchDefaults,omitempty"` // named search presets: preset → source names
    ProjectRoot    string                      `json:"-"`
    Project        string                      `json:"project,omitempty"`
}

type CollectionConfig struct {
    SourceConfig
    ExcludeFromDefault bool `json:"excludeFromDefault"`
}

type WikiConfig struct {
    SourceConfig
    WikiDir            string             `json:"wikiDir"`
    RawDir             string             `json:"rawDir"`
    IndexFile          string             `json:"indexFile"`
    LogFile            string             `json:"logFile"`
    GraphLinks         bool               `json:"graphLinks"`
    ExcludeFromDefault bool               `json:"excludeFromDefault"`
    SourceRefs         []string           `json:"sourceRefs,omitempty"`
    Frontmatter        *FrontmatterConfig `json:"frontmatter,omitempty"` // wiki-specific, nested
}
```

### `FrontmatterConfig` — kept as wiki-specific nested field

The old `WikiConfig` struct (with `Enabled`, `IndexFile`, `LogFile`, `GraphLinks`,
`Frontmatter`) is replaced by the new `WikiConfig` above. `FrontmatterConfig` is kept
as a **nested** field on `WikiConfig` only — not on `CollectionConfig`. This avoids
name collisions between wiki page frontmatter keys and gmd's own indexing field names.
Wiki frontmatter fields under `frontmatter.fields` are separate from `SourceConfig.fields`,
which controls Typesense-level field indexing for both collections and wikis.

**Implementation note:** `wikiDir` and `rawDir` default to `"wiki"` and `"raw"` in the
CUE schema (`*"wiki"`, `*"raw"`). Verify during implementation that CUE defaults survive
JSON decode into Go — if CUE omits fields with default values during export, the Go
struct must handle zero values by applying defaults at decode time.

### `CollectionKey` stays — works for both collections and wikis

```go
func (c *Config) CollectionKey(name string) string {
    if c.Project == "" {
        return name
    }
    return c.Project + "-" + name
}
```

Both collections and wikis share the same Typesense `chunks` collection. The `collection`
facet field distinguishes them. `CollectionKey("myresearch")` works identically whether
`myresearch` is a collection or a wiki.

### Helper functions (signatures to be refined at implementation)

```go
// SourceKey returns a prefixed key for any named source.
func (c *Config) SourceKey(name string) string {
    return c.CollectionKey(name)
}

// IsWiki reports whether name is a wiki (not a collection).
func (c *Config) IsWiki(name string) bool {
    _, ok := c.Wikis[name]
    return ok
}

// IsCollection reports whether name is a collection (not a wiki).
func (c *Config) IsCollection(name string) bool {
    _, ok := c.Collections[name]
    return ok
}

// SourceKeysForSearch returns the set of Typesense collection keys to query
// when searching a named source. If the source is a wiki with sourceRefs,
// the result includes the wiki's own key plus keys for all referenced sources.
// Returns an error if any referenced source does not exist in collections or wikis.
func (c *Config) SourceKeysForSearch(name string) ([]string, error) {
    keys := []string{c.CollectionKey(name)}
    wc, ok := c.Wikis[name]
    if !ok {
        return keys, nil // collection — single key
    }
    for _, ref := range wc.SourceRefs {
        if _, ok := c.Collections[ref]; ok {
            keys = append(keys, c.CollectionKey(ref))
        } else if _, ok := c.Wikis[ref]; ok {
            keys = append(keys, c.CollectionKey(ref))
        } else {
            return nil, fmt.Errorf("wiki %q references source %q which does not exist in collections or wikis", name, ref)
        }
    }
    return keys, nil
}
```

---

## 3. Indexing — How Wikis Get Into Typesense

### Wiki files are indexed like collections

Both `CollectionConfig` and `WikiConfig` embed `SourceConfig`, so the indexer can treat
them uniformly. Wiki pages are scanned, chunked, embedded, and upserted into Typesense with
`collection: <wiki_name>` — the same mechanism as collections.

**Frontmatter indexing for wiki pages.** The indexer reads `SourceConfig.fields` to determine
which YAML frontmatter keys to extract and send to Typesense as typed fields. Wiki pages
may also have wiki-specific frontmatter defined under `WikiConfig.frontmatter.fields` —
these are consumed by the wiki agent (ingest, lint, query) but are **not** sent to Typesense
as indexed fields. The two field domains serve different purposes and are never merged.

**rawDir is indexed by default.** Most raw source material is markdown and should be
searchable. Users who want to exclude raw content can opt out via `ignore: ["raw/**"]`.

```go
// In indexer or update command — unified loop over all sources:
func indexSources(ctx context.Context, cfg *Config) {
    for name := range cfg.Collections {
        indexSource(ctx, name, &cfg.Collections[name].SourceConfig)
    }
    for name, wc := range cfg.Wikis {
        // Build an ignore list that excludes the meta files but preserves
        // the original wc.Ignore slice (no mutation). rawDir content is
        // indexed by default; users opt out via ignore patterns if desired.
        scannerIgnore := make([]string, 0, len(wc.Ignore)+2)
        scannerIgnore = append(scannerIgnore, wc.Ignore...)
        scannerIgnore = append(scannerIgnore,
            filepath.Join(wc.WikiDir, wc.IndexFile),
            filepath.Join(wc.WikiDir, wc.LogFile),
        )
        // Pass scannerIgnore locally — do NOT mutate wc.Ignore
        indexSourceWithIgnore(ctx, name, &wc.SourceConfig, scannerIgnore)
    }
}
```

Wikis without explicit patterns use default patterns (`<wikiDir>/**/*.md`,
`<rawDir>/**/*.md`), so agent-created pages in `wikiDir` are always indexed.
SourceRef'd collections provide additional searchable content.

### `gmd embed` also works on wikis

Same loop — wiki files are re-chunked and re-embedded alongside collections.

---

## 4. Search — How Wikis Appear in Queries

### Unified Typesense `chunks` collection

All content — collections and wikis — lives in the single Typesense `chunks` collection.
The `collection` facet field identifies the source:

```
{ collection: "myproject-docs",       path: "README.md", ... }
{ collection: "myproject-myresearch", path: "wiki/concepts/transformers.md", ... }
```

### `gmd query <text>` — searches ALL sources by default

Current behavior: searches all collections. New behavior: searches all collections + wikis
where `excludeFromDefault` is `false` (the default). **Note:** `excludeFromDefault` filtering
is new runtime behavior that must be implemented from scratch — the current `includeByDefault`
field exists in config but is never read by the indexer or search pipeline. Default search
filtering (step 10) must build the source list by iterating `Collections` and `Wikis` maps,
omitting any source with `excludeFromDefault: true`. When `searchDefaults` presets are
invoked via `--search`, the preset's source list is used directly in place of default
resolution.

### `gmd query <text> --collection <name>` — scoped search with auto-expansion

If `<name>` resolves to a collection, the filter is `collection == CollectionKey(name)`.

If `<name>` resolves to a wiki with `sourceRefs`, the filter expands to:

```
collection IN [CollectionKey(name), CollectionKey(ref1), CollectionKey(ref2), ...]
```

If any `sourceRef` does not exist in `collections` or `wikis`, `SourceKeysForSearch`
returns an error explaining which reference is unknown.

This is transparent to the user — no separate `--wiki` flag needed. The pipeline
determines expansion by looking up the name in `cfg.Wikis`. Pass `--no-expansion`
to search only the wiki's own content without expanding `sourceRefs`.

### No `--wiki` flag

Collections and wikis are interchangeable as search targets. The `--collection` flag
handles both. The pipeline internally calls `SourceKeysForSearch(name)` to determine
the set of Typesense collection keys.

### `gmd search "/ text" --collection <name>` — text-only with auto-expansion

Same as `query` but limited to keyword search.

### `gmd get <path>` — resolves relative to the source's path

If `path` is a wiki page, `gmd get wiki/concepts/foo.md` returns its content.

**Path disambiguation is new logic.** Currently `gmd get` resolves via Typesense lookup
using a single filter value. With wikis and collections sharing the same Typesense
collection, the same relative path may exist in multiple sources (producing multiple
Typesense documents with different `collection` values). When this occurs, `gmd get`
errors with a disambiguation message:

```
path "wiki/concepts/foo.md" exists in multiple sources:
  - docs (collection)
  - myresearch (wiki)
Use --collection to disambiguate: gmd get --collection docs wiki/concepts/foo.md
```

---

## 5. CLI Design

### Wiki lifecycle commands (mirroring collections)

All use **positional args** — no `--name` flags.

| Command | Effect | Collections Equivalent |
|---|---|---|
| `gmd wiki create <name> --path <path> [--skills] [--wiki-dir <dir>] [--raw-dir <dir>]` | Scaffold dirs + add wiki config + validate (name uniqueness, cycle-free sourceRefs). `--skills` writes agent skill templates for discovery. | `gmd collection create <name>` |
| `gmd wiki list` | List wikis | `gmd collection list` |
| `gmd wiki show <name>` | Show wiki details + source refs | `gmd collection show <name>` |
| `gmd wiki remove <name>` | Remove wiki + delete Typesense chunks | `gmd collection remove <name>` |
| `gmd wiki rename <old> <new>` | Rename wiki in config | `gmd collection rename <old> <new>` |
| `gmd wiki include <name> <patterns...>` | Add file patterns (proxy) | `gmd collection include` |
| `gmd wiki exclude <name> <patterns...>` | Add ignore patterns (proxy) | `gmd collection exclude` |
| `gmd wiki context add <name> "text"` | Set wiki context (proxy) | `gmd context add` |
| `gmd wiki context rm <name>` | Remove wiki context (proxy) | `gmd context rm` |
| `gmd wiki context list` | List wiki contexts (proxy) | `gmd context list` |
| `gmd wiki ref add <name> <source>` | Add source reference (validates: must exist in collections or wikis, no cycles) | — |
| `gmd wiki ref rm <name> <source>` | Remove source reference | — |
| `gmd wiki ref list <name>` | List source references | — |

### Wiki create — configurable directory names

```sh
gmd wiki create mywiki                                    # path defaults to project root
gmd wiki create mywiki --path ./docs                      # explicit path
gmd wiki create mywiki --wiki-dir pages --raw-dir inputs  # custom subdir names
gmd wiki create mywiki --skills                           # also write agent skill templates
```

Scaffolds `<path>/<wikiDir>/` (subdirs: entities, concepts, comparisons, synthesis, sources)
and `<path>/<rawDir>/`. Creates default `_index.md` and `_log.md` in `<wikiDir>/`.
Does **not** create `WIKI_SCHEMA.md` — this is deferred for a future generalized
`AGENTS.md`-style capability.

**Validation performed at create time:**
- Name must not already exist in `collections` or `wikis` maps
- `wikiDir` and `rawDir` must be distinct (`wikiDir != rawDir`)
- `wikiDir` and `rawDir` must not overlap with resolved paths of existing collections or wikis
- Each `sourceRefs` entry must exist in the `collections` or `wikis` map
- `sourceRefs` must contain no cycles (DFS traversal of the reference graph)

**Conflict checks:** before scaffolding, verify `<wikiDir>` and `<rawDir>` don't overlap
with resolved paths of any existing collection or other wiki. Error if they do —
the user must rename or choose different paths.

### Wiki agent operations (unchanged semantics, positional wiki name)

```sh
gmd wiki ingest mywiki paper.md                 # positional wiki name
gmd wiki ingest mywiki paper.md --batch         # batch mode
gmd wiki query mywiki "key findings"            # positional wiki name
gmd wiki query mywiki "key findings" --save     # save as synthesis page
gmd wiki graph mywiki [--format dot|mermaid|json]
gmd wiki lint mywiki
gmd wiki doctor mywiki [--fix]
gmd wiki skills list
gmd wiki skills show <name>
gmd wiki skills write [--target <agent>]
```

### Search commands — no `--wiki` flag

```sh
gmd query "some text" --collection mywiki        # auto-expands sourceRefs if mywiki is a wiki
gmd query "some text" --collection mywiki --no-expansion  # wiki's own content only
gmd search "some text" --collection mywiki       # text-only
gmd vsearch "some text" --collection mywiki      # vector-only
gmd get wiki/concepts/foo.md                     # resolves via Typesense collection field
gmd ls mywiki                                    # list files indexed in wiki
```

### Collection commands accept wiki names

Wikis are also collections. The same CLI verbs work on both:

```sh
gmd collection show mywiki                       # works: shows wiki as a source
gmd collection include mywiki "docs/**/*.md"     # works: adds patterns to wiki
gmd collection exclude mywiki "**/_*.md"         # works: adds ignore patterns to wiki
gmd context add mywiki "AI context"              # works: sets wiki context
```

The `gmd wiki *` proxy commands call the same underlying implementation as `gmd collection *`.
Users can use whichever path reads more naturally for the task at hand.

### `collection list` — show references

```sh
gmd collection list
  docs
    path:    ./docs
    patterns: ["**/*.md"]
    referenced by: mywiki    # ← NEW: shows which wikis reference this collection
```

The "referenced by" field is computed by scanning all wikis' `sourceRefs` for the
collection name. This is an O(n*m) scan on each `collection list` call, which is
acceptable for typical config sizes (tens of sources; no persisted index).

### `gmd cleanup` with sourceRefs

`gmd cleanup` removes stale chunks from deleted files in individual collections.
When a wiki references a collection via `sourceRefs`, cleanup of the referenced
collection naturally reduces the content visible through the wiki's search.
No special handling is needed — the wiki always searches whatever is currently
indexed in Typesense.

---

## 6. Config Editing (`pkg/config/edit.go`)

The existing collection CRUD functions operate on the `collections:` CUE AST block.
New wiki functions operate on the `wikis:` block. The `include`/`exclude`/`context`
functions are updated to find the named entity in either `collections` or `wikis`.

| Function | Purpose |
|---|---|---|
| `CreateWiki(cfg, name, path, patterns)` | Scaffold dirs + add wiki to CUE + in-memory config |
| `RemoveWiki(cfg, name)` | Remove wiki from CUE + in-memory |
| `RenameWiki(cfg, old, new)` | Rename wiki in CUE + in-memory |
| `AddSourceRef(cfg, wikiName, srcName)` | Add source reference (validated: must exist, no cycles) |
| `RemoveSourceRef(cfg, wikiName, srcName)` | Remove source reference |
| `ListSourceRefs(cfg, wikiName)` | List source references |

Existing functions updated to search both maps:

| Function | Change |
|---|---|
| `AddCollectionPatterns(cfg, name, ...)` | Look up name in `Collections` first, then `Wikis` |
| `AddIgnorePatterns(cfg, name, ...)` | Same |
| `RemoveIgnorePattern(cfg, name, ...)` | Same |
| `AddContextDoc(cfg, name, ...)` | Same |
| `RemoveContextDoc(cfg, name)` | Same |
| `ListContextDocs(cfg)` | Return both collection and wiki contexts |

Sample CUE output for `gmd wiki create mywiki`:

```cue
Config: {
    project: "myproject"
    // ... llm, typesense, pipeline ...
    collections: {
        docs: { path: ".", patterns: ["**/*.md"] }
    }
    wikis: {
        mywiki: {
            path:       "."
            wikiDir:    "wiki"
            rawDir:     "raw"
            patterns:   ["wiki/**/*.md"]
            ignore:     ["wiki/_index.md", "wiki/_log.md"]
            sourceRefs: ["docs"]
        }
    }
}
```

---

## 7. Breaking Change (No Migration)

This is a breaking schema change: `wiki` is removed from `CollectionConfig`, and the
new `wikis` top-level map is added. Existing config files will fail CUE validation.

**No migration command is provided.** The tool is still alpha; users should manually
update their `.gmd/config.cue` files.

---

## 8. Implementation Order

1. **CUE schema** — add `#Source` constraint, `WikiConfig` top-level (with nested
   `frontmatter`, `excludeFromDefault`, `wikiDir`, `rawDir`, `sourceRefs`), `wikis: [string]:
   WikiConfig` in `ProjectConfig`, `searchDefaults`. Remove `wiki` from `CollectionConfig`.
   Change `CollectionConfig.includeByDefault` to `excludeFromDefault`.
2. **Go types — Config layer** — update `Config` (add `Wikis`, `SearchDefaults`),
   `SourceConfig`, `CollectionConfig` (`ExcludeFromDefault`), `WikiConfig` (new fields,
   nested `Frontmatter`). Keep existing collection-only code working through this step.
3. **Name uniqueness** — validate at `collection create` and `wiki create` that the name
   doesn't exist in either map.
4. **Path collision validation** — cross-wiki collisions, wikiDir==rawDir check,
   wiki-vs-collection overlap detection. Wired into `wiki create` and include operations.
5. **Config editing — wiki CRUD** — `CreateWiki`, `RemoveWiki`, `RenameWiki`,
   `AddSourceRef` (validates target exists, rejects cycles), `RemoveSourceRef`. Update
   existing include/exclude/context functions to search both `Collections` and `Wikis` maps.
6. **Wiki CLI** — refactor to positional args, replace `wiki init` with `wiki create`,
    add lifecycle commands (`create`, `list`, `show`, `remove`, `rename`, `include`,
    `exclude`, `context`, `ref`). `wiki create` supports `--skills` for convenience
    (writes agent skill templates; also available via `wiki skills write`). Wiki
    commands are proxies delegating to the same collection CRUD internals.
7. **Wiki struct + agent** — update `Wiki` struct to use `*config.WikiConfig` directly
   (no `CollectionConfig` wrapper). Update `agent.go` to reference `WikiConfig` fields
   directly. Ensure `frontmatter.fields` (nested) and `SourceConfig.fields` (Typesense
   indexing) are accessed from the correct location.
8. **Validation — cycle detection + ref existence** — cycle detection for sourceRefs
   graph. Reject unknown sourceRef names at `wiki ref add` time with a clear error.
9. **Indexer** — update `gmd update` and `gmd embed` to iterate wikis via shared
    `SourceConfig`. Wikis without explicit patterns use defaults (`<wikiDir>/**/*.md`,
    `<rawDir>/**/*.md`) and are never skipped. `--collection` flag scopes update/embed
    to a single source. rawDir is indexed by default; meta files (`_index.md`, `_log.md`)
    are excluded from scanning. User opt-out via `ignore: ["raw/**"]`.
10. **Search** — implement `SourceKeysForSearch` (errors on unknown refs), auto-expand
     sourceRefs in pipeline when `--collection` resolves to a wiki, add `--no-expansion`
     flag to opt out of sourceRefs expansion and search only the wiki's own content,
     implement `excludeFromDefault` filtering for unscoped searches (new runtime
     behavior — the current `includeByDefault` field is config-only with no runtime
     effect), implement `searchDefaults` preset resolution via `--search` flag (preset's
     source list replaces default resolution), path collision detection and
     disambiguation in `gmd get` (new logic: multiple Typesense documents with different
     `collection` values for the same relative path require `--collection` to
     disambiguate).
11. **Cleanup** — remove dead code (old `WikiConfig.Enabled`, old `Wiki`
    struct shape, `CollectionConfig.Wiki`, `IncludeByDefault`), update docs/AGENTS.md.


