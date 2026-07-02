package llm

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm/auth"
)

// Standard role names. Consumers reference these. Config can define any number
// of additional roles; these are the conventional defaults.
const (
	RoleEmbedding    = "embedding"
	RoleExpansion    = "expansion"
	RoleRerank       = "rerank"
	RoleSummarizing  = "summarizing"
	RoleGeneralBig   = "general_big"
	RoleGeneralMid   = "general_mid"
	RoleGeneralSmall = "general_small"
)

// ProviderHealth holds the health check result for one LLM role binding.
type ProviderHealth struct {
	Label    string   `json:"label"`
	URL      string   `json:"url"`
	Model    string   `json:"model"`
	Provider string   `json:"provider,omitempty"`
	OK       bool     `json:"ok"`
	Models   []string `json:"models,omitempty"`
	Err      string   `json:"err,omitempty"`
}

// Registry holds resolved LLM clients indexed by role, plus the shared
// embedder and reranker. Built once from config at startup. Read-only after
// construction; safe for concurrent use.
type Registry struct {
	models         map[string]ChatModel
	embed          Embedder
	rerank         Reranker
	providerModels map[string]*OpenAIModel
	closers        []func() error
}

// Model returns the ChatModel for the given role, or nil if unset.
func (r *Registry) Model(role string) ChatModel {
	return r.models[role]
}

// Embedder returns the shared Embedder, or nil if embedding is unset.
func (r *Registry) Embedder() Embedder {
	return r.embed
}

// Reranker returns the shared Reranker, or nil if rerank is unset.
func (r *Registry) Reranker() Reranker {
	return r.rerank
}

// Roles returns the sorted list of configured role names.
func (r *Registry) Roles() []string {
	roles := make([]string, 0, len(r.models))
	for role := range r.models {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}

// CheckProviders runs health checks against roles referenced by the active profile.
// Each entry corresponds to one role binding (provider + optional model override).
func (r *Registry) CheckProviders(ctx context.Context) []ProviderHealth {
	var results []ProviderHealth
	providerModels := make(map[string][]string)
	for _, m := range r.providerModels {
		if m != nil {
			name := m.Name()
			if name != "" {
				providerModels[name] = nil
			}
		}
	}
	for _, m := range r.providerModels {
		if m == nil {
			continue
		}
		key := m.Name()
		if key == "" {
			continue
		}
		if _, exists := providerModels[key]; !exists {
			models, _ := m.ListModels(ctx)
			providerModels[key] = models
		}
	}
	for name, m := range r.models {
		h := ProviderHealth{
			Label: name,
			OK:    true,
		}
		if om, ok := m.(*OpenAIModel); ok {
			h.Model = om.Name()
			pname := ""
			for pn, pm := range r.providerModels {
				if pm == om || (pm != nil && om.client == pm.client && om.modelName == pm.modelName) {
					pname = pn
					break
				}
			}
			if pname == "" {
				for pn, pm := range r.providerModels {
					if pm != nil && om.client == pm.client {
						pname = pn
						break
					}
				}
			}
			h.Provider = pname
			if h.Model != "" {
				allModels := providerModels[pname]
				if allModels == nil {
					allModels, _ = om.ListModels(context.Background())
					providerModels[pname] = allModels
				}
				found := false
				for _, am := range allModels {
					if am == h.Model {
						found = true
						break
					}
				}
				if !found {
					h.Err = fmt.Sprintf("model %q not in provider's available models (%d total)", h.Model, len(allModels))
				}
			}
		}
		results = append(results, h)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Label < results[j].Label })
	return results
}

