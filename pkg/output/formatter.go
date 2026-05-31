package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/verdverm/gmd/pkg/search"
)

func FormatResults(results []search.Result, format string) (string, error) {
	switch format {
	case "json":
		return formatJSON(results)
	case "cli", "text":
		return formatCLI(results), nil
	default:
		return formatCLI(results), nil
	}
}

func formatJSON(results []search.Result) (string, error) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func formatCLI(results []search.Result) string {
	if len(results) == 0 {
		return "No results found."
	}

	var b strings.Builder
	for i, r := range results {
		fmt.Fprintf(&b, "%d. %s\n", i+1, r.Path)
		if r.Title != "" {
			fmt.Fprintf(&b, "   Title: %s\n", r.Title)
		}
		if r.Collection != "" {
			fmt.Fprintf(&b, "   Collection: %s\n", r.Collection)
		}
		fmt.Fprintf(&b, "   Score: %.4f\n", r.Score)

		snippet := makeSnippet(r.Content, 200)
		if snippet != "" {
			fmt.Fprintf(&b, "   ---\n")
			fmt.Fprintf(&b, "   %s\n", snippet)
			fmt.Fprintf(&b, "   ---\n")
		}
		fmt.Fprintln(&b)
	}
	return strings.TrimRight(b.String(), "\n")
}

func makeSnippet(content string, maxLen int) string {
	cleaned := strings.TrimSpace(content)
	if cleaned == "" {
		return ""
	}
	lines := strings.Split(cleaned, "\n")
	var nonEmpty []string
	for _, line := range lines {
		tr := strings.TrimSpace(line)
		if tr != "" {
			nonEmpty = append(nonEmpty, tr)
		}
	}
	if len(nonEmpty) == 0 {
		return ""
	}
	text := strings.Join(nonEmpty, " ")
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
