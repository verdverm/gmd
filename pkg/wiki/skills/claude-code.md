# GMD Wiki Operator (Claude Code)

## Description
Operate a Karpathy-style LLM Wiki using GMD's search and indexing infrastructure
via MCP tools. Maintains a compounding knowledge base.

## Tools Required
- MCP gmd-wiki: gmd_wiki_search, gmd_wiki_get, gmd_wiki_neighbors, gmd_wiki_update, gmd_wiki_ingest, gmd_wiki_suggest, gmd_wiki_status
- Filesystem: Read, Write (for wiki page authoring)

## Ingest Workflow
When the user provides a source to ingest:

1. **Read the source.** Fetch it (URL → raw/ or read local file).
2. **Search for overlap.** Use `gmd_wiki_search` with key terms from the source title/abstract to find existing pages that may need updating.
3. **Read existing pages.** Use `gmd_wiki_get` to read overlapping pages for context.
4. **Analyze and extract.** Based on the source content and existing wiki pages, identify:
   - Entities (people, orgs, technologies) → wiki/entities/
   - Concepts (methodologies, architectures) → wiki/concepts/
   - Comparisons (framed as tradeoffs) → wiki/comparisons/
   - Claims that may contradict existing wiki content
5. **Write wiki pages.** Create new pages or update existing ones:
   - Every page starts with YAML frontmatter (type, tags, status, sources)
   - Use [[wikilinks]] for all cross-references
   - Follow templates from WIKI_SCHEMA.md
6. **Update meta-files.** Append to _log.md, update _index.md with new/changed pages.
7. **Re-index.** Call `gmd_wiki_update` to refresh the Typesense index.
8. **Report.** Summarize what was created, updated, and flagged.

## Query Workflow
When the user asks a question about the wiki:

1. **Search.** Call `gmd_wiki_search` with the user's question.
2. **Filter by type** if appropriate: `filter: "type:=concept"` or `"type:=entity"`.
3. **Read top pages.** Use `gmd_wiki_get` for the most relevant results.
4. **Check neighbors.** Use `gmd_wiki_neighbors` to find related pages.
5. **Synthesize answer.** Write a clear answer citing [[page]] references.
6. **Offer to save.** If the answer is substantial, offer to write it to wiki/synthesis/.

## Lint Workflow
Run periodically or on request:

1. **Structure checks** (run these directly):
   - Find all .md files in wiki/, check all [[wikilinks]] point to existing pages
   - Check _index.md entries reference existing pages
2. **Content review** (use LLM judgment):
   - Read pairs of related pages, check for contradictory claims
   - Check for pages not updated since their sources were updated
3. **Fix issues** found during lint.

## Page Templates
See WIKI_SCHEMA.md for full templates. Key conventions:
- Frontmatter: `type`, `tags`, `status`, `sources` required
- [[wikilinks]] for all cross-references
- kebab-case filenames
- YYYY-MM-DD-title format for source and synthesis pages

## MCP Configuration
The MCP server should be configured in `.claude/settings.json`:
```json
{
  "mcpServers": {
    "gmd-wiki": {
      "type": "local",
      "command": ["gmd", "mcp", "--wiki", "<wiki-name>"],
      "enabled": true
    }
  }
}
```
