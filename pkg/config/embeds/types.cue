package gmd

// LLMProviderConfig defines a named LLM service endpoint.
LLMProviderConfig: {
	provider:       "openai" | "anthropic" | "vertex" | "opencode" | string
	base_url?:      string
	auth:           "none" | "apikey" | "service-account" | *"apikey"
	project_id?:    string
	location?:      string
	features?: {
		embed?:  bool | *true
		chat?:   bool | *true
		rerank?: bool | *false
	}
}

// LLMRoleConfig maps a role to a provider + model.
LLMRoleConfig: {
	provider?:  string
	model?:     string
}

// LLMProfile bundles all roles into a named preset.
LLMProfile: {
	embedding?:       LLMRoleConfig
	expansion?:       LLMRoleConfig
	rerank?:          LLMRoleConfig
	summarizing?:     LLMRoleConfig
	general_big?:     LLMRoleConfig
	general_mid?:     LLMRoleConfig
	general_small?:   LLMRoleConfig
}

// LLMConfig defines the OpenAI-compatible provider settings.
LLMConfig: {
	providers?:  [string]: LLMProviderConfig
	profile?:    string | *"default"
	profiles?:   [string]: LLMProfile
}

// TypesenseConfig defines the search engine connection settings.
TypesenseConfig: {
	host?:    string
}

// ChunkConfig defines heading-aware chunking parameters.
ChunkConfig: {
	targetTokens:     int | *900
	overlap:          number | *0.15
	headingWeights: {
		h1: int | *100
		h2: int | *90
		h3: int | *80
		h4: int | *70
		h5: int | *60
		h6: int | *50
	}
	codeFenceWeight: int    | *10
	newlineWeight:   number | *1
}

// StrongSignalConfig defines the heuristic for detecting strong query signals.
StrongSignalConfig: {
	minScore: number | *0.85
	minGap:   number | *0.15
}

// RRFConfig defines rank fusion parameters.
RRFConfig: {
	k:               int | *60
	originalWeight:  number | *2.0
	expansionWeight: number | *1.0
}

// RerankConfig defines reranking pipeline parameters.
RerankConfig: {
	candidateLimit: int    | *40
	contextSize:    int    | *4096
}

// BlendingConfig defines position-aware blending thresholds and weights.
BlendingConfig: {
	thresholds: {
		top:    int | *3
		middle: int | *10
	}
	weights: {
		top:    number | *0.75
		middle: number | *0.60
		bottom: number | *0.40
	}
}

// OutputConfig defines result formatting parameters.
OutputConfig: {
	defaultFormat: "cli" | "json" | "csv" | "md" | "xml" | "files" | *"cli"
	maxResults:    int | *5
}

// PipelineConfig groups all pipeline knobs.
PipelineConfig: {
	chunk:        ChunkConfig
	strongSignal: StrongSignalConfig
	rrf:          RRFConfig
	rerank:       RerankConfig
	blending:     BlendingConfig
	output:       OutputConfig
}

// FrontmatterField defines a typed key extracted from YAML frontmatter.
// The type names align with Typesense field types (with Go YAML parsing in mind):
//   string  — YAML string
//   string[] — YAML string array
//   int32   — YAML int
//   float   — YAML float64
//   bool    — YAML bool
FrontmatterField: {
	type:  "string" | "string[]" | "int32" | "float" | "bool"
	facet?: bool | *false
	sort?:  bool | *false
}

// Source defines shared file-indexing configuration used by both
// collections and wikis. Both entity types are indexed into the same
// Typesense chunks collection.
Source: {
	path:     string
	patterns: [...string]
	ignore?:  [...string]
	context?: string
	fields?:  [string]: FrontmatterField
}

// WikiConfig defines an LLM wiki — a compounding knowledge base with
// agent-driven content generation, wikilinks, and optional collection aggregation.
// Collection commands (show, include, exclude, context) accept wiki names
// identically. Wiki CLI commands delegate to the same collection CRUD internals.
WikiConfig: Source & {
	wikiDir:     string | *"wiki"        // subdirectory for wiki content pages
	rawDir:      string | *"raw"         // subdirectory for raw source material
	indexFile:   string | *"index.md"
	logFile:     string | *"log.md"
	okfVersion:  string | *"0.1"
	graphLinks:  bool | *true
	excludeFromDefault?: bool | *false    // opt-out of default (unscoped) searches

	// Aggregation: when searching this wiki, also search these named sources
	// (collections or other wikis). Each entry must be a key in the top-level
	// collections or wikis map — validation at create/add time rejects
	// unknown names and circular references.
	sourceRefs?: [...string]

	// OKF frontmatter fields. Only type is required (OKF §4.1).
	// Unknown fields pass through via ... per OKF extensibility.
	frontmatter?: {
		type!:        string               // REQUIRED by OKF — any non-empty string
		title?:       string
		description?: string
		resource?:    string
		tags?:        [...string]
		timestamp?:   string
		status?:      string               // free-form, no fixed enum
		sources?:     [...string]
		...                                // unknown fields pass through (OKF §4.1)
	}
}

