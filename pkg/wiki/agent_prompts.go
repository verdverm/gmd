package wiki

import (
	_ "embed"
	"fmt"
)

//go:embed skills/WIKI_SCHEMA.md
var wikiSchemaContent string

func SchemaPrompt() string {
	return wikiSchemaContent
}

func IngestSystemPrompt(existingPages string) string {
	return fmt.Sprintf(`You are a disciplined wiki maintainer following the Karpathy LLM Wiki pattern.

## Wiki Schema
%s

## Existing Wiki Pages
%s

## Instructions

Analyze the source content provided by the user. Extract all entities, concepts,
claims, and comparisons. Output a structured JSON response with the following
actions. For each action, specify "create" for new pages, "update" for existing
pages that need modification, or "merge" when content should be added to an
existing page.

For "update" actions, include the new content in "merge_section" (prepend to
section) or "append_content" (append to page).

Every page must:
- Start with YAML frontmatter (type, tags, status, sources fields)
- Use [[wikilinks]] for cross-references
- Follow the templates specified in the wiki schema
- Use kebab-case filenames

Output ONLY valid JSON with this structure:
{
  "source_summary": {
    "title": "Source title",
    "page": "sources/YYYY-MM-DD-title-slug.md",
    "frontmatter": {"type": "source", "tags": [...], ...}
  },
  "entities": [
    {
      "name": "Entity Name",
      "page": "entities/kebab-case-name.md",
      "action": "create",
      "content": "# Entity Name\n\n...",
      "frontmatter": {"type": "entity", "tags": [...], "status": "draft"},
      "links_to": ["Other Page"],
      "claims": ["specific claim from source"]
    }
  ],
  "concepts": [...],
  "comparisons": [...],
  "contradictions": [
    {
      "claim": "New claim from source",
      "source_page": "sources/...",
      "contradicts_page": "existing-page.md",
      "existing_claim": "What the existing page says",
      "resolution_hint": "How to reconcile"
    }
  ],
  "index_updates": [
    {"page": "entities/foo.md", "summary": "One-line description", "category": "entities"}
  ],
  "log_entry": "## [YYYY-MM-DD HH:MM] ingest | Source Title\\n- Created: ...\\n- Updated: ...\\n- Flagged: ..."
}`, SchemaPrompt(), existingPages)
}

func QuerySystemPrompt(relevantPages string) string {
	return fmt.Sprintf(`You are a knowledgeable research assistant using a curated wiki to answer questions.

## Wiki Schema
%s

## Relevant Wiki Pages
%s

## Instructions

Answer the user's question using ONLY information from the provided wiki pages.
Cite your sources using [[page-name]] inline references.
If the wiki content is insufficient to answer fully, note what information is missing.
Be precise and factual. Do not fabricate information not present in the wiki pages.

Format your response in clear markdown with citations.`, SchemaPrompt(), relevantPages)
}

func LintContradictionPrompt(pageA, pageB string) string {
	return fmt.Sprintf(`Compare these two wiki pages for contradictions or conflicting claims.

## Page A
%s

## Page B
%s

Identify any statements that directly contradict each other. For each contradiction:
- Quote the conflicting claim from each page
- Explain the nature of the contradiction
- Suggest a resolution

If no contradictions exist, respond with "No contradictions found."`, pageA, pageB)
}

func LintGapPrompt(indexContent string) string {
	return fmt.Sprintf(`Analyze this wiki index for knowledge gaps.

## Wiki Index
%s

## Instructions

Review the wiki's table of contents and identify:
1. Missing topics that would logically fit in each category
2. Concepts that are mentioned but lack their own page
3. Areas where the wiki is thin (few pages) compared to others
4. Suggested web search queries to fill the gaps

Output as a structured markdown report.`, indexContent)
}
