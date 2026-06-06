package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/shared"

	"github.com/verdverm/gmd/pkg/config"
)

type roleClient struct {
	client *openai.Client
	model  string
	url    string
}

type Config struct {
	APIKey string

	EmbeddingModel  string
	EmbeddingAPIKey string
	EmbedURL        string

	ExpansionModel  string
	ExpansionAPIKey string
	ExpandURL       string

	RerankModel  string
	RerankAPIKey string
	RerankURL    string

	SummarizingModel   string
	SummarizingAPIKey  string
	SummarizingBaseURL string

	GeneralBigModel   string
	GeneralBigAPIKey  string
	GeneralBigBaseURL string

	GeneralMidModel   string
	GeneralMidAPIKey  string
	GeneralMidBaseURL string

	GeneralSmallModel   string
	GeneralSmallAPIKey  string
	GeneralSmallBaseURL string
}

type Client struct {
	embedder     roleClient
	expander     roleClient
	reranker     roleClient
	summarizer   roleClient
	generalBig   roleClient
	generalMid   roleClient
	generalSmall roleClient

	providers map[string]*openai.Client
}

func (c *Client) ProviderClients() map[string]*openai.Client {
	return c.providers
}

func (c *Client) RoleClient(role string) *openai.Client {
	switch role {
	case "embedding":
		return c.embedder.client
	case "expansion":
		return c.expander.client
	case "rerank":
		return c.reranker.client
	case "summarizing":
		return c.summarizer.client
	case "general_big":
		return c.generalBig.client
	case "general_mid":
		return c.generalMid.client
	case "general_small":
		return c.generalSmall.client
	}
	return nil
}

func (c *Client) RoleModel(role string) string {
	switch role {
	case "embedding":
		return c.embedder.model
	case "expansion":
		return c.expander.model
	case "rerank":
		return c.reranker.model
	case "summarizing":
		return c.summarizer.model
	case "general_big":
		return c.generalBig.model
	case "general_mid":
		return c.generalMid.model
	case "general_small":
		return c.generalSmall.model
	}
	return ""
}

func (c *Client) RoleURL(role string) string {
	switch role {
	case "embedding":
		return c.embedder.url
	case "expansion":
		return c.expander.url
	case "rerank":
		return c.reranker.url
	case "summarizing":
		return c.summarizer.url
	case "general_big":
		return c.generalBig.url
	case "general_mid":
		return c.generalMid.url
	case "general_small":
		return c.generalSmall.url
	}
	return ""
}

func keyOrFallback(key, fallback string) string {
	if key != "" {
		return key
	}
	return fallback
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

func ConfigFromProject(cfg *config.Config) Config {
	return Config{
		APIKey:              cfg.LLM.APIKey,
		EmbeddingModel:      cfg.LLM.EmbeddingModel,
		ExpansionModel:      cfg.LLM.ExpansionModel,
		RerankModel:         cfg.LLM.RerankModel,
		EmbedURL:            cfg.LLM.EmbeddingBaseURL,
		ExpandURL:           cfg.LLM.ExpansionBaseURL,
		RerankURL:           cfg.LLM.RerankBaseURL,
		EmbeddingAPIKey:     cfg.LLM.EmbeddingAPIKey,
		ExpansionAPIKey:     cfg.LLM.ExpansionAPIKey,
		RerankAPIKey:        cfg.LLM.RerankAPIKey,
		SummarizingModel:    cfg.LLM.SummarizingModel,
		SummarizingBaseURL:  cfg.LLM.SummarizingBaseURL,
		SummarizingAPIKey:   cfg.LLM.SummarizingAPIKey,
		GeneralBigModel:     cfg.LLM.GeneralBigModel,
		GeneralBigBaseURL:   cfg.LLM.GeneralBigBaseURL,
		GeneralBigAPIKey:    cfg.LLM.GeneralBigAPIKey,
		GeneralMidModel:     cfg.LLM.GeneralMidModel,
		GeneralMidBaseURL:   cfg.LLM.GeneralMidBaseURL,
		GeneralMidAPIKey:    cfg.LLM.GeneralMidAPIKey,
		GeneralSmallModel:   cfg.LLM.GeneralSmallModel,
		GeneralSmallBaseURL: cfg.LLM.GeneralSmallBaseURL,
		GeneralSmallAPIKey:  cfg.LLM.GeneralSmallAPIKey,
	}
}

func New(cfg Config) *Client {
	c := &Client{
		providers: make(map[string]*openai.Client),
	}

	embedClient := newOpenAIClient(cfg.EmbedURL, keyOrFallback(cfg.EmbeddingAPIKey, cfg.APIKey))
	expandClient := newOpenAIClient(cfg.ExpandURL, keyOrFallback(cfg.ExpansionAPIKey, cfg.APIKey))
	rerankClient := newOpenAIClient(cfg.RerankURL, keyOrFallback(cfg.RerankAPIKey, cfg.APIKey))
	summarizeClient := newOpenAIClient(cfg.SummarizingBaseURL, keyOrFallback(cfg.SummarizingAPIKey, cfg.APIKey))
	bigClient := newOpenAIClient(cfg.GeneralBigBaseURL, keyOrFallback(cfg.GeneralBigAPIKey, cfg.APIKey))
	midClient := newOpenAIClient(cfg.GeneralMidBaseURL, keyOrFallback(cfg.GeneralMidAPIKey, cfg.APIKey))
	smallClient := newOpenAIClient(cfg.GeneralSmallBaseURL, keyOrFallback(cfg.GeneralSmallAPIKey, cfg.APIKey))

	c.embedder = roleClient{client: &embedClient, model: cfg.EmbeddingModel, url: cfg.EmbedURL}
	c.expander = roleClient{client: &expandClient, model: cfg.ExpansionModel, url: cfg.ExpandURL}
	c.reranker = roleClient{client: &rerankClient, model: cfg.RerankModel, url: cfg.RerankURL}
	c.summarizer = roleClient{client: &summarizeClient, model: cfg.SummarizingModel, url: cfg.SummarizingBaseURL}
	c.generalBig = roleClient{client: &bigClient, model: cfg.GeneralBigModel, url: cfg.GeneralBigBaseURL}
	c.generalMid = roleClient{client: &midClient, model: cfg.GeneralMidModel, url: cfg.GeneralMidBaseURL}
	c.generalSmall = roleClient{client: &smallClient, model: cfg.GeneralSmallModel, url: cfg.GeneralSmallBaseURL}

	c.providers["embedding"] = &embedClient
	c.providers["expansion"] = &expandClient
	c.providers["rerank"] = &rerankClient
	c.providers["summarizing"] = &summarizeClient
	c.providers["general_big"] = &bigClient
	c.providers["general_mid"] = &midClient
	c.providers["general_small"] = &smallClient

	return c
}

func (c *Client) Embed(ctx context.Context, text string) ([]float64, error) {
	return c.EmbedWithModel(ctx, text, c.embedder.model)
}

func (c *Client) EmbedWithModel(ctx context.Context, text, model string) ([]float64, error) {
	resp, err := c.embedder.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: model,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: param.NewOpt(text),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("embedding: %w", err)
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, fmt.Errorf("embedding: no data")
	}
	return resp.Data[0].Embedding, nil
}

