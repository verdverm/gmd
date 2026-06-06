# Multi-LLM Providers & Profiles

## Summary

Currently, gmd uses a flat `LLMConfig` with 7 role-specific fields (model, base_url, api_key
per role) — all backed by the `openai-go` SDK. We need to:

1. Support multiple LLM **providers** (OpenAI, Anthropic, Vertex, vLLM, opencode proxy)
2. Support multiple **auth methods** (none, apikey, service-account)
3. Turn each role into a structured object referencing a provider + model
4. Support named **profiles** (presets) for quick configuration

## Key Insight: No Per-Provider Go Packages Needed

Both **Anthropic** and **Google Vertex AI** provide official OpenAI-compatible API
endpoints. This means we can use the existing `openai-go` SDK (v3) for all providers
— just with different base URLs and auth headers. No separate provider Go packages,
no new SDK dependencies. The work becomes primarily config + an auth layer.

| Provider | OpenAI-compat endpoint | Auth |
|---|---|---|
| vLLM / Ollama / local | `http://host:port/v1` | none or apikey |
| OpenAI | `https://api.openai.com/v1` | apikey |
| Anthropic | `https://api.anthropic.com/v1` | apikey (x-api-key via openai compat) |
| Vertex AI | `https://{loc}-aiplatform.googleapis.com/v1beta1/projects/{proj}/locations/{loc}/endpoints/openapi` | service-account (GCP OAuth2) |
| opencode proxy | TBD | apikey |

References:
- Anthropic OpenAI compat: https://docs.anthropic.com/en/api/openai-sdk
- Vertex OpenAI compat: https://cloud.google.com/vertex-ai/generative-ai/docs/start/openai

## Current Architecture

### Config (`pkg/config/embeds/types.cue:3-34`, `pkg/config/config.go:397-427`)

Flat `LLMConfig` — 7 roles × (model, base_url, api_key) = 21 fields. API keys resolved
from env vars (`OPENAI_API_KEY`, `GMD_EMBEDDING_API_KEY`, etc.) in `config.go:630-637`.

### Client (`pkg/llm/client.go`)

Monolithic `Client` struct with 7 `openai.Client` instances (one per role).
`newOpenAIClient()` hardcodes `openai.NewClient()` — no provider abstraction.
Operations: `Embed`, `EmbedBatch`, `Chat`, `Summarize`, `GeneralBigChat`,
`GeneralMidChat`, `GeneralSmallChat`, `Rerank` (raw POST to `/rerank`),
`CheckEndpoint`/`CheckAll` (via `Models.List()`).

### Callers

| Caller | Roles used |
|---|---|
| `pkg/search/pipeline.go` | embed, expansion chat, rerank |
| `pkg/indexer/indexer.go` | batch embed |
| `pkg/wiki/agent.go` | chat (general big) |
| `cmd/gmd/doctor.go` | CheckAll health checks |
| `cmd/gmd/web_agent.go` | chat |
| `cmd/gmd/wiki_*.go` | chat (various) |
| `cmd/gmd/update.go` | embed |

## Design

### 1. Auth Methods (orthogonal to provider)

| Auth | Description | Implementation |
|---|---|---|
| `none` | No authentication (local vLLM/Ollama) | No headers |
| `apikey` | Static API key as Bearer token | `option.WithAPIKey(key)` |
| `service-account` | GCP OAuth2 token with auto-refresh | Custom `http.RoundTripper` wrapping `golang.org/x/oauth2/google` token source |

Auth is resolved per-provider entry in config. The LLM client builder picks
the right `option.RequestOption` (API key) or `http.Client` (OAuth2 token
transport) based on the auth method.

### 2. Provider Config (CUE + Go)

A provider is a named endpoint with auth configuration — no runtime code per provider.

```cue
// LLMProviderConfig defines a named LLM service endpoint
LLMProviderConfig: {
    provider:       "openai" | "anthropic" | "vertex" | "opencode" | string
    base_url?:      string    // override default for this provider type
    auth:           "none" | "apikey" | "service-account" | *"apikey"

    // For service-account auth (vertex):
    project_id?:    string    // GCP project ID
    location?:      string    // e.g. "us-central1"

    // Capability hints (informational, not enforced at wire level):
    features?: {
        embed?:  bool | *true
        chat?:   bool | *true
        rerank?: bool | *false
    }
}
```

