package output

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/verdverm/gmd/pkg/search"
	"github.com/verdverm/gmd/pkg/ts"
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

// FormatLS formats a list of documents grouped by collection with sorted paths.
func FormatLS(results []ts.HybridSearchResult) string {
	if len(results) == 0 {
		return "No indexed documents."
	}

	type colGroup struct{ paths []string }
	byCol := make(map[string]*colGroup)
	colOrder := make([]string, 0)
	for _, res := range results {
		g, ok := byCol[res.Collection]
		if !ok {
			g = &colGroup{}
			byCol[res.Collection] = g
			colOrder = append(colOrder, res.Collection)
		}
		g.paths = append(g.paths, res.Path)
	}
	sort.Strings(colOrder)

	var b strings.Builder
	for _, col := range colOrder {
		g := byCol[col]
		if g == nil {
			continue
		}
		sort.Strings(g.paths)
		b.WriteString(col + ":\n")
		for _, p := range g.paths {
			b.WriteString("  " + p + "\n")
		}
	}
	b.WriteString(fmt.Sprintf("\n%d document(s) indexed\n", len(results)))
	return b.String()
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
