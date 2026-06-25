package llm

import (
	"context"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// ChatModel is ADK's model.LLM plus gmd ergonomic helpers.
// The OpenAI adapter implements this. Future Gemini/Vertex adapters can too.
type ChatModel interface {
	model.LLM

	// Chat is a simple text-in/text-out helper for the common system+user pattern.
	Chat(ctx context.Context, system, user string) (string, error)

	// ChatMessages is the full-control path for multi-turn chat with genai.Content.
	ChatMessages(ctx context.Context, contents []*genai.Content) (string, error)
}
