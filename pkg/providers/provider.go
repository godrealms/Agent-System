// Package providers defines a unified interface for multiple LLM backends.
// All providers implement the Provider interface so the agent can switch
// between Claude, ChatGPT, Kimi, MINIMAX, OpenRouter, Groq, Gemini,
// HuggingFace, Fireworks and Cloudflare Workers AI without changing
// any business logic.
package providers

import (
	"context"
	"encoding/json"
)

// ProviderType identifies an LLM backend.
type ProviderType string

const (
	ProviderClaude      ProviderType = "claude"
	ProviderOpenAI      ProviderType = "openai"
	ProviderKimi        ProviderType = "kimi"
	ProviderMiniMax     ProviderType = "minimax"
	ProviderOpenRouter  ProviderType = "openrouter"
	ProviderGroq        ProviderType = "groq"
	ProviderGemini      ProviderType = "gemini"
	ProviderHuggingFace ProviderType = "huggingface"
	ProviderFireworks   ProviderType = "fireworks"
	ProviderCloudflare  ProviderType = "cloudflare"
)

// Role is the participant role in a conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolCall represents a single tool invocation requested by the model.
type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

// Message is the provider-agnostic conversation unit.
type Message struct {
	Role      Role
	Content   string     // text (user/assistant) or tool result text (tool)
	ToolCalls []ToolCall // non-empty only when Role == RoleAssistant

	// Tool result metadata (Role == RoleTool)
	ToolCallID string // matches the ToolCall.ID this is answering
	ToolName   string // tool name (required by Gemini)
}

// ToolDef describes a tool the model may call.
type ToolDef struct {
	Name        string
	Description string
	Properties  map[string]interface{} // JSON Schema property map
	Required    []string
}

// ChatResponse is the model's reply to a single Chat call.
type ChatResponse struct {
	Content   string
	ToolCalls []ToolCall
	Usage     TokenUsage
}

// TokenUsage tracks API token consumption.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Provider is the common interface every LLM backend must satisfy.
type Provider interface {
	// ProviderType returns the backend identifier.
	ProviderType() ProviderType

	// Chat sends the conversation to the model and returns its response.
	// system is the system-prompt string; messages is the full history so far.
	// If tools is non-empty the model may respond with ToolCalls instead of text.
	Chat(ctx context.Context, system string, messages []Message, tools []ToolDef) (*ChatResponse, error)
}
