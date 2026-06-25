# Rewrite pkg/llm on ADK

| | |
|---|---|
| Created | 2026-06-22 |
| Last Update | 2026-06-22 |
| Phase | Phases 1-5 implemented (2026-06-22); Phases 6-10 pending |
| Supersedes | `.design/llm-providers.md` (that design was implemented; this rewrites the result) |

## Context

`pkg/llm` is poorly written and needs a from-scratch rewrite. Three problems, per
`TASK.md`:

1. It must be built on **ADK** (`.extern/adk-go`), with the OpenAI client pattern from
   `~/hof/hof/lib/agent/models/openai.go` as a reference.
2. The `Client` "interface" is trash - it is a concrete struct, not a real interface, with
   leaky abstractions and hardcoded roles.
3. All consumers must be re-thought. There will be significant changes outside `pkg/llm`.

This document designs the rewrite. It is design-only: no implementation here.

## Goal

A new `pkg/llm` that:
- Implements ADK's `model.LLM` interface for chat/generation (the ADK-native path)
- Provides real Go interfaces for the three distinct LLM concerns: chat, embed, rerank
- Eliminates `*openai.Client` leakage from the package boundary
- Replaces 7 hardcoded role struct fields with a config-driven role registry
- Preserves the tape-replay test pattern (HTTP transport injection)
- Keeps the existing CUE providers/profiles config shape (it is sound; the runtime layer is what rots)
- Drops `LLMProviderFeatures` (code rot, no possible feature overlap)

And the agent migration:
- Rewrites `pkg/wiki/agent.go` from hand-rolled pipelines to ADK `llmagent` agents with
  tools (search, read, write, index)
- Rewrites `pkg/web/agent.go` from Go-orchestrated loops to ADK `llmagent` with
  `WebSearch`/`WebFetch` tools routing through the unified provider interfaces
- All tapes regenerated (major refactor)

## Summary of Findings

### Current `pkg/llm` (what we are replacing)

- `Client` is a **concrete struct** with 7 hardcoded `roleClient` fields (`embedder`,
  `expander`, `reranker`, `summarizer`, `generalBig`, `generalMid`, `generalSmall`) -
  `pkg/llm/client.go:20-30`. Not an interface.
- **Leaky:** `Client.ProviderClients() map[string]*openai.Client` and
  `Client.RoleClient(role) *openai.Client` hand raw openai-go objects to callers
  (`client.go:32-94`). `BuildClient` returns `*openai.Client` (`builder.go:38`). Two CLI
  commands (`cmd/gmd/llm_status.go:36`, `cmd/gmd/llm_testcmd.go:70-75`) call
  `client.Models.List` / `client.Chat.Completions.New` directly on the leaked object.
- **Three naming conventions** for the same 7 roles: struct fields (`embedder`...),
  role-string keys (`"embedding"`...), and `Profile` fields (`Embedding`...). Three
  near-identical switch statements (`RoleClient`, `RoleModel`, `RoleURL`) each switch on
  the same 7 magic strings (`client.go:36-94`).
- **Duplicate config structs:** `llm.Profile`/`llm.RoleConfig`/`llm.ProviderConfig`
  (`builder.go`) mirror `config.LLMProfileConfig`/`LLMRoleConfig`/`LLMProviderConfig`
  (`pkg/config/config.go:434-477`). `resolveStructured` (`pkg/llm/config.go:13-52`) is a
  manual field-by-field copy kept in sync by hand.
- **Thin types:** `ChatMessage{Role, Content string}` (`client.go:149-152`) - no tool
  calls, no images, no streaming, no temperature/max_tokens. All chat methods return a
  bare `string`. No request/response structs.
- **Bugs:** `BuildClient` double-builds and discards the OAuth2 client when an
  `HTTPClient` is also injected (`builder.go:71-75`). `ChatWithModel` routes through
  `c.expander.client` regardless of the model argument (`client.go:186-188`).
- **Dead code:** `RoleConfig.Client` field (`builder.go:23`), `GeneralBigChat`/
  `GeneralMidChat`/`GeneralSmallChat`/`ChatWithModel` (no external callers),
  `denormalizeToolCallID` in hof's reference. `LLMProviderConfig.Features`
  (`embed`/`chat`/`rerank` booleans) is parsed from CUE but ignored by `pkg/llm`.
- **No lifecycle:** no `Close()`. OAuth2 HTTP clients from service-account auth are never
  shut down. No typed errors (all `fmt.Errorf` stringly).

### ADK (`.extern/adk-go`) - the foundation

- **Module:** `google.golang.org/adk`, Go 1.25.0, depends on `google.golang.org/genai`
  v1.57.0. ADK has **no** openai-go dependency. gmd already has openai-go v3.37.0 and
  Go 1.25.6. gmd must add `google.golang.org/adk` + `google.golang.org/genai`.
- **The entire `model.LLM` interface** (`.extern/adk-go/model/llm.go:26-29`):
  ```go
  type LLM interface {
      Name() string
      GenerateContent(ctx context.Context, req *LLMRequest, stream bool) iter.Seq2[*LLMResponse, error]
  }
  ```
  Two methods. That is the whole contract.
- **`LLMRequest`** (`llm.go:32-38`): `Model string`, `Contents []*genai.Content`,
  `Config *genai.GenerateContentConfig`, `Tools map[string]any`.
- **`LLMResponse`** (`llm.go:42-68`): `Content *genai.Content`, `UsageMetadata`,
  `FinishReason`, streaming flags `Partial`/`TurnComplete`/`Interrupted`, `ModelVersion`,
  error fields `ErrorCode`/`ErrorMessage`, plus citation/grounding/logprobs/transcription.
- **Lingua franca is `google.golang.org/genai`:** `genai.Content` (`Role` + `Parts
  []*genai.Part`), `genai.Part` (text / `FunctionCall` / `FunctionResponse` /
  `InlineData`), `genai.GenerateContentConfig` (temperature, tools, system instruction,
  safety, thinking config), `genai.Schema`, `genai.FinishReason`, etc. Any non-Gemini
  adapter must translate to/from these types.
- **Streaming** is Go 1.23 `iter.Seq2[*LLMResponse, error]` (range-over-func). Yield many
  `Partial: true` chunks, then one `TurnComplete: true` final event. Consumer early-exits
  by returning `false` from `yield`.
- **ADK ships NO OpenAI adapter.** Only `model/gemini` and `model/apigee` (proxy over
  gemini). gmd builds the OpenAI adapter from scratch.
- **No model registry.** A model is constructed and passed by value into
  `llmagent.Config{Model: mdl}` (`.extern/adk-go/agent/llmagent/llmagent.go:178`).
- **ADK has ZERO embedding support.** Grep for `Embed`/`Embedding` across all of
  `.extern/adk-go` returns only `//go:embed` directives and Go struct embedding. gmd's
  embeddings **must** live outside `model.LLM`.

### Hof reference (`~/hof/hof/lib/agent/models`)

- `openai.go` (804 lines): implements `model.LLM` for OpenAI-compatible providers on
  openai-go v3. Pattern: `Config{APIKey, BaseURL, ModelName, HTTPOptions}` -> `New(cfg)
  *Model` -> `Name()` + `GenerateContent(ctx, req, stream)` dispatching to
  `generate`/`generateStream`. Uses `openai.ChatCompletionAccumulator` for streaming.
  Converts `genai.Content`/`Part` <-> openai-go messages. Handles tool calls, schemas,
  multi-modal. `var _ model.LLM = &Model{}` compile-time assertion.
- `vertex.go` (37 lines): thin factory delegating to ADK's `gemini.NewModel`.
- **Gaps to fix in gmd's port:** no `*http.Client`/`http.RoundTripper` injection
  (deal-breaker for tape replay - hof only does headers via `HTTPOptions`), stray
  `fmt.Println` debug prints, hardcoded `ReasoningEffort: "low"`, `parseJSONArgs`
  swallows errors, `denormalizeToolCallID` is dead code, `ctx` ignored in factory.

### Consumers (15 non-test files + 8 test files)

Full map in the exploration report. Summary by call pattern:

