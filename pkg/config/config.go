package config

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SourceConfig holds file-indexing fields shared by collections and wikis.
type SourceConfig struct {
	Path     string                      `json:"path"`
	Patterns []string                    `json:"patterns"`
	Ignore   []string                    `json:"ignore,omitempty"`
	Context  string                      `json:"context,omitempty"`
	Fields   map[string]FrontmatterField `json:"fields,omitempty"`
}

// Config is the validated Go representation of the unified CUE configuration.
type Config struct {
	LLM            LLMConfig                   `json:"llm"`
	Typesense      TypesenseConfig             `json:"typesense"`
	Web            WebConfig                   `json:"web,omitempty"`
	Pipeline       PipelineConfig              `json:"pipeline"`
	Collections    map[string]CollectionConfig `json:"collections"`
	Wikis          map[string]WikiConfig       `json:"wikis"`
	SearchDefaults map[string][]string         `json:"searchDefaults,omitempty"`
	ProjectRoot    string                      `json:"-"`
	Project        string                      `json:"project,omitempty"`
}

// WebConfig groups all web search provider configurations.
type WebConfig struct {
	Group      string                      `json:"group"`
	Groups     map[string]WebProviderGroup `json:"groups,omitempty"`
	EXA        EXAConfig                   `json:"exa,omitempty"`
	Tavily     TavilyConfig                `json:"tavily,omitempty"`
	SearXNG    SearXNGConfig               `json:"searxng,omitempty"`
	Local      LocalConfig                 `json:"local,omitempty"`
	Cloudflare CloudflareConfig            `json:"cloudflare,omitempty"`
	Search     WebSearchConfig             `json:"search,omitempty"`
}

// EXAConfig maps from the CUE EXAConfig schema.
type EXAConfig struct {
	APIKey string `json:"-"`
}

// LocalConfig maps from the CUE LocalConfig schema.
type LocalConfig struct {
	ChromiumPath         string `json:"chromium_path,omitempty"`
	NoBrowser            bool   `json:"no_browser,omitempty"`
	HTMLMaxSize          int    `json:"html_max_size,omitempty"`
	CrawlDelayMs         int    `json:"crawl_delay_ms,omitempty"`
	MaxConcurrentDomains int    `json:"max_concurrent_domains,omitempty"`
	MaxPagesPerDomain    int    `json:"max_pages_per_domain,omitempty"`
	CacheEnabled         bool   `json:"cache_enabled,omitempty"`
	CacheDir             string `json:"cache_dir,omitempty"`
	CacheMaxSize         int    `json:"cache_max_size,omitempty"`
	CacheTTL             string `json:"cache_ttl,omitempty"`
}

// CloudflareConfig maps from the CUE CloudflareConfig schema.
type CloudflareConfig struct {
	APIKey    string `json:"-"`
	AccountID string `json:"-"`
}

// TavilyConfig maps from the CUE TavilyConfig schema.
type TavilyConfig struct {
	APIKey string `json:"-"`
}

// SearXNGConfig maps from the CUE SearXNGConfig schema.
type SearXNGConfig struct {
	BaseURL string `json:"base_url,omitempty"`
}

// WebProviderGroup maps a preset name to search/browser provider selections.
type WebProviderGroup struct {
	Search  []string `json:"search,omitempty"`
	Browser string   `json:"browser,omitempty"`
}

// WebSearchConfig controls multi-provider search behavior.
type WebSearchConfig struct {
	Dedup           string `json:"dedup,omitempty"`
	Synthesize      bool   `json:"synthesize"`
	SynthesisPrompt string `json:"synthesis_prompt,omitempty"`
}

