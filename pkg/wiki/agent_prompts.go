package wiki

import (
	"embed"
	"fmt"
)

//go:embed embeds
var wikiEmbedsFS embed.FS

func SchemaPrompt() string {
	data, err := wikiEmbedsFS.ReadFile("embeds/wiki_schema.md")
	if err != nil {
		return ""
	}
	return string(data)
}

func IngestSystemPrompt(existingPages string) string {
	tmpl, _ := wikiEmbedsFS.ReadFile("embeds/ingest_system.md")
	return fmt.Sprintf(string(tmpl), SchemaPrompt(), existingPages)
}

func QuerySystemPrompt(relevantPages string) string {
	tmpl, _ := wikiEmbedsFS.ReadFile("embeds/query_system.md")
	return fmt.Sprintf(string(tmpl), SchemaPrompt(), relevantPages)
}

func LintContradictionPrompt(pageA, pageB string) string {
	tmpl, _ := wikiEmbedsFS.ReadFile("embeds/lint_contradiction.md")
	return fmt.Sprintf(string(tmpl), pageA, pageB)
}

func LintGapPrompt(indexContent string) string {
	tmpl, _ := wikiEmbedsFS.ReadFile("embeds/lint_gap.md")
	return fmt.Sprintf(string(tmpl), indexContent)
}
