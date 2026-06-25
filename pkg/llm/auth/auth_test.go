package auth

import (
	"testing"
)

func TestAuth_DefaultBaseURL(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{"openai", "https://api.openai.com/v1"},
		{"anthropic", "https://api.anthropic.com/v1"},
		{"vertex", ""},
		{"opencode", ""},
		{"custom", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := DefaultBaseURL(tt.provider)
		if got != tt.want {
			t.Errorf("DefaultBaseURL(%q) = %q, want %q", tt.provider, got, tt.want)
		}
	}
}

func TestBuildHTTPClient_None(t *testing.T) {
	hc, err := BuildHTTPClient(Config{Method: AuthNone})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hc != nil {
		t.Error("expected nil client for none auth")
	}
}

func TestBuildHTTPClient_APIKey(t *testing.T) {
	hc, err := BuildHTTPClient(Config{Method: AuthAPIKey, APIKey: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hc != nil {
		t.Error("expected nil client for apikey auth")
	}
}

func TestBuildHTTPClient_UnknownMethod(t *testing.T) {
	_, err := BuildHTTPClient(Config{Method: "invalid"})
	if err == nil {
		t.Fatal("expected error for unknown method")
	}
}

func TestAuth_MethodConstants(t *testing.T) {
	if AuthNone != "none" {
		t.Error("AuthNone mismatch")
	}
	if AuthAPIKey != "apikey" {
		t.Error("AuthAPIKey mismatch")
	}
	if AuthServiceAccount != "service-account" {
		t.Error("AuthServiceAccount mismatch")
	}
}
