package web

import "embed"

//go:embed embeds
var webEmbedsFS embed.FS

func agentSystemPrompt() string {
	data, _ := webEmbedsFS.ReadFile("embeds/agent_system.md")
	return string(data)
}

func agentSynthesizePrompt() string {
	data, _ := webEmbedsFS.ReadFile("embeds/agent_synthesize.md")
	return string(data)
}
