package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"iter"
	"net/http"
	"sync"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

const maxToolCallIDLength = 40

var (
	_ ChatModel = (*OpenAIModel)(nil)
	_ model.LLM = (*OpenAIModel)(nil)
)

// OpenAIConfig holds configuration for constructing an OpenAIModel.
type OpenAIConfig struct {
	APIKey     string
	BaseURL    string
	ModelName  string
	HTTPClient *http.Client
	Headers    http.Header
}

// OpenAIModel implements ChatModel (and thus model.LLM) for any
// OpenAI-compatible endpoint.
type OpenAIModel struct {
	client    *openai.Client
	modelName string

	toolCallIDMap   map[string]string
	toolCallIDMapMu sync.RWMutex
}

// NewOpenAIModel creates a new OpenAIModel with the given configuration.
func NewOpenAIModel(cfg OpenAIConfig) *OpenAIModel {
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
	return &OpenAIModel{
		client:        &client,
		modelName:     cfg.ModelName,
		toolCallIDMap: make(map[string]string),
	}
}

// Name returns the model name.
func (m *OpenAIModel) Name() string {
	return m.modelName
}

// ListModels queries the provider for available model names.
func (m *OpenAIModel) ListModels(ctx context.Context) ([]string, error) {
	resp, err := m.client.Models.List(ctx)
	if err != nil {
		return nil, err
	}
	models := make([]string, 0, len(resp.Data))
	for _, md := range resp.Data {
		models = append(models, md.ID)
	}
	return models, nil
}

// GenerateContent sends a request to the LLM and returns responses.
func (m *OpenAIModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	if stream {
		return m.generateStream(ctx, req)
	}
	return m.generate(ctx, req)
}

// Chat is a simple text-in/text-out helper for the common 2-message pattern.
func (m *OpenAIModel) Chat(ctx context.Context, system, user string) (string, error) {
	var contents []*genai.Content
	if user != "" {
		contents = append(contents, genai.NewContentFromText(user, genai.RoleUser))
	}
	cfg := &genai.GenerateContentConfig{}
	if system != "" {
		cfg.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: system}},
		}
	}
	req := &model.LLMRequest{
		Model:    m.modelName,
		Contents: contents,
		Config:   cfg,
	}
	for resp, err := range m.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", err
		}
		if resp.Content != nil {
			return extractText(resp.Content), nil
		}
	}
	return "", ErrNoChoicesInResponse
}

// ChatMessages is the full-control path for multi-turn chat.
func (m *OpenAIModel) ChatMessages(ctx context.Context, contents []*genai.Content) (string, error) {
	req := &model.LLMRequest{
		Model:    m.modelName,
		Contents: contents,
	}
	for resp, err := range m.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", err
		}
		if resp.Content != nil {
			return extractText(resp.Content), nil
		}
	}
	return "", ErrNoChoicesInResponse
}

func (m *OpenAIModel) generate(ctx context.Context, req *model.LLMRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		params, err := m.buildChatCompletionParams(req)
		if err != nil {
			yield(nil, err)
			return
		}
		resp, err := m.client.Chat.Completions.New(ctx, params)
		if err != nil {
			yield(nil, err)
			return
		}
		llmResp, err := convertResponse(resp)
		if err != nil {
			yield(nil, err)
			return
		}
		yield(llmResp, nil)
	}
}

func (m *OpenAIModel) generateStream(ctx context.Context, req *model.LLMRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		params, err := m.buildChatCompletionParams(req)
		if err != nil {
			yield(nil, err)
			return
		}
		stream := m.client.Chat.Completions.NewStreaming(ctx, params)
		acc := openai.ChatCompletionAccumulator{}

		for stream.Next() {
			chunk := stream.Current()
			acc.AddChunk(chunk)
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				llmResp := &model.LLMResponse{
					Content: &genai.Content{
						Role:  genai.RoleModel,
						Parts: []*genai.Part{{Text: chunk.Choices[0].Delta.Content}},
					},
					Partial:      true,
					TurnComplete: false,
				}
				if !yield(llmResp, nil) {
					return
				}
			}
		}
		if err := stream.Err(); err != nil {
			yield(nil, err)
			return
		}
		yield(m.buildStreamFinalResponse(&acc), nil)
	}
}

func (m *OpenAIModel) buildStreamFinalResponse(acc *openai.ChatCompletionAccumulator) *model.LLMResponse {
	content := &genai.Content{
		Role:  genai.RoleModel,
		Parts: []*genai.Part{},
	}
	if len(acc.Choices) > 0 {
		choice := acc.Choices[0]
		if choice.Message.Content != "" {
			content.Parts = append(content.Parts, &genai.Part{Text: choice.Message.Content})
		}
		for _, tc := range choice.Message.ToolCalls {
			content.Parts = append(content.Parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					ID:   tc.ID,
					Name: tc.Function.Name,
					Args: parseJSONArgs(tc.Function.Arguments),
				},
			})
		}
	}
	var finishReason genai.FinishReason
	if len(acc.Choices) > 0 {
		finishReason = convertFinishReason(string(acc.Choices[0].FinishReason))
	}
	return &model.LLMResponse{
		Content:       content,
		UsageMetadata: convertUsageMetadata(acc.Usage),
		FinishReason:  finishReason,
		Partial:       false,
		TurnComplete:  true,
	}
}

func (m *OpenAIModel) normalizeToolCallID(id string) string {
	if len(id) <= maxToolCallIDLength {
		return id
	}
	hash := sha256.Sum256([]byte(id))
	shortID := "tc_" + hex.EncodeToString(hash[:])[:maxToolCallIDLength-3]
	m.toolCallIDMapMu.Lock()
	m.toolCallIDMap[shortID] = id
	m.toolCallIDMapMu.Unlock()
	return shortID
}
