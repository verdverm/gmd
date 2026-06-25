package llm

import (
	"testing"

	"google.golang.org/genai"
)

func TestLLM_ConvertRole(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"model", "assistant"},
		{"user", "user"},
		{"system", "system"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := convertRole(tt.input); got != tt.want {
			t.Errorf("convertRole(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLLM_ConvertFinishReason(t *testing.T) {
	tests := []struct {
		input string
		want  genai.FinishReason
	}{
		{"stop", genai.FinishReasonStop},
		{"tool_calls", genai.FinishReasonStop},
		{"function_call", genai.FinishReasonStop},
		{"length", genai.FinishReasonMaxTokens},
		{"content_filter", genai.FinishReasonSafety},
		{"unknown", genai.FinishReasonUnspecified},
		{"", genai.FinishReasonUnspecified},
	}
	for _, tt := range tests {
		if got := convertFinishReason(tt.input); got != tt.want {
			t.Errorf("convertFinishReason(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLLM_SchemaTypeToString(t *testing.T) {
	tests := []struct {
		input genai.Type
		want  string
	}{
		{genai.TypeString, "string"},
		{genai.TypeNumber, "number"},
		{genai.TypeInteger, "integer"},
		{genai.TypeBoolean, "boolean"},
		{genai.TypeArray, "array"},
		{genai.TypeObject, "object"},
		{genai.TypeUnspecified, "string"},
	}
	for _, tt := range tests {
		if got := schemaTypeToString(tt.input); got != tt.want {
			t.Errorf("schemaTypeToString(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLLM_ExtractText(t *testing.T) {
	tests := []struct {
		name    string
		content *genai.Content
		want    string
	}{
		{"nil", nil, ""},
		{"empty", &genai.Content{}, ""},
		{"single", &genai.Content{Parts: []*genai.Part{{Text: "hello"}}}, "hello"},
		{"multi", &genai.Content{Parts: []*genai.Part{{Text: "a"}, {Text: "b"}}}, "a\nb"},
		{"mixed", &genai.Content{Parts: []*genai.Part{{Text: "a"}, {}, {Text: "b"}}}, "a\nb"},
	}
	for _, tt := range tests {
		if got := extractText(tt.content); got != tt.want {
			t.Errorf("extractText(%s) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestLLM_JoinTexts(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{nil, ""},
		{[]string{"a"}, "a"},
		{[]string{"a", "b"}, "a\nb"},
	}
	for _, tt := range tests {
		if got := joinTexts(tt.input); got != tt.want {
			t.Errorf("joinTexts(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLLM_ParseJSONArgs(t *testing.T) {
	tests := []struct {
		input string
		want  map[string]any
	}{
		{"", map[string]any{}},
		{"{}", map[string]any{}},
		{`{"key":"value"}`, map[string]any{"key": "value"}},
		{"invalid", map[string]any{}},
	}
	for _, tt := range tests {
		got := parseJSONArgs(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parseJSONArgs(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for k, v := range tt.want {
			if got[k] != v {
				t.Errorf("parseJSONArgs(%q)[%q] = %v, want %v", tt.input, k, got[k], v)
			}
		}
	}
}

func TestLLM_EnsureObjectProperties(t *testing.T) {
	tests := []struct {
		name   string
		schema map[string]any
		want   map[string]any
	}{
		{"nil", nil, nil},
		{"object no props", map[string]any{"type": "object"}, map[string]any{"type": "object", "properties": map[string]any{}}},
		{"object with props", map[string]any{"type": "object", "properties": map[string]any{"x": map[string]any{"type": "string"}}}, map[string]any{"type": "object", "properties": map[string]any{"x": map[string]any{"type": "string"}}}},
		{"nested object", map[string]any{"type": "object", "properties": map[string]any{"x": map[string]any{"type": "object"}}}, map[string]any{"type": "object", "properties": map[string]any{"x": map[string]any{"type": "object", "properties": map[string]any{}}}}},
		{"array items", map[string]any{"type": "array", "items": map[string]any{"type": "object"}}, map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{}}}},
	}
	for _, tt := range tests {
		schema := make(map[string]any)
		for k, v := range tt.schema {
			schema[k] = v
		}
		ensureObjectProperties(schema)
		for k, v := range tt.want {
			if schema[k] == nil && v != nil {
				t.Errorf("ensureObjectProperties(%s): key %q is nil, want %v", tt.name, k, v)
			}
		}
	}
}

func TestLLM_ConvertSchema(t *testing.T) {
	schema := &genai.Schema{
		Type:        genai.TypeObject,
		Description: "test schema",
		Required:    []string{"name"},
		Properties: map[string]*genai.Schema{
			"name": {Type: genai.TypeString, Description: "the name"},
			"age":  {Type: genai.TypeInteger},
		},
	}
	result, err := convertSchema(schema)
	if err != nil {
		t.Fatalf("convertSchema() error: %v", err)
	}
	if result["type"] != "object" {
		t.Errorf("type = %v, want object", result["type"])
	}
	if result["description"] != "test schema" {
		t.Errorf("description = %v, want 'test schema'", result["description"])
	}
	props := result["properties"].(map[string]any)
	if props["name"] == nil {
		t.Error("missing name property")
	}
	if props["age"] == nil {
		t.Error("missing age property")
	}
}

func TestLLM_ConvertSchemaNil(t *testing.T) {
	result, err := convertSchema(nil)
	if err != nil {
		t.Fatalf("convertSchema(nil) error: %v", err)
	}
	if result["type"] != "object" {
		t.Errorf("type = %v, want object", result["type"])
	}
}

func TestLLM_ConvertToFunctionParams(t *testing.T) {
	params := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"q": map[string]any{"type": "string"},
		},
	}
	result := convertToFunctionParams(params)
	if result == nil {
		t.Fatal("convertToFunctionParams returned nil")
	}
	if result["type"] != "object" {
		t.Errorf("type = %v, want object", result["type"])
	}
}

func TestLLM_ConvertToFunctionParamsNil(t *testing.T) {
	if got := convertToFunctionParams(nil); got != nil {
		t.Errorf("convertToFunctionParams(nil) = %v, want nil", got)
	}
}

func TestLLM_BuildUserMessage(t *testing.T) {
	msg := buildUserMessage([]string{"hello"}, nil)
	if msg == nil {
		t.Fatal("buildUserMessage returned nil")
	}
	if msg.OfUser == nil {
		t.Error("expected OfUser message")
	}
}

func TestLLM_BuildAssistantMessage(t *testing.T) {
	msg := buildAssistantMessage([]string{"response"}, nil)
	if msg == nil {
		t.Fatal("buildAssistantMessage returned nil")
	}
	if msg.OfAssistant == nil {
		t.Error("expected OfAssistant message")
	}
}

func TestLLM_BuildRoleMessage(t *testing.T) {
	tests := []struct {
		role string
		want string
	}{
		{"user", "user"},
		{"assistant", "assistant"},
		{"system", "system"},
		{"model", "assistant"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		msg := buildRoleMessage(tt.role, []string{"text"}, nil, nil)
		if tt.want == "" {
			if msg != nil {
				t.Errorf("buildRoleMessage(%q) returned non-nil, want nil", tt.role)
			}
			continue
		}
		if msg == nil {
			t.Errorf("buildRoleMessage(%q) returned nil", tt.role)
		}
	}
}

func TestLLM_OpenAIModelName(t *testing.T) {
	m := &OpenAIModel{modelName: "test-model"}
	if got := m.Name(); got != "test-model" {
		t.Errorf("Name() = %q, want %q", got, "test-model")
	}
}

func TestLLM_NormalizeToolCallID(t *testing.T) {
	m := &OpenAIModel{toolCallIDMap: make(map[string]string)}
	short := m.normalizeToolCallID("short")
	if short != "short" {
		t.Errorf("short ID unchanged: got %q", short)
	}
	long := "this_is_a_very_long_tool_call_id_that_exceeds_forty_characters_limit"
	normalized := m.normalizeToolCallID(long)
	if normalized == long {
		t.Error("long ID was not normalized")
	}
	if len(normalized) > maxToolCallIDLength {
		t.Errorf("normalized ID too long: %d > %d", len(normalized), maxToolCallIDLength)
	}
}
