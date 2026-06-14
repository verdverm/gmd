# GMD Wiki Operator

## Description
Operate a Karpathy-style LLM Wiki using GMD's search and indexing infrastructure.
Maintains a compounding knowledge base: ingest sources, query the wiki,
lint for health, and export results.

## Required Tools
- MCP: gmd_wiki_search, gmd_wiki_get, gmd_wiki_neighbors, gmd_wiki_update,
       gmd_wiki_ingest, gmd_wiki_suggest, gmd_wiki_status
- Filesystem: read_file, write_file (for wiki page authoring)

## Ingest Workflow
When user provides a source to ingest:
1. Read the source file (or fetch URL, save to raw/)
2. Call gmd_wiki_search to find existing pages that overlap
3. Read related wiki pages for context
4. Extract entities, concepts, claims, contradictions
5. Write/update wiki pages in the appropriate directories:
   - entities/  # people, orgs, products, technologies
   - concepts/  # methodologies, architectures, theories
   - comparisons/  # X vs Y analyses
   - sources/  # summaries of ingested content
6. Update _index.md with new/updated page entries
7. Append entry to _log.md
8. Call gmd_wiki_update to re-index
9. Report summary to user

## Query Workflow
When user asks a question:
1. Call gmd_wiki_search with the question
2. Read top matching wiki pages with gmd_wiki_get
3. Synthesize answer with citations using [[page]] links
4. Offer to save answer to wiki/synthesis/

## Page Templates

### Entity Page (entities/name.md)
```markdown
---
type: entity
tags: [tag1, tag2]
status: draft
sources: [source-page.md]
---
# Entity Name

## Overview
Brief description of the entity.

## Properties
- Property 1
- Property 2

## Relationships
- Related to [[other-entity]]
- Part of [[broader-concept]]

## Sources
- [[source-page]] -- key claim or quote
```

### Concept Page (concepts/name.md)
```markdown
---
type: concept
tags: [tag1, tag2]
status: draft
sources: [source-page.md]
---
# Concept Name

## Definition
Clear, concise definition.

## Key Principles
1. Principle one
2. Principle two

## Examples
- Example with [[related-entity]]

## See Also
- [[related-concept]]
- [[comparison-page]]
```

### Comparison Page (comparisons/a-vs-b.md)
```markdown
---
type: comparison
tags: [tag1, tag2]
status: draft
---
# A vs B

| Dimension | A | B |
|---|---|---|
| ... | ... | ... |

## Analysis
Narrative comparison.

## When to Use Which
Guidance for choosing between them.
```

### Source Summary (sources/YYYY-MM-DD-title-slug.md)
```markdown
---
type: source
tags: [tag1, tag2]
source_url: https://...
status: draft
---
# Source Title

## Summary
One-paragraph summary.

## Key Takeaways
- Takeaway 1
- Takeaway 2

## Entities Referenced
- [[entity-1]]
- [[entity-2]]

## Concepts Introduced
- [[concept-1]]

## Claims
> Direct quote or paraphrased claim.
```

## Frontmatter Conventions

| Field | Description | Example |
|---|---|---|
| type | Page category | entity, concept, comparison, source, synthesis |
| tags | Searchable labels | [kubernetes, deployment] |
| status | Review state | draft, reviewed, needs-update |
| sources | Pages this page derives from | [source-page.md] |
| difficulty | Complexity rating (1-5) | 3 |
| source_url | URL of original source | https://... |

## Lint & Maintenance
Periodically run gmd_wiki_lint to:
- Find orphan pages (zero inbound [[wikilinks]])
- Detect broken wikilinks (targets with no matching page)
- Check _index.md for stale entries
- Flag potential contradictions between pages
- Identify knowledge gaps

When fixing issues:
- Orphan pages: add links from related pages, or merge into parent concept
- Broken links: create missing page, or remove the link
- Stale entries: update or remove from _index.md
- Contradictions: add a note in both pages, create a comparison if warranted
