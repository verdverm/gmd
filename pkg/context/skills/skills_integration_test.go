//go:build integration

package skills

import (
	"os"
	"testing"
)

func TestIntegrationSkills_ListSkillNames(t *testing.T) {
	names, err := ListSkillNames()
	if err != nil {
		t.Fatalf("ListSkillNames error: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("expected at least 1 skill")
	}
	if names[0] != "gmd-wiki" {
		t.Errorf("expected gmd-wiki, got %v", names)
	}
}

func TestIntegrationGetSkillContent_Found(t *testing.T) {
	content, err := GetSkillContent("gmd-wiki")
	if err != nil {
		t.Fatalf("GetSkillContent error: %v", err)
	}
	if content == "" {
		t.Error("expected non-empty content")
	}
}

func TestIntegrationGetSkillContent_NotFound(t *testing.T) {
	_, err := GetSkillContent("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent skill")
	}
}

func TestIntegrationHarnessDir_Unknown(t *testing.T) {
	_, err := harnessDir("/tmp", true, "nonexistent")
	if err == nil {
		t.Error("expected error for unknown harness")
	}
}

func TestIntegrationCheckHarnessInstalled_None(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("no home directory: %v", err)
	}
	for _, name := range HarnessNames() {
		installed, err := CheckHarnessInstalled(home, true, name)
		if err != nil {
			t.Errorf("CheckHarnessInstalled(%q) error: %v", name, err)
		}
		if installed {
			t.Logf("harness %q is installed (unexpected in test env)", name)
		}
	}
}

func TestIntegrationSkillInstalled_None(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("no home directory: %v", err)
	}
	skills, err := ListSkillNames()
	if err != nil {
		t.Fatalf("ListSkillNames error: %v", err)
	}
	for _, name := range HarnessNames() {
		for _, sn := range skills {
			installed, err := SkillInstalled(home, true, name, sn)
			if err != nil {
				t.Errorf("SkillInstalled(%q, %q) error: %v", name, sn, err)
			}
			if installed {
				t.Logf("skill %q for %q is installed (unexpected in test env)", sn, name)
			}
		}
	}
}

func TestIntegrationSkillInstalled_UnknownHarness(t *testing.T) {
	_, err := SkillInstalled("/tmp", true, "nonexistent", "gmd-wiki")
	if err == nil {
		t.Error("expected error for unknown harness")
	}
}