func (c *Client) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	resp, err := c.embedder.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: c.embedder.model,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("batch embedding: %w", err)
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, fmt.Errorf("batch embedding: no data")
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

func (c *Client) chatWithClient(ctx context.Context, messages []ChatMessage, model string, client *openai.Client) (string, error) {
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
		Model:    shared.ChatModel(model), //nolint:unconvert // required for string→ChatModel conversion
		Messages: chatMsgs,
	})
	if err != nil {
		return "", fmt.Errorf("chat: %w", err)
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", fmt.Errorf("chat: no choices")
	}
	return resp.Choices[0].Message.Content, nil
}

func (c *Client) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.chatWithClient(ctx, messages, c.expander.model, c.expander.client)
}

func (c *Client) ChatWithModel(ctx context.Context, messages []ChatMessage, model string) (string, error) {
	return c.chatWithClient(ctx, messages, model, c.expander.client)
}

func (c *Client) Summarize(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.chatWithClient(ctx, messages, c.summarizer.model, c.summarizer.client)
}

func (c *Client) GeneralBigChat(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.chatWithClient(ctx, messages, c.generalBig.model, c.generalBig.client)
}

func (c *Client) GeneralMidChat(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.chatWithClient(ctx, messages, c.generalMid.model, c.generalMid.client)
}

func (c *Client) GeneralSmallChat(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.chatWithClient(ctx, messages, c.generalSmall.model, c.generalSmall.client)
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
	if page == nil {
		s.Err = "no data from models endpoint"
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
		{"embedding", c.embedder.url, c.embedder.model},
		{"expansion", c.expander.url, c.expander.model},
		{"rerank", c.reranker.url, c.reranker.model},
		{"summarizing", c.summarizer.url, c.summarizer.model},
		{"general_big", c.generalBig.url, c.generalBig.model},
		{"general_mid", c.generalMid.url, c.generalMid.model},
		{"general_small", c.generalSmall.url, c.generalSmall.model},
	}

	results := make([]EndpointStatus, 0, len(endpoints))
	for _, ep := range endpoints {
		s := c.CheckEndpoint(ctx, ep.label, ep.url, ep.model)
		results = append(results, s)
	}
	return results
}

func (c *Client) CheckProvider(ctx context.Context, name string, provider ProviderConfig) EndpointStatus {
	return c.CheckEndpoint(ctx, name, provider.BaseURL, "")
}

func (c *Client) CheckAllProviders(ctx context.Context, providers map[string]ProviderConfig) []EndpointStatus {
	results := make([]EndpointStatus, 0, len(providers))
	for name, pc := range providers {
		s := c.CheckProvider(ctx, name, pc)
		results = append(results, s)
	}
	return results
}

func (c *Client) Rerank(ctx context.Context, query string, documents []string) ([]RerankResult, error) {
	body := rerankRequest{
		Model:     c.reranker.model,
		Query:     query,
		Documents: documents,
	}
	var resp rerankResponse
	err := c.reranker.client.Post(ctx, "/rerank", body, &resp)
	if err != nil {
		return nil, fmt.Errorf("rerank: %w", err)
	}
	results := make([]RerankResult, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = RerankResult{Index: r.Index, Score: r.RelevanceScore}
	}
	return results, nil
}