// ResolveProvider resolves the provider name for a given role using the active group
// or defaults. role is "search" or "browser". For "search", use ResolveSearchProviders instead.
func (w WebConfig) ResolveProvider(role string, cmdOverride string) string {
	if cmdOverride != "" {
		return cmdOverride
	}
	groupName := w.Group
	if groupName == "" {
		groupName = "default"
	}
	if g, ok := w.Groups[groupName]; ok {
		switch role {
		case "search":
			if len(g.Search) > 0 {
				return g.Search[0]
			}
		case "browser":
			if g.Browser != "" {
				return g.Browser
			}
		}
	}
	switch role {
	case "search":
		return "exa"
	case "browser":
		return "exa"
	}
	return ""
}

// ResolveSearchProviders resolves the list of search providers from the active group
// or defaults. cmdOverride is a comma-separated list of provider names.
func (w WebConfig) ResolveSearchProviders(cmdOverride string) []string {
	if cmdOverride != "" {
		return splitAndTrim(cmdOverride, ",")
	}
	groupName := w.Group
	if groupName == "" {
		groupName = "default"
	}
	if g, ok := w.Groups[groupName]; ok {
		if len(g.Search) > 0 {
			return g.Search
		}
	}
	return []string{"exa"}
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return []string{"exa"}
	}
	return result
}

// CollectionKey returns the project-prefixed key for a collection name.
// This avoids name collisions on shared Typesense instances.
func (c *Config) CollectionKey(name string) string {
	if c.Project == "" {
		return name
	}
	return c.Project + "-" + name
}

// IsWiki reports whether name is a wiki (not a collection).
func (c *Config) IsWiki(name string) bool {
	_, ok := c.Wikis[name]
	return ok
}

// IsCollection reports whether name is a collection (not a wiki).
func (c *Config) IsCollection(name string) bool {
	_, ok := c.Collections[name]
	return ok
}

// SourceKeysForSearch returns the set of Typesense collection keys to query
// when searching a named source. If the source is a wiki with sourceRefs,
// the result includes the wiki's own key plus keys for all referenced sources.
// Returns an error if any referenced source does not exist in collections or wikis.
func (c *Config) SourceKeysForSearch(name string) ([]string, error) {
	keys := []string{c.CollectionKey(name)}
	wc, ok := c.Wikis[name]
	if !ok {
		return keys, nil
	}
	for _, ref := range wc.SourceRefs {
		if _, ok := c.Collections[ref]; ok {
			keys = append(keys, c.CollectionKey(ref))
		} else if _, ok := c.Wikis[ref]; ok {
			keys = append(keys, c.CollectionKey(ref))
		} else {
			return nil, fmt.Errorf("wiki %q references source %q which does not exist in collections or wikis", name, ref)
		}
	}
	return keys, nil
}

// HasSourceRefsCycle performs a global DFS from every wiki to detect any cycle.
// Returns the first cycle path found, or nil if the graph is acyclic.
func (c *Config) HasSourceRefsCycle() ([]string, bool) {
	for name := range c.Wikis {
		visited := make(map[string]bool)
		path := make([]string, 0)
		if cycle, found := c.dfsSourceRefs(name, visited, path); found {
			return cycle, true
		}
	}
	return nil, false
}

func (c *Config) dfsSourceRefs(current string, visited map[string]bool, path []string) ([]string, bool) {
	if visited[current] {
		cycle := make([]string, 0)
		inCycle := false
		for _, p := range path {
			if p == current {
				inCycle = true
			}
			if inCycle {
				cycle = append(cycle, p)
			}
		}
		cycle = append(cycle, current)
		return cycle, true
	}

	wc, ok := c.Wikis[current]
	if !ok {
		return nil, false
	}

	visited[current] = true
	path = append(path, current)

	for _, ref := range wc.SourceRefs {
		if _, ok := c.Wikis[ref]; ok {
			if cycle, found := c.dfsSourceRefs(ref, visited, path); found {
				return cycle, true
			}
		}
	}

	visited[current] = false
	return nil, false
}

// WouldCreateSourceRefsCycle checks whether adding edge src -> target would create a
// cycle. DFS from target to see if src is reachable.
func (c *Config) WouldCreateSourceRefsCycle(src, target string) bool {
	visited := make(map[string]bool)
	return c.isReachable(target, src, visited)
}

