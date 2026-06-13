package agentsmd

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed embeds
var agentsEmbedsFS embed.FS

func GetContent(name string) (string, error) {
	data, err := agentsEmbedsFS.ReadFile("embeds/" + name + ".md")
	if err != nil {
		return "", fmt.Errorf("reading agents content for %q: %w", name, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func ValidNames() ([]string, error) {
	entries, err := agentsEmbedsFS.ReadDir("embeds")
	if err != nil {
		return nil, fmt.Errorf("reading embeds directory: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		names = append(names, name)
	}
	return names, nil
}
