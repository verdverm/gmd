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
	embedClient    openai.Client
	expandClient   openai.Client
	rerankClient   openai.Client
	embeddingModel string
	expansionModel string
	rerankModel    string
	embedURL       string
	expandURL      string
	rerankURL      string
}

type Config struct {
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
	return &Client{
		embedClient:    newOpenAIClient(cfg.EmbedURL, cfg.APIKey),
		expandClient:   newOpenAIClient(cfg.ExpandURL, cfg.APIKey),
		rerankClient:   newOpenAIClient(cfg.RerankURL, cfg.APIKey),
		embeddingModel: cfg.EmbeddingModel,
		expansionModel: cfg.ExpansionModel,
		rerankModel:    cfg.RerankModel,
		embedURL:       cfg.EmbedURL,
		expandURL:      cfg.ExpandURL,
		rerankURL:      cfg.RerankURL,
	}
}

func (c *Client) clientForEmbed() openai.Client  { return c.embedClient }
func (c *Client) clientForExpand() openai.Client { return c.expandClient }
func (c *Client) clientForRerank() openai.Client { return c.rerankClient }

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

type EndpointStatus struct {
	Label  string
	URL    string
	Model  string
	OK     bool
	Models []string
	Err    string
}

func (c *Client) CheckEndpoint(ctx context.Context, label, baseURL, model string) EndpointStatus {
	s := EndpointStatus{Label: label, URL: baseURL, Model: model}
	cli := newOpenAIClient(baseURL, "")
	page, err := cli.Models.List(ctx)
	if err != nil {
		s.Err = err.Error()
		return s
	}
	s.OK = true
	for _, m := range page.Data {
		s.Models = append(s.Models, m.ID)
	}
	if model != "" {
		found := false
		for _, m := range s.Models {
			if m == model {
				found = true
				break
			}
		}
		if !found {
			s.Err = "model not found on endpoint"
			s.OK = false
		}
	}
	return s
}

func (c *Client) CheckAll(ctx context.Context) []EndpointStatus {
	endpoints := []struct {
		label string
		url   string
		model string
	}{
		{"embedding", c.embedURL, c.embeddingModel},
		{"expansion", c.expandURL, c.expansionModel},
		{"rerank", c.rerankURL, c.rerankModel},
	}

	var results []EndpointStatus
	for _, ep := range endpoints {
		s := c.CheckEndpoint(ctx, ep.label, ep.url, ep.model)
		results = append(results, s)
	}
	return results
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
