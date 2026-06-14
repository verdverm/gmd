package llm

import (
	"net/http"
	"testing"

	"github.com/verdverm/gmd/pkg/testutil"
)

func TestEmbedReplay(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/001_embed.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	providers := map[string]ProviderConfig{
		"test-provider": {
			Name:       "test-provider",
			BaseURL:    "https://api.openai.com/v1",
			Auth:       "apikey",
			AuthData:   map[string]string{"api_key": "test-key"},
			HTTPClient: &http.Client{Transport: tape.Transport()},
		},
	}
	profile := Profile{
		Embedding: RoleConfig{Provider: "test-provider", Model: "text-embedding-3-small"},
	}
	client, err := BuildAllClients(providers, profile)
	if err != nil {
		t.Fatal(err)
	}

	vec, err := client.Embed(t.Context(), "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if len(vec) == 0 {
		t.Error("empty embedding vector")
	}

	vecs, err := client.EmbedBatch(t.Context(), []string{"hello", "world"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vecs) != 2 {
		t.Errorf("expected 2 vectors, got %d", len(vecs))
	}

	_, err = client.Embed(t.Context(), "exhausted")
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}

func TestChatReplay(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/002_chat_expand.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	providers := map[string]ProviderConfig{
		"test-provider": {
			Name:       "test-provider",
			BaseURL:    "https://api.openai.com/v1",
			Auth:       "apikey",
			AuthData:   map[string]string{"api_key": "test-key"},
			HTTPClient: &http.Client{Transport: tape.Transport()},
		},
	}
	profile := Profile{
		Expansion: RoleConfig{Provider: "test-provider", Model: "gpt-4o-mini"},
	}
	client, err := BuildAllClients(providers, profile)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Chat(t.Context(), []ChatMessage{
		{Role: "user", Content: "Say hello in one word."},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp == "" {
		t.Error("empty chat response")
	}
}

func TestRerankReplay(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/003_rerank.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	providers := map[string]ProviderConfig{
		"test-provider": {
			Name:       "test-provider",
			BaseURL:    "https://api.openai.com/v1",
			Auth:       "apikey",
			AuthData:   map[string]string{"api_key": "test-key"},
			HTTPClient: &http.Client{Transport: tape.Transport()},
		},
	}
	profile := Profile{
		Rerank: RoleConfig{Provider: "test-provider", Model: "bge-reranker-v2-m3"},
	}
	client, err := BuildAllClients(providers, profile)
	if err != nil {
		t.Fatal(err)
	}

	results, err := client.Rerank(t.Context(), "test query", []string{"doc one", "doc two"})
	if err != nil {
		t.Fatal(err)
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
