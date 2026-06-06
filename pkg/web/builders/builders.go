package builders

import (
	"github.com/verdverm/gmd/pkg/web"
	cfprovider "github.com/verdverm/gmd/pkg/web/providers/cloudflare"
	exaprovider "github.com/verdverm/gmd/pkg/web/providers/exa"
	searxngprovider "github.com/verdverm/gmd/pkg/web/providers/searxng"
	tavilyprovider "github.com/verdverm/gmd/pkg/web/providers/tavily"
)

func DefaultSearchConstructors() map[string]web.ProviderConstructor {
	return map[string]web.ProviderConstructor{
		"exa": func(cfg web.ProviderConfig) (any, error) {
			return exaprovider.NewSearchAdapter(cfg)
		},
		"tavily": func(cfg web.ProviderConfig) (any, error) {
			return tavilyprovider.NewSearchClient(cfg)
		},
		"searxng": func(cfg web.ProviderConfig) (any, error) {
			return searxngprovider.NewSearchClient(cfg)
		},
	}
}

func DefaultBrowserConstructors() map[string]web.ProviderConstructor {
	return map[string]web.ProviderConstructor{
		"exa": func(cfg web.ProviderConfig) (any, error) {
			return exaprovider.NewBrowserAdapter(cfg)
		},
		"cloudflare": func(cfg web.ProviderConfig) (any, error) {
			return cfprovider.NewBrowserClient(cfg)
		},
		"local": func(cfg web.ProviderConfig) (any, error) {
			return nil, web.ErrProviderNotFound
		},
	}
}

func DefaultRegistry() *web.ProviderRegistry {
	return web.NewRegistry(DefaultSearchConstructors(), DefaultBrowserConstructors())
}
