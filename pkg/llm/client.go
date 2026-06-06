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

type Client struct {
	embedClient        openai.Client
	expandClient       openai.Client
	rerankClient       openai.Client
	summarizeClient    openai.Client
	generalBigClient   openai.Client
	generalMidClient   openai.Client
	generalSmallClient openai.Client

	embeddingModel    string
	expansionModel    string
	rerankModel       string
	summarizingModel  string
	generalBigModel   string
	generalMidModel   string
	generalSmallModel string

	embedURL        string
	expandURL       string
	rerankURL       string
	summarizeURL    string
	generalBigURL   string
	generalMidURL   string
	generalSmallURL string
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

// ConfigFromProject builds an llm.Config from a project-level gmd Config.
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
	return &Client{
		embedClient:        newOpenAIClient(cfg.EmbedURL, keyOrFallback(cfg.EmbeddingAPIKey, cfg.APIKey)),
		expandClient:       newOpenAIClient(cfg.ExpandURL, keyOrFallback(cfg.ExpansionAPIKey, cfg.APIKey)),
		rerankClient:       newOpenAIClient(cfg.RerankURL, keyOrFallback(cfg.RerankAPIKey, cfg.APIKey)),
		summarizeClient:    newOpenAIClient(cfg.SummarizingBaseURL, keyOrFallback(cfg.SummarizingAPIKey, cfg.APIKey)),
		generalBigClient:   newOpenAIClient(cfg.GeneralBigBaseURL, keyOrFallback(cfg.GeneralBigAPIKey, cfg.APIKey)),
		generalMidClient:   newOpenAIClient(cfg.GeneralMidBaseURL, keyOrFallback(cfg.GeneralMidAPIKey, cfg.APIKey)),
		generalSmallClient: newOpenAIClient(cfg.GeneralSmallBaseURL, keyOrFallback(cfg.GeneralSmallAPIKey, cfg.APIKey)),

		embeddingModel:    cfg.EmbeddingModel,
		expansionModel:    cfg.ExpansionModel,
		rerankModel:       cfg.RerankModel,
		summarizingModel:  cfg.SummarizingModel,
		generalBigModel:   cfg.GeneralBigModel,
		generalMidModel:   cfg.GeneralMidModel,
		generalSmallModel: cfg.GeneralSmallModel,

		embedURL:        cfg.EmbedURL,
		expandURL:       cfg.ExpandURL,
		rerankURL:       cfg.RerankURL,
		summarizeURL:    cfg.SummarizingBaseURL,
		generalBigURL:   cfg.GeneralBigBaseURL,
		generalMidURL:   cfg.GeneralMidBaseURL,
		generalSmallURL: cfg.GeneralSmallBaseURL,
	}
}

func (c *Client) clientForEmbed() openai.Client        { return c.embedClient }
func (c *Client) clientForExpand() openai.Client       { return c.expandClient }
func (c *Client) clientForRerank() openai.Client       { return c.rerankClient }
func (c *Client) clientForSummarize() openai.Client    { return c.summarizeClient }
func (c *Client) clientForGeneralBig() openai.Client   { return c.generalBigClient }
func (c *Client) clientForGeneralMid() openai.Client   { return c.generalMidClient }
func (c *Client) clientForGeneralSmall() openai.Client { return c.generalSmallClient }

func (c *Client) Embed(ctx context.Context, text string) ([]float64, error) {
	return c.EmbedWithModel(ctx, text, c.embeddingModel)
}

func (c *Client) EmbedWithModel(ctx context.Context, text, model string) ([]float64, error) {
	client := c.clientForEmbed()
	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
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
	client := c.clientForEmbed()
	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: c.embeddingModel,
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

func (c *Client) chatWithClient(ctx context.Context, messages []ChatMessage, model string, client openai.Client) (string, error) {
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
	return c.chatWithClient(ctx, messages, c.expansionModel, c.clientForExpand())
}

func (c *Client) ChatWithModel(ctx context.Context, messages []ChatMessage, model string) (string, error) {
	return c.chatWithClient(ctx, messages, model, c.clientForExpand())
}

func (c *Client) Summarize(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.chatWithClient(ctx, messages, c.summarizingModel, c.clientForSummarize())
}

func (c *Client) GeneralBigChat(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.chatWithClient(ctx, messages, c.generalBigModel, c.clientForGeneralBig())
}

func (c *Client) GeneralMidChat(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.chatWithClient(ctx, messages, c.generalMidModel, c.clientForGeneralMid())
}

func (c *Client) GeneralSmallChat(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.chatWithClient(ctx, messages, c.generalSmallModel, c.clientForGeneralSmall())
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
		{"embedding", c.embedURL, c.embeddingModel},
		{"expansion", c.expandURL, c.expansionModel},
		{"rerank", c.rerankURL, c.rerankModel},
		{"summarizing", c.summarizeURL, c.summarizingModel},
		{"general_big", c.generalBigURL, c.generalBigModel},
		{"general_mid", c.generalMidURL, c.generalMidModel},
		{"general_small", c.generalSmallURL, c.generalSmallModel},
	}

	results := make([]EndpointStatus, 0, len(endpoints))
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