Provider type determines the default base URL and which env var provides the API key:

| Provider type | Default base URL | API key env var |
|---|---|---|
| `openai` | `https://api.openai.com/v1` | `OPENAI_API_KEY` |
| `anthropic` | `https://api.anthropic.com/v1` | `ANTHROPIC_API_KEY` |
| `vertex` | (constructed from project_id + location) | `GOOGLE_APPLICATION_CREDENTIALS` |
| `opencode` | (from opencode config) | `OPENCODE_API_KEY` |
| Custom | (must specify base_url) | `GMD_LLM_API_KEY` or provider-specific |

### 3. Role Config & Profiles

```cue
// LLMRoleConfig maps a role to a provider + model
LLMRoleConfig: {
    provider?:  string   // name from LLMConfig.providers
    model?:     string
}

// LLMProfile bundles all roles into a named preset
LLMProfile: {
    embedding?:       LLMRoleConfig
    expansion?:       LLMRoleConfig
    rerank?:          LLMRoleConfig
    summarizing?:     LLMRoleConfig
    general_big?:     LLMRoleConfig
    general_mid?:     LLMRoleConfig
    general_small?:   LLMRoleConfig
}

// New LLMConfig
LLMConfig: {
    providers?:  [string]: LLMProviderConfig
    profile?:    string | *"default"
    profiles?:   [string]: LLMProfile
}
```

**Example config (mixed local + cloud):**

```cue
llm: {
    providers: {
        vllm8000: {
            provider: "openai"
            base_url: "http://192.168.4.31:8000/v1"
            auth: "none"
            features: { chat: true }
        }
        vllm8001: {
            provider: "openai"
            base_url: "http://192.168.4.31:8001/v1"
            auth: "none"
            features: { embed: true, chat: false, rerank: false }
        }
        vllm8002: {
            provider: "openai"
            base_url: "http://192.168.4.31:8002/v1"
            auth: "none"
            features: { embed: false, chat: true, rerank: false }
        }
        anthro: {
            provider: "anthropic"
            auth: "apikey"
            features: { embed: false, rerank: false }
        }
    }
    profiles: {
        default: {
            embedding:     { provider: "vllm8001", model: "google/embeddinggemma-300m" }
            expansion:     { provider: "vllm8002", model: "Qwen/Qwen3-1.7B" }
            rerank:        { provider: "vllm8002", model: "Qwen/Qwen3-Reranker-0.6B" }
            summarizing:   { provider: "vllm8000", model: "Qwen/Qwen3.6-27B-FP8" }
            general_big:   { provider: "vllm8000", model: "Qwen/Qwen3.6-27B-FP8" }
            general_mid:   { provider: "vllm8000", model: "Qwen/Qwen3.6-27B-FP8" }
            general_small: { provider: "vllm8000", model: "Qwen/Qwen3.6-27B-FP8" }
        }
        hybrid: {
            embedding:     { provider: "vllm8001", model: "google/embeddinggemma-300m" }
            expansion:     { provider: "anthro", model: "claude-3-5-haiku-20241022" }
            summarizing:   { provider: "anthro", model: "claude-3-5-sonnet-20241022" }
        }
    }
}
```

### 4. Client Architecture

The `Client` struct stays mostly the same shape but providers are composed
from config at construction time:

```go
// pkg/llm/client.go

type Client struct {
    embedder    *roleClient    // openai.Client + model name + auth
    expander    *roleClient
    reranker    *roleClient
    summarizer  *roleClient
    generalBig   *roleClient
    generalMid   *roleClient
    generalSmall *roleClient

    // Providers indexed by name (built once, shared across roles)
    providers map[string]*openai.Client
}

type roleClient struct {
    client *openai.Client
    model  string
    url    string
}

type Config struct {
    // Active profile with resolved role→(client, model) mappings.
    // Built from config during New().
    Embedding        RoleConfig
    Expansion        RoleConfig
    Rerank           RoleConfig
    Summarizing      RoleConfig
    GeneralBig       RoleConfig
    GeneralMid       RoleConfig
    GeneralSmall     RoleConfig

    // Raw provider configs for health checks
    Providers map[string]ProviderConfig
}

type RoleConfig struct {
    Client *openai.Client
    Model  string
    URL    string
}

type ProviderConfig struct {
    Name     string
    BaseURL  string
    Auth     string
    AuthData map[string]string // key, project_id, location, etc.
}
```

