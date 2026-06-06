package web

import "fmt"

type ProviderConstructor func(cfg ProviderConfig) (any, error)

type ProviderRegistry struct {
	search  map[string]ProviderConstructor
	browser map[string]ProviderConstructor
}

func NewRegistry(searchCtors, browserCtors map[string]ProviderConstructor) *ProviderRegistry {
	return &ProviderRegistry{
		search:  searchCtors,
		browser: browserCtors,
	}
}

func (r *ProviderRegistry) Resolve(role, name string, cfg ProviderConfig) (any, error) {
	var m map[string]ProviderConstructor
	switch role {
	case "search":
		m = r.search
	case "browser":
		m = r.browser
	default:
		return nil, &ProviderError{Provider: name, Err: ErrProviderNotFound, Detail: "unknown role: " + role}
	}

	ctor, ok := m[name]
	if !ok {
		return nil, &ProviderError{Provider: name, Err: ErrProviderNotFound, Detail: "not a known " + role + " provider"}
	}

	return ctor(cfg)
}

func (r *ProviderRegistry) ValidateName(role, name string) error {
	var m map[string]ProviderConstructor
	switch role {
	case "search":
		m = r.search
	case "browser":
		m = r.browser
	default:
		return fmt.Errorf("unknown role: %s", role)
	}

	if _, ok := m[name]; !ok {
		return fmt.Errorf("unknown %s provider: %s", role, name)
	}
	return nil
}
