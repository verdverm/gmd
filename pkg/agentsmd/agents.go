package agentsmd

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed content/*.md
var contentFS embed.FS

func GetContent(name string) (string, error) {
	data, err := contentFS.ReadFile("content/" + name + ".md")
	if err != nil {
		return "", fmt.Errorf("reading agents content for %q: %w", name, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func ValidNames() ([]string, error) {
	entries, err := contentFS.ReadDir("content")
	if err != nil {
		return nil, fmt.Errorf("reading content directory: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".md")
		names = append(names, name)
	}
	return names, nil
}