| Pattern | Call sites | Files |
|---|---|---|
| `Embed`/`EmbedBatch` | 3 | `pkg/indexer/indexer.go:279`, `pkg/search/pipeline.go:92,269` |
| `Chat`/`Summarize` | 10 | `pkg/search/pipeline.go:255`, `pkg/wiki/agent.go:116,348`, `pkg/wiki/postprocess.go:41`, `pkg/wiki/lint.go:233,261`, `pkg/web/agent.go:179,222`, `pkg/web/fusion/fusion.go:201,249` |
| `Rerank` | 1 | `pkg/search/pipeline.go:373` |
| `CheckAll`/`EndpointStatus` | 2 | `pkg/wiki/doctor.go:52`, `cmd/gmd/doctor.go:114` |
| `BuildClient` + raw openai-go | 2 | `cmd/gmd/llm_status.go:30,36`, `cmd/gmd/llm_testcmd.go:40,70-75` |
| Wiring (threads client to constructor) | 3 | `pkg/mcp/wiki_tools.go:24` (-> `wiki.NewAgent`), `pkg/wiki/watch.go:24` (-> `wiki.NewAgent`), `pkg/web/fusion/fusion.go:19` (`Config.LLMClient` field) |
| Wiring (`ResolveLLMConfig`) | 1 + ~10 cmd | `cmd/gmd/llm_helper.go:9`, all `cmd/gmd/*` that build a client |

Not affected: `pkg/runtime` (owns no LLM lifecycle), `cmd/gmd/llm_providers.go`/
`llm_profiles.go`/`llm_show.go` (read `pkg/config` types only), `cmd/gmd/cleanup.go`
(passes nil).

## Key Decisions

### D1: Three concerns, three interfaces (not one god-struct)

ADK's `model.LLM` covers chat/generation only. Embeddings and rerank are fundamentally
different operations that ADK deliberately excludes. The rewrite exposes them as
separate, small interfaces rather than bolting them onto `model.LLM`:

```go
// pkg/llm/chat.go
// ChatModel is ADK's model.LLM plus gmd ergonomic helpers.
// The OpenAI adapter implements this. Future Gemini/Vertex adapters can too.
type ChatModel interface {
    model.LLM  // Name() + GenerateContent(ctx, *LLMRequest, stream) iter.Seq2[...]

    // Simple text-in/text-out helpers for gmd's common patterns.
    // These wrap GenerateContent with genai.Content construction.
    Chat(ctx context.Context, system, user string) (string, error)
    ChatMessages(ctx context.Context, contents []*genai.Content) (string, error)
}

// pkg/llm/embed.go
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float64, error)
}

// pkg/llm/rerank.go
type Reranker interface {
    Rerank(ctx context.Context, query string, documents []string) ([]RerankResult, error)
}
```

**Rationale:** `model.LLM` is the ADK-native contract - implementing it means gmd's model
can plug into ADK's `llmagent` in the future without adaptation. The `Chat`/`ChatMessages`
helpers preserve gmd's simple `Chat(ctx, system, user) (string, error)` ergonomics for the
~10 call sites that just want text in / text out, without forcing every consumer to build
`*genai.Content`/`*genai.Part` manually. Consumers that need tools/streaming/multi-modal
call `GenerateContent` directly.

**Rejected alternative:** make `ChatModel` a pure wrapper around `model.LLM` without
embedding ADK's interface. Rejected because it breaks the "built on ADK" requirement and
prevents future `llmagent.Config{Model: ...}` wiring.

### D2: Config-driven role registry (not 7 hardcoded fields)

Replace the 7 named struct fields on `Client`/`Profile` with a map. Standard role names
are constants, not struct fields:

```go
// pkg/llm/registry.go

// Standard role names. Consumers reference these. Config can define any number
// of additional roles; these are the conventional defaults. All role sizes
// are kept (general_big/mid/small) - do not collapse them.
const (
    RoleEmbedding    = "embedding"
    RoleExpansion    = "expansion"
    RoleRerank       = "rerank"
    RoleSummarizing  = "summarizing"
    RoleGeneralBig   = "general_big"
    RoleGeneralMid   = "general_mid"
    RoleGeneralSmall = "general_small"
)

// Registry holds resolved LLM clients indexed by role, plus the shared
// embedder and reranker. Built once from config at startup. Read-only after
// construction; safe for concurrent use.
type Registry struct {
    models  map[string]ChatModel   // role name -> chat model
    embed   Embedder
    rerank  Reranker
    health  []ProviderHealth       // for CheckProviders
    closers []func() error         // OAuth2 HTTP clients to shut down
}

func (r *Registry) Model(role string) ChatModel       // nil if role unset
func (r *Registry) Embedder() Embedder                 // nil if embedding unset
func (r *Registry) Reranker() Reranker                 // nil if rerank unset
func (r *Registry) Roles() []string                   // sorted role names
func (r *Registry) CheckProviders(ctx context.Context) []ProviderHealth
func (r *Registry) Close() error                      // closes OAuth2-backed HTTP clients

// RegistryOption is a functional option for NewRegistry.
type RegistryOption func(*registryConfig)

// registryConfig holds test-time overrides applied before building clients.
type registryConfig struct {
    providerTransports map[string]*http.Client   // provider name -> custom HTTP client
}

// WithProviderTransport injects a custom HTTP client for a specific provider.
// Used by tests to inject tape-replay transports.
func WithProviderTransport(provider string, client *http.Client) RegistryOption
```

**Rationale:** eliminates the 3 switch statements (`RoleClient`/`RoleModel`/`RoleURL`),
the 3 naming conventions, and the 5-location edit problem. Adding a role is a config
change, not a code change. `Registry.Model(role)` returns `nil` for unset roles (matching
the current nil-tolerant pattern in `pkg/wiki/lint.go:198,251` - but see the nil-check
note in Consumer Migration Details below). `Close()` shuts down OAuth2 token-refresh
goroutines from service-account auth, fixing the current lifecycle leak.

**Provider client sharing:** `NewRegistry` caches the underlying `*openai.Client` per
provider name (keyed by `provider` name in `cfg.LLM.Providers`), so multiple roles pointing
at the same provider share one HTTP connection pool - preserving the optimization in the
current `clientBuilder.getOrBuild` (`builder.go:139-153`). The defaults have 4+ roles on
`"default"` (`config.go:801-807`); without sharing they would spawn 4+ separate pools.

### D3: OpenAI adapter as a separate type implementing `ChatModel` (+ `model.LLM`)

```go
// pkg/llm/openai_model.go

// OpenAIModel implements ChatModel (and thus model.LLM) for any
// OpenAI-compatible endpoint: OpenAI, vLLM, Ollama, Anthropic-compat, Vertex-compat.
type OpenAIModel struct {
    client    *openai.Client
    modelName string
    // ...toolCallIDMap for >40-char IDs, etc. (from hof pattern)
}

var _ ChatModel = (*OpenAIModel)(nil)
var _ model.LLM = (*OpenAIModel)(nil)

// Config for constructing an OpenAIModel. Fixes hof's gaps:
// - HTTPClient/RoundTripper injection (required for tape replay)
// - No debug prints, no hardcoded ReasoningEffort
type OpenAIConfig struct {
    APIKey      string
    BaseURL     string
    ModelName   string
    HTTPClient  *http.Client   // injected transport (tape replay, service-account oauth)
    Headers     http.Header
}

func NewOpenAIModel(cfg OpenAIConfig) *OpenAIModel

// model.LLM interface
func (m *OpenAIModel) Name() string
func (m *OpenAIModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error]

// ChatModel helpers (wrap GenerateContent)
func (m *OpenAIModel) Chat(ctx context.Context, system, user string) (string, error)
func (m *OpenAIModel) ChatMessages(ctx context.Context, contents []*genai.Content) (string, error)
```

The conversion logic (`buildChatCompletionParams`, `convertContentToMessages`,
`convertResponse`, `convertTools`, `ensureObjectProperties`, `convertSchema`,
`convertRole`, `convertFinishReason`, `convertUsageMetadata`) is ported from
`~/hof/hof/lib/agent/models/openai.go` with the fixes noted above.

