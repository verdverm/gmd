package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// Embedder computes text embeddings.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float64, error)
}

type openaiEmbedder struct {
	client    *openai.Client
	modelName string
}

// NewEmbedder creates an Embedder backed by an OpenAI-compatible embeddings endpoint.
func NewEmbedder(cfg OpenAIConfig) Embedder {
	return &openaiEmbedder{
		client:    newClientFromConfig(cfg),
		modelName: cfg.ModelName,
	}
}

func (e *openaiEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	resp, err := e.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: e.modelName,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("embedding: %w", err)
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, fmt.Errorf("embedding: no data")
	}
	results := make([][]float64, len(resp.Data))
	for i, d := range resp.Data {
		results[i] = d.Embedding
	}
	return results, nil
}

// newClientFromConfig builds an *openai.Client from OpenAIConfig.
// Self-contained: does not depend on old client.go helpers.
func newClientFromConfig(cfg OpenAIConfig) *openai.Client {
	var opts []option.RequestOption
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	if cfg.HTTPClient != nil {
		opts = append(opts, option.WithHTTPClient(cfg.HTTPClient))
	}
	for k, vals := range cfg.Headers {
		for _, v := range vals {
			opts = append(opts, option.WithHeaderAdd(k, v))
		}
	}
	client := openai.NewClient(opts...)
	return &client
}

// EmbedSingle is a convenience wrapper for single-text embedding.
func EmbedSingle(ctx context.Context, e Embedder, text string) ([]float64, error) {
	results, err := e.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("embedding: no data")
	}
	return results[0], nil
}
