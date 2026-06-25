//go:build integration

package llm

import (
	"context"
	"testing"
	"time"

	"github.com/verdverm/gmd/pkg/config"
)

func TestIntegrationLLM_ListModels(t *testing.T) {
	config.LoadEnvFiles(config.FindProjectRoot("."), nil, nil)
	cfg, err := config.Load(".")
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	for name, pc := range cfg.LLM.Providers {
		t.Run(name, func(t *testing.T) {
			if pc.BaseURL == "" {
				t.Fatalf("no base_url configured for provider %q", name)
			}

			m := NewOpenAIModel(OpenAIConfig{
				APIKey:  cfg.LLM.APIKey,
				BaseURL: pc.BaseURL,
			})

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			start := time.Now()
			models, listErr := m.ListModels(ctx)
			elapsed := time.Since(start)

			if listErr != nil {
				t.Logf("FAIL after %v: %v", elapsed, listErr)
			} else {
				t.Logf("OK after %v: %d models: %v", elapsed, len(models), models)
			}
		})
	}
}