**Rationale:** one adapter type for all OpenAI-compatible providers (the "no per-provider
Go packages needed" insight from `.design/llm-providers.md` still holds). Auth and base
URL are config concerns, not code concerns. The adapter is the only place openai-go types
appear; they never escape `pkg/llm`.

### D4: Embeddings and rerank stay as direct OpenAI-compatible HTTP calls

```go
// pkg/llm/embed.go
type openaiEmbedder struct {
    client    *openai.Client
    modelName string
}
func NewEmbedder(cfg OpenAIConfig) Embedder   // shares OpenAIConfig shape

// pkg/llm/rerank.go
type openaiReranker struct {
    client    *openai.Client   // uses openai-go's generic POST for /rerank
    modelName string
}
func NewReranker(cfg OpenAIConfig) Reranker

type RerankResult struct {
    Index int
    Score float64
}
```

**Rationale:** ADK has no embedding interface, so `Embedder` is gmd-owned. Rerank is a
non-standard endpoint (vLLM-specific `/rerank`); keeping it as a raw POST via openai-go's
generic helper is the simplest correct approach. Both share the `OpenAIConfig` shape so
auth/base-URL/HTTPClient injection is uniform. `RerankResult` is the only rerank type that
escapes the package - it is already clean.

**Batch-first `Embedder`.** The current `Client` has `Embed(ctx, text) ([]float64, error)`
(single) and `EmbedBatch(ctx, texts) ([][]float64, error)` (batch). The new interface has
only `Embed(ctx, texts []string) ([][]float64, error)` - batch-first. Single-embedding
callers (`pipeline.go:92,269`) wrap in `[]string{text}` and take `[0]`. The indexer's batch
call (`indexer.go:279`) maps directly. This eliminates the redundant single/batch split.

### D5: Eliminate duplicate config structs

`pkg/llm` stops defining its own `ProviderConfig`/`RoleConfig`/`Profile`. Config
resolution reads `pkg/config` types directly:

```go
// pkg/llm/config.go
func NewRegistry(ctx context.Context, cfg *config.Config, opts ...RegistryOption) (*Registry, error)
```

This function: reads `cfg.LLM.Providers` + `cfg.LLM.Profiles[activeProfile]`, builds an
`OpenAIModel` (or `OpenAIEmbedder`/`OpenAIReranker`) per role using `OpenAIConfig`
populated from the provider entry (base URL, auth, HTTP client from `auth.BuildHTTPClient`),
and returns a populated `*Registry`. No manual field copying, no parallel structs.

**Rationale:** `pkg/config` already owns the CUE-facing structs (`LLMProviderConfig`,
`LLMRoleConfig`, `LLMProfileConfig` at `config.go:434-477`). Duplicating them in
`pkg/llm` created the manual-copy boilerplate and sync burden. `pkg/llm` depends on
`pkg/config` already (via `ResolveLLMConfig`); consuming the types directly is cleaner.

### D6: Keep the tape-replay test pattern, fix HTTPClient injection

The tape pattern (`pkg/testutil/tape.go`) injects an `http.RoundTripper` via the provider's
`HTTPClient` field. The new `OpenAIConfig.HTTPClient` field makes this work identically.
The `auth` double-build bug (`builder.go:71-75`) is gone because auth and HTTPClient are
merged into a single `openai.NewClient(opts...)` call:

```go
func newOpenAIClient(cfg OpenAIConfig) *openai.Client {
    var opts []option.RequestOption
    if cfg.APIKey != ""      { opts = append(opts, option.WithAPIKey(cfg.APIKey)) }
    if cfg.BaseURL != ""     { opts = append(opts, option.WithBaseURL(cfg.BaseURL)) }
    if cfg.HTTPClient != nil { opts = append(opts, option.WithHTTPClient(cfg.HTTPClient)) }
    for k, vals := range cfg.Headers { for _, v := range vals { opts = append(opts, option.WithHeaderAdd(k, v)) } }
    client := openai.NewClient(opts...)
    return &client
}
```

**Tape replay is sequential, not request-matched.** `tape.replayRoundTrip`
(`pkg/testutil/tape.go:184-200`) returns `exchanges[pos]` in order and increments `pos` -
it does **not** inspect the request method, URL, or body. The recording side
(`recordRoundTrip`, `:152-182`) captures method/URL/body for audit, but replay ignores
them. This means existing tapes replay correctly as long as the number and order of HTTP
round-trips is unchanged (1 POST per chat, 1 per embed, 1 per rerank - unchanged by the
rewrite). Tapes should still be **regenerated** during integration recording so the
captured request bodies reflect the new adapter's request shape, but old tapes work for
unit tests in the interim.

### D7: Auth subpackage stays, bug fixed

`pkg/llm/auth/` keeps its `Method`/`Config`/`BuildHTTPClient`/`DefaultBaseURL` API. The
fix: `BuildHTTPClient` is the single place that produces an `*http.Client` for
service-account auth, and that client is passed as `OpenAIConfig.HTTPClient`. For
`apikey` auth, the key goes into `OpenAIConfig.APIKey` (not into `auth.Config.APIKey`,
which is currently dead). The double-build bug disappears because there is only one
`openai.NewClient` call per provider.

## Planned Architecture

### New `pkg/llm` layout

```
pkg/llm/
  openai_model.go     OpenAIModel: implements ChatModel + model.LLM (ADK adapter)
  openai_convert.go   genai <-> openai-go conversion (ported from hof, fixed)
  embed.go            Embedder interface + openaiEmbedder impl
  rerank.go           Reranker interface + openaiReranker impl + RerankResult
  registry.go         Registry: role -> ChatModel map, Embedder, Reranker, health, Close
  config.go           NewRegistry(ctx, *config.Config, opts ...RegistryOption) (*Registry, error)
  errors.go           typed sentinel errors
  auth/
    auth.go           unchanged: Method, Config, BuildHTTPClient, DefaultBaseURL
    auth_test.go      unchanged
  testdata/
    001_embed.json    reused (sequential replay, adapter-agnostic)
    002_chat_expand.json
    003_rerank.json
  openai_model_test.go       unit tests (conversion helpers, param building)
  openai_model_replay_test.go    tape replay (Embed, Chat, Rerank via new interfaces)
  openai_model_integration_test.go   +build integration (records tapes)
  registry_test.go          unit tests (role resolution, nil-unset, NewRegistry with taped transports)
```

**Deleted:** `client.go` (god-struct), `builder.go` (duplicate structs + buggy builder),
`config.go` (manual copy). Their functionality moves to `registry.go` + `config.go` +
`openai_model.go`.

### Type relationships

```
                    model.LLM  (ADK: Name + GenerateContent)
                        ^
                        |
                    ChatModel  (gmd: + Chat + ChatMessages helpers)
                        ^
                        |
                   OpenAIModel  (concrete: openai-go v3 adapter)

    Embedder <---- openaiEmbedder   (concrete: openai-go /v1/embeddings)
    Reranker <---- openaiReranker   (concrete: openai-go POST /rerank)

    Registry ---models---> map[string]ChatModel  (role -> model)
            \--- embed ---> Embedder
             \-- rerank --> Reranker
              \- health --> []ProviderHealth

    config.Config ---NewRegistry()---> *Registry
```

### Consumer migration shape

Every consumer changes. Simple consumers swap `*llm.Client` for specific interfaces
(Phase 4-5). Wiki and web agents are rewritten as ADK `llmagent` agents with tools
(Phases 6-8). New files created in `pkg/wiki/` and `pkg/web/`: `tools.go`, `runner.go`,
`prompts.go`. `pkg/wiki/postprocess.go` is deleted.

| Consumer | Current | New |
|---|---|---|
| `pkg/indexer` | `llmClient *llm.Client` -> `EmbedBatch` | `embedder llm.Embedder` -> `Embed` |
| `pkg/search` | `llmClient *llm.Client` -> `Embed`, `Chat`, `Rerank` | `embedder llm.Embedder`, `expander llm.ChatModel`, `reranker llm.Reranker` |
| `pkg/wiki/agent.go` | `llmClient *llm.Client` -> hand-rolled Chat pipelines | ADK `llmagent` with tools (ingest/query), run via `runner.Run` |
| `pkg/wiki/lint.go` | `llmClient *llm.Client` -> pairwise Chat calls | ADK `llmagent` lint agent; `lintStructure` stays pure Go pre-pass |
| `pkg/wiki/postprocess.go` | `llmClient *llm.Client` -> `generateDescription` | **deleted** — ingest agent writes description directly |
| `pkg/wiki/doctor.go` | `llmClient *llm.Client` -> `CheckAll` + `EndpointStatus` | `registry *llm.Registry` -> `CheckProviders` |
| `pkg/wiki/watch.go` | `llmClient *llm.Client` (plumbing to Agent) | `*Runner` (or deps) (plumbing to Agent) |
| `pkg/web/agent.go` | `llmClient *llm.Client` -> Go-orchestrated loop + free-text parsing | ADK `llmagent` with `WebSearch`/`WebFetch` tools |
| `pkg/web/fusion/fusion.go` | `Config.LLMClient *llm.Client` -> `Summarize` | `Config.Summarizer llm.ChatModel` -> `.Chat` (not an agent) |
| `pkg/mcp/wiki_tools.go` | `llmClient *llm.Client` -> wires into `wiki.NewAgent` | `*Runner` (or deps) -> wires into `wiki.NewAgent` |
| `cmd/gmd/search.go` (handles search, query, vsearch modes) | build `*llm.Client`, pass to `search.New` | `registry.Model(RoleExpansion)`, `registry.Embedder()`, `registry.Reranker()` -> `search.New` |
| `cmd/gmd/update.go` (handles both update and embed) | build `*llm.Client`, pass to `indexer.New` | `registry.Embedder()` -> `indexer.New` |
| `cmd/gmd/wiki_*.go` | build `*llm.Client`, pass to `wiki.NewAgent` | build `*Runner` (via `llm.NewRegistry`), pass to `wiki.NewAgent` |
| `cmd/gmd/web_agent.go` | build `*llm.Client`, pass to `web.NewAgent` | wire `SearchProvider`s + `BrowserProvider` + `registry.Model` into new `NewAgent` |
| `cmd/gmd/web_search.go` | build `*llm.Client` only if synthesize/dedup -> `fusion.Config` | `registry.Model(RoleSummarizing)` only if needed -> `fusion.Config.Summarizer` |
| `cmd/gmd/doctor.go` | `client.CheckAll` + `client.RoleModel(...)` (lines 141-143) | `registry.CheckProviders` + `registry.Model(role)` |
| `cmd/gmd/llm_status.go` | `BuildClient` + raw `client.Models.List` | `registry.CheckProviders(ctx)` |
| `cmd/gmd/llm_testcmd.go` | `BuildClient` + raw `client.Chat.Completions.New` | `registry.Model(role).Chat(ctx, system, user)` |

**`ChatMessage` is gone.** Consumers that build `[]llm.ChatMessage{{Role:"system",...},...}`
switch to either:
- `chatModel.Chat(ctx, system, user)` for the common 2-message system+user pattern (covers
  ~7 of 10 chat call sites), or
- `chatModel.ChatMessages(ctx, []*genai.Content{...})` for multi-turn, building
  `genai.Content` with `genai.NewContentFromText(text, genai.RoleUser)`.

**`Summarize`/`GeneralBigChat`/etc. are gone.** They were thin wrappers over `Chat` with a
different role client. Now the caller gets the right `ChatModel` for the role from the
registry: `registry.Model(llm.RoleSummarizing)` or `registry.Model(llm.RoleGeneralBig)`.
This is functionally equivalent: `Summarize` currently routes through
`c.summarizer.client` with the summarizer's model - the new code calls `.Chat()` on a
`ChatModel` built from the same provider+model. The defaults (`config.go:801-807`) give
`Summarizing` a different provider than `Expansion`, so `pkg/web/agent` and
`pkg/web/fusion` receive two separate `ChatModel` instances (one for `RoleGeneralBig`, one
for `RoleSummarizing`), matching the current routing.

### Consumer migration details

**Nil-check pattern change.** Three call sites currently nil-check the whole `*llm.Client`
(`pkg/wiki/lint.go:198,251` checks `a.llmClient == nil`; `pkg/web/fusion/fusion.go:183,222`
checks `cfg.LLMClient == nil`). With the new design, they receive a `chat llm.ChatModel`
(or `cfg.Summarizer llm.ChatModel`) and must nil-check that interface value before
calling `.Chat(...)`. A nil `ChatModel` is the "role unset" signal from
`Registry.Model(role)`. This is a behavior change in the nil-check location, not just a
type swap - the 3 call sites need their guards updated from `a.llmClient == nil` to
`a.chat == nil` (etc.).

**`Chat(ctx, system, user)` system-prompt mapping.** The `system` string maps to
`req.Config.SystemInstruction` (a `*genai.Content` with `Role: "system"`), NOT to a
`genai.Content` entry in `req.Contents`. This matches ADK's convention: hof's
`buildChatCompletionParams` extracts `req.Config.SystemInstruction` and prepends it as an
`openai.SystemMessage` (`openai.go:268-272`). The `ChatMessages` helper, by contrast,
passes `contents` through as `req.Contents` and leaves `SystemInstruction` nil - so a
caller wanting a system prompt via `ChatMessages` includes a `genai.Content{Role:
"system", ...}` in the slice. The two helpers are consistent: `Chat` is the 2-message
shortcut (system -> `SystemInstruction`), `ChatMessages` is the full-control path. All 10
current chat call sites pass at most a system+user pair, so `Chat(ctx, system, user)`
covers them (user-only sites pass `""` for system, which the helper skips if empty).

**`Chat(ctx, system, user)` is sufficient for all current call sites.** Verified: every
chat/summarize call site passes either 2 messages (system+user) or 1 message (user-only).
`pkg/wiki/lint.go:233,261` and `pkg/wiki/postprocess.go:41` pass user-only -> `system =
""`. None require multi-turn. `ChatMessages` is reserved for future use.

## Design Considerations & Trade-offs

### ADK agent migration scope

This rewrite builds the model client **on** ADK (`model.LLM`) AND migrates gmd's
wiki/web agent loops to ADK's `llmagent` framework. Both are in scope. The model rewrite
makes the agent migration possible (the model is an `model.LLM` that plugs into
`llmagent.Config{Model: ...}`). See the "ADK Agent Migration" section below for the full
design of the wiki and web agent rewrites.

### `genai.Content` vs a gmd-specific message type

Consumers must now speak `genai.Content`/`genai.Part` for multi-turn chat (via
`ChatMessages`). This pulls `google.golang.org/genai` into every chat consumer's import
graph. Trade-off:
- **Pro:** ADK-native, no translation layer, future tool-use/multi-modal works
  out-of-the-box, one message type across the whole ADK ecosystem.
- **Con:** consumers depend on genai types; `genai.Content` is more ceremony than
  `ChatMessage{Role, Content}` for simple cases.
- **Mitigation:** the `Chat(ctx, system, user)` helper covers the common case with zero
  genai exposure. Only multi-turn callers (rare in gmd today) touch `genai.Content`.

### Role map vs fixed roles

The registry uses `map[string]ChatModel` keyed by role name. This is more flexible than 7
struct fields but means a typo in a role name returns `nil` instead of a compile error.
Trade-off accepted: the config already uses string-keyed roles, and `nil` returns are
already the de-facto pattern (`lint.go` nil-checks). We define `Role*` constants so
consumers reference names symbolically.

### `openai-go` still a dependency

ADK does not depend on openai-go, but gmd's OpenAI adapter does (for the
OpenAI-compatible HTTP API). This is correct: ADK speaks genai; the adapter speaks
openai-go; the adapter translates. openai-go is an implementation detail of
`pkg/llm/openai_model.go` and does not escape the package.

