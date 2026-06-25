//go:build integration

package fusion

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/testutil"
	"github.com/verdverm/gmd/pkg/web"
	exaprovider "github.com/verdverm/gmd/pkg/web/providers/exa"
	searxngprovider "github.com/verdverm/gmd/pkg/web/providers/searxng"
	tavilyprovider "github.com/verdverm/gmd/pkg/web/providers/tavily"
)

func TestMain(m *testing.M) {
	root := config.FindProjectRoot(".")
	config.LoadEnvFiles(root, nil, nil)
	if _, err := config.Load("."); err != nil {
		fmt.Fprintf(os.Stderr, "fusion integration: config load failed (%v)\n", err)
	}
	os.Exit(m.Run())
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		if os.Getenv("GMD_WEB_INTEGRATION_FAIL") == "1" {
			t.Fatalf("%s not set — integration test requires credentials", key)
		}
		t.Skipf("%s not set — skipping integration test", key)
	}
	return v
}

func maybeNewTape(t *testing.T, filePath, upstreamURL string) *testutil.Tape {
	t.Helper()
	if os.Getenv("GMD_NORECORD") == "1" {
		return nil
	}
	return testutil.NewTape(filePath, upstreamURL, nil, testutil.ModeRecord)
}

func tapedSearchProviders(t *testing.T, exaHC, tavilyHC, searxngHC *http.Client) []web.SearchProvider {
	t.Helper()
	var providers []web.SearchProvider

	if key := os.Getenv("EXA_API_KEY"); key != "" {
		adapter, err := exaprovider.NewSearchAdapter(web.ProviderConfig{
			Name: "exa", Extra: map[string]any{"api_key": key},
			HTTPClient: exaHC,
		})
		if err == nil {
			providers = append(providers, adapter)
		}
	}

	if key := os.Getenv("TAVILY_API_KEY"); key != "" {
		client, err := tavilyprovider.NewSearchClient(web.ProviderConfig{
			Name: "tavily", Extra: map[string]any{"api_key": key},
			HTTPClient: tavilyHC,
		})
		if err == nil {
			providers = append(providers, client)
		}
	}

	if url := os.Getenv("SEARXNG_BASE_URL"); url != "" {
		client, err := searxngprovider.NewSearchClient(web.ProviderConfig{
			Name: "searxng", Extra: map[string]any{"base_url": url},
			HTTPClient: searxngHC,
		})
		if err == nil {
			providers = append(providers, client)
		}
	}

	return providers
}

func availableSearchProviders(t *testing.T) []web.SearchProvider {
	t.Helper()
	var providers []web.SearchProvider

	if key := os.Getenv("EXA_API_KEY"); key != "" {
		adapter, err := exaprovider.NewSearchAdapter(web.ProviderConfig{
			Name: "exa", Extra: map[string]any{"api_key": key},
		})
		if err == nil {
			providers = append(providers, adapter)
		}
	}

	if key := os.Getenv("TAVILY_API_KEY"); key != "" {
		client, err := tavilyprovider.NewSearchClient(web.ProviderConfig{
			Name: "tavily", Extra: map[string]any{"api_key": key},
		})
		if err == nil {
			providers = append(providers, client)
		}
	}

	if url := os.Getenv("SEARXNG_BASE_URL"); url != "" {
		client, err := searxngprovider.NewSearchClient(web.ProviderConfig{
			Name: "searxng", Extra: map[string]any{"base_url": url},
		})
		if err == nil {
			providers = append(providers, client)
		}
	}

	return providers
}

func llmClientOrSkip(t *testing.T) llm.ChatModel {
	t.Helper()

	cfg, err := config.Load(".")
	if err != nil {
		t.Skipf("config load failed: %v", err)
	}

	reg, err := llm.NewRegistry(context.Background(), cfg)
	if err != nil {
		t.Skipf("LLM registry build failed: %v", err)
	}

	chat := reg.Model(llm.RoleSummarizing)
	if chat == nil {
		t.Skip("no LLM model configured for summarizing")
	}

	return chat
}

