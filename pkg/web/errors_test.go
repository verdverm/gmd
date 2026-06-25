package web

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrors_SentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrNotSupported", ErrNotSupported, "gmd/web: operation not supported by provider"},
		{"ErrProviderNotFound", ErrProviderNotFound, "gmd/web: provider not found in registry"},
		{"ErrProviderNotConfigured", ErrProviderNotConfigured, "gmd/web: provider referenced by group but not configured"},
		{"ErrBrowserNotAvailable", ErrBrowserNotAvailable, "gmd/web: browser not available on this machine"},
		{"ErrAuthMissing", ErrAuthMissing, "gmd/web: required credentials not set — check env vars"},
		{"ErrAuthFailed", ErrAuthFailed, "gmd/web: authentication failed"},
		{"ErrRateLimited", ErrRateLimited, "gmd/web: rate limited by provider"},
		{"ErrTimeout", ErrTimeout, "gmd/web: request timed out"},
		{"ErrSSRFBlocked", ErrSSRFBlocked, "gmd/web: request blocked — private/internal IP"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.msg {
				t.Errorf("expected %q, got %q", tt.msg, tt.err.Error())
			}
		})
	}
}

func TestErrors_ProviderError(t *testing.T) {
	cases := []struct {
		name     string
		pe       *ProviderError
		expected string
	}{
		{
			name:     "with detail",
			pe:       &ProviderError{Provider: "exa", Err: ErrRateLimited, Detail: "search"},
			expected: "gmd/web: exa: search: gmd/web: rate limited by provider",
		},
		{
			name:     "without detail",
			pe:       &ProviderError{Provider: "tavily", Err: ErrAuthFailed},
			expected: "gmd/web: tavily: gmd/web: authentication failed",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.pe.Error() != c.expected {
				t.Errorf("expected %q, got %q", c.expected, c.pe.Error())
			}
		})
	}
}

func TestErrors_ProviderErrorUnwrap(t *testing.T) {
	pe := &ProviderError{Provider: "exa", Err: ErrRateLimited}
	if !errors.Is(pe, ErrRateLimited) {
		t.Error("expected errors.Is(pe, ErrRateLimited) to be true")
	}
	if errors.Is(pe, ErrAuthFailed) {
		t.Error("expected errors.Is(pe, ErrAuthFailed) to be false")
	}
}

func TestErrors_WrapProviderError(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		if err := WrapProviderError("exa", "search", nil); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("wraps error", func(t *testing.T) {
		cause := fmt.Errorf("connection refused")
		err := WrapProviderError("tavily", "failed", cause)
		pe, ok := err.(*ProviderError)
		if !ok {
			t.Fatalf("expected *ProviderError, got %T", err)
		}
		if pe.Provider != "tavily" || pe.Detail != "failed" || pe.Err != cause {
			t.Errorf("unexpected fields: provider=%s detail=%s err=%v", pe.Provider, pe.Detail, pe.Err)
		}
	})
}
