# Wiki Schema & Conventions

## Directory Structure

```
raw/                  Immutable source files (not indexed)
wiki/
  entities/           People, orgs, products, technologies
  concepts/           Methodologies, architectures, theories
  comparisons/        X vs Y analyses
  synthesis/          Cross-source analysis, saved answers
  sources/            Summaries of ingested content
  _index.md           Content catalog (LLM-maintained)
  _log.md             Chronological record (LLM-maintained)
WIKI_SCHEMA.md        This file
.gmd/config.cue       GMD configuration
```

## Naming Conventions

- Use kebab-case for page filenames: `transformer-architecture.md`
- Entity pages: singular noun (`kubernetes.md`, `attention-mechanism.md`)
- Concept pages: descriptive phrase (`scaled-dot-product-attention.md`)
- Comparison pages: `a-vs-b.md` format
- Source pages: `YYYY-MM-DD-short-title.md` format
- Synthesis pages: `YYYY-MM-DD-question-slug.md` format

## Page Format

Every wiki page must:
1. Start with YAML frontmatter (between `---` delimiters)
2. Have a single H1 heading matching the page title
3. Use [[wikilinks]] for cross-references (no relative paths)
4. End with a "See Also" section where applicable

## Frontmatter Schema

```yaml
---
type: concept          # Required: entity | concept | comparison | source | synthesis
tags: [tag1, tag2]     # Required: lowercase, kebab-case
status: draft          # Optional: draft | reviewed | needs-update
sources: [page.md]     # Optional: pages this content derives from
difficulty: 3          # Optional: 1 (intro) to 5 (expert)
source_url: https://   # Optional: for source pages only
---
```

## The Index File (_index.md)

The index is the wiki's table of contents. It is maintained by the ingest agent
and should not be manually edited. Format:

```markdown
# Wiki Index

## Entities
- [[entity-name]] — One-line description
- ...

## Concepts
- [[concept-name]] — One-line description
- ...

## Comparisons
- [[a-vs-b]] — One-line description
- ...

## Sources
- [[source-page]] — Summary and date ingested
- ...

## Last Updated
YYYY-MM-DD HH:MM
```

## The Log File (_log.md)

The log records every ingest operation chronologically. It is maintained by the
ingest agent and should not be manually edited. Format:

```markdown
# Wiki Log

## [2026-05-31 14:30] ingest | Source Title
- Created: entities/foo.md, concepts/bar.md
- Updated: entities/baz.md
- Flagged contradiction: claim X vs existing page Y

## [2026-05-30 10:15] query-save | "question text"
- Saved: wiki/synthesis/2026-05-30-question-slug.md
```

## Status Lifecycle

```
draft → reviewed → needs-update → reviewed → ...
```

- draft: Initial creation, needs review
- reviewed: Verified for accuracy
- needs-update: New information available, needs revision