### Streaming

`ChatModel` inherits `GenerateContent(..., stream bool)` from `model.LLM`, so streaming is
available to any consumer that wants it via the ADK interface. The `Chat`/`ChatMessages`
helpers are non-streaming (collect the final response). No current consumer needs
streaming, but the capability is there for future use (e.g. MCP streaming responses).

### Health checks

`CheckAll`/`CheckEndpoint`/`CheckProvider`/`CheckAllProviders` (4 methods, `client.go:233-
299`) collapse into `Registry.CheckProviders(ctx) []ProviderHealth`. `ProviderHealth`
replaces `EndpointStatus` with proper JSON tags. The CLI `gmd llm status` and `gmd doctor`
call this. `cmd/gmd/llm_status.go` and `llm_testcmd.go` stop calling
`client.Models.List` directly; the health check wraps that internally.

### Backward compatibility

Per AGENTS.md: "The project is still in alpha / just-me state, we do not need to worry
about backwards compatibility." The CUE config shape (`providers`/`profiles`/`roles`)
stays the same because it is sound; the Go runtime layer is rewritten wholesale. No
legacy shims.

### Error handling

The current package uses `fmt.Errorf` stringly errors with no way for callers to
distinguish rate-limit / auth / missing-model / network failures. The rewrite introduces
typed sentinel errors and an option for structured errors:

```go
// pkg/llm/errors.go
var (
    ErrProviderNotConfigured = errors.New("llm: provider not configured")
    ErrRoleUnset             = errors.New("llm: role not set in profile")
    ErrModelNotFound         = errors.New("llm: model not found on provider")
)
```