### 5. Auth Implementation (`pkg/llm/auth/`)

```go
// pkg/llm/auth/auth.go

type Method string

const (
    AuthNone          Method = "none"
    AuthAPIKey        Method = "apikey"
    AuthServiceAccount Method = "service-account"
)

type Config struct {
    Method          Method
    APIKey          string
    // Service account fields:
    ProjectID       string
    Location        string
    CredentialsFile string // path to JSON key file, or "" for ADC
}

// BuildHTTPClient returns an *http.Client that handles auth transparently.
// For "apikey": returns nil (use option.WithAPIKey directly).
// For "service-account": returns an http.Client with GCP OAuth2 transport.
// For "none": returns nil (no auth).
func BuildHTTPClient(cfg Config) (*http.Client, error)

// DefaultBaseURL returns the default base URL for a provider type.
func DefaultBaseURL(provider string) string
```

For `service-account`, the implementation wraps `golang.org/x/oauth2/google`:

```go
import "golang.org/x/oauth2/google"

func gcpTokenClient(ctx context.Context, credentialsFile string) (*http.Client, error) {
    var tokenSource oauth2.TokenSource
    if credentialsFile != "" {
        data, err := os.ReadFile(credentialsFile)
        if err != nil {
            return nil, err
        }
        creds, err := google.CredentialsFromJSON(ctx, data, "https://www.googleapis.com/auth/cloud-platform")
        if err != nil {
            return nil, err
        }
        tokenSource = creds.TokenSource
    } else {
        // Application Default Credentials (metadata server / env var)
        tokenSource, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
        if err != nil {
            return nil, err
        }
    }
    return oauth2.NewClient(ctx, tokenSource), nil
}
```

### 6. Provider Client Builder

```go
// pkg/llm/builder.go

func buildClient(provider ProviderConfig) (*openai.Client, error) {
    baseURL := provider.BaseURL
    if baseURL == "" {
        baseURL = auth.DefaultBaseURL(provider.Name)
    }

    opts := []option.RequestOption{option.WithBaseURL(baseURL)}

    switch provider.Auth {
    case "apikey":
        key := provider.AuthData["api_key"]
        // key resolved from env earlier in config loading
        opts = append(opts, option.WithAPIKey(key))
    case "service-account":
        httpClient, err := auth.BuildHTTPClient(auth.Config{
            Method:          auth.AuthServiceAccount,
            ProjectID:       provider.AuthData["project_id"],
            Location:        provider.AuthData["location"],
            CredentialsFile: provider.AuthData["credentials_file"],
        })
        if err != nil {
            return nil, fmt.Errorf("building service-account client: %w", err)
        }
        opts = append(opts, option.WithHTTPClient(httpClient))
    case "none":
        // no auth
    }

    return openai.NewClient(opts...), nil
}
```

### 7. Config Resolution (Config → llm.Config)

The `ConfigFromProject` function (currently in `pkg/llm/client.go:91`) becomes the
translation layer:

1. If structured `providers` + `profiles` exist, build `llm.Config` from active profile
2. If legacy flat fields exist (and no structured config), synthesize a default
   provider for each unique base_url + a "default" profile
3. API keys resolved from env vars based on provider type and auth method

### 8. Rerank Handling

Rerank is the one operation that doesn't use the OpenAI SDK directly — it's a
raw POST to `/rerank`. This endpoint is vLLM-specific. For providers that
support it (`features.rerank: true`), we keep the current raw POST behavior.
For providers that don't, `Rerank()` returns an error. Future: add Cohere
rerank support if needed (Cohere has a different API).

## Backward Compatibility

Legacy flat fields continue to work. The loader translates them:

```
if legacy flat fields present AND no providers/profiles configured:
    → group by unique base_url → create a provider per base_url
    → create a "default" profile mapping each role to its provider + model
    → no deprecation warning if ONLY legacy fields are used
    → deprecation warning if BOTH legacy and structured are present
```

## Implementation Plan

### Phase 1: Auth package (`pkg/llm/auth/`)

- `auth.go`: `Method` type, `Config` struct, `BuildHTTPClient()`, `DefaultBaseURL()`
- `auth_test.go`: unit tests with mock HTTP server for token refresh
- Only new external dep: `golang.org/x/oauth2` (already transitive via openai-go)

