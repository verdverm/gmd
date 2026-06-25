package llm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

func (m *OpenAIModel) buildChatCompletionParams(req *model.LLMRequest) (openai.ChatCompletionNewParams, error) {
	var messages []openai.ChatCompletionMessageParamUnion

	if req.Config != nil && req.Config.SystemInstruction != nil {
		if text := extractText(req.Config.SystemInstruction); text != "" {
			messages = append(messages, openai.SystemMessage(text))
		}
	}

	for _, content := range req.Contents {
		msgs, err := m.convertContentToMessages(content)
		if err != nil {
			return openai.ChatCompletionNewParams{}, err
		}
		messages = append(messages, msgs...)
	}

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(m.modelName),
		Messages: messages,
	}

	if req.Config != nil {
		applyGenerationConfig(&params, req.Config)
	}

	return params, nil
}

func applyGenerationConfig(params *openai.ChatCompletionNewParams, cfg *genai.GenerateContentConfig) {
	if cfg.Temperature != nil {
		params.Temperature = openai.Float(float64(*cfg.Temperature))
	}
	if cfg.MaxOutputTokens > 0 {
		params.MaxTokens = openai.Int(int64(cfg.MaxOutputTokens))
	}
	if cfg.TopP != nil {
		params.TopP = openai.Float(float64(*cfg.TopP))
	}

	if len(cfg.StopSequences) == 1 {
		params.Stop = openai.ChatCompletionNewParamsStopUnion{
			OfString: openai.String(cfg.StopSequences[0]),
		}
	} else if len(cfg.StopSequences) > 1 {
		params.Stop = openai.ChatCompletionNewParamsStopUnion{
			OfStringArray: cfg.StopSequences,
		}
	}

	if cfg.ThinkingConfig != nil {
		params.ReasoningEffort = convertThinkingLevel(cfg.ThinkingConfig.ThinkingLevel)
	}

	if cfg.ResponseMIMEType == "application/json" {
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &openai.ResponseFormatJSONObjectParam{},
		}
	}

	if cfg.ResponseSchema != nil {
		if schemaMap, err := convertSchema(cfg.ResponseSchema); err == nil {
			params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
				OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
					JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
						Name:        "response",
						Description: openai.String(cfg.ResponseSchema.Description),
						Schema:      schemaMap,
						Strict:      openai.Bool(true),
					},
				},
			}
		}
	}

	if len(cfg.Tools) > 0 {
		if tools, err := convertTools(cfg.Tools); err == nil {
			params.Tools = tools
		}
	}
}

func (m *OpenAIModel) convertContentToMessages(content *genai.Content) ([]openai.ChatCompletionMessageParamUnion, error) {
	var messages []openai.ChatCompletionMessageParamUnion
	var textParts []string
	var toolCalls []openai.ChatCompletionMessageToolCallUnionParam
	var mediaParts []openai.ChatCompletionContentPartUnionParam

	for _, part := range content.Parts {
		switch {
		case part.FunctionResponse != nil:
			responseJSON, err := json.Marshal(part.FunctionResponse.Response)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal function response: %w", err)
			}
			normalizedID := m.normalizeToolCallID(part.FunctionResponse.ID)
			messages = append(messages, openai.ToolMessage(string(responseJSON), normalizedID))

		case part.FunctionCall != nil:
			argsJSON, err := json.Marshal(part.FunctionCall.Args)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal function args: %w", err)
			}
			normalizedID := m.normalizeToolCallID(part.FunctionCall.ID)
			toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallUnionParam{
				OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
					ID: normalizedID,
					Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
						Name:      part.FunctionCall.Name,
						Arguments: string(argsJSON),
					},
				},
			})

		case part.Text != "":
			textParts = append(textParts, part.Text)

		case part.InlineData != nil:
			p, err := convertInlineDataToPart(part.InlineData)
			if err != nil {
				return nil, err
			}
			mediaParts = append(mediaParts, *p)
		}
	}

	if len(textParts) > 0 || len(mediaParts) > 0 || len(toolCalls) > 0 {
		msg := buildRoleMessage(content.Role, textParts, mediaParts, toolCalls)
		if msg != nil {
			messages = append(messages, *msg)
		}
	}

	return messages, nil
}