func (c *Config) isReachable(from, target string, visited map[string]bool) bool {
	if from == target {
		return true
	}
	if visited[from] {
		return false
	}
	wc, ok := c.Wikis[from]
	if !ok {
		return false
	}
	visited[from] = true
	for _, ref := range wc.SourceRefs {
		if _, ok := c.Wikis[ref]; ok {
			if c.isReachable(ref, target, visited) {
				return true
			}
		}
	}
	return false
}

// AllSearchableSources returns the names of all collections and wikis
// where excludeFromDefault is false.
func (c *Config) AllSearchableSources() []string {
	var sources []string
	for name, col := range c.Collections {
		if !col.ExcludeFromDefault {
			sources = append(sources, name)
		}
	}
	for name, wc := range c.Wikis {
		if !wc.ExcludeFromDefault {
			sources = append(sources, name)
		}
	}
	return sources
}

// ResolvedWikiDir returns the absolute path for a wiki's wikiDir.
func (c *Config) ResolvedWikiDir(name string) string {
	wc, ok := c.Wikis[name]
	if !ok {
		return ""
	}
	return c.resolveSubDir(wc, wc.WikiDir)
}

// ResolvedRawDir returns the absolute path for a wiki's rawDir.
func (c *Config) ResolvedRawDir(name string) string {
	wc, ok := c.Wikis[name]
	if !ok {
		return ""
	}
	return c.resolveSubDir(wc, wc.RawDir)
}

// ResolvedCollectionPath returns the absolute path for a collection.
func (c *Config) ResolvedCollectionPath(name string) string {
	col, ok := c.Collections[name]
	if !ok {
		return ""
	}
	colPath := col.Path
	if !filepath.IsAbs(colPath) {
		colPath = filepath.Join(c.ProjectRoot, colPath)
	}
	return filepath.Clean(colPath)
}

func (c *Config) resolveSubDir(wc WikiConfig, subDir string) string {
	basePath := wc.Path
	if !filepath.IsAbs(basePath) {
		basePath = filepath.Join(c.ProjectRoot, basePath)
	}
	return filepath.Clean(filepath.Join(basePath, subDir))
}

// HasWikiDirRawDirCollision checks whether a wiki's wikiDir and rawDir are the same.
func (c *Config) HasWikiDirRawDirCollision(name string) bool {
	wc, ok := c.Wikis[name]
	if !ok {
		return false
	}
	return wc.WikiDir == wc.RawDir
}

// FindPathConflicts checks whether a wiki's wikiDir and rawDir overlap with
// any existing collection or other wiki. Returns a list of conflict descriptions.
func (c *Config) FindPathConflicts(name string) []string {
	wc, ok := c.Wikis[name]
	if !ok {
		return nil
	}
	wikiDirAbs := c.resolveSubDir(wc, wc.WikiDir)
	rawDirAbs := c.resolveSubDir(wc, wc.RawDir)

	var conflicts []string

	// Check against other wikis
	for otherName, otherWC := range c.Wikis {
		if otherName == name {
			continue
		}
		otherWikiDir := c.resolveSubDir(otherWC, otherWC.WikiDir)
		otherRawDir := c.resolveSubDir(otherWC, otherWC.RawDir)
		if pathsOverlap(wikiDirAbs, otherWikiDir) || pathsOverlap(wikiDirAbs, otherRawDir) {
			conflicts = append(conflicts, fmt.Sprintf("%s wikiDir overlaps with wiki %q", name, otherName))
		}
		if pathsOverlap(rawDirAbs, otherWikiDir) || pathsOverlap(rawDirAbs, otherRawDir) {
			conflicts = append(conflicts, fmt.Sprintf("%s rawDir overlaps with wiki %q", name, otherName))
		}
	}

	// Check against collections
	for colName := range c.Collections {
		colPath := c.ResolvedCollectionPath(colName)
		if pathsOverlap(wikiDirAbs, colPath) || pathsOverlap(rawDirAbs, colPath) {
			conflicts = append(conflicts, fmt.Sprintf("wiki %q directory overlaps with collection %q", name, colName))
		}
	}

	return conflicts
}