`NewRegistry` returns `ErrProviderNotConfigured` / `ErrRoleUnset` during resolution
(fail fast at startup, not at first call). `CheckProviders` populates
`ProviderHealth.Err` with the typed error. Chat/Embed/Rerank call sites wrap provider
errors with `fmt.Errorf("chat: %w", err)` as today, but callers can `errors.Is` the
sentinels. This is a light touch - not a full error-type hierarchy, just enough to
distinguish the failure modes that callers might branch on. (Tape exhaustion is a
`testutil` concern using `fmt.Errorf` today; if a sentinel is needed there, it would be
added to `pkg/testutil` separately, not re-exported from `pkg/llm`.)

### Performance / conversion overhead

The genai<->openai-go conversion in `OpenAIModel.GenerateContent` adds per-request
allocation (building `[]openai.ChatCompletionMessageParamUnion` from `[]*genai.Content`).
For gmd's common case (system + user, no tools, no media), the path is:
`extractText(SystemInstruction)` -> `openai.SystemMessage(text)` + one
`convertContentToMessages` -> `openai.UserMessage(joinedTexts)`. This produces the same
simple string content as the current code - overhead is negligible (a few string
concatenations and one slice allocation). The full conversion machinery (tool calls,
schemas, multi-media) is only exercised when `LLMRequest.Config` or `Contents` carry those
part types, which no current consumer does.

### Migration safety

Phases are ordered so that the package compiles and tests pass between phases:

- **After Phase 1-3:** `pkg/llm` has the new types (`OpenAIModel`, `Embedder`, `Reranker`,
  `Registry`) but no consumers use them yet. Old `client.go`/`builder.go` still exist.
  `make test` passes (old tests, new unit tests for conversion helpers).
- **Phases 4-5 (consumer migration):** Old `client.go`/`builder.go`/old `config.go` are
  deleted (Phase 4); all consumers switch in the same commit (or a short series of commits
  within the phase). Between individual consumer edits, the tree may not compile. This is
  acceptable per the alpha/no-backcompat policy. The risk is bounded: each consumer change
  is mechanical (swap `*llm.Client` -> specific interface). Phase 5 wires the new CLI.
- **Phases 6-8 (agent migration):** Wiki and web agents are rewritten as ADK `llmagent`
  agents with tools. Highest-risk changes: hand-rolled pipelines become LLM-driven loops.
  Tests use mock `model.LLM` instances for loop behavior.
- **After Phase 9-10:** Tapes regenerated, docs updated, `make check` green.

A transitional shim (old + new APIs coexisting) was considered and rejected: it would
require keeping the god-struct alive during migration, doubling the maintenance surface
for a short-lived alpha project. The big-bang is cleaner here.

## Implementation Plan

### Phase 1: Dependencies + OpenAI adapter

- Add `google.golang.org/adk` (local replace -> `.extern/adk-go`) and
  `google.golang.org/genai v1.57.0` to `go.mod`.
- Write `pkg/llm/openai_model.go`: `OpenAIConfig`, `NewOpenAIModel`, `OpenAIModel` struct
  implementing `ChatModel` (+ `model.LLM`). Port conversion logic from hof's `openai.go`
  into `openai_convert.go`: `buildChatCompletionParams`, `convertContentToMessages`,
  `buildUserMessage`, `buildAssistantMessage`, `convertResponse`, `convertTools`,
  `convertToFunctionParams`, `ensureObjectProperties`, `convertSchema`, `convertRole`,
  `convertFinishReason`, `convertUsageMetadata`, `extractText`, `joinTexts`. Fix: no
  `fmt.Println`, no hardcoded `ReasoningEffort`, `parseJSONArgs` returns errors,
  `HTTPClient` injection, drop unused `denormalizeToolCallID` reverse map.
- Write `pkg/llm/openai_model_test.go`: unit tests for conversion helpers (pure, no HTTP).

### Phase 2: Embedder + Reranker

- Write `pkg/llm/embed.go`: `Embedder` interface, `openaiEmbedder`, `NewEmbedder`,
  `Embed(ctx, texts) ([][]float64, error)`.
- Write `pkg/llm/rerank.go`: `Reranker` interface, `openaiReranker`, `NewReranker`,
  `Rerank(ctx, query, documents) ([]RerankResult, error)`, `RerankResult` struct.
- Both use `newOpenAIClient(cfg OpenAIConfig)` (shared helper, fixes the double-build bug).

### Phase 3: Registry + config resolution

- Write `pkg/llm/registry.go`: `Registry` struct, `Model`/`Embedder`/`Reranker`/
  `Roles`/`CheckProviders`/`Close` methods, `ProviderHealth` type, role name constants.
  `Registry` is read-only after construction (safe for concurrent use).
- Write `pkg/llm/config.go`: `NewRegistry(ctx, *config.Config, opts ...RegistryOption)
  (*Registry, error)` - reads `cfg.LLM`, caches `*openai.Client` per provider name
  (shared across roles on the same provider, preserving the current `getOrBuild`
  optimization), builds per-role `OpenAIModel`/`OpenAIEmbedder`/`OpenAIReranker` via
  `OpenAIConfig` + `auth.BuildHTTPClient`. `RegistryOption` allows test-time
  `HTTPClient` injection per provider. Registers OAuth2 closers for `Close()`.
  No duplicate structs.
- Write `pkg/llm/errors.go`: typed sentinel errors (`ErrProviderNotConfigured`,
  `ErrRoleUnset`, `ErrModelNotFound`).
- Remove `LLMProviderFeatures` (`embed`/`chat`/`rerank` booleans) from `pkg/config`
  structs and CUE schema - code rot, no possible feature overlap.
- Old `client.go`/`builder.go`/old `config.go` are NOT yet deleted - they coexist with
  the new types so `make test` passes (old tests + new unit tests). Deletion happens in
  Phase 4.

### Phase 4: Migrate library consumers (big-bang)

- Delete `pkg/llm/client.go`, `pkg/llm/builder.go`, old `pkg/llm/config.go` - the old
  `*llm.Client` is gone. All consumers must switch in this phase (the tree may not
  compile between individual edits; acceptable per alpha policy).
- `pkg/indexer`: `New(..., embedder llm.Embedder)`.
- `pkg/search`: `New(..., embedder llm.Embedder, expander llm.ChatModel, reranker llm.Reranker)`.
- `pkg/wiki/doctor`: `Doctor(..., registry *llm.Registry)`.
- `pkg/web/fusion`: `Config.Summarizer llm.ChatModel`.
- Wiki and web agent consumers get temporary constructors taking `chat llm.ChatModel` (or
  `*Registry` for doctor). These are interfaces — they compile against the new types now
  and stay valid through the ADK agent rewrite in Phases 6-7.

### Phase 5: Migrate cmd/gmd wiring + CLI

- `cmd/gmd/llm_helper.go`: `newRegistry(cfg) (*llm.Registry, error)` replaces
  `llmConfigFromConfig`.
- All `cmd/gmd/*.go` that build a client: `registry, _ := newRegistry(cfg)`, then pull
  `registry.Model(llm.RoleExpansion)`, `registry.Embedder()`, `registry.Reranker()`, etc.
  per command.
- `cmd/gmd/llm_status.go`: `registry.CheckProviders(ctx)`.
- `cmd/gmd/llm_testcmd.go`: `registry.Model(role).Chat(ctx, "You are a test", "ping")`.
- `cmd/gmd/doctor.go`: `registry.CheckProviders(ctx)`.

### Phase 6: Wiki ADK agent

- Write `pkg/wiki/tools.go`: `functiontool.New` wrappers for `SearchWiki`, `ReadPage`,
  `ReadIndex`, `CreatePage`, `UpdatePage`, `UpdateIndex`, `AppendLog`, `ListPages`,
  `IndexPage`, `ReadSource`. Each wraps an existing Go function (`readWikiPage`,
  `createWikiPage`, etc.) with the `tsClient`/`wiki`/`indexer` dependencies captured in
  the tool's closure.
- Write `pkg/wiki/prompts.go`: `Instruction` strings for ingest/query/lint agents. The
  static parts of `ingest_system.md`/`query_system.md`/`lint_*.md` become the
  `Instruction`; the dynamic parts (existing pages, search results) become tool results.
- Write `pkg/wiki/runner.go`: `NewRunner(registry, wiki, tsClient, indexer, sessionSvc)`
  builds three `llmagent` agents (ingest/query/lint) composed under a root agent, plus
  a `runner.Runner`. Tools capture `tsClient`/`wiki`/`indexer` in closures. Expose
  `Ingest(ctx, sourcePath)`, `Query(ctx, question)`, `Lint(ctx)` methods that create a
  session, run, collect final event text + session state, and assemble
  `IngestReport`/`QueryResult`/`LintResult` from tool-emitted state (NOT text parsing).