func buildRoleMessage(role string, texts []string, media []openai.ChatCompletionContentPartUnionParam, toolCalls []openai.ChatCompletionMessageToolCallUnionParam) *openai.ChatCompletionMessageParamUnion {
	switch convertRole(role) {
	case "user":
		return buildUserMessage(texts, media)
	case "assistant":
		return buildAssistantMessage(texts, toolCalls)
	case "system":
		msg := openai.SystemMessage(joinTexts(texts))
		return &msg
	}
	return nil
}

func buildUserMessage(texts []string, media []openai.ChatCompletionContentPartUnionParam) *openai.ChatCompletionMessageParamUnion {
	if len(media) == 0 {
		msg := openai.UserMessage(joinTexts(texts))
		return &msg
	}
	var parts []openai.ChatCompletionContentPartUnionParam
	for _, text := range texts {
		parts = append(parts, openai.ChatCompletionContentPartUnionParam{
			OfText: &openai.ChatCompletionContentPartTextParam{Text: text},
		})
	}
	parts = append(parts, media...)
	return &openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				OfArrayOfContentParts: parts,
			},
		},
	}
}

func buildAssistantMessage(texts []string, toolCalls []openai.ChatCompletionMessageToolCallUnionParam) *openai.ChatCompletionMessageParamUnion {
	msg := openai.ChatCompletionAssistantMessageParam{}
	if len(texts) > 0 {
		msg.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
			OfString: openai.String(joinTexts(texts)),
		}
	}
	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}
	return &openai.ChatCompletionMessageParamUnion{OfAssistant: &msg}
}

func convertResponse(resp *openai.ChatCompletion) (*model.LLMResponse, error) {
	if len(resp.Choices) == 0 {
		return nil, ErrNoChoicesInResponse
	}
	choice := resp.Choices[0]
	content := &genai.Content{
		Role:  genai.RoleModel,
		Parts: []*genai.Part{},
	}
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
	return &model.LLMResponse{
		Content:       content,
		UsageMetadata: convertUsageMetadata(resp.Usage),
		FinishReason:  convertFinishReason(string(choice.FinishReason)),
		TurnComplete:  true,
	}, nil
}

func convertTools(genaiTools []*genai.Tool) ([]openai.ChatCompletionToolUnionParam, error) {
	var tools []openai.ChatCompletionToolUnionParam
	for _, genaiTool := range genaiTools {
		if genaiTool == nil {
			continue
		}
		for _, funcDecl := range genaiTool.FunctionDeclarations {
			params := funcDecl.ParametersJsonSchema
			if params == nil {
				params = funcDecl.Parameters
			}
			tools = append(tools, openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
				Name:        funcDecl.Name,
				Description: openai.String(funcDecl.Description),
				Parameters:  convertToFunctionParams(params),
			}))
		}
	}
	return tools, nil
}

func convertToFunctionParams(params any) shared.FunctionParameters {
	if params == nil {
		return nil
	}
	var m map[string]any
	if dm, ok := params.(map[string]any); ok {
		m = dm
	} else {
		jsonBytes, err := json.Marshal(params)
		if err != nil {
			return nil
		}
		if json.Unmarshal(jsonBytes, &m) != nil {
			return nil
		}
	}
	ensureObjectProperties(m)
	return shared.FunctionParameters(m)
}

func ensureObjectProperties(schema map[string]any) {
	if schema == nil {
		return
	}
	if t, ok := schema["type"].(string); ok && t == "object" {
		if _, hasProps := schema["properties"]; !hasProps {
			schema["properties"] = map[string]any{}
		}
	}
	if props, ok := schema["properties"].(map[string]any); ok {
		for _, prop := range props {
			if propMap, ok := prop.(map[string]any); ok {
				ensureObjectProperties(propMap)
			}
		}
	}
	if items, ok := schema["items"].(map[string]any); ok {
		ensureObjectProperties(items)
	}
}

func convertSchema(schema *genai.Schema) (map[string]any, error) {
	if schema == nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}, nil
	}
	result := make(map[string]any)
	if schema.Type != genai.TypeUnspecified {
		result["type"] = schemaTypeToString(schema.Type)
	}
	if schema.Description != "" {
		result["description"] = schema.Description
	}
	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}
	if len(schema.Enum) > 0 {
		result["enum"] = schema.Enum
	}
	if len(schema.Properties) > 0 {
		props := make(map[string]any)
		for name, propSchema := range schema.Properties {
			converted, err := convertSchema(propSchema)
			if err != nil {
				return nil, err
			}
			props[name] = converted
		}
		result["properties"] = props
	}
	if schema.Items != nil {
		items, err := convertSchema(schema.Items)
		if err != nil {
			return nil, err
		}
		result["items"] = items
	}
	return result, nil
}

