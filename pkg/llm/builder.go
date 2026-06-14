package llm

import (
	"fmt"
	"net/http"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/verdverm/gmd/pkg/llm/auth"
)

type ProviderConfig struct {
	Name       string
	BaseURL    string
	Auth       string
	AuthData   map[string]string
	HTTPClient *http.Client
}

type RoleConfig struct {
	Provider string
	Client   *openai.Client
	Model    string
	URL      string
}

type Profile struct {
	Embedding    RoleConfig
	Expansion    RoleConfig
	Rerank       RoleConfig
	Summarizing  RoleConfig
	GeneralBig   RoleConfig
	GeneralMid   RoleConfig
	GeneralSmall RoleConfig
}

func BuildClient(provider ProviderConfig) (*openai.Client, error) {
	baseURL := provider.BaseURL
	if baseURL == "" {
		baseURL = auth.DefaultBaseURL(provider.Name)
	}
	if baseURL == "" && provider.Name == "vertex" {
		return nil, fmt.Errorf("vertex provider requires explicit base_url")
	}

	opts := []option.RequestOption{option.WithBaseURL(baseURL)}

	switch provider.Auth {
	case "apikey":
		key := provider.AuthData["api_key"]
		if key != "" {
			opts = append(opts, option.WithAPIKey(key))
		}
	case "service-account":
		httpClient, err := auth.BuildHTTPClient(auth.Config{
			Method:          auth.AuthServiceAccount,
			ProjectID:       provider.AuthData["project_id"],
			Location:        provider.AuthData["location"],
			CredentialsFile: provider.AuthData["credentials_file"],
		})
		if err != nil {
			return nil, fmt.Errorf("building service-account client for %q: %w", provider.Name, err)
		}
		opts = append(opts, option.WithHTTPClient(httpClient))
	case "none":
	default:
		return nil, fmt.Errorf("unknown auth method %q for provider %q", provider.Auth, provider.Name)
	}

	client := openai.NewClient(opts...)

	if provider.HTTPClient != nil {
		client = openai.NewClient(append(opts, option.WithHTTPClient(provider.HTTPClient))...)
	}

	return &client, nil
}

func BuildAllClients(providers map[string]ProviderConfig, profile Profile) (*Client, error) {
	b := &clientBuilder{
		providers: providers,
		built:     make(map[string]*openai.Client),
	}

	c := &Client{
		providers: b.built,
	}

	if err := b.buildRole(&c.embedder, profile.Embedding); err != nil {
		return nil, fmt.Errorf("embedding role: %w", err)
	}
	if err := b.buildRole(&c.expander, profile.Expansion); err != nil {
		return nil, fmt.Errorf("expansion role: %w", err)
	}
	if err := b.buildRole(&c.reranker, profile.Rerank); err != nil {
		return nil, fmt.Errorf("rerank role: %w", err)
	}
	if err := b.buildRole(&c.summarizer, profile.Summarizing); err != nil {
		return nil, fmt.Errorf("summarizing role: %w", err)
	}
	if err := b.buildRole(&c.generalBig, profile.GeneralBig); err != nil {
		return nil, fmt.Errorf("general_big role: %w", err)
	}
	if err := b.buildRole(&c.generalMid, profile.GeneralMid); err != nil {
		return nil, fmt.Errorf("general_mid role: %w", err)
	}
	if err := b.buildRole(&c.generalSmall, profile.GeneralSmall); err != nil {
		return nil, fmt.Errorf("general_small role: %w", err)
	}

	return c, nil
}

type clientBuilder struct {
	providers map[string]ProviderConfig
	built     map[string]*openai.Client
}

func (b *clientBuilder) buildRole(target *roleClient, rc RoleConfig) error {
	if rc.Model == "" && rc.Provider == "" {
		return nil
	}
	if rc.Provider == "" {
		return fmt.Errorf("role has model but no provider")
	}
	client, err := b.getOrBuild(rc.Provider)
	if err != nil {
		return err
	}
	target.client = client
	target.model = rc.Model
	if pc, ok := b.providers[rc.Provider]; ok {
		target.url = pc.BaseURL
	}
	return nil
}

func (b *clientBuilder) getOrBuild(name string) (*openai.Client, error) {
	if c, ok := b.built[name]; ok {
		return c, nil
	}
	pc, ok := b.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not found in config", name)
	}
	c, err := BuildClient(pc)
	if err != nil {
		return nil, err
	}
	b.built[name] = c
	return c, nil
}
