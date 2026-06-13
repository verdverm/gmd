package skills

import (
	"testing"
)

func TestListSkillNames(t *testing.T) {
	names, err := ListSkillNames()
	if err != nil {
		t.Fatalf("ListSkillNames error: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("expected at least one skill")
	}
	if names[0] != "gmd-wiki" {
		t.Errorf("expected gmd-wiki, got %v", names)
	}
}

func TestGetSkillContent(t *testing.T) {
	t.Run("existing skill", func(t *testing.T) {
		content, err := GetSkillContent("gmd-wiki")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content == "" {
			t.Error("expected non-empty content")
		}
	})

	t.Run("missing skill", func(t *testing.T) {
		_, err := GetSkillContent("nonexistent")
		if err == nil {
			t.Fatal("expected error for missing skill")
		}
	})
}

func TestHarnessNames(t *testing.T) {
	names := HarnessNames()
	if len(names) != 3 {
		t.Errorf("expected 3 harnesses, got %d: %v", len(names), names)
	}
	seen := make(map[string]bool)
	for _, n := range names {
		seen[n] = true
	}
	for _, want := range []string{"claude", "codex", "opencode"} {
		if !seen[want] {
			t.Errorf("expected harness %q", want)
		}
	}
}