func convertInlineDataToPart(data *genai.Blob) (*openai.ChatCompletionContentPartUnionParam, error) {
	if data == nil {
		return nil, fmt.Errorf("inline data is nil")
	}
	mediaType := data.MIMEType
	base64Data := base64.StdEncoding.EncodeToString(data.Data)

	switch {
	case mediaType == "image/jpeg" || mediaType == "image/jpg" || mediaType == "image/png" ||
		mediaType == "image/gif" || mediaType == "image/webp":
		return &openai.ChatCompletionContentPartUnionParam{
			OfImageURL: &openai.ChatCompletionContentPartImageParam{
				ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
					URL:    fmt.Sprintf("data:%s;base64,%s", mediaType, base64Data),
					Detail: "auto",
				},
			},
		}, nil
	case mediaType == "audio/wav" || mediaType == "audio/mp3" ||
		mediaType == "audio/mpeg" || mediaType == "audio/webm":
		format := "wav"
		if mediaType == "audio/mp3" || mediaType == "audio/mpeg" {
			format = "mp3"
		}
		return &openai.ChatCompletionContentPartUnionParam{
			OfInputAudio: &openai.ChatCompletionContentPartInputAudioParam{
				InputAudio: openai.ChatCompletionContentPartInputAudioInputAudioParam{
					Data:   base64Data,
					Format: format,
				},
			},
		}, nil
	case mediaType == "application/pdf" || strings.HasPrefix(mediaType, "text/"):
		return &openai.ChatCompletionContentPartUnionParam{
			OfFile: &openai.ChatCompletionContentPartFileParam{
				File: openai.ChatCompletionContentPartFileFileParam{
					FileData: openai.String(fmt.Sprintf("data:%s;base64,%s", mediaType, base64Data)),
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported inline data MIME type for OpenAI: %s", mediaType)
	}
}

func convertUsageMetadata(usage openai.CompletionUsage) *genai.GenerateContentResponseUsageMetadata {
	if usage.TotalTokens == 0 {
		return nil
	}
	return &genai.GenerateContentResponseUsageMetadata{
		PromptTokenCount:     int32(usage.PromptTokens),
		CandidatesTokenCount: int32(usage.CompletionTokens),
		TotalTokenCount:      int32(usage.TotalTokens),
	}
}

func convertRole(role string) string {
	if role == "model" {
		return "assistant"
	}
	return role
}

func convertFinishReason(reason string) genai.FinishReason {
	switch reason {
	case "stop", "tool_calls", "function_call":
		return genai.FinishReasonStop
	case "length":
		return genai.FinishReasonMaxTokens
	case "content_filter":
		return genai.FinishReasonSafety
	default:
		return genai.FinishReasonUnspecified
	}
}

func convertThinkingLevel(level genai.ThinkingLevel) shared.ReasoningEffort {
	switch level {
	case genai.ThinkingLevelLow:
		return shared.ReasoningEffortLow
	case genai.ThinkingLevelHigh:
		return shared.ReasoningEffortHigh
	default:
		return shared.ReasoningEffortMedium
	}
}

func schemaTypeToString(t genai.Type) string {
	types := map[genai.Type]string{
		genai.TypeString:  "string",
		genai.TypeNumber:  "number",
		genai.TypeInteger: "integer",
		genai.TypeBoolean: "boolean",
		genai.TypeArray:   "array",
		genai.TypeObject:  "object",
	}
	if s, ok := types[t]; ok {
		return s
	}
	return "string"
}

func extractText(content *genai.Content) string {
	if content == nil {
		return ""
	}
	var texts []string
	for _, part := range content.Parts {
		if part.Text != "" {
			texts = append(texts, part.Text)
		}
	}
	return joinTexts(texts)
}

func joinTexts(texts []string) string {
	return strings.Join(texts, "\n")
}

func parseJSONArgs(argsJSON string) map[string]any {
	if argsJSON == "" {
		return make(map[string]any)
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return make(map[string]any)
	}
	return args
}
