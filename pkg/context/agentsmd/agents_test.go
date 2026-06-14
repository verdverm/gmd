package agentsmd

import (
	"testing"
)

func TestValidNames(t *testing.T) {
	names, err := ValidNames()
	if err != nil {
		t.Fatalf("ValidNames error: %v", err)
	}

	expected := map[string]bool{
		"oneline":  false,
		"summary":  false,
		"detailed": false,
		"full":     false,
	}
	if len(names) != len(expected) {
		t.Errorf("expected %d names, got %d: %v", len(expected), len(names), names)
	}

	for _, n := range names {
		if _, ok := expected[n]; !ok {
			t.Errorf("unexpected name %q", n)
		} else {
			expected[n] = true
		}
		if n == "" {
			t.Error("name should not be empty")
		}
	}

	for k, seen := range expected {
		if !seen {
			t.Errorf("expected name %q not found", k)
		}
	}
}

func TestGetContent(t *testing.T) {
	allNames, err := ValidNames()
	if err != nil {
		t.Fatalf("ValidNames error: %v", err)
	}

	for _, name := range allNames {
		t.Run(name, func(t *testing.T) {
			content, err := GetContent(name)
			if err != nil {
				t.Fatalf("GetContent(%q) error: %v", name, err)
			}
			if content == "" {
				t.Errorf("GetContent(%q) returned empty string", name)
			}
		})
	}

	t.Run("invalid name", func(t *testing.T) {
		_, err := GetContent("nonexistent")
		if err == nil {
			t.Fatal("expected error for nonexistent detail level")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := GetContent("")
		if err == nil {
			t.Fatal("expected error for empty name")
		}
	})

	t.Run("trimmed", func(t *testing.T) {
		content, err := GetContent("oneline")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Content should be trimmed of leading/trailing whitespace
		if len(content) > 0 && (content[0] == ' ' || content[0] == '\n') {
			t.Error("content should not start with whitespace")
		}
		if len(content) > 0 && (content[len(content)-1] == ' ' || content[len(content)-1] == '\n') {
			t.Error("content should not end with whitespace")
		}
	})
}
