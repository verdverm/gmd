# GMD Wiki Operator

Operate a Karpathy-style LLM Wiki using GMD's search and indexing infrastructure.

## Ingest
1. Read source file
2. Search wiki for existing overlap (gmd_wiki_search)
3. Extract entities, concepts, claims, contradictions
4. Write/update wiki pages with [[wikilinks]] and YAML frontmatter
5. Update _index.md and _log.md
6. Run gmd_wiki_update to re-index

## Query
1. Search wiki (gmd_wiki_search)
2. Read pages (gmd_wiki_get)
3. Synthesize answer with [[page]] citations
4. Optionally save to wiki/synthesis/

## See WIKI_SCHEMA.md for page templates and conventions.
