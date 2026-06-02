package wiki

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/verdverm/gmd/pkg/chunking"
	"github.com/verdverm/gmd/pkg/ts"
)

type GraphEdge struct {
	From string
	To   string
}

type WikiGraph struct {
	Nodes    []string
	Edges    []GraphEdge
	AdjList  map[string][]string
	InDegree map[string]int
}

func (a *Agent) BuildGraph(ctx context.Context) (*WikiGraph, error) {
	g := &WikiGraph{
		AdjList:  make(map[string][]string),
		InDegree: make(map[string]int),
	}

	wikiDir := a.wiki.WikiPath
	err := filepath.Walk(wikiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		if strings.HasPrefix(filepath.Base(path), "_") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		content := string(data)
		_, stripped, _ := ParseFrontmatter(content)
		links := chunking.ExtractWikilinks(stripped)

		from := pageName(wikiDir, path)
		g.Nodes = append(g.Nodes, from)

		for _, to := range links {
			g.Edges = append(g.Edges, GraphEdge{From: from, To: to})
			g.AdjList[from] = append(g.AdjList[from], to)
			g.InDegree[to]++
		}

		return nil
	})

	if err != nil {
		return g, err
	}

	sort.Strings(g.Nodes)
	return g, nil
}

func (a *Agent) Neighbors(ctx context.Context, page string, direction string) ([]string, error) {
	g, err := a.BuildGraph(ctx)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)

	switch direction {
	case "out":
		for _, to := range g.AdjList[page] {
			seen[to] = true
		}
	case "in":
		for _, edge := range g.Edges {
			if edge.To == page {
				seen[edge.From] = true
			}
		}
	default:
		for _, to := range g.AdjList[page] {
			seen[to] = true
		}
		for _, edge := range g.Edges {
			if edge.To == page {
				seen[edge.From] = true
			}
		}
	}

	var result []string
	for k := range seen {
		result = append(result, k)
	}
	sort.Strings(result)
	return result, nil
}

func (a *Agent) NeighborsFromTS(ctx context.Context, page string) ([]string, error) {
	filterBy := fmt.Sprintf("links:=[%s]", page)
	searchResults, err := a.tsClient.TextSearch(ctx, ts.HybridSearchParams{
		Query:      "*",
		FilterBy:   filterBy,
		Limit:      50,
		GroupLimit: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("finding inbound links: %w", err)
	}

	var inbound []string
	seen := make(map[string]bool)
	for _, r := range searchResults {
		p := r.Path
		if !seen[p] {
			inbound = append(inbound, p)
			seen[p] = true
		}
	}
	return inbound, nil
}

func FormatGraph(g *WikiGraph, format string) string {
	switch format {
	case "mermaid":
		var b strings.Builder
		b.WriteString("graph TD\n")
		for _, edge := range g.Edges {
			from := sanitizeNode(edge.From)
			to := sanitizeNode(edge.To)
			b.WriteString(fmt.Sprintf("  %s --> %s\n", from, to))
		}
		return b.String()

	case "json":
		var b strings.Builder
		b.WriteString("{\n  \"nodes\": [\n")
		for i, n := range g.Nodes {
			if i > 0 {
				b.WriteString(",\n")
			}
			b.WriteString(fmt.Sprintf("    \"%s\"", n))
		}
		b.WriteString("\n  ],\n  \"edges\": [\n")
		for i, e := range g.Edges {
			if i > 0 {
				b.WriteString(",\n")
			}
			b.WriteString(fmt.Sprintf("    {\"from\": \"%s\", \"to\": \"%s\"}", e.From, e.To))
		}
		b.WriteString("\n  ]\n}")
		return b.String()

	default:
		var b strings.Builder
		b.WriteString("digraph wiki {\n")
		for _, edge := range g.Edges {
			from := sanitizeNode(edge.From)
			to := sanitizeNode(edge.To)
			b.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n", from, to))
		}
		b.WriteString("}\n")
		return b.String()
	}
}

func pageName(wikiDir, path string) string {
	rel, _ := filepath.Rel(wikiDir, path)
	name := strings.TrimSuffix(rel, ".md")
	return name
}

func sanitizeNode(name string) string {
	s := strings.ReplaceAll(name, "-", "_")
	s = strings.ReplaceAll(s, "/", "_")
	return s
}
