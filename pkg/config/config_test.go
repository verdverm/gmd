package config

import (
	"testing"
)

func TestWebPersistenceConfig_NilDefaults(t *testing.T) {
	cfg := defaultConfig()
	cfg.Web.Persistence = nil

	if cfg.Web.Persistence == nil {
		cfg.Web.Persistence = &WebPersistenceConfig{
			Enabled: true,
			Dir:     ".gmd/web",
		}
	}

	if cfg.Web.Persistence == nil {
		t.Fatal("Persistence should not be nil after applying defaults")
	}
	if !cfg.Web.Persistence.Enabled {
		t.Error("Enabled should default to true when block is absent")
	}
	if cfg.Web.Persistence.Dir != ".gmd/web" {
		t.Errorf("Dir = %q, want %q", cfg.Web.Persistence.Dir, ".gmd/web")
	}
}

func TestWebPersistenceConfig_UserExplicitFalse(t *testing.T) {
	cfg := defaultConfig()
	cfg.Web.Persistence = &WebPersistenceConfig{
		Enabled: false,
		Dir:     ".gmd/web",
	}

	if cfg.Web.Persistence.Enabled {
		t.Error("Enabled should remain false when user explicitly set it")
	}
	if cfg.Web.Persistence.Dir != ".gmd/web" {
		t.Errorf("Dir = %q, want %q", cfg.Web.Persistence.Dir, ".gmd/web")
	}
}

func TestWebPersistenceConfig_UserCustomDir(t *testing.T) {
	cfg := defaultConfig()
	cfg.Web.Persistence = &WebPersistenceConfig{
		Enabled: true,
		Dir:     "my-web-dir",
	}

	if !cfg.Web.Persistence.Enabled {
		t.Error("Enabled should stay true")
	}
	if cfg.Web.Persistence.Dir != "my-web-dir" {
		t.Errorf("Dir = %q, want %q", cfg.Web.Persistence.Dir, "my-web-dir")
	}
}

func TestWebPersistenceConfig_Merge_ProjectOverridesGlobal(t *testing.T) {
	dst := defaultConfig()
	dst.Web.Persistence = &WebPersistenceConfig{Enabled: true, Dir: ".gmd/web"}

	src := defaultConfig()
	src.Web.Persistence = &WebPersistenceConfig{Enabled: false, Dir: ".gmd/web"}

	mergeConfigs(dst, src)

	if dst.Web.Persistence == nil {
		t.Fatal("Persistence should not be nil after merge")
	}
	if dst.Web.Persistence.Enabled {
		t.Error("Project's enabled: false should override global's true")
	}
}

func TestWebPersistenceConfig_Merge_ProjectAbsentKeepsGlobal(t *testing.T) {
	dst := defaultConfig()
	dst.Web.Persistence = &WebPersistenceConfig{Enabled: true, Dir: "/custom/path"}

	src := defaultConfig()
	src.Web.Persistence = nil

	mergeConfigs(dst, src)

	if dst.Web.Persistence == nil {
		t.Fatal("Persistence should not be nil after merge")
	}
	if !dst.Web.Persistence.Enabled {
		t.Error("Global's enabled: true should persist when project is absent")
	}
	if dst.Web.Persistence.Dir != "/custom/path" {
		t.Errorf("Dir = %q, want %q", dst.Web.Persistence.Dir, "/custom/path")
	}
}

func TestWebPersistenceConfig_Merge_ProjectSetsDir(t *testing.T) {
	dst := defaultConfig()
	dst.Web.Persistence = &WebPersistenceConfig{Enabled: true, Dir: ".gmd/web"}

	src := defaultConfig()
	src.Web.Persistence = &WebPersistenceConfig{Enabled: true, Dir: "alt-dir"}

	mergeConfigs(dst, src)

	if dst.Web.Persistence == nil {
		t.Fatal("Persistence should not be nil")
	}
	if dst.Web.Persistence.Dir != "alt-dir" {
		t.Errorf("Dir = %q, want %q", dst.Web.Persistence.Dir, "alt-dir")
	}
}

func TestWebPersistenceConfig_JSONTags(t *testing.T) {
	p := WebPersistenceConfig{Enabled: true, Dir: "test"}
	if p.Enabled != true {
		t.Error("Enabled should be true")
	}
	if p.Dir != "test" {
		t.Error("Dir should be test")
	}
}

func TestMergeProviders_Additive(t *testing.T) {
	dst := defaultConfig()

	src := &Config{
		LLM: LLMConfig{
			Providers: map[string]LLMProviderConfig{
				"vllm8000": {Name: "vllm8000", BaseURL: "http://192.168.4.31:8000/v1"},
				"vllm8001": {Name: "vllm8001", BaseURL: "http://192.168.4.31:8001/v1"},
				"vllm8002": {Name: "vllm8002", BaseURL: "http://192.168.4.31:8002/v1"},
				"vllm8003": {Name: "vllm8003", BaseURL: "http://192.168.4.31:8003/v1"},
				"vertex":   {Name: "vertex", BaseURL: "https://us-central1-aiplatform.googleapis.com/v1beta1/..."},
			},
		},
	}

	mergeConfigs(dst, src)

	for name, pc := range dst.LLM.Providers {
		t.Logf("%s -> %s", name, pc.BaseURL)
	}

	if len(dst.LLM.Providers) != 9 {
		t.Errorf("got %d providers, want 9 (4 defaults + 5 user): %v", len(dst.LLM.Providers), providerNames(dst))
	}
}

func providerNames(cfg *Config) []string {
	var names []string
	for name := range cfg.LLM.Providers {
		names = append(names, name)
	}
	return names
}