// Close shuts down OAuth2-backed HTTP clients and other resources.
func (r *Registry) Close() error {
	var firstErr error
	for _, closer := range r.closers {
		if err := closer(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// RegistryOption is a functional option for NewRegistry.
type RegistryOption func(*registryConfig)

type registryConfig struct {
	providerTransports map[string]*http.Client
}

// WithProviderTransport injects a custom HTTP client for a specific provider.
// Used by tests to inject tape-replay transports.
func WithProviderTransport(provider string, client *http.Client) RegistryOption {
	return func(rc *registryConfig) {
		if rc.providerTransports == nil {
			rc.providerTransports = make(map[string]*http.Client)
		}
		rc.providerTransports[provider] = client
	}
}

// NewRegistry builds a Registry from the resolved gmd config.
func NewRegistry(ctx context.Context, cfg *config.Config, opts ...RegistryOption) (*Registry, error) {
	rc := &registryConfig{}
	for _, opt := range opts {
		opt(rc)
	}

	profileName := cfg.LLM.Profile
	if profileName == "" {
		profileName = "default"
	}
	profileCfg, ok := cfg.LLM.Profiles[profileName]
	if !ok {
		return nil, ErrRoleUnset
	}

	// Cache openai clients per provider name to share connection pools.
	type cachedProvider struct {
		model  *OpenAIModel
		embed  Embedder
		rerank Reranker
	}
	providerCache := make(map[string]*cachedProvider)
	providerModels := make(map[string]*OpenAIModel)

	getProvider := func(providerName string) (*cachedProvider, error) {
		if c, ok := providerCache[providerName]; ok {
			return c, nil
		}
		pc, ok := cfg.LLM.Providers[providerName]
		if !ok {
			return nil, ErrProviderNotConfigured
		}

		baseURL := pc.BaseURL
		if baseURL == "" {
			baseURL = auth.DefaultBaseURL(pc.Name)
		}

		apiKey := ""
		httpClient, err := buildHTTPClient(pc)
		if err != nil {
			return nil, err
		}

		// API key from auth data or top-level config.
		if pc.Auth == string(auth.AuthAPIKey) {
			if pc.AuthData != nil {
				apiKey = pc.AuthData["api_key"]
			}
			if apiKey == "" {
				apiKey = cfg.LLM.APIKey
			}
		}

		// Apply test-time transport override.
		if rc.providerTransports != nil {
			if tc, ok := rc.providerTransports[providerName]; ok {
				httpClient = tc
			}
		}

		oci := OpenAIConfig{
			APIKey:     apiKey,
			BaseURL:    baseURL,
			HTTPClient: httpClient,
		}

		cp := &cachedProvider{}
		cp.model = NewOpenAIModel(oci)
		cp.embed = NewEmbedder(oci)
		cp.rerank = NewReranker(oci)
		providerCache[providerName] = cp
		providerModels[providerName] = cp.model
		return cp, nil
	}

	reg := &Registry{
		models:         make(map[string]ChatModel),
		providerModels: providerModels,
	}

	// Build role -> model mappings from the profile.
	type roleBinding struct {
		name    string
		roleCfg *config.LLMRoleConfig
	}
	roles := []roleBinding{
		{RoleEmbedding, profileCfg.Embedding},
		{RoleExpansion, profileCfg.Expansion},
		{RoleRerank, profileCfg.Rerank},
		{RoleSummarizing, profileCfg.Summarizing},
		{RoleGeneralBig, profileCfg.GeneralBig},
		{RoleGeneralMid, profileCfg.GeneralMid},
		{RoleGeneralSmall, profileCfg.GeneralSmall},
	}

	for _, rb := range roles {
		if rb.roleCfg == nil || (rb.roleCfg.Provider == "" && rb.roleCfg.Model == "") {
			continue
		}
		if rb.roleCfg.Provider == "" {
			return nil, ErrProviderNotConfigured
		}
		cp, err := getProvider(rb.roleCfg.Provider)
		if err != nil {
			return nil, err
		}
		if rb.roleCfg.Model != "" {
			m := NewOpenAIModelWithModel(cp.model, rb.roleCfg.Model)
			reg.models[rb.name] = m
		} else {
			reg.models[rb.name] = cp.model
		}
	}

	// Set embedder from the embedding role's provider.
	if rc := profileCfg.Embedding; rc != nil && rc.Provider != "" {
		cp, err := getProvider(rc.Provider)
		if err != nil {
			return nil, err
		}
		reg.embed = cp.embed
	}

	// Set reranker from the rerank role's provider.
	if rc := profileCfg.Rerank; rc != nil && rc.Provider != "" {
		cp, err := getProvider(rc.Provider)
		if err != nil {
			return nil, err
		}
		reg.rerank = cp.rerank
	}

	return reg, nil
}

// NewOpenAIModelWithModel creates an OpenAIModel with a different model name
// but the same underlying client configuration.
func NewOpenAIModelWithModel(base *OpenAIModel, modelName string) *OpenAIModel {
	return &OpenAIModel{
		client:        base.client,
		modelName:     modelName,
		toolCallIDMap: make(map[string]string),
	}
}

func buildHTTPClient(pc config.LLMProviderConfig) (*http.Client, error) {
	cfg := auth.Config{
		Method:    auth.Method(pc.Auth),
		ProjectID: pc.ProjectID,
		Location:  pc.Location,
	}
	if pc.AuthData != nil {
		cfg.CredentialsFile = pc.AuthData["credentials_file"]
	}
	return auth.BuildHTTPClient(cfg)
}
