package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/shared"
)

type Client struct {
	defaultClient  openai.Client
	embedClient    *openai.Client
	expandClient   *openai.Client
	rerankClient   *openai.Client
	embeddingModel string
	expansionModel string
	rerankModel    string
}

type Config struct {
	BaseURL        string
	APIKey         string
	EmbeddingModel string
	ExpansionModel string
	RerankModel    string
	EmbedURL       string
	ExpandURL      string
	RerankURL      string
}

func newOpenAIClient(baseURL, apiKey string) openai.Client {
	opts := []option.RequestOption{
		option.WithBaseURL(baseURL),
	}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return openai.NewClient(opts...)
}

func New(cfg Config) *Client {
	c := &Client{
		defaultClient:  newOpenAIClient(cfg.BaseURL, cfg.APIKey),
		embeddingModel: cfg.EmbeddingModel,
		expansionModel: cfg.ExpansionModel,
		rerankModel:    cfg.RerankModel,
	}
	if cfg.EmbedURL != "" {
		cli := newOpenAIClient(cfg.EmbedURL, cfg.APIKey)
		c.embedClient = &cli
	}
	if cfg.ExpandURL != "" {
		cli := newOpenAIClient(cfg.ExpandURL, cfg.APIKey)
		c.expandClient = &cli
	}
	if cfg.RerankURL != "" {
		cli := newOpenAIClient(cfg.RerankURL, cfg.APIKey)
		c.rerankClient = &cli
	}
	return c
}

func (c *Client) clientForEmbed() openai.Client {
	if c.embedClient != nil {
		return *c.embedClient
	}
	return c.defaultClient
}

func (c *Client) clientForExpand() openai.Client {
	if c.expandClient != nil {
		return *c.expandClient
	}
	return c.defaultClient
}

func (c *Client) clientForRerank() openai.Client {
	if c.rerankClient != nil {
		return *c.rerankClient
	}
	return c.defaultClient
}

func (c *Client) Embed(ctx context.Context, text string) ([]float64, error) {
	return c.EmbedWithModel(ctx, text, c.embeddingModel)
}

func (c *Client) EmbedWithModel(ctx context.Context, text, model string) ([]float64, error) {
	client := c.clientForEmbed()
	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(model),
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: param.NewOpt(text),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("embedding: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("embedding: no data")
	}
	return resp.Data[0].Embedding, nil
}

func (c *Client) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	client := c.clientForEmbed()
	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(c.embeddingModel),
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("batch embedding: %w", err)
	}
	results := make([][]float64, len(resp.Data))
	for i, d := range resp.Data {
		results[i] = d.Embedding
	}
	return results, nil
}

type ChatMessage struct {
	Role    string
	Content string
}

func (c *Client) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.ChatWithModel(ctx, messages, c.expansionModel)
}

func (c *Client) ChatWithModel(ctx context.Context, messages []ChatMessage, model string) (string, error) {
	client := c.clientForExpand()
	chatMsgs := make([]openai.ChatCompletionMessageParamUnion, len(messages))
	for i, m := range messages {
		switch m.Role {
		case "system":
			chatMsgs[i] = openai.SystemMessage(m.Content)
		case "user":
			chatMsgs[i] = openai.UserMessage(m.Content)
		case "assistant":
			chatMsgs[i] = openai.AssistantMessage(m.Content)
		default:
			chatMsgs[i] = openai.UserMessage(m.Content)
		}
	}

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(model),
		Messages: chatMsgs,
	})
	if err != nil {
		return "", fmt.Errorf("chat: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("chat: no choices")
	}
	return resp.Choices[0].Message.Content, nil
}

type RerankResult struct {
	Index int
	Score float64
}

type rerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
}

type rerankResponse struct {
	Results []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
	} `json:"results"`
}

func (c *Client) Rerank(ctx context.Context, query string, documents []string) ([]RerankResult, error) {
	client := c.clientForRerank()
	body := rerankRequest{
		Model:     c.rerankModel,
		Query:     query,
		Documents: documents,
	}
	var resp rerankResponse
	err := client.Post(ctx, "/rerank", body, &resp)
	if err != nil {
		return nil, fmt.Errorf("rerank: %w", err)
	}
	results := make([]RerankResult, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = RerankResult{Index: r.Index, Score: r.RelevanceScore}
	}
	return results, nil
}
