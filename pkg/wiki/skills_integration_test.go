//go:build integration

package wiki

import (
	"testing"
)

func TestIntegrationListSkillTemplates(t *testing.T) {
	templates, err := ListSkillTemplates()
	if err != nil {
		t.Fatalf("ListSkillTemplates error: %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("expected at least 1 skill template")
	}

	found := make(map[string]bool)
	for _, tmpl := range templates {
		found[tmpl.Name] = true
		if tmpl.Name == "" {
			t.Error("found template with empty name")
		}
		if tmpl.Content == "" {
			t.Errorf("template %q has empty content", tmpl.Name)
		}
	}

	expected := []string{"AGENTS.md", "WIKI_SCHEMA.md", "claude-code.md", "codex-cli.md", "opencode.md", "generic.md"}
	for _, name := range expected {
		if !found[name] {
			t.Errorf("expected template %q not found", name)
		}
	}
}

func TestIntegrationGetSkillTemplate_Found(t *testing.T) {
	tmpl, err := GetSkillTemplate("AGENTS.md")
	if err != nil {
		t.Fatalf("GetSkillTemplate error: %v", err)
	}
	if tmpl.Name != "AGENTS.md" {
		t.Errorf("name = %q, want AGENTS.md", tmpl.Name)
	}
	if tmpl.Target != "universal" {
		t.Errorf("target = %q, want universal", tmpl.Target)
	}
	if tmpl.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestIntegrationGetSkillTemplate_NotFound(t *testing.T) {
	_, err := GetSkillTemplate("nonexistent-template")
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
}

func TestIntegrationGetSkillTemplate_AllTargets(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{"AGENTS.md", "universal"},
		{"WIKI_SCHEMA.md", "reference"},
		{"claude-code.md", "claude"},
		{"codex-cli.md", "codex"},
		{"opencode.md", "opencode"},
		{"generic.md", "generic"},
	}
	for _, tc := range tests {
		tmpl, err := GetSkillTemplate(tc.name)
		if err != nil {
			t.Errorf("GetSkillTemplate(%q) error: %v", tc.name, err)
			continue
		}
		if tmpl.Target != tc.target {
			t.Errorf("GetSkillTemplate(%q).Target = %q, want %q", tc.name, tmpl.Target, tc.target)
		}
	}
}

func TestIntegrationAgentDiscoveryPaths(t *testing.T) {
	paths := AgentDiscoveryPaths()
	if len(paths) == 0 {
		t.Fatal("expected non-empty discovery paths")
	}
	for name, p := range paths {
		if p == "" {
			t.Errorf("agent %q has empty path", name)
		}
	}
	expectedAgents := []string{"claude", "codex", "opencode"}
	for _, name := range expectedAgents {
		if _, ok := paths[name]; !ok {
			t.Errorf("expected agent %q in discovery paths", name)
		}
	}
}

func TestIntegrationCheckAgentInstalled_None(t *testing.T) {
	// In the test environment, no agents should be installed
	for _, name := range []string{"claude", "codex", "opencode"} {
		if CheckAgentInstalled(name) {
			t.Logf("agent %q is installed (unexpected in test env)", name)
		}
	}
}

func TestIntegrationCheckSkillInstalled_None(t *testing.T) {
	for _, name := range []string{"claude", "codex", "opencode"} {
		if CheckSkillInstalled(name) {
			t.Logf("skill for %q is installed (unexpected in test env)", name)
		}
	}
}

func TestIntegrationCheckSkillInstalled_UnknownAgent(t *testing.T) {
	if CheckSkillInstalled("nonexistent") {
		t.Error("expected false for nonexistent agent")
	}
}