- Rewrite `pkg/wiki/agent.go`: delete `Ingest`/`Query` pipelines, `searchOverlap`,
  `extractKeyTerms`, `cleanJSON`, `generateDescription`. The `Agent` struct becomes a
  thin wrapper around the ADK runner, retaining the same public `Ingest`/`Query`/`Lint`
  method signatures. Keep `createWikiPage`/`updateWikiPage`/
  `updateIndexFile`/`appendLogFile`/`readWikiPage`/`readSource`/`loadIndexContext` as
  the tool implementations.
- Rewrite `pkg/wiki/lint.go`: `lintStructure` stays pure Go. `lintContent`/`lintGaps`
  delegate to the lint ADK agent.
- Delete `pkg/wiki/postprocess.go`.
- Update `cmd/gmd/wiki_*.go` and `pkg/mcp/wiki_tools.go` wiring to pass `*Runner` (or
  deps) to `wiki.NewAgent`.

### Phase 7: Web ADK agent

- Write `pkg/web/tools.go`: `WebSearch` (wraps `SearchProvider` with inline fan-out/dedup,
  NOT via fusion to avoid import cycle), `WebFetch` (wraps `BrowserProvider.GetContent`).
- Write `pkg/web/prompts.go`: de-duplicated `Instruction` from `agent_system.md` +
  `agent_synthesize.md`.
- Rewrite `pkg/web/agent.go`: `NewAgent` takes `searchProviders []SearchProvider`,
  `browserProvider BrowserProvider`, `chat llm.ChatModel` instead of raw `*exa.Client`
  + `*llm.Client`. `Run` builds an `llmagent` + runner, sends the question, collects
  events. Delete `analyzeResults`, `synthesize`, `formatResultsForLLM`, the `## ACTION`
  parser. `AgentResult` stays (assembled from the final event).
- Update `cmd/gmd/web_agent.go`: wire `SearchProvider`s + `BrowserProvider` +
  `registry.Model(RoleGeneralBig)` into the new `NewAgent`.

### Phase 8: Fusion update (not an agent)

- `pkg/web/fusion/fusion.go`: `Config.LLMClient *llm.Client` ->
  `Config.Summarizer llm.ChatModel`. `dedupLLM`/`Synthesize` call `.Chat` instead of
  `.Summarize`. Update `cmd/gmd/web_search.go` wiring.

### Phase 9: Tests + tapes (all tapes regenerated)

- `pkg/llm/openai_model_replay_test.go`: replay `001_embed.json` / `002_chat_expand.json`
  / `003_rerank.json` via `OpenAIConfig{HTTPClient: &http.Client{Transport: tape}}`.
- `pkg/llm/openai_model_integration_test.go`: `//go:build integration`, records new tapes.
- Update `pkg/wiki/*_test.go`, `pkg/web/*_test.go`, `pkg/web/fusion/*_test.go` helpers
  that build `*llm.Client` via `BuildAllClients` -> build `*llm.Registry` via
  `NewRegistry` or direct `NewOpenAIModel`/`NewEmbedder` with taped transport.
- **All tapes MUST be regenerated** - this is a major refactor. The old tapes are stale.
  The new adapter builds different request bodies (genai->openai conversion path), and
  the ADK agent migration changes the number and shape of LLM calls per workflow (multi-
  step tool-calling loops instead of single-shot). Regenerate every tape file:
  `pkg/llm/testdata/*.json`, `pkg/wiki/testdata/*.json`, `pkg/web/fusion/testdata/*.json`,
  and any new tapes for the ADK agent workflows.

### Phase 10: Cleanup + docs

- Verify no dead code remains: `GeneralBigChat`/`GeneralMidChat`/
  `GeneralSmallChat`/`ChatWithModel`, `ProviderClients`/`RoleClient`/`RoleURL` accessors
  (all removed in Phase 4 with the old files).
- Update `AGENTS.md` architecture section + CLI command list.
- Update `.design/llm-providers.md` -> mark as superseded by this doc.
- `make lint && make lint-all && make test`.

### Tests to add/update

| Test | Type | What it covers |
|---|---|---|
| `openai_model_test.go` | unit (pure) | genai<->openai conversion, param building, schema normalization, `ensureObjectProperties` |
| `openai_model_replay_test.go` | unit (tape) | Embed, Chat, Rerank via new interfaces against existing tapes (sequential replay) |
| `openai_model_integration_test.go` | integration | records new tapes (chat with genai Content shape, no `ReasoningEffort: "low"`) |
| `registry_test.go` | unit (pure + tape) | role resolution, nil-unset roles, `NewRegistry` from a test `config.Config` with per-provider taped `HTTPClient`, `Close()` |
| `embed_test.go` / `rerank_test.go` | unit (pure) | input validation, empty-input handling |
| `config_test.go` | unit (pure) | `NewRegistry` error paths: missing provider, unset role |
| consumer tests | existing replay | updated helpers, same tapes (sequential replay works regardless of body shape) |

**Testing `NewRegistry` without real HTTP:** build a `config.Config` with test
`LLMProviderConfig` entries (apikey auth, `AuthData["api_key"]="test-key"`), then for each
provider inject a taped `*http.Client` via the `RegistryOption` functional option
(`WithProviderTransport(name, *http.Client)`) which sets the `OpenAIConfig.HTTPClient`
for that provider. `NewRegistry` applies options before building clients, mirroring how
the current `ProviderConfig.HTTPClient` field works. This keeps `NewRegistry` testable
without a separate test-only constructor.

## ADK Agent Migration

Both `pkg/wiki/agent.go` and `pkg/web/agent.go` are currently hand-rolled
prompt-and-parse pipelines with single-shot LLM calls and Go code doing all
orchestration. Neither is a real agent loop. The rewrite migrates both to ADK's
`llmagent` framework, giving the LLM actual tools and letting ADK's `Flow.Run`
(`.extern/adk-go/internal/llminternal/base_flow.go:123-149`) handle the automatic
re-prompting loop.

### ADK agent mechanics (how it works)

An ADK LLM agent is constructed via `llmagent.New(llmagent.Config{Name, Model, Instruction,
Tools, ...})` and run via `runner.New(runner.Config{Agent, SessionService,
AutoCreateSession})`. The runner's `Run(ctx, userID, sessionID, msg, RunConfig{})`
returns `iter.Seq2[*session.Event, error]`.

The `Flow.Run` loop is automatic: `for { runOneStep(); if lastEvent.IsFinalResponse()
{ return } }`. One step = one LLM call + any tool calls. If the model returns
`FunctionCall` parts, ADK executes the tools, persists the function-call and
function-response events to the session, and re-prompts the model with the full history
(including tool results) on the next step. The loop terminates when the model produces a
plain text answer with no function calls. The caller never manages the loop.

Tools are defined via `functiontool.New[TArgs, TResults](cfg, handler)` - a Go function
wrapped as a tool with auto-inferred JSON schema. Tool results (`map[string]any`) are
fed back to the model automatically via session events.

### Current wiki agent (what we replace)

`pkg/wiki/agent.go` is not an agent - it is a fixed linear pipeline per workflow:

- **Ingest** (`agent.go:89-173`): `readSource` -> `loadIndexContext` ->
  `searchOverlap` (5 bigram TextSearch calls) -> `readWikiPage` for each overlap ->
  stuff all context into one system prompt -> **single `Chat` call** -> parse JSON ->
  Go code executes create/update/merge actions -> `generateDescription` (N more single
  `Chat` calls) -> `updateIndexFile` -> `appendLogFile`. No tools, no loop, no retry.
- **Query** (`agent.go:311-367`): `TextSearch` -> `readWikiPage` for each hit -> stuff
  all page bodies into system prompt -> **single `Chat` call**. No vector search, no
  rerank, no expansion, no truncation cap.
- **Lint** (`lint.go:197-268`): `lintStructure` (pure Go, no LLM) +
  `lintContent` (pairwise `Chat` calls, O(N^2) capped at 10 pages, parsed by string-
  contains "no contradictions found") + `lintGaps` (single `Chat` call, raw response).

