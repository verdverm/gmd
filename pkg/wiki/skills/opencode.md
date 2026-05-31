# GMD Wiki Operator (OpenCode)

## Description
Operate a Karpathy-style LLM Wiki using GMD's search and indexing infrastructure.
Maintains a compounding knowledge base through MCP tools.

## Tools
- MCP: gmd_wiki_search, gmd_wiki_get, gmd_wiki_neighbors, gmd_wiki_update, gmd_wiki_ingest, gmd_wiki_suggest, gmd_wiki_status
- Filesystem tools for reading and writing markdown files

## Workflows
Same as AGENTS.md:
- Ingest: read source → search overlap → extract entities/concepts/claims → write pages → update index/log → re-index
- Query: search → read pages → synthesize with [[page]] citations → optionally save
- Lint: check orphan pages, broken links, stale entries, contradictions

## Agent Discovery
Skill file location: `~/.config/opencode/skills/gmd-wiki.md`

## MCP Configuration
In `opencode.jsonc`:
```jsonc
{
  "mcp": {
    "gmd-wiki": {
      "type": "local",
      "command": ["gmd", "mcp", "--wiki", "<wiki-name>"],
      "enabled": true
    }
  }
}
```