// CollectionConfig defines a document collection to index.
CollectionConfig: Source & {
	excludeFromDefault?: bool | *false
}

// EXAConfig defines the EXA web search API settings.
EXAConfig: {
	api_key: string | *"" // from EXA_API_KEY env var
}

// TavilyConfig defines the Tavily search API settings.
TavilyConfig: {
	api_key: string | *"" // from TAVILY_API_KEY env var
}

// SearXNGConfig defines the SearXNG search API settings.
SearXNGConfig: {
	base_url: string | *"" // from SEARXNG_BASE_URL env var
	engines?:  string | *"" // comma-separated engine list (e.g. "google,bing")
}

// LocalConfig defines local execution settings.
LocalConfig: {
	chromium_path?: string | *""
	no_browser?:    bool   | *false
	html_max_size?: int    | *10485760
	crawl_delay_ms?: int | *1000
	max_concurrent_domains?: int | *2
	max_pages_per_domain?: int | *200
	cache_enabled?:  bool   | *false
	cache_dir?:      string | *"~/.cache/gmd/web"
	cache_max_size?: int    | *536870912
	cache_ttl?:      string | *"24h"
}

// CloudflareConfig defines Cloudflare Browser Run settings.
CloudflareConfig: {
	api_key:    string | *"" // from CLOUDFLARE_API_KEY env var
	account_id: string | *"" // from CLOUDFLARE_ACCOUNT_ID env var
}

// WebProviderGroup maps a preset name to search/browser provider selections.
WebProviderGroup: {
	search?:  [...string]
	browser?: string
}

// WebSearchConfig controls multi-provider search behavior.
WebSearchConfig: {
	dedup:      string | *"heuristic"   // "heuristic", "llm", "none"
	synthesize: bool   | *false           // synthesize results via LLM
	synthesis_prompt:  string | *""      // path to custom system prompt file
}

// WebConfig groups all web search provider configurations.
WebConfig: {
	group?:  string | *"default"
	groups?: [string]: WebProviderGroup
	exa?:       EXAConfig
	tavily?:    TavilyConfig
	searxng?:   SearXNGConfig
	local?:     LocalConfig
	cloudflare?: CloudflareConfig
	search:    WebSearchConfig
}

// AgentHarnessConfig defines a named executable harness (e.g., opencode, claude).
AgentHarnessConfig: {
	bin:        string
	flagStyle?: string | *"double-dash"
	env?:       [string]: string
}

// AgentHarnessProfile defines a named preset for launching a harness.
AgentHarnessProfile: {
	harness?:    string
	message?:    string
	flags?:      [string]: string
	args?:       [...string]
	env?:        [string]: string
	configFile?: string
	cwd?:        string
	tmux?:       bool | *false
	workspace?:  bool | *false
	async?:      bool | *false
}

// AgentConfig defines configuration for launching external AI agent harnesses.
AgentConfig: {
	defaultHarness?: string
	harnesses?:      [string]: AgentHarnessConfig
	profiles?:       [string]: AgentHarnessProfile
}

// ProjectConfig is the root configuration object.
ProjectConfig: {
	project?:    string
	llm?:        LLMConfig
	typesense?:  TypesenseConfig
	web?:        WebConfig
	pipeline?:   PipelineConfig
	agent?:      AgentConfig
	collections: [string]: CollectionConfig
	wikis:       [string]: WikiConfig

	// searchDefaults defines named search presets. Each key is a preset name
	// used with --search, and the value is the list of source names
	// (collections and/or wikis) to search in that preset. When a search uses
	// --search <preset>, only the listed sources are included, overriding
	// the default behavior. When --search is not used, unscoped search
	// includes all sources where excludeFromDefault is false. searchDefaults
	// does NOT intersect with or override excludeFromDefault for unscoped
	// searches — it only takes effect when explicitly invoked via --search.
	searchDefaults?: [string]: [...string]
}
