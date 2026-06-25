package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAgents_ProjectAgentsDir(t *testing.T) {
	got := ProjectAgentsDir("/project/root")
	expected := filepath.Join("/project/root", ".gmd", "agents")
	if got != expected {
		t.Errorf("ProjectAgentsDir = %q, want %q", got, expected)
	}
}

func TestAgents_ResolveDir(t *testing.T) {
	t.Run("global", func(t *testing.T) {
		dir, err := ResolveDir(true, "/irrelevant")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir == "" {
			t.Fatal("expected non-empty global dir")
		}
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".config", "gmd", "agents")
		if dir != expected {
			t.Errorf("global dir = %q, want %q", dir, expected)
		}
	})

	t.Run("project", func(t *testing.T) {
		dir, err := ResolveDir(false, "/project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join("/project", ".gmd", "agents")
		if dir != expected {
			t.Errorf("project dir = %q, want %q", dir, expected)
		}
	})
}

func TestAgents_ListAgents(t *testing.T) {
	t.Run("non-existent dir returns empty", func(t *testing.T) {
		dir := t.TempDir()
		names, err := ListAgents(false, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(names) != 0 {
			t.Errorf("expected 0 agent names, got %v", names)
		}
	})

	t.Run("dir with agent subdirs", func(t *testing.T) {
		dir := t.TempDir()
		agentsDir := ProjectAgentsDir(dir)
		if err := os.MkdirAll(filepath.Join(agentsDir, "foo"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(agentsDir, "bar"), 0755); err != nil {
			t.Fatal(err)
		}

		names, err := ListAgents(false, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(names) != 2 {
			t.Errorf("expected 2 agent names, got %d: %v", len(names), names)
		}
	})

	t.Run("files in agents dir are ignored", func(t *testing.T) {
		dir := t.TempDir()
		agentsDir := ProjectAgentsDir(dir)
		if err := os.MkdirAll(agentsDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(agentsDir, "not-an-agent.txt"), []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(agentsDir, "real-agent"), 0755); err != nil {
			t.Fatal(err)
		}

		names, err := ListAgents(false, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(names) != 1 {
			t.Errorf("expected 1 agent name (ignoring files), got %d: %v", len(names), names)
		}
		if names[0] != "real-agent" {
			t.Errorf("expected real-agent, got %s", names[0])
		}
	})
}

func TestAgents_ShowAgent(t *testing.T) {
	t.Run("existing agent with files", func(t *testing.T) {
		dir := t.TempDir()
		agentsDir := ProjectAgentsDir(dir)
		agentDir := filepath.Join(agentsDir, "wiki-writer")
		if err := os.MkdirAll(agentDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(agentDir, "prompt.md"), []byte("you are a wiki writer"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(agentDir, "config.json"), []byte(`{"key":"val"}`), 0644); err != nil {
			t.Fatal(err)
		}

		files, err := ShowAgent("wiki-writer", false, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 2 {
			t.Errorf("expected 2 files, got %d: %v", len(files), files)
		}
		if files["prompt.md"] != "you are a wiki writer" {
			t.Errorf("prompt.md = %q, want %q", files["prompt.md"], "you are a wiki writer")
		}
		if files["config.json"] != `{"key":"val"}` {
			t.Errorf("config.json = %q, want %q", files["config.json"], `{"key":"val"}`)
		}
	})

	t.Run("non-existent agent", func(t *testing.T) {
		dir := t.TempDir()
		_, err := ShowAgent("missing-agent", false, dir)
		if err == nil {
			t.Fatal("expected error for missing agent")
		}
	})

	t.Run("name is a file not a directory", func(t *testing.T) {
		dir := t.TempDir()
		agentsDir := ProjectAgentsDir(dir)
		if err := os.MkdirAll(agentsDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(agentsDir, "bad-agent"), []byte("not a dir"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := ShowAgent("bad-agent", false, dir)
		if err == nil {
			t.Fatal("expected error when name is a file not a directory")
		}
	})

	t.Run("agent dir with subdirectory is ignored", func(t *testing.T) {
		dir := t.TempDir()
		agentsDir := ProjectAgentsDir(dir)
		agentDir := filepath.Join(agentsDir, "my-agent")
		if err := os.MkdirAll(filepath.Join(agentDir, "nested"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(agentDir, "prompt.md"), []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}

		files, err := ShowAgent("my-agent", false, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 1 {
			t.Errorf("expected 1 file (ignoring subdirs), got %d", len(files))
		}
		if files["prompt.md"] != "hello" {
			t.Errorf("prompt.md = %q, want %q", files["prompt.md"], "hello")
		}
	})

	t.Run("unreadable file in agent dir", func(t *testing.T) {
		dir := t.TempDir()
		agentsDir := ProjectAgentsDir(dir)
		agentDir := filepath.Join(agentsDir, "broken-agent")
		if err := os.MkdirAll(agentDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(agentDir, "readable.md"), []byte("ok"), 0644); err != nil {
			t.Fatal(err)
		}
		badFile := filepath.Join(agentDir, "unreadable")
		if err := os.WriteFile(badFile, []byte("secret"), 0000); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(badFile, 0644)

		files, err := ShowAgent("broken-agent", false, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if files["readable.md"] != "ok" {
			t.Errorf("readable.md content mismatch: %q", files["readable.md"])
		}
		if !containsPrefix(files["unreadable"], "[error reading:") {
			t.Errorf("expected error placeholder for unreadable, got %q", files["unreadable"])
		}
	})
}

func containsPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
