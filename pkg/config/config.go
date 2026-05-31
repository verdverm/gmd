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

// Config is the validated Go representation of the unified CUE configuration.
type Config struct {
	LLM         LLMConfig                   `json:"llm"`
	Typesense   TypesenseConfig             `json:"typesense"`
	EXA         EXAConfig                   `json:"exa,omitempty"`
	Pipeline    PipelineConfig              `json:"pipeline"`
	Collections map[string]CollectionConfig `json:"collections"`
	ProjectRoot string                      `json:"-"`
	Project     string                      `json:"project,omitempty"`
}

// EXAConfig maps from the CUE EXAConfig schema.
type EXAConfig struct {
	APIKey string `json:"-"`
}

// CollectionKey returns the project-prefixed key for a collection name.
// This avoids name collisions on shared Typesense instances.
func (c *Config) CollectionKey(name string) string {
	if c.Project == "" {
		return name
	}
	return c.Project + "-" + name
}

// LLMConfig maps from the CUE LLMConfig schema.
type LLMConfig struct {
	APIKey           string `json:"-"`
	EmbeddingModel   string `json:"embedding_model"`
	ExpansionModel   string `json:"expansion_model"`
	RerankModel      string `json:"rerank_model"`
	EmbeddingBaseURL string `json:"embedding_base_url"`
	ExpansionBaseURL string `json:"expansion_base_url"`
	RerankBaseURL    string `json:"rerank_base_url"`
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
	Enabled     bool               `json:"enabled"`
	IndexFile   string             `json:"indexFile"`
	LogFile     string             `json:"logFile"`
	GraphLinks  bool               `json:"graphLinks"`
	Frontmatter *FrontmatterConfig `json:"frontmatter,omitempty"`
}

// FrontmatterConfig maps from the CUE frontmatter field config.
type FrontmatterConfig struct {
	Fields map[string]FrontmatterField `json:"fields"`
}

// FrontmatterField maps from the CUE FrontmatterField schema.
type FrontmatterField struct {
	Type  string `json:"type"`
	Facet bool   `json:"facet"`
	Sort  bool   `json:"sort"`
}

// CollectionConfig maps from the CUE CollectionConfig schema.
// The collection name is the map key in Config.Collections, not a field in this struct.
type CollectionConfig struct {
	Path             string      `json:"path"`
	Pattern          string      `json:"pattern"`
	Ignore           []string    `json:"ignore,omitempty"`
	Context          string      `json:"context,omitempty"`
	IncludeByDefault bool        `json:"includeByDefault"`
	Wiki             *WikiConfig `json:"wiki,omitempty"`
}

//go:embed schema/*.cue
var cueSchema embed.FS

// Load loads and validates the unified configuration.
// It embeds the built-in schema, loads optional global config (~/.config/gmd/config.cue),
// detects the project root, loads optional project-local config, unifies them,
// validates against the schema, and exports to a Go struct.
func Load(cwd string) (*Config, error) {
	ctx := cuecontext.New()

	var allCUEContent string
	entries, err := cueSchema.ReadDir("schema")
	if err != nil {
		return nil, fmt.Errorf("reading embedded schema dir: %w", err)
	}
	for i, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := cueSchema.ReadFile("schema/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}
		content := string(data)
		if i > 0 {
			content = stripPackageDecl(content)
		}
		allCUEContent += content + "\n"
	}

	val := ctx.CompileString(allCUEContent)
	if val.Err() != nil {
		return nil, fmt.Errorf("compiling embedded schema: %w", val.Err())
	}

	if data, err := tryReadGlobalConfig(); err == nil {
		gv := ctx.CompileString(data)
		if gv.Err() != nil {
			return nil, fmt.Errorf("compiling global config: %w", gv.Err())
		}
		val = val.Unify(gv)
		if val.Err() != nil {
			return nil, fmt.Errorf("unifying global config: %w", val.Err())
		}
	}

	projectRoot := FindProjectRoot(cwd)
	if projectRoot != "" {
		if data, err := tryReadProjectConfig(projectRoot); err == nil {
			pv := ctx.CompileString(data)
			if pv.Err() != nil {
				return nil, fmt.Errorf("compiling project config: %w", pv.Err())
			}
			val = val.Unify(pv)
			if val.Err() != nil {
				return nil, fmt.Errorf("unifying project config: %w", val.Err())
			}
		}
	}

	configVal := val.LookupPath(cue.ParsePath("Config"))
	if configVal.Err() != nil {
		return nil, fmt.Errorf("lookup Config: %w", configVal.Err())
	}

	var cfg Config
	if err := configVal.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decoding config: %w", err)
	}

	cfg.LLM.APIKey = os.Getenv("OPENAI_API_KEY")
	cfg.Typesense.APIKey = os.Getenv("GMD_TYPESENSE_API_KEY")
	cfg.EXA.APIKey = os.Getenv("EXA_API_KEY")

	cfg.ProjectRoot = projectRoot
	if cfg.Project == "" && projectRoot != "" {
		cfg.Project = filepath.Base(projectRoot)
	}

	return &cfg, nil
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
	var out []string
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
