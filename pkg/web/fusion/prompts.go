package fusion

import "embed"

//go:embed embeds
var fusionEmbedsFS embed.FS

func searchSynthesisPrompt() string {
	data, _ := fusionEmbedsFS.ReadFile("embeds/search_synthesis.md")
	return string(data)
}