func pathsOverlap(a, b string) bool {
	a = filepath.Clean(a) + string(filepath.Separator)
	b = filepath.Clean(b) + string(filepath.Separator)
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}

// SourceExists returns true if a name exists in either collections or wikis.
func (c *Config) SourceExists(name string) bool {
	_, inCol := c.Collections[name]
	_, inWiki := c.Wikis[name]
	return inCol || inWiki
}

// LLMConfig maps from the CUE LLMConfig schema.
type LLMConfig struct {
	APIKey string `json:"-"`

	EmbeddingModel   string `json:"embedding_model"`
	EmbeddingBaseURL string `json:"embedding_base_url"`
	EmbeddingAPIKey  string `json:"embedding_api_key"`

	ExpansionModel   string `json:"expansion_model"`
	ExpansionBaseURL string `json:"expansion_base_url"`
	ExpansionAPIKey  string `json:"expansion_api_key"`

	RerankModel   string `json:"rerank_model"`
	RerankBaseURL string `json:"rerank_base_url"`
	RerankAPIKey  string `json:"rerank_api_key"`

	SummarizingModel   string `json:"summarizing_model"`
	SummarizingBaseURL string `json:"summarizing_base_url"`
	SummarizingAPIKey  string `json:"summarizing_api_key"`

	GeneralBigModel   string `json:"general_big_model"`
	GeneralBigBaseURL string `json:"general_big_base_url"`
	GeneralBigAPIKey  string `json:"general_big_api_key"`

	GeneralMidModel   string `json:"general_mid_model"`
	GeneralMidBaseURL string `json:"general_mid_base_url"`
	GeneralMidAPIKey  string `json:"general_mid_api_key"`

	GeneralSmallModel   string `json:"general_small_model"`
	GeneralSmallBaseURL string `json:"general_small_base_url"`
	GeneralSmallAPIKey  string `json:"general_small_api_key"`
}

// TypesenseConfig maps from the CUE TypesenseConfig schema.
type TypesenseConfig struct {
	Host   string `json:"host"`
	APIKey string `json:"-"`
}

// ChunkConfig maps from the CUE ChunkConfig schema.
type ChunkConfig struct {
	TargetTokens    int            `json:"targetTokens"`
	Overlap         float64        `json:"overlap"`
	HeadingWeights  HeadingWeights `json:"headingWeights"`
	CodeFenceWeight int            `json:"codeFenceWeight"`
	NewlineWeight   float64        `json:"newlineWeight"`
}

// HeadingWeights maps heading-level breakpoint scores.
type HeadingWeights struct {
	H1 int `json:"h1"`
	H2 int `json:"h2"`
	H3 int `json:"h3"`
	H4 int `json:"h4"`
	H5 int `json:"h5"`
	H6 int `json:"h6"`
}

// StrongSignalConfig maps from the CUE StrongSignalConfig schema.
type StrongSignalConfig struct {
	MinScore float64 `json:"minScore"`
	MinGap   float64 `json:"minGap"`
}

// RRFConfig maps from the CUE RRFConfig schema.
type RRFConfig struct {
	K               int     `json:"k"`
	OriginalWeight  float64 `json:"originalWeight"`
	ExpansionWeight float64 `json:"expansionWeight"`
}

// RerankConfig maps from the CUE RerankConfig schema.
type RerankConfig struct {
	CandidateLimit int `json:"candidateLimit"`
	ContextSize    int `json:"contextSize"`
}

// BlendingConfig maps from the CUE BlendingConfig schema.
type BlendingConfig struct {
	Thresholds BlendingThresholds `json:"thresholds"`
	Weights    BlendingWeights    `json:"weights"`
}

type BlendingThresholds struct {
	Top    int `json:"top"`
	Middle int `json:"middle"`
}

