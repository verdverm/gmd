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

type Graph struct {
	Nodes    []string
	Edges    []GraphEdge
	AdjList  map[string][]string
	InDegree map[string]int
}

func (a *Agent) BuildGraph(ctx context.Context) (*Graph, error) {
	g := &Graph{
		AdjList:  make(map[string][]string),
		InDegree: make(map[string]int),
	}

	wikiDir := a.wiki.WikiPath

	pageNameToID := make(map[string]string)

	type pageData struct {
		conceptID string
		rawLinks  []string
	}
	var pages []pageData

	_ = filepath.Walk(wikiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		if filepath.Base(path) == a.wiki.WikiConfig.IndexFile || filepath.Base(path) == a.wiki.WikiConfig.LogFile {
			return nil
		}

		conceptID := pageName(wikiDir, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		fm, stripped, _ := ParseFrontmatter(content)

		pn := getPageName(fm, stripped)
		if pn != "" {
			pageNameToID[pn] = conceptID
		}

		wLinks := chunking.ExtractWikilinks(stripped)
		mLinks := chunking.ExtractMarkdownLinks(stripped)
		allLinks := make([]string, 0, len(wLinks)+len(mLinks))
		allLinks = append(allLinks, wLinks...)
		allLinks = append(allLinks, mLinks...)

		pages = append(pages, pageData{conceptID: conceptID, rawLinks: allLinks})
		g.Nodes = append(g.Nodes, conceptID)

		return nil
	})

	sourceDir := func(cid string) string {
		return filepath.Dir(filepath.Join(wikiDir, cid+".md"))
	}

	for _, page := range pages {
		from := page.conceptID
		seen := make(map[string]bool)
		seen[from] = true

		for _, rawLink := range page.rawLinks {
			var to string
			if strings.HasSuffix(rawLink, ".md") {
				to = chunking.NormalizeConceptID(rawLink, sourceDir(from))
			} else {
				if id, ok := pageNameToID[rawLink]; ok {
					to = id
				} else {
					to = rawLink
				}
			}

			if to != "" && !seen[to] && to != from {
				seen[to] = true
				g.Edges = append(g.Edges, GraphEdge{From: from, To: to})
				g.AdjList[from] = append(g.AdjList[from], to)
				g.InDegree[to]++
			}
		}
	}

	sort.Strings(g.Nodes)
	return g, nil
}

func getPageName(fm map[string]interface{}, stripped string) string {
	if fm != nil {
		if t, ok := fm["title"]; ok {
			if s, ok := t.(string); ok && s != "" {
				return s
			}
		}
	}
	lines := strings.SplitN(stripped, "\n", 2)
	if len(lines) > 0 {
		h1 := strings.TrimPrefix(lines[0], "# ")
		if h1 != lines[0] && h1 != "" {
			return h1
		}
	}
	return ""
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

	result := make([]string, 0, len(seen))
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

func FormatGraph(g *Graph, format string) string {
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
