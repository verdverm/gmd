package llm

import (
	"fmt"

	"github.com/verdverm/gmd/pkg/config"
)

func ResolveLLMConfig(cfg *config.Config) (*Client, error) {
	if len(cfg.LLM.Providers) > 0 {
		return resolveStructured(cfg)
	}
	return resolveLegacy(cfg), nil
}

func resolveStructured(cfg *config.Config) (*Client, error) {
	providers := make(map[string]ProviderConfig)
	for name, pc := range cfg.LLM.Providers {
		providers[name] = ProviderConfig{
			Name:     pc.Name,
			BaseURL:  pc.BaseURL,
			Auth:     pc.Auth,
			AuthData: pc.AuthData,
		}
	}

	profileName := cfg.LLM.Profile
	if profileName == "" {
		profileName = "default"
	}

	profileCfg, ok := cfg.LLM.Profiles[profileName]
	if !ok {
		return nil, fmt.Errorf("profile %q not found in config", profileName)
	}

	profile := Profile{}

	setRole := func(target *RoleConfig, rc *config.LLMRoleConfig) {
		if rc == nil {
			return
		}
		target.Provider = rc.Provider
		target.Model = rc.Model
	}

	setRole(&profile.Embedding, profileCfg.Embedding)
	setRole(&profile.Expansion, profileCfg.Expansion)
	setRole(&profile.Rerank, profileCfg.Rerank)
	setRole(&profile.Summarizing, profileCfg.Summarizing)
	setRole(&profile.GeneralBig, profileCfg.GeneralBig)
	setRole(&profile.GeneralMid, profileCfg.GeneralMid)
	setRole(&profile.GeneralSmall, profileCfg.GeneralSmall)

	return BuildAllClients(providers, profile)
}

func resolveLegacy(cfg *config.Config) *Client {
	return New(ConfigFromProject(cfg))
}
