package web

import (
	"testing"
)

func TestRegistry_NewRegistry(t *testing.T) {
	search := map[string]ProviderConstructor{
		"exa": func(cfg ProviderConfig) (any, error) { return "exa", nil },
	}
	browser := map[string]ProviderConstructor{
		"cloudflare": func(cfg ProviderConfig) (any, error) { return "cf", nil },
	}

	r := NewRegistry(search, browser)
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if len(r.search) != 1 || len(r.browser) != 1 {
		t.Errorf("expected 1 search and 1 browser constructor")
	}
}

func TestRegistry_Resolve(t *testing.T) {
	r := NewRegistry(
		map[string]ProviderConstructor{
			"exa":    func(cfg ProviderConfig) (any, error) { return "exa-search", nil },
			"tavily": func(cfg ProviderConfig) (any, error) { return "tavily-search", nil },
		},
		map[string]ProviderConstructor{
			"exa":        func(cfg ProviderConfig) (any, error) { return "exa-browser", nil },
			"cloudflare": func(cfg ProviderConfig) (any, error) { return "cf-browser", nil },
		},
	)

	t.Run("resolve known search provider", func(t *testing.T) {
		v, err := r.Resolve("search", "exa", ProviderConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "exa-search" {
			t.Errorf("expected exa-search, got %v", v)
		}
	})

	t.Run("resolve known browser provider", func(t *testing.T) {
		v, err := r.Resolve("browser", "cloudflare", ProviderConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "cf-browser" {
			t.Errorf("expected cf-browser, got %v", v)
		}
	})

	t.Run("resolve unknown provider", func(t *testing.T) {
		_, err := r.Resolve("search", "unknown", ProviderConfig{})
		if err == nil {
			t.Fatal("expected error for unknown provider")
		}
	})

	t.Run("resolve unknown role", func(t *testing.T) {
		_, err := r.Resolve("invalid", "exa", ProviderConfig{})
		if err == nil {
			t.Fatal("expected error for unknown role")
		}
	})

	t.Run("resolve browser for search-only provider", func(t *testing.T) {
		_, err := r.Resolve("browser", "tavily", ProviderConfig{})
		if err == nil {
			t.Fatal("expected error for provider not registered in browser role")
		}
	})
}

func TestRegistry_ValidateName(t *testing.T) {
	r := NewRegistry(
		map[string]ProviderConstructor{"exa": nil},
		map[string]ProviderConstructor{"cloudflare": nil},
	)

	t.Run("valid search provider", func(t *testing.T) {
		if err := r.ValidateName("search", "exa"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("valid browser provider", func(t *testing.T) {
		if err := r.ValidateName("browser", "cloudflare"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("unknown search provider", func(t *testing.T) {
		if err := r.ValidateName("search", "unknown"); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("unknown role", func(t *testing.T) {
		if err := r.ValidateName("invalid", "exa"); err == nil {
			t.Error("expected error")
		}
	})
}

func TestRegistry_NilMaps(t *testing.T) {
	r := NewRegistry(nil, nil)
	if r == nil {
		t.Fatal("expected non-nil registry with nil maps")
	}
	_, err := r.Resolve("search", "exa", ProviderConfig{})
	if err == nil {
		t.Fatal("expected error when resolving from nil map")
	}
}
