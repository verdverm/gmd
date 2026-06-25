//go:build integration

package wiki

import (
	"testing"

	"github.com/verdverm/gmd/pkg/config"
)

func TestIntegrationParseFrontmatter_NoFrontmatter(t *testing.T) {
	fm, body, err := ParseFrontmatter("# Just content\n\nNo frontmatter.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm != nil {
		t.Errorf("expected nil frontmatter, got %v", fm)
	}
	if body != "# Just content\n\nNo frontmatter." {
		t.Errorf("body mismatch: got %q", body)
	}
}

func TestIntegrationParseFrontmatter_EmptyFrontmatter(t *testing.T) {
	// The regex requires at least one char between the --- markers
	// so "---\n---\n\nContent" is treated as having no frontmatter
	fm, body, err := ParseFrontmatter("---\n---\n\nContent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm != nil {
		t.Errorf("expected nil frontmatter for empty delimiter body, got %v", fm)
	}
	if body != "---\n---\n\nContent" {
		t.Errorf("body mismatch: got %q", body)
	}
}

func TestIntegrationParseFrontmatter_Valid(t *testing.T) {
	input := "---\ntype: entity\ntags: [ai, ml]\nstatus: draft\n---\n\n# Page\nBody."
	fm, body, err := ParseFrontmatter(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm["type"] != "entity" {
		t.Errorf("type = %v, want entity", fm["type"])
	}
	if body != "# Page\nBody." {
		t.Errorf("body mismatch: got %q", body)
	}
}

func TestIntegrationParseFrontmatter_InvalidYAML(t *testing.T) {
	_, _, err := ParseFrontmatter("---\n: invalid yaml\n---\n\nbody")
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestIntegrationFrontmatter_Strip(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"---\ntype: x\n---\n\nbody", "body"},
		{"no frontmatter", "no frontmatter"},
		{"", ""},
	}
	for _, tc := range tests {
		got := StripFrontmatter(tc.input)
		if got != tc.expected {
			t.Errorf("StripFrontmatter(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestIntegrationValidateFrontmatter_NilConfig(t *testing.T) {
	err := ValidateFrontmatter(map[string]interface{}{"type": "entity"}, nil)
	if err != nil {
		t.Errorf("expected nil error for nil config, got %v", err)
	}
}

func TestIntegrationValidateFrontmatter_EmptyConfig(t *testing.T) {
	err := ValidateFrontmatter(map[string]interface{}{"type": "entity"}, &config.FrontmatterConfig{})
	if err != nil {
		t.Errorf("expected nil error for empty config, got %v", err)
	}
}

func TestIntegrationValidateFrontmatter_StringField(t *testing.T) {
	cfg := &config.FrontmatterConfig{
		Fields: map[string]config.FrontmatterField{
			"type": {Type: "string"},
		},
	}
	if err := ValidateFrontmatter(map[string]interface{}{"type": "entity"}, cfg); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if err := ValidateFrontmatter(map[string]interface{}{"type": 42}, cfg); err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestIntegrationValidateFrontmatter_StringArrayField(t *testing.T) {
	cfg := &config.FrontmatterConfig{
		Fields: map[string]config.FrontmatterField{
			"tags": {Type: "string[]"},
		},
	}
	if err := ValidateFrontmatter(map[string]interface{}{"tags": []interface{}{"a", "b"}}, cfg); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if err := ValidateFrontmatter(map[string]interface{}{"tags": []interface{}{"a", 42}}, cfg); err == nil {
		t.Error("expected error for non-string element")
	}
	if err := ValidateFrontmatter(map[string]interface{}{"tags": "not-array"}, cfg); err == nil {
		t.Error("expected error for non-array value")
	}
}

func TestIntegrationValidateFrontmatter_IntField(t *testing.T) {
	cfg := &config.FrontmatterConfig{
		Fields: map[string]config.FrontmatterField{
			"priority": {Type: "int32"},
		},
	}
	if err := ValidateFrontmatter(map[string]interface{}{"priority": 5}, cfg); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if err := ValidateFrontmatter(map[string]interface{}{"priority": "high"}, cfg); err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestIntegrationValidateFrontmatter_FloatField(t *testing.T) {
	cfg := &config.FrontmatterConfig{
		Fields: map[string]config.FrontmatterField{
			"score": {Type: "float"},
		},
	}
	if err := ValidateFrontmatter(map[string]interface{}{"score": 3.14}, cfg); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if err := ValidateFrontmatter(map[string]interface{}{"score": "high"}, cfg); err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestIntegrationValidateFrontmatter_BoolField(t *testing.T) {
	cfg := &config.FrontmatterConfig{
		Fields: map[string]config.FrontmatterField{
			"published": {Type: "bool"},
		},
	}
	if err := ValidateFrontmatter(map[string]interface{}{"published": true}, cfg); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if err := ValidateFrontmatter(map[string]interface{}{"published": "yes"}, cfg); err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestIntegrationValidateFrontmatter_UnknownType(t *testing.T) {
	cfg := &config.FrontmatterConfig{
		Fields: map[string]config.FrontmatterField{
			"custom": {Type: "unknown"},
		},
	}
	err := ValidateFrontmatter(map[string]interface{}{"custom": "val"}, cfg)
	if err == nil {
		t.Error("expected error for unknown type")
	}
}

func TestIntegrationFrontmatterToFilter_NilConfig(t *testing.T) {
	if s := FrontmatterToFilter(map[string]interface{}{"type": "entity"}, nil); s != "" {
		t.Errorf("expected empty, got %q", s)
	}
}

func TestIntegrationFrontmatterToFilter_StringField(t *testing.T) {
	cfg := &config.FrontmatterConfig{
		Fields: map[string]config.FrontmatterField{
			"type": {Type: "string"},
		},
	}
	s := FrontmatterToFilter(map[string]interface{}{"type": "entity"}, cfg)
	if s != "type:=entity" {
		t.Errorf("got %q, want %q", s, "type:=entity")
	}
}

func TestIntegrationFrontmatterToFilter_StringArray(t *testing.T) {
	cfg := &config.FrontmatterConfig{
		Fields: map[string]config.FrontmatterField{
			"tags": {Type: "string[]"},
		},
	}
	s := FrontmatterToFilter(map[string]interface{}{"tags": []interface{}{"ai", "ml"}}, cfg)
	if s != "tags:=[ai,ml]" {
		t.Errorf("got %q, want %q", s, "tags:=[ai,ml]")
	}
}

func TestIntegrationFrontmatterToFilter_IntField(t *testing.T) {
	cfg := &config.FrontmatterConfig{
		Fields: map[string]config.FrontmatterField{
			"priority": {Type: "int32"},
		},
	}
	s := FrontmatterToFilter(map[string]interface{}{"priority": 5}, cfg)
	if s != "priority:=5" {
		t.Errorf("got %q, want %q", s, "priority:=5")
	}
}

func TestIntegrationFrontmatterToFilter_FloatField(t *testing.T) {
	cfg := &config.FrontmatterConfig{
		Fields: map[string]config.FrontmatterField{
			"score": {Type: "float"},
		},
	}
	s := FrontmatterToFilter(map[string]interface{}{"score": 3.14}, cfg)
	if s != "score:=3.14" {
		t.Errorf("got %q, want %q", s, "score:=3.14")
	}
}

func TestIntegrationFrontmatterToFilter_BoolField(t *testing.T) {
	cfg := &config.FrontmatterConfig{
		Fields: map[string]config.FrontmatterField{
			"published": {Type: "bool"},
		},
	}
	s := FrontmatterToFilter(map[string]interface{}{"published": true}, cfg)
	if s != "published:=true" {
		t.Errorf("got %q, want %q", s, "published:=true")
	}
}
