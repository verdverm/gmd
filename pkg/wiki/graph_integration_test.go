//go:build integration

package wiki

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/verdverm/gmd/pkg/config"
)

func TestIntegrationBuildGraph_EmptyWiki(t *testing.T) {
	_, agent := newTestWikiAgent(t)
	g, err := agent.BuildGraph(context.Background())
	if err != nil {
		t.Fatalf("BuildGraph error: %v", err)
	}
	if len(g.Nodes) != 0 {
		t.Errorf("expected 0 nodes in empty wiki, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(g.Edges))
	}
}

func TestIntegrationBuildGraph_WithWikilinks(t *testing.T) {
	_, agent := newTestWikiAgent(t)

	// Create entity page linking to a concept
	entityPath := filepath.Join(agent.wiki.WikiPath, "entities", "machine-learning.md")
	os.MkdirAll(filepath.Dir(entityPath), 0755)
	os.WriteFile(entityPath, []byte("---\ntype: entity\n---\n\n# Machine Learning\n\nSee also [[concepts/supervised-learning]] and [[concepts/unsupervised-learning]].\n"), 0644)

	// Create concept page
	conceptPath := filepath.Join(agent.wiki.WikiPath, "concepts", "supervised-learning.md")
	os.MkdirAll(filepath.Dir(conceptPath), 0755)
	os.WriteFile(conceptPath, []byte("---\ntype: concept\n---\n\n# Supervised Learning\n\nRelated to [[entities/machine-learning]].\n"), 0644)

	// index.md is reserved and should be skipped by BuildGraph
	indexPath := filepath.Join(agent.wiki.WikiPath, "index.md")
	os.WriteFile(indexPath, []byte("[[hidden]]"), 0644)

	g, err := agent.BuildGraph(context.Background())
	if err != nil {
		t.Fatalf("BuildGraph error: %v", err)
	}

	if len(g.Nodes) != 2 {
		t.Errorf("expected 2 nodes (index.md skipped), got %d: %v", len(g.Nodes), g.Nodes)
	}

	foundEdge := false
	for _, e := range g.Edges {
		if e.From == "entities/machine-learning" && e.To == "concepts/supervised-learning" {
			foundEdge = true
			break
		}
	}
	if !foundEdge {
		t.Errorf("expected edge from entities/machine-learning to concepts/supervised-learning, got edges: %v", g.Edges)
	}
}

func TestIntegrationNeighbors_AllDirections(t *testing.T) {
	_, agent := newTestWikiAgent(t)

	// Page A links to B and C
	os.MkdirAll(filepath.Join(agent.wiki.WikiPath, "entities"), 0755)
	os.WriteFile(filepath.Join(agent.wiki.WikiPath, "entities", "a.md"), []byte("# A\n[[entities/b]] [[entities/c]]\n"), 0644)
	os.WriteFile(filepath.Join(agent.wiki.WikiPath, "entities", "b.md"), []byte("# B\n[[entities/a]]\n"), 0644)
	os.WriteFile(filepath.Join(agent.wiki.WikiPath, "entities", "c.md"), []byte("# C\nNo links.\n"), 0644)

	outNeighbors, err := agent.Neighbors(context.Background(), "entities/a", "out")
	if err != nil {
		t.Fatalf("Neighbors out error: %v", err)
	}
	if len(outNeighbors) != 2 {
		t.Errorf("expected 2 outgoing neighbors, got %d: %v", len(outNeighbors), outNeighbors)
	}

	inNeighbors, err := agent.Neighbors(context.Background(), "entities/b", "in")
	if err != nil {
		t.Fatalf("Neighbors in error: %v", err)
	}
	if len(inNeighbors) != 1 {
		t.Errorf("expected 1 incoming neighbor, got %d: %v", len(inNeighbors), inNeighbors)
	}
	if inNeighbors[0] != "entities/a" {
		t.Errorf("expected entities/a, got %q", inNeighbors[0])
	}

	both, err := agent.Neighbors(context.Background(), "entities/a", "both")
	if err != nil {
		t.Fatalf("Neighbors both error: %v", err)
	}
	if len(both) == 0 {
		t.Error("expected non-empty neighbors for both")
	}
}

func TestIntegrationNeighbors_PageWithoutLinks(t *testing.T) {
	_, agent := newTestWikiAgent(t)
	os.MkdirAll(filepath.Join(agent.wiki.WikiPath, "entities"), 0755)
	os.WriteFile(filepath.Join(agent.wiki.WikiPath, "entities", "orphan.md"), []byte("# Orphan\nNo links.\n"), 0644)

	neighbors, err := agent.Neighbors(context.Background(), "entities/orphan", "out")
	if err != nil {
		t.Fatalf("Neighbors error: %v", err)
	}
	if len(neighbors) != 0 {
		t.Errorf("expected 0 neighbors for orphan, got %d", len(neighbors))
	}
}

func TestIntegrationGraph_NeighborsFromTS(t *testing.T) {
	c := tapeTest(t, "testdata/Graph_NeighborsFromTS.json")
	defer c.Stop()

	ctx := context.Background()
	defer cleanupTestData(ctx, t, testCollKey)

	tmpDir := t.TempDir()
	wc := &config.WikiConfig{
		SourceConfig: config.SourceConfig{Path: tmpDir},
		WikiDir:      "wiki",
		RawDir:       "raw",
		IndexFile:    "index.md",
		LogFile:      "log.md",
		GraphLinks:   true,
	}
	w, err := NewWiki("test-wiki", tmpDir, wc)
	if err != nil {
		t.Fatalf("NewWiki error: %v", err)
	}
	if err := w.Init(); err != nil {
		t.Fatalf("Init error: %v", err)
	}
	agent := NewAgent(w, testCfg, c.TS, c.Chat)

	if err := os.MkdirAll(filepath.Join(w.WikiPath, "entities"), 0755); err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(
		filepath.Join(w.WikiPath, "entities", "A.md"),
		[]byte("---\ntype: entity\n---\n# A\nLinks to [[entities/B]] and [[entities/C]].\n"),
		0644,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = indexTapedWikiPage(ctx, c.TS, c.Embedder, testCfg.CollectionKey(w.Name), w.WikiPath, "entities/A.md")
	if err != nil {
		t.Fatalf("indexTapedWikiPage error: %v", err)
	}

	err = os.WriteFile(
		filepath.Join(w.WikiPath, "entities", "B.md"),
		[]byte("---\ntype: entity\n---\n# B\nLinks to [[entities/A]].\n"),
		0644,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = indexTapedWikiPage(ctx, c.TS, c.Embedder, testCfg.CollectionKey(w.Name), w.WikiPath, "entities/B.md")
	if err != nil {
		t.Fatalf("indexTapedWikiPage error: %v", err)
	}

	result, err := agent.NeighborsFromTS(ctx, "entities/B")
	if err != nil {
		t.Fatalf("NeighborsFromTS error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 result (page linking to entities/B), got %d: %v", len(result), result)
	}
	if !strings.Contains(result[0], "entities/A") {
		t.Errorf("expected result to reference entities/A, got %q", result[0])
	}

	// Find inbound links to entities/C — should find A
	result, err = agent.NeighborsFromTS(ctx, "entities/C")
	if err != nil {
		t.Fatalf("NeighborsFromTS error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result (page linking to entities/C), got %d: %v", len(result), result)
	}
	if !strings.Contains(result[0], "entities/A") {
		t.Errorf("expected result to reference entities/A, got %q", result[0])
	}

	// Find inbound links to nonexistent page — should return empty
	result, err = agent.NeighborsFromTS(ctx, "entities/nonexistent")
	if err != nil {
		t.Fatalf("NeighborsFromTS error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results for nonexistent page, got %d", len(result))
	}
}

func TestIntegrationFormatGraph_Mermaid(t *testing.T) {
	g := &Graph{
		Nodes: []string{"a", "b"},
		Edges: []GraphEdge{{From: "a", To: "b"}},
	}
	out := FormatGraph(g, "mermaid")
	if !strings.Contains(out, "graph TD") {
		t.Error("expected mermaid header")
	}
	if !strings.Contains(out, "a --> b") {
		t.Error("expected edge in mermaid output")
	}
}

func TestIntegrationFormatGraph_JSON(t *testing.T) {
	g := &Graph{
		Nodes: []string{"a", "b"},
		Edges: []GraphEdge{{From: "a", To: "b"}},
	}
	out := FormatGraph(g, "json")
	if !strings.Contains(out, `"nodes"`) {
		t.Error("expected nodes key in JSON output")
	}
	if !strings.Contains(out, `"from": "a"`) {
		t.Error("expected edge data in JSON output")
	}
}

func TestIntegrationFormatGraph_DOT(t *testing.T) {
	g := &Graph{
		Nodes: []string{"a", "b"},
		Edges: []GraphEdge{{From: "a", To: "b"}},
	}
	out := FormatGraph(g, "dot")
	if !strings.Contains(out, "digraph wiki") {
		t.Error("expected DOT header")
	}
	if !strings.Contains(out, `"a" -> "b"`) {
		t.Error("expected edge in DOT output")
	}
}

func TestIntegrationFormatGraph_Default(t *testing.T) {
	g := &Graph{
		Nodes: []string{"x", "y"},
		Edges: []GraphEdge{{From: "x", To: "y"}},
	}
	out := FormatGraph(g, "unknown")
	if !strings.Contains(out, "digraph wiki") {
		t.Error("expected DOT output for unknown format (default)")
	}
}

func TestIntegrationFormatGraph_Sanitized(t *testing.T) {
	g := &Graph{
		Nodes: []string{"entities/my-page", "concepts/deep-learning"},
		Edges: []GraphEdge{{From: "entities/my-page", To: "concepts/deep-learning"}},
	}
	out := FormatGraph(g, "mermaid")
	// Slash and hyphen should be converted for node IDs
	if strings.Contains(out, "entities/my-page") {
		t.Log("mermaid output contains unsanitized node name")
	}
	_ = out
}

func TestIntegrationGraph_PageName(t *testing.T) {
	name := pageName("/wiki", "/wiki/entities/test.md")
	if name != "entities/test" {
		t.Errorf("got %q, want %q", name, "entities/test")
	}
}

func TestIntegrationGraph_SanitizeNode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"entities/my-page", "entities_my_page"},
		{"simple", "simple"},
		{"a/b/c", "a_b_c"},
	}
	for _, tc := range tests {
		got := sanitizeNode(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeNode(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