type BlendingWeights struct {
	Top    float64 `json:"top"`
	Middle float64 `json:"middle"`
	Bottom float64 `json:"bottom"`
}

// OutputConfig maps from the CUE OutputConfig schema.
type OutputConfig struct {
	DefaultFormat string `json:"defaultFormat"`
	MaxResults    int    `json:"maxResults"`
}

// PipelineConfig maps from the CUE PipelineConfig schema.
type PipelineConfig struct {
	Chunk        ChunkConfig        `json:"chunk"`
	StrongSignal StrongSignalConfig `json:"strongSignal"`
	RRF          RRFConfig          `json:"rrf"`
	Rerank       RerankConfig       `json:"rerank"`
	Blending     BlendingConfig     `json:"blending"`
	Output       OutputConfig       `json:"output"`
}

// WikiConfig maps from the CUE WikiConfig schema.
type WikiConfig struct {
	SourceConfig
	WikiDir            string             `json:"wikiDir"`
	RawDir             string             `json:"rawDir"`
	IndexFile          string             `json:"indexFile"`
	LogFile            string             `json:"logFile"`
	GraphLinks         bool               `json:"graphLinks"`
	ExcludeFromDefault bool               `json:"excludeFromDefault"`
	SourceRefs         []string           `json:"sourceRefs,omitempty"`
	Frontmatter        *FrontmatterConfig `json:"frontmatter,omitempty"`
}

// FrontmatterConfig maps from the CUE frontmatter field config.
type FrontmatterConfig struct {
	Fields map[string]FrontmatterField `json:"fields"`
}

// FrontmatterField maps from the CUE FrontmatterField schema.
// The Type field uses Typesense-compatible type names (string, string[], int32, float, bool).
type FrontmatterField struct {
	Type  string `json:"type"`
	Facet bool   `json:"facet"`
	Sort  bool   `json:"sort"`
}

// CollectionConfig maps from the CUE CollectionConfig schema.
// The collection name is the map key in Config.Collections, not a field in this struct.
type CollectionConfig struct {
	SourceConfig
	ExcludeFromDefault bool `json:"excludeFromDefault"`
}

//go:embed embeds
var configEmbedsFS embed.FS

