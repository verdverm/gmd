package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
)

// Reranker re-ranks documents by relevance to a query.
type Reranker interface {
	Rerank(ctx context.Context, query string, documents []string) ([]RerankResult, error)
}

// RerankResult holds a single rerank result.
type RerankResult struct {
	Index int     `json:"index"`
	Score float64 `json:"score"`
}

type openaiReranker struct {
	client    *openai.Client
	modelName string
}

// NewReranker creates a Reranker backed by an OpenAI-compatible rerank endpoint.
func NewReranker(cfg OpenAIConfig) Reranker {
	return &openaiReranker{
		client:    newClientFromConfig(cfg),
		modelName: cfg.ModelName,
	}
}

type rerankWireRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
}

type rerankWireResponse struct {
	Results []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
	} `json:"results"`
}

func (r *openaiReranker) Rerank(ctx context.Context, query string, documents []string) ([]RerankResult, error) {
	if len(documents) == 0 {
		return nil, nil
	}
	body := rerankWireRequest{
		Model:     r.modelName,
		Query:     query,
		Documents: documents,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("rerank marshal: %w", err)
	}
	var resp rerankWireResponse
	err = r.client.Post(ctx, "/rerank", bodyBytes, &resp)
	if err != nil {
		return nil, fmt.Errorf("rerank: %w", err)
	}
	results := make([]RerankResult, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = RerankResult{Index: r.Index, Score: r.RelevanceScore}
	}
	return results, nil
}