func TestMultiSearch_Integration(t *testing.T) {
	exaTape := maybeNewTape(t, "testdata/fusion_exa.json", "https://api.exa.ai")
	tavilyTape := maybeNewTape(t, "testdata/fusion_tavily.json", "https://api.tavily.com")

	searxngURL := os.Getenv("SEARXNG_BASE_URL")
	var searxngTape *testutil.Tape
	if searxngURL != "" {
		searxngTape = maybeNewTape(t, "testdata/fusion_searxng.json", searxngURL)
	}

	var exaHC, tavilyHC, searxngHC *http.Client
	startTape := func(tape *testutil.Tape) *http.Client {
		if tape == nil {
			return nil
		}
		tape.Start()
		return &http.Client{Transport: tape.Transport()}
	}
	exaHC = startTape(exaTape)
	tavilyHC = startTape(tavilyTape)
	searxngHC = startTape(searxngTape)

	defer func() {
		stopTape := func(tape *testutil.Tape) {
			if tape == nil {
				return
			}
			if err := tape.Stop(); err != nil {
				t.Fatal(err)
			}
		}
		stopTape(exaTape)
		stopTape(tavilyTape)
		stopTape(searxngTape)
	}()

	providers := tapedSearchProviders(t, exaHC, tavilyHC, searxngHC)
	if len(providers) == 0 {
		t.Skip("no search providers available")
	}

	results, _, _, err := MultiSearch(context.Background(), "golang standard library features",
		providers, web.SearchOptions{Query: "golang standard library features", NumResults: 3})
	if err != nil {
		t.Fatalf("MultiSearch: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}

	t.Logf("got %d results from %d providers", len(results), len(providers))

	providersSeen := map[string]int{}
	for _, r := range results {
		if r.Title == "" {
			t.Error("result has empty title")
		}
		if r.URL == "" {
			t.Error("result has empty URL")
		}
		if p, ok := r.Extra["_provider"].(string); ok {
			providersSeen[p]++
		} else {
			t.Error("result missing _provider tag")
		}
		t.Logf("  [%s] %s — %s (score=%.2f)", r.Extra["_provider"], r.Title, r.URL, r.Score)
	}

	t.Logf("provider distribution: %v", providersSeen)
}

func TestRun_HeuristicDedup_Integration(t *testing.T) {
	providers := availableSearchProviders(t)
	if len(providers) == 0 {
		t.Skip("no search providers available")
	}

	result, err := Run(context.Background(), "react vs vue comparison",
		providers,
		web.SearchOptions{Query: "react vs vue comparison", NumResults: 5},
		Config{Dedup: "heuristic", Synthesize: false},
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(result.Results) == 0 {
		t.Fatal("expected at least 1 result after dedup")
	}

	t.Logf("deduped to %d results (from multiple providers)", len(result.Results))

	urls := map[string]bool{}
	for _, r := range result.Results {
		key := strings.TrimRight(strings.ToLower(r.URL), "/")
		if urls[key] {
			t.Errorf("duplicate URL after dedup: %s", r.URL)
		}
		urls[key] = true
		t.Logf("  [%s] %s — %s (score=%.2f)", r.Extra["_provider"], r.Title, r.URL, r.Score)
	}

	t.Logf("no duplicate URLs found")
}

func TestRun_Synthesis_Integration(t *testing.T) {
	providers := availableSearchProviders(t)
	if len(providers) == 0 {
		t.Skip("no search providers available")
	}

	summarizer := llmClientOrSkip(t)
	if summarizer == nil {
		t.Skip("llm client not available")
	}

	result, err := Run(context.Background(), "python vs javascript for web development",
		providers,
		web.SearchOptions{Query: "python vs javascript for web development", NumResults: 3},
		Config{Dedup: "heuristic", Synthesize: true, Summarizer: summarizer},
	)
	if err != nil {
		t.Fatalf("Run with synthesis: %v", err)
	}

	if result.Answer == "" {
		t.Fatal("expected synthesized answer, got empty string")
	}

	t.Logf("=== SYNTHESIZED ANSWER ===\n%s\n=== END ANSWER ===", result.Answer)
	t.Logf("based on %d deduped results", len(result.Results))

	if len(result.Results) == 0 {
		t.Fatal("expected at least 1 result")
	}
}

func TestRun_LLMDedup_Integration(t *testing.T) {
	providers := availableSearchProviders(t)
	if len(providers) == 0 {
		t.Skip("no search providers available")
	}

	summarizer := llmClientOrSkip(t)

	result, err := Run(context.Background(), "rust programming language features",
		providers,
		web.SearchOptions{Query: "rust programming language features", NumResults: 3},
		Config{Dedup: "llm", Synthesize: false, Summarizer: summarizer},
	)
	if err != nil {
		t.Fatalf("Run with LLM dedup: %v", err)
	}

	if len(result.Results) == 0 {
		t.Fatal("expected at least 1 result after LLM dedup")
	}

	t.Logf("LLM deduped to %d results", len(result.Results))
	for _, r := range result.Results {
		t.Logf("  [%s] %s — %s", r.Extra["_provider"], r.Title, r.URL)
	}
}

func TestRun_CustomSynthesisPrompt_Integration(t *testing.T) {
	providers := availableSearchProviders(t)
	if len(providers) == 0 {
		t.Skip("no search providers available")
	}

	summarizer := llmClientOrSkip(t)

	customPrompt := `You are a concise search synthesizer. Answer in exactly 3 bullet points. Cite sources with [](url).`

	result, err := Run(context.Background(), "typescript advantages",
		providers,
		web.SearchOptions{Query: "typescript advantages", NumResults: 3},
		Config{
			Dedup:           "heuristic",
			Synthesize:      true,
			SynthesisPrompt: customPrompt,
			Summarizer:      summarizer,
		},
	)
	if err != nil {
		t.Fatalf("Run with custom prompt: %v", err)
	}

	if result.Answer == "" {
		t.Fatal("expected synthesized answer")
	}

	t.Logf("=== CUSTOM PROMPT ANSWER ===\n%s\n=== END ===", result.Answer)

	if !strings.Contains(result.Answer, "-") && !strings.Contains(result.Answer, "*") {
		t.Log("warning: answer doesn't look like bullet points (may still be valid)")
	}
}

func TestRun_SingleProvider_Integration(t *testing.T) {
	key := os.Getenv("EXA_API_KEY")
	if key == "" {
		t.Skip("EXA_API_KEY not set")
	}

	provider, err := exaprovider.NewSearchAdapter(web.ProviderConfig{
		Name: "exa", Extra: map[string]any{"api_key": key},
	})
	if err != nil {
		t.Fatalf("NewSearchAdapter: %v", err)
	}

	result, err := Run(context.Background(), "kubernetes basics",
		[]web.SearchProvider{provider},
		web.SearchOptions{Query: "kubernetes basics", NumResults: 3},
		Config{Dedup: "heuristic", Synthesize: false},
	)
	if err != nil {
		t.Fatalf("Run with single provider: %v", err)
	}

	if len(result.Results) == 0 {
		t.Fatal("expected at least 1 result")
	}

	t.Logf("single provider returned %d results", len(result.Results))
	for _, r := range result.Results {
		if p, ok := r.Extra["_provider"].(string); ok {
			t.Logf("  [%s] %s — %s", p, r.Title, r.URL)
		}
	}
}

func TestRun_DedupNone_Integration(t *testing.T) {
	providers := availableSearchProviders(t)
	if len(providers) < 2 {
		t.Skip("need at least 2 providers for dedup-none test")
	}

	result, err := Run(context.Background(), "docker containers",
		providers,
		web.SearchOptions{Query: "docker containers", NumResults: 3},
		Config{Dedup: "none", Synthesize: false},
	)
	if err != nil {
		t.Fatalf("Run with dedup=none: %v", err)
	}

	totalAvailable := len(providers) * 3
	t.Logf("dedup=none preserved %d results (max possible: %d)", len(result.Results), totalAvailable)

	if len(result.Results) == 0 {
		t.Fatal("expected results")
	}
}