// Load loads and validates the unified configuration.
// It embeds the built-in schema, loads optional global config (UserConfigDir/gmd/config.cue),
// detects the project root, loads optional project-local config, merges project over global,
// and exports to a Go struct. Project values take precedence over global values.
func Load(cwd string) (*Config, error) {
	ctx := cuecontext.New()

	var allCUEContent string
	entries, err := configEmbedsFS.ReadDir("embeds")
	if err != nil {
		return nil, fmt.Errorf("reading embedded schema dir: %w", err)
	}
	for i, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := configEmbedsFS.ReadFile("embeds/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}
		content := string(data)
		if i > 0 {
			content = stripPackageDecl(content)
		}
		allCUEContent += content + "\n"
	}

	decodeCUE := func(cueData string) (*Config, error) {
		data := stripPackageDecl(cueData)
		full := allCUEContent + "\n" + data
		val := ctx.CompileString(full)
		if val.Err() != nil {
			return nil, val.Err()
		}
		configVal := val.LookupPath(cue.ParsePath("Config"))
		if configVal.Err() != nil {
			return nil, configVal.Err()
		}
		var decoded Config
		if err := configVal.Decode(&decoded); err != nil {
			return nil, err
		}
		return &decoded, nil
	}

	projectRoot := FindProjectRoot(cwd)

	// Start with sensible defaults (used when no global or project config)
	cfg := defaultConfig()

	// Load global config
	if data, err := tryReadGlobalConfig(); err == nil {
		globalCfg, err := decodeCUE(data)
		if err != nil {
			return nil, fmt.Errorf("loading global config: %w", err)
		}
		mergeConfigs(cfg, globalCfg)
	}

	// Load project config and overlay on top (project takes precedence)
	if projectRoot != "" {
		if data, err := tryReadProjectConfig(projectRoot); err == nil {
			projCfg, err := decodeCUE(data)
			if err != nil {
				return nil, fmt.Errorf("loading project config: %w", err)
			}
			mergeConfigs(cfg, projCfg)
		}
	}

	// Apply defaults for wikiDir/rawDir if CUE defaults didn't survive decode
	for name, wc := range cfg.Wikis {
		if wc.WikiDir == "" {
			wc.WikiDir = "wiki"
		}
		if wc.RawDir == "" {
			wc.RawDir = "raw"
		}
		if wc.IndexFile == "" {
			wc.IndexFile = "_index.md"
		}
		if wc.LogFile == "" {
			wc.LogFile = "_log.md"
		}
		cfg.Wikis[name] = wc
	}

	// Apply API keys from env vars
	cfg.LLM.APIKey = os.Getenv("OPENAI_API_KEY")
	cfg.LLM.EmbeddingAPIKey = envOrFallback("GMD_EMBEDDING_API_KEY", cfg.LLM.APIKey)
	cfg.LLM.ExpansionAPIKey = envOrFallback("GMD_EXPANSION_API_KEY", cfg.LLM.APIKey)
	cfg.LLM.RerankAPIKey = envOrFallback("GMD_RERANK_API_KEY", cfg.LLM.APIKey)
	cfg.LLM.SummarizingAPIKey = envOrFallback("GMD_SUMMARIZING_API_KEY", cfg.LLM.APIKey)
	cfg.LLM.GeneralBigAPIKey = envOrFallback("GMD_GENERAL_BIG_API_KEY", cfg.LLM.APIKey)
	cfg.LLM.GeneralMidAPIKey = envOrFallback("GMD_GENERAL_MID_API_KEY", cfg.LLM.APIKey)
	cfg.LLM.GeneralSmallAPIKey = envOrFallback("GMD_GENERAL_SMALL_API_KEY", cfg.LLM.APIKey)
	cfg.Typesense.APIKey = os.Getenv("GMD_TYPESENSE_API_KEY")
	if v := os.Getenv("EXA_API_KEY"); v != "" {
		cfg.Web.EXA.APIKey = v
	}
	if v := os.Getenv("CLOUDFLARE_API_KEY"); v != "" {
		cfg.Web.Cloudflare.APIKey = v
	}
	if v := os.Getenv("CLOUDFLARE_ACCOUNT_ID"); v != "" {
		cfg.Web.Cloudflare.AccountID = v
	}
	if v := os.Getenv("TAVILY_API_KEY"); v != "" {
		cfg.Web.Tavily.APIKey = v
	}
	if v := os.Getenv("SEARXNG_BASE_URL"); v != "" {
		cfg.Web.SearXNG.BaseURL = v
	}

	cfg.ProjectRoot = projectRoot
	if cfg.Project == "" && projectRoot != "" {
		cfg.Project = filepath.Base(projectRoot)
	}

	return cfg, nil
}

// defaultConfig returns sensible defaults for when no global config exists.
func defaultConfig() *Config {
	return &Config{
		LLM: LLMConfig{
			EmbeddingModel:      "google/embeddinggemma-300m",
			EmbeddingBaseURL:    "http://localhost:8001/v1",
			ExpansionModel:      "Qwen/Qwen3-1.7B",
			ExpansionBaseURL:    "http://localhost:8002/v1",
			RerankModel:         "Qwen/Qwen3-Reranker-0.6B",
			RerankBaseURL:       "http://localhost:8003/v1",
			SummarizingBaseURL:  "http://localhost:8000/v1",
			GeneralBigBaseURL:   "http://localhost:8000/v1",
			GeneralMidBaseURL:   "http://localhost:8000/v1",
			GeneralSmallBaseURL: "http://localhost:8000/v1",
		},
		Typesense: TypesenseConfig{
			Host: "http://localhost:8108",
		},
		Collections:    make(map[string]CollectionConfig),
		Wikis:          make(map[string]WikiConfig),
		SearchDefaults: make(map[string][]string),
	}
}