Dead code to drop: `indexCache`, `LinksTo`, `Claims`, `ConceptKind`,
`IngestOpts.Batch`, `IngestOpts.Interactive`, `cleanJSON` (no longer needed - the
old code stripped DeepSeek `ihadk` reasoning blocks and ```json fences from a raw text
response to parse JSON; ADK tool calls return structured `FunctionCall` parts via
openai-go, so there is no raw text to clean), `extractKeyTerms` (the LLM searches
itself), `marshalYAML`
(replace with real YAML writer), blank imports of `chunking`/`search`.

### Wiki ADK agent design

Three ADK agents, each an `llmagent` with tools. All share the `wiki_schema.md`
instruction content as their base `Instruction`.

#### Wiki tools (shared across all three agents)

```go
// pkg/wiki/tools.go - ADK function tools wrapping existing Go functions

SearchWiki(tc agent.ToolContext, args SearchWikiArgs) (SearchWikiResult, error)
  // wraps tsClient.TextSearch (or hybrid) against the wiki's collection
  // args: { query: string, limit?: int }
  // returns: { results: [{ path, title, snippet, score }] }

ReadPage(tc agent.ToolContext, args ReadPageArgs) (ReadPageResult, error)
  // wraps readWikiPage - reads a wiki page, strips frontmatter
  // args: { path: string }
  // returns: { content: string, frontmatter: map[string]any }

ReadIndex(tc agent.ToolContext, args ReadIndexArgs) (ReadIndexResult, error)
  // wraps loadIndexContext - reads index.md
  // returns: { content: string }

CreatePage(tc agent.ToolContext, args CreatePageArgs) (CreatePageResult, error)
  // wraps createWikiPage - mkdir, write YAML frontmatter + body
  // args: { path, content, frontmatter }
  // returns: { path: string, created: bool }

UpdatePage(tc agent.ToolContext, args UpdatePageArgs) (UpdatePageResult, error)
  // wraps updateWikiPage - merge section or append content
  // args: { path, merge_section?, append_content? }
  // returns: { path: string, updated: bool }

UpdateIndex(tc agent.ToolContext, args UpdateIndexArgs) (UpdateIndexResult, error)
  // wraps updateIndexFile
  // args: { updates: [{ page, summary, category }] }

AppendLog(tc agent.ToolContext, args AppendLogArgs) (AppendLogResult, error)
  // wraps appendLogFile
  // args: { entry: string }

ListPages(tc agent.ToolContext, args ListPagesArgs) (ListPagesResult, error)
  // new - walks wiki dir, returns page list (currently only in lintStructure)
  // returns: { pages: [{ path, title, type }] }

IndexPage(tc agent.ToolContext, args IndexPageArgs) (IndexPageResult, error)
  // new - indexes a page into Typesense (currently done outside the package)
  // args: { path: string }
  // wraps the indexer for the wiki collection

ReadSource(tc agent.ToolContext, args ReadSourceArgs) (ReadSourceResult, error)
  // wraps readSource - reads a raw/ source file
  // args: { path: string }
```

#### Wiki Ingest agent

```go
// ptr returns a pointer to v. Helper for genai config fields that require *float32 etc.
func ptr[T any](v T) *T { return &v }

agent, _ := llmagent.New(llmagent.Config{
    Name:        "wiki-ingest",
    Model:       registry.Model(llm.RoleGeneralBig),   // the big model for complex extraction
    Instruction: ingestInstruction,                    // wiki_schema.md + ingest role + output format
    Tools: []tool.Tool{
        searchWikiTool, readPageTool, readIndexTool, readSourceTool,
        createPageTool, updatePageTool, updateIndexTool, appendLogTool,
        indexPageTool,
    },
    GenerateContentConfig: &genai.GenerateContentConfig{
        Temperature: ptr(float32(0.3)),   // deterministic-ish for extraction
    },
})
```

**Flow change:** instead of one giant JSON blob, the LLM reads the source (via
`ReadSource`), searches for overlap (via `SearchWiki`), reads existing pages (via
`ReadPage`), then creates/updates pages one at a time (via `CreatePage`/`UpdatePage`),
verifying as it goes. ADK's loop handles the iteration. `generateDescription` becomes
unnecessary - the LLM writes the `description` frontmatter directly when creating a page
(the old code generated it post-hoc because the LLM didn't have page-write access). The
agent calls `IndexPage` after writing each page, fixing the current gap where pages are
written but not indexed.

**Input:** user message = `"Ingest source: {sourcePath}"`. The agent calls `ReadSource`
to get the content.

**Termination:** the LLM calls `UpdateIndex` + `AppendLog`, then produces a final text
summary of what it did. ADK detects no more function calls -> `IsFinalResponse()` ->
loop exits.

#### Wiki Query agent

```go
agent, _ := llmagent.New(llmagent.Config{
    Name:        "wiki-query",
    Model:       registry.Model(llm.RoleGeneralMid),
    Instruction: queryInstruction,     // wiki_schema.md + query role + citation format
    Tools: []tool.Tool{
        searchWikiTool, readPageTool,
    },
    GenerateContentConfig: &genai.GenerateContentConfig{ Temperature: ptr(float32(0.2)) },
})
```

**Flow change:** the LLM calls `SearchWiki` with its own formulated query (not the raw
user question), reads the most relevant pages via `ReadPage`, and synthesizes an answer
with inline citations. If the first search misses, it can search again with refined
terms - ADK's loop handles this. No more stuffing all page bodies into one prompt; the
agent reads selectively.

**Input:** user message = the question.

**Termination:** the LLM produces a cited markdown answer with no function calls.

#### Wiki Lint agent

```go
agent, _ := llmagent.New(llmagent.Config{
    Name:        "wiki-lint",
    Model:       registry.Model(llm.RoleGeneralMid),
    Instruction: lintInstruction,      // wiki_schema.md + lint role
    Tools: []tool.Tool{
        listPagesTool, readPageTool, readIndexTool,
    },
})
```

**Flow change:** instead of brute-force O(N^2) pairwise comparisons capped at 10 pages,
the LLM calls `ListPages` to see the full wiki, calls `ReadIndex` to understand
structure, then selectively reads pages it suspects of contradicting or overlapping.
It decides its own comparison strategy. `lintStructure` (orphan/broken-link/stale
detection) stays as pure Go - it runs before the agent as a pre-pass, and its results
are injected into the agent's user message as context.

#### Wiki runner wiring

```go
// pkg/wiki/runner.go - shared runner factory

type Runner struct {
    runner     *runner.Runner
    wiki       *Wiki
    tsClient   *ts.Client
    indexer    *indexer.Indexer
}

// NewRunner builds the three llmagent agents (ingest, query, lint) with their tools,
// composing them as sub-agents under a root "wiki" agent that transfers to the
// appropriate sub-agent based on the user message. Tools capture tsClient/wiki/indexer
// in their closures.
func NewRunner(registry *llm.Registry, wiki *Wiki, tsClient *ts.Client,
    idx *indexer.Indexer, sessionSvc session.Service) (*Runner, error)
```

The three agents are composed under a root agent via `SubAgents` + ADK's agent-transfer
mechanism. The root agent's instruction routes to the right sub-agent based on the
task. Alternatively, expose three separate runners - but composition under one root is
cleaner for the CLI (one runner, one session, route by user message).

**Result assembly (not text parsing):** the `IngestReport`/`QueryResult`/`LintResult`
are NOT parsed from the agent's final text response (that would reintroduce the fragile
text parsing the migration eliminates). Instead, the tools write structured results into
session state (`tc.Actions().StateDelta["created_pages"] = [...]` etc., where `tc` is the
`agent.ToolContext` parameter), and the runner reads state after the run completes. The
agent's final text response is a human-readable summary shown to the CLI user; the typed
report is assembled from tool-emitted state.

Each CLI command (`gmd wiki ingest`, `gmd wiki query`, `gmd wiki lint`) creates a runner
+ session, sends the appropriate user message, and collects events until the final
response. `session.InMemoryService()` is sufficient (no cross-session persistence
needed for wiki workflows). `AutoCreateSession: true` simplifies the call.

**Message construction:** `runner.Run` takes `msg *genai.Content`, not a string. The
runner wraps user message strings via `genai.NewContentFromText(msg, genai.RoleUser)`
before passing to `Run`. The `agent.RunConfig{}` is passed as-is (streaming mode
defaults to `StreamingModeNone`).

**Public API:** the `Agent` struct retains its `Ingest(ctx, sourcePath) (*IngestReport,
error)`, `Query(ctx, question) (*QueryResult, error)`, `Lint(ctx) (*LintResult, error)`
method signatures. Internally each creates a session, runs the ADK runner, collects the
final event text + session state, and assembles the typed result. Callers (`cmd/gmd/
wiki_*`, `pkg/mcp/wiki_tools`, `pkg/wiki/watch`) see the same API - only the constructor
changes (`NewAgent` takes `*Runner` or the dependencies to build one).

### Current web agent (what we replace)

`pkg/web/agent.go` is a Go-orchestrated loop:

1. EXA search (raw `*exa.Client`, bypasses `SearchProvider` interface) - unconditional
2. Loop (`maxSteps-1` times): `Chat` call -> parse free-text `## ACTION`/`## QUERIES`
   -> if `SEARCH_MORE`, Go executes each query string as a new EXA search
3. Optional batched `/contents` fetch (all URLs at once, index-parallel patch - fragile)
4. `Summarize` call -> final answer

The LLM never calls search as a tool. It outputs query strings that Go executes. Fetch
is all-or-nothing, post-loop. The decision mechanism is free-text parsing (broken: empty
`SEARCH_MORE` with no queries loops until `maxSteps`).

### Web ADK agent design

One ADK agent with `web_search` and `web_fetch` tools. The LLM drives the entire loop.

#### Web tools

```go
// pkg/web/tools.go - ADK function tools

WebSearch(tc agent.ToolContext, args WebSearchArgs) (WebSearchResult, error)
  // wraps the SearchProvider interface (NOT raw exa.Client)
  // calls all configured SearchProviders in parallel, merges + dedups inline
  // (NOT via pkg/web/fusion - that would create an import cycle web->fusion->web.
  //  The fan-out/dedup logic is implemented directly in tools.go or a shared
  //  helper in pkg/web that fusion also imports, avoiding the cycle.)
  // args: { query: string, num_results?: int }
  // returns: { results: [{ title, url, snippet, score }] }

WebFetch(tc agent.ToolContext, args WebFetchArgs) (WebFetchResult, error)
  // wraps the BrowserProvider interface (NOT raw exa /contents)
  // args: { url: string, max_chars?: int }
  // returns: { title, content, content_type }
```

Key change: the agent routes through the unified `SearchProvider`/`BrowserProvider`
interfaces, not the raw EXA client. This gives multi-provider support (EXA, Cloudflare,
Tavily, SearXNG) for free, and makes the agent provider-agnostic. The fan-out/dedup
logic is implemented inline in `pkg/web/tools.go` (or a shared `pkg/web/searchutil`
helper that both `tools.go` and `fusion` import), NOT by calling `pkg/web/fusion`
directly - that would create an import cycle (`web` -> `fusion` -> `web`).

#### Web research agent

```go
agent, _ := llmagent.New(llmagent.Config{
    Name:        "web-research",
    Model:       registry.Model(llm.RoleGeneralBig),
    Instruction: researchInstruction,  // de-duplicated synthesis guidelines from
                                        // agent_system.md + agent_synthesize.md
    Tools: []tool.Tool{
        webSearchTool, webFetchTool,
    },
    GenerateContentConfig: &genai.GenerateContentConfig{
        Temperature: ptr(float32(0.4)),
    },
})
```

**Flow change:** the LLM calls `WebSearch` with the user's question, reads snippets,
decides whether to fetch full content via `WebFetch` for specific URLs, searches again
with refined queries if needed, and synthesizes a cited answer. ADK's loop handles all
iteration. No `maxSteps` cap (the LLM terminates when satisfied) - or enforce one via a
`BeforeModelCallback` that counts steps and forces termination. No more `## ACTION`/
`## QUERIES` free-text parsing. No more batched all-or-nothing fetch.

**Input:** user message = the research question.

**Termination:** the LLM produces a cited markdown answer with no function calls.

#### Web fusion (not an agent, stays single-shot)

`pkg/web/fusion/fusion.go` is not an agent loop - it is parallel fan-out + dedup + one
synthesis call. It stays as-is, but its `Config.LLMClient *llm.Client` becomes
`Config.Summarizer llm.ChatModel`, and its `Summarize` calls become `.Chat`. The
`dedupLLM` and `Synthesize` functions are single-shot and do not benefit from becoming
ADK agents.

### Agent migration tests

- **Mock `model.LLM` for unit tests.** ADK's `model.LLM` is a 2-method interface. A
  test mock yields scripted `*model.LLMResponse` values (first a `FunctionCall` to
  simulate tool selection, then a text response to simulate the final answer). This
  tests the agent loop, tool dispatch, and event collection without real HTTP or tapes.
  The mock is defined in `pkg/wiki/agent_test.go` / `pkg/web/agent_test.go`.
- **Tool dispatch tests.** Each `functiontool.New` wrapper has a unit test verifying
  the handler is invoked with correctly marshaled args and returns the expected
  `map[string]any`. The `functiontool` schema inference (JSON schema from Go struct
  types) is tested by checking `Declaration()` produces the expected parameter names.
- **Loop termination tests.** A mock `model.LLM` that always returns `FunctionCall`
  (never terminates) + a `BeforeModelCallback` step cap (default 20) verifies the loop
  terminates. Test that the cap is hit and a meaningful error/result is produced.
- **Event collection tests.** Verify the runner collects `iter.Seq2[*session.Event,
  error]` correctly: tool-call events, function-response events, and the final text
  event. Verify `IngestReport`/`QueryResult`/`LintResult` are assembled from session
  state (not text parsing).
- **Integration tests (tapes regenerated).** Wiki agent: existing tape files
  (`ingest_flow.json`, `query_flow.json`, `lint_content.json`) are regenerated. The new
  agent makes multiple LLM calls per workflow (one per ADK loop step), so tapes have
  more exchanges. Replay is sequential (position-based). Web agent: new tapes for the
  multi-step search/fetch/synthesize loop. All old tapes are stale and replaced.

### Agent migration risks

1. **ADK loop termination.** The LLM agent loop has no built-in max-iterations cap
   (unlike `loopagent`). A confused model could loop forever calling tools. Mitigation:
   a `BeforeModelCallback` that counts steps and forces termination after N (configurable,
   default 20). Or wrap the `llmagent` in a `loopagent.New` with `MaxIterations`.

2. **Structured output.** The current ingest agent expects a JSON response. ADK's
   tool-calling replaces this - the LLM creates pages one at a time via `CreatePage`
   tool calls, not a JSON blob. This is a workflow change: the `IngestReport` is
   assembled from the tool-call events (which pages were created/updated), not parsed
   from a JSON response. The final text response is a human-readable summary.

3. **Context window.** The current query agent stuffs all page bodies into one prompt.
   The ADK agent reads pages selectively via `ReadPage`, but the session history grows
   with each tool call. For large wikis with many search/read cycles, the history could
   exceed the context window. Mitigation: `IncludeContents: "none"` is too aggressive
   (loses tool results); instead, rely on the model's context window and cap
   `SearchWiki` results. ADK's `ContentsRequestProcessor` includes full history - if
   this becomes a problem, a custom `RequestProcessor` could truncate old events.

4. **Session lifecycle.** Wiki/web workflows are single-turn (one `Run` call). Sessions
   are created per invocation and discarded. `session.InMemoryService()` is sufficient.
   No cross-session memory needed.

5. **Provider migration for web agent.** The web agent currently uses raw `*exa.Client`.
   Routing through `SearchProvider` changes the search behavior (multi-provider fan-out
   instead of EXA-only). This is an improvement but changes test expectations - tapes
   must reflect the new provider fan-out shape.

## Resolved Decisions

1. **ADK agent migration scope.** IN SCOPE. `pkg/wiki/agent.go` and `pkg/web/agent.go`
   are migrated to ADK's `llmagent` framework as part of this rewrite. See the "ADK Agent
   Migration" section above.

2. **Role sizes.** All role sizes (`general_big`/`general_mid`/`general_small`) are kept
   as separate roles. No consolidation.

3. **Gemini/Vertex.** Out of scope. No Gemini/Vertex native provider support. Only
   OpenAI-compatible endpoints. (ADK itself is still used as the agent framework.)

4. **`Features` field.** Dropped. The `LLMProviderConfig.Features` (`embed`/`chat`/`rerank`
   booleans) is code rot - there is no feature overlap possible. Remove it from the
   config schema and Go structs. No validation needed.

5. **Tapes.** All tapes and recordings MUST be regenerated. This is a major refactor;
   old tapes are stale. The replay mechanism is sequential (position-based,
   `tape.go:184-200`), so old tapes would technically replay, but the captured request
   bodies no longer reflect the new adapter's shape. Regenerate everything in the
   integration phase.
