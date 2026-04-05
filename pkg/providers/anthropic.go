package providers

// Anthropic (Claude) provider.
// Uses the official anthropic-sdk-go with streaming + adaptive thinking.

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicProvider implements Provider using the Anthropic Claude API.
type AnthropicProvider struct {
	client anthropic.Client
	model  string
}

// NewAnthropicProvider creates an Anthropic provider.
// model defaults to "claude-opus-4-6" when empty.
func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	if model == "" {
		model = "claude-opus-4-6"
	}
	return &AnthropicProvider{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}
}

func (p *AnthropicProvider) ProviderType() ProviderType { return ProviderClaude }

// Chat converts the provider-agnostic conversation to Anthropic format,
// streams the response, and returns a provider-agnostic ChatResponse.
func (p *AnthropicProvider) Chat(ctx context.Context, system string, messages []Message, tools []ToolDef) (*ChatResponse, error) {
	// --- convert messages ---
	anthMessages, err := toAnthropicMessages(messages)
	if err != nil {
		return nil, fmt.Errorf("convert messages: %w", err)
	}

	// --- convert tools ---
	anthTools := toAnthropicTools(tools)

	// --- adaptive thinking (recommended for Opus 4.6) ---
	adaptiveParam := anthropic.NewThinkingConfigAdaptiveParam()
	thinking := anthropic.ThinkingConfigParamUnion{OfAdaptive: &adaptiveParam}

	// --- stream request ---
	stream := p.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 8192,
		System:    []anthropic.TextBlockParam{{Text: system}},
		Tools:     anthTools,
		Messages:  anthMessages,
		Thinking:  thinking,
	})

	var response anthropic.Message
	for stream.Next() {
		if err := response.Accumulate(stream.Current()); err != nil {
			return nil, fmt.Errorf("accumulate: %w", err)
		}
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("stream: %w", err)
	}

	// --- build ChatResponse ---
	chatResp := &ChatResponse{
		Usage: TokenUsage{
			PromptTokens:     int(response.Usage.InputTokens),
			CompletionTokens: int(response.Usage.OutputTokens),
			TotalTokens:      int(response.Usage.InputTokens) + int(response.Usage.OutputTokens),
		},
	}

	for _, block := range response.Content {
		switch v := block.AsAny().(type) {
		case anthropic.TextBlock:
			chatResp.Content += v.Text
		case anthropic.ToolUseBlock:
			chatResp.ToolCalls = append(chatResp.ToolCalls, ToolCall{
				ID:        v.ID,
				Name:      v.Name,
				Arguments: v.Input,
			})
		}
	}

	return chatResp, nil
}

// toAnthropicMessages converts []Message → []anthropic.MessageParam.
// Consecutive RoleTool messages are grouped into a single user turn
// (Anthropic accepts multiple tool_result blocks in one user message).
func toAnthropicMessages(messages []Message) ([]anthropic.MessageParam, error) {
	var result []anthropic.MessageParam
	i := 0
	for i < len(messages) {
		msg := messages[i]

		switch msg.Role {
		case RoleUser:
			result = append(result, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
			i++

		case RoleAssistant:
			var parts []anthropic.ContentBlockParamUnion
			if msg.Content != "" {
				parts = append(parts, anthropic.NewTextBlock(msg.Content))
			}
			for _, tc := range msg.ToolCalls {
				parts = append(parts, anthropic.NewToolUseBlock(tc.ID, tc.Arguments, tc.Name))
			}
			result = append(result, anthropic.NewAssistantMessage(parts...))
			i++

		case RoleTool:
			// Collect all consecutive tool results into one user message.
			var toolResults []anthropic.ContentBlockParamUnion
			for i < len(messages) && messages[i].Role == RoleTool {
				toolResults = append(toolResults,
					anthropic.NewToolResultBlock(messages[i].ToolCallID, messages[i].Content, false))
				i++
			}
			result = append(result, anthropic.NewUserMessage(toolResults...))

		default:
			i++
		}
	}
	return result, nil
}

// toAnthropicTools converts []ToolDef → []anthropic.ToolUnionParam.
func toAnthropicTools(tools []ToolDef) []anthropic.ToolUnionParam {
	if len(tools) == 0 {
		return nil
	}
	result := make([]anthropic.ToolUnionParam, len(tools))
	for i, t := range tools {
		tp := anthropic.ToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: t.Properties,
				Required:   t.Required,
			},
		}
		result[i] = anthropic.ToolUnionParam{OfTool: &tp}
	}
	return result
}
