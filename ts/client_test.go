package ts

import (
	"math"
	"strings"
	"testing"
)

func TestBuildCollectionFilter(t *testing.T) {
	tests := []struct {
		name     string
		cols     []string
		expected string
	}{
		{
			name:     "single collection",
			cols:     []string{"docs"},
			expected: "collection:=[docs]",
		},
		{
			name:     "multiple collections",
			cols:     []string{"docs", "notes"},
			expected: "collection:=[docs,notes]",
		},
		{
			name:     "three collections",
			cols:     []string{"a", "b", "c"},
			expected: "collection:=[a,b,c]",
		},
		{
			name:     "empty collections",
			cols:     []string{},
			expected: "collection:=[]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCollectionFilter(tt.cols)
			if got != tt.expected {
				t.Errorf("buildCollectionFilter() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFormatVector(t *testing.T) {
	tests := []struct {
		name     string
		input    []float64
		checkLen bool
	}{
		{
			name:     "empty vector",
			input:    []float64{},
			checkLen: false,
		},
		{
			name:     "single element",
			input:    []float64{0.5},
			checkLen: false,
		},
		{
			name:     "multiple elements",
			input:    []float64{0.1, 0.2, 0.3},
			checkLen: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatVector(tt.input)
			if tt.name == "empty vector" && got != "" {
				t.Errorf("expected empty string, got %q", got)
			}
			if len(tt.input) > 0 && got == "" {
				t.Errorf("expected non-empty vector string")
			}
		})
	}

	t.Run("trailing comma not present", func(t *testing.T) {
		got := formatVector([]float64{0.1})
		if strings.HasSuffix(got, ",") {
			t.Errorf("should not end with comma: %q", got)
		}
	})

	t.Run("comma separated", func(t *testing.T) {
		got := formatVector([]float64{1.0, 2.0, 3.0})
		expected := "1.000000,2.000000,3.000000"
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
	})
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		input interface{}
		want  float64
	}{
		{float64(3.14), 3.14},
		{int(42), 42.0},
		{int64(100), 100.0},
		{int32(50), 50.0},
		{uint64(200), 200.0},
		{uint32(75), 75.0},
		{string("not a number"), 0},
		{nil, 0},
		{true, 0},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := toFloat64(tt.input)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("toFloat64(%v) = %f, want %f", tt.input, got, tt.want)
			}
		})
	}
}

func TestGroupedHitsToResults(t *testing.T) {
	t.Run("nil response", func(t *testing.T) {
		results := groupedHitsToResults(nil)
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})
}

func TestBoolPtr(t *testing.T) {
	b := boolPtr(true)
	if b == nil {
		t.Fatal("boolPtr returned nil")
	}
	if *b != true {
		t.Errorf("got %v, want true", *b)
	}
}

func TestIntPtr(t *testing.T) {
	i := intPtr(42)
	if i == nil {
		t.Fatal("intPtr returned nil")
	}
	if *i != 42 {
		t.Errorf("got %d, want 42", *i)
	}
}

func TestStringPtr(t *testing.T) {
	s := stringPtr("hello")
	if s == nil {
		t.Fatal("stringPtr returned nil")
	}
	if *s != "hello" {
		t.Errorf("got %q, want %q", *s, "hello")
	}
}

func TestNewClient(t *testing.T) {
	cfg := Config{Host: "http://localhost:8108", APIKey: "test-key"}
	c := New(cfg)
	if c == nil {
		t.Fatal("New() returned nil")
	}
	if c.config.Host != "http://localhost:8108" {
		t.Errorf("host = %q, want %q", c.config.Host, "http://localhost:8108")
	}
}
