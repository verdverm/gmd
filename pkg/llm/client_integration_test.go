//go:build integration

package llm

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/testutil"
)

var testCfg *config.Config

func TestMain(m *testing.M) {
	config.LoadEnvFiles(config.FindProjectRoot("."), nil, nil)
	cfg, err := config.Load(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "llm integration: config load failed (%v)\n", err)
		os.Exit(1)
	}
	testCfg = cfg
	os.Exit(m.Run())
}

func maybeNewTape(t *testing.T, filePath, upstreamURL string) *testutil.Tape {
	t.Helper()
	if os.Getenv("GMD_NORECORD") == "1" {
		return nil
	}
	return testutil.NewTape(filePath, upstreamURL, nil, testutil.ModeRecord)
}

func providerBaseURL(t *testing.T, role string) string {
	t.Helper()
	profileName := testCfg.LLM.Profile
	if profileName == "" {
		profileName = "default"
	}
	pc, ok := testCfg.LLM.Profiles[profileName]
	if !ok {
		t.Fatalf("profile %q not found", profileName)
	}
	var provName string
	switch role {
	case "embedding":
		if pc.Embedding == nil {
			t.Fatal("no embedding role configured")
		}
		provName = pc.Embedding.Provider
	case "expansion":
		if pc.Expansion == nil {
			t.Fatal("no expansion role configured")
		}
		provName = pc.Expansion.Provider
	case "rerank":
		if pc.Rerank == nil {
			t.Fatal("no rerank role configured")
		}
		provName = pc.Rerank.Provider
	default:
		t.Fatalf("unknown role %q", role)
	}
	p, ok := testCfg.LLM.Providers[provName]
	if !ok {
		t.Fatalf("provider %q not found", provName)
	}
	return p.BaseURL
}

func buildSingleRoleClient(t *testing.T, role string, httpClient *http.Client) *Client {
	t.Helper()
	providers := make(map[string]ProviderConfig)
	for name, pc := range testCfg.LLM.Providers {
		providers[name] = ProviderConfig{
			Name:       pc.Name,
			BaseURL:    pc.BaseURL,
			Auth:       pc.Auth,
			AuthData:   pc.AuthData,
			HTTPClient: httpClient,
		}
	}

	profileName := testCfg.LLM.Profile
	if profileName == "" {
		profileName = "default"
	}
	pc, ok := testCfg.LLM.Profiles[profileName]
	if !ok {
		t.Fatalf("profile %q not found", profileName)
	}

	profile := Profile{}
	switch role {
	case "embedding":
		if pc.Embedding != nil {
			profile.Embedding.Provider = pc.Embedding.Provider
			profile.Embedding.Model = pc.Embedding.Model
		}
	case "expansion":
		if pc.Expansion != nil {
			profile.Expansion.Provider = pc.Expansion.Provider
			profile.Expansion.Model = pc.Expansion.Model
		}
	case "rerank":
		if pc.Rerank != nil {
			profile.Rerank.Provider = pc.Rerank.Provider
			profile.Rerank.Model = pc.Rerank.Model
		}
	default:
		t.Fatalf("unknown role %q", role)
	}

	c, err := BuildAllClients(providers, profile)
	if err != nil {
		t.Fatalf("BuildAllClients(%s): %v", role, err)
	}
	return c
}

func TestIntegrationEmbed(t *testing.T) {
	upstreamURL := providerBaseURL(t, "embedding")
	tape := maybeNewTape(t, "testdata/001_embed.json", upstreamURL)
	var httpClient *http.Client
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
		httpClient = &http.Client{Transport: tape.Transport()}
	}
	client := buildSingleRoleClient(t, "embedding", httpClient)

	vec, err := client.Embed(t.Context(), "hello world")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) == 0 {
		t.Fatal("expected non-empty embedding vector")
	}

	vecs, err := client.EmbedBatch(t.Context(), []string{"hello", "world"})
	if err != nil {
		t.Fatalf("EmbedBatch: %v", err)
	}
	if len(vecs) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vecs))
	}
}

func TestIntegrationChatExpand(t *testing.T) {
	upstreamURL := providerBaseURL(t, "expansion")
	tape := maybeNewTape(t, "testdata/002_chat_expand.json", upstreamURL)
	var httpClient *http.Client
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
		httpClient = &http.Client{Transport: tape.Transport()}
	}
	client := buildSingleRoleClient(t, "expansion", httpClient)

	resp, err := client.Chat(t.Context(), []ChatMessage{
		{Role: "user", Content: "Say hello in one word."},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp == "" {
		t.Fatal("expected non-empty chat response")
	}
}

func TestIntegrationRerank(t *testing.T) {
	upstreamURL := providerBaseURL(t, "rerank")
	tape := maybeNewTape(t, "testdata/003_rerank.json", upstreamURL)
	var httpClient *http.Client
	if tape != nil {
		tape.Start()
		defer func() {
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}()
		httpClient = &http.Client{Transport: tape.Transport()}
	}
	client := buildSingleRoleClient(t, "rerank", httpClient)

	results, err := client.Rerank(t.Context(), "test query", []string{"doc one", "doc two"})
	if err != nil {
		t.Fatalf("Rerank: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, r := range results {
		if r.Index < 0 {
			t.Errorf("result[%d]: negative index %d", i, r.Index)
		}
		if r.Score <= 0 {
			t.Errorf("result[%d]: non-positive score %f", i, r.Score)
		}
	}
}