### Phase 2: Provider builder (`pkg/llm/builder.go`)

- `buildClient(cfg ProviderConfig) (*openai.Client, error)` — creates
  `openai.Client` with correct base URL + auth from provider config
- `buildReranker(client *openai.Client, model string)` — wraps raw POST for vLLM
- `buildAllClients(providers map[string]ProviderConfig, profile Profile) (*Client, error)`

### Phase 3: Refactor `Client` struct (`pkg/llm/client.go`)

- Replace 7 `openai.Client` fields with 7 `*roleClient` fields (client + model + url)
- Simplify constructor: takes resolved `Config` struct, builds clients via builder
- Keep public methods (`Embed`, `Chat`, `Rerank`, etc.) unchanged
- Remove `ConfigFromProject` — moves to separate config resolution function
- Remove per-role `clientFor*()` helpers — use `roleClient.client` directly

### Phase 4: New CUE schema + Go config structs

- Define `LLMProviderConfig`, `LLMRoleConfig`, `LLMProfile` in `types.cue`
- Update `newLLMConfig` CUE type
- Add Go structs in `pkg/config/config.go`: `LLMProviderConfig`, `LLMRoleConfig`,
  `LLMProfileConfig`, new fields on `LLMConfig`
- Add env var resolution for provider API keys (keyed by provider type, not role):
  `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GOOGLE_APPLICATION_CREDENTIALS`, etc.
- Backward-compat translation from legacy flat fields in `Config.Load()`

### Phase 5: Config resolution (`pkg/llm/config.go` or `pkg/config/llm_resolve.go`)

- `ResolveLLMConfig(cfg *config.Config) (llm.Config, error)` — translates
  resolved CUE config into runtime `llm.Config` with built clients
- Profile resolution: active profile → role configs, with fallback chain
- Role resolution: per-role → provider entry → built `openai.Client`

### Phase 6: Update callers

- `cmd/gmd/llm_helper.go`: `llmConfigFromConfig` → calls `ResolveLLMConfig`,
  error handling for resolution failures
- All `llm.New(llmConfigFromConfig(cfg))` call sites (~12) get error handling
  (construction can now fail due to auth config issues)
- `pkg/search/pipeline.go`, `pkg/indexer/indexer.go`, `pkg/wiki/agent.go`,
  `cmd/gmd/web_agent.go`, `cmd/gmd/doctor.go`, `cmd/gmd/wiki_*.go`

### Phase 7: Profiles CLI

- `gmd llm status` — health check all providers (uses `CheckEndpoint`)
- `gmd llm providers` — list configured providers with auth info (keys masked)
- `gmd llm profiles` — list profiles
- `gmd llm profile show <name>` — show role→provider mappings
- `gmd llm test <provider>` — quick chat test against a provider

### Phase 8: Legacy deprecation

- Warning on `stderr` if both legacy and structured config present
- Update `AGENTS.md` and `docs/` with new config format
- Eventually remove legacy field support in a future major version

## Dependencies

Only one new dependency:

| Module | Purpose |
|---|---|
| `golang.org/x/oauth2` | GCP OAuth2 token source for service-account auth |

Already a transitive dependency of `openai-go` v3.

## Risks & Open Questions

- **Rerank**: vLLM's `/rerank` is non-standard. Cohere offers a rerank API with
  different request/response shapes. For now, rerank is gated behind
  `features.rerank: true` and only works with vLLM/OpenAI-compat providers.
  Future: add a Reranker interface with Cohere/other implementations if needed.
- **Embedding model names**: Vertex uses different model IDs than OpenAI
  (`textembedding-gecko` vs `text-embedding-3-small`). The config just passes
  the model name through — the user must specify the correct name for their
  provider.
- **Anthropic compat limitations**: No structured output, no seed, no logprobs,
  `n` must be 1. These aren't used by gmd today, so no impact.
- **Vertex endpoint URL construction**: The pattern
  `https://{loc}-aiplatform.googleapis.com/v1beta1/projects/{proj}/locations/{loc}/endpoints/openapi`
  must be verified against the actual Vertex OpenAI-compat docs. If the format
  changes, providers can override with explicit `base_url`.
- **opencode provider**: Requires opencode API details (endpoint URL, auth
  header format). Deferred until opencode's LLM proxy API stabilizes.