// mergeConfigs overlays src onto dst. Non-zero fields in src take precedence.
func mergeConfigs(dst, src *Config) {
	mergeStringField(&src.Project, &dst.Project)

	// Merge LLM
	l := &src.LLM
	d := &dst.LLM
	mergeStringField(&l.EmbeddingModel, &d.EmbeddingModel)
	mergeStringField(&l.EmbeddingBaseURL, &d.EmbeddingBaseURL)
	mergeStringField(&l.ExpansionModel, &d.ExpansionModel)
	mergeStringField(&l.ExpansionBaseURL, &d.ExpansionBaseURL)
	mergeStringField(&l.RerankModel, &d.RerankModel)
	mergeStringField(&l.RerankBaseURL, &d.RerankBaseURL)
	mergeStringField(&l.SummarizingModel, &d.SummarizingModel)
	mergeStringField(&l.SummarizingBaseURL, &d.SummarizingBaseURL)
	mergeStringField(&l.GeneralBigModel, &d.GeneralBigModel)
	mergeStringField(&l.GeneralBigBaseURL, &d.GeneralBigBaseURL)
	mergeStringField(&l.GeneralMidModel, &d.GeneralMidModel)
	mergeStringField(&l.GeneralMidBaseURL, &d.GeneralMidBaseURL)
	mergeStringField(&l.GeneralSmallModel, &d.GeneralSmallModel)
	mergeStringField(&l.GeneralSmallBaseURL, &d.GeneralSmallBaseURL)

	// Merge Typesense
	mergeStringField(&src.Typesense.Host, &dst.Typesense.Host)

	// Merge Pipeline
	if src.Pipeline.Chunk.TargetTokens != 0 {
		dst.Pipeline.Chunk = src.Pipeline.Chunk
	}

	// Merge Collections
	if src.Collections != nil {
		if dst.Collections == nil {
			dst.Collections = make(map[string]CollectionConfig)
		}
		for k, v := range src.Collections {
			dst.Collections[k] = v
		}
	}
	// Merge Wikis
	if src.Wikis != nil {
		if dst.Wikis == nil {
			dst.Wikis = make(map[string]WikiConfig)
		}
		for k, v := range src.Wikis {
			dst.Wikis[k] = v
		}
	}
	// Merge SearchDefaults
	if src.SearchDefaults != nil {
		if dst.SearchDefaults == nil {
			dst.SearchDefaults = make(map[string][]string)
		}
		for k, v := range src.SearchDefaults {
			dst.SearchDefaults[k] = v
		}
	}

	// Merge Web config
	mergeStringField(&src.Web.Group, &dst.Web.Group)
	if src.Web.Groups != nil {
		if dst.Web.Groups == nil {
			dst.Web.Groups = make(map[string]WebProviderGroup)
		}
		for k, v := range src.Web.Groups {
			dst.Web.Groups[k] = v
		}
	}
	mergeStringField(&src.Web.Search.Dedup, &dst.Web.Search.Dedup)
	if src.Web.Search.Synthesize {
		dst.Web.Search.Synthesize = src.Web.Search.Synthesize
	}
	mergeStringField(&src.Web.Search.SynthesisPrompt, &dst.Web.Search.SynthesisPrompt)
	mergeStringField(&src.Web.SearXNG.BaseURL, &dst.Web.SearXNG.BaseURL)
}

func mergeStringField(src, dst *string) {
	if *src != "" {
		*dst = *src
	}
}

func tryReadGlobalConfig() (string, error) {
	p, err := GlobalConfigPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func stripPackageDecl(content string) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "package ") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func tryReadProjectConfig(root string) (string, error) {
	p := filepath.Join(root, sentinelDir, "config.cue")
	data, err := os.ReadFile(p)
	if err == nil {
		return string(data), nil
	}
	return "", fmt.Errorf("no project config found")
}

func envOrFallback(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
