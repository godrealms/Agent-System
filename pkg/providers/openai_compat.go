package providers

// OpenAI-compatible provider
//
// Covers: ChatGPT (OpenAI), Kimi (Moonshot), MINIMAX, OpenRouter,
// GroqCloud, Hugging Face Inference Providers, Fireworks AI,
// Cloudflare Workers AI — all of these expose the same
// POST /v1/chat/completions endpoint.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAICompatConfig configures a single OpenAI-compatible endpoint.
type OpenAICompatConfig struct {
	ProvType     ProviderType
	BaseURL      string
	APIKey       string
	Model        string
	ExtraHeaders map[string]string // e.g. HTTP-Referer for OpenRouter
}

// Preset constructors ---------------------------------------------------

func OpenAIPreset(apiKey, model string) OpenAICompatConfig {
	if model == "" {
		model = "gpt-4o"
	}
	return OpenAICompatConfig{
		ProvType: ProviderOpenAI,
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   apiKey,
		Model:    model,
	}
}

func KimiPreset(apiKey, model string) OpenAICompatConfig {
	if model == "" {
		model = "moonshot-v1-8k"
	}
	return OpenAICompatConfig{
		ProvType: ProviderKimi,
		BaseURL:  "https://api.moonshot.cn/v1",
		APIKey:   apiKey,
		Model:    model,
	}
}

func MiniMaxPreset(apiKey, model string) OpenAICompatConfig {
	if model == "" {
		model = "MiniMax-Text-01"
	}
	return OpenAICompatConfig{
		ProvType: ProviderMiniMax,
		BaseURL:  "https://api.minimax.chat/v1",
		APIKey:   apiKey,
		Model:    model,
	}
}

func OpenRouterPreset(apiKey, model string) OpenAICompatConfig {
	if model == "" {
		model = "openai/gpt-4o"
	}
	return OpenAICompatConfig{
		ProvType: ProviderOpenRouter,
		BaseURL:  "https://openrouter.ai/api/v1",
		APIKey:   apiKey,
		Model:    model,
		ExtraHeaders: map[string]string{
			"HTTP-Referer": "https://github.com/godrealms/Agent-System",
			"X-Title":      "AI Agent System",
		},
	}
}

func GroqPreset(apiKey, model string) OpenAICompatConfig {
	if model == "" {
		model = "llama-3.3-70b-versatile"
	}
	return OpenAICompatConfig{
		ProvType: ProviderGroq,
		BaseURL:  "https://api.groq.com/openai/v1",
		APIKey:   apiKey,
		Model:    model,
	}
}

func HuggingFacePreset(apiKey, model string) OpenAICompatConfig {
	if model == "" {
		model = "meta-llama/Llama-3.3-70B-Instruct"
	}
	return OpenAICompatConfig{
		ProvType: ProviderHuggingFace,
		BaseURL:  "https://api-inference.huggingface.co/v1",
		APIKey:   apiKey,
		Model:    model,
	}
}

func FireworksPreset(apiKey, model string) OpenAICompatConfig {
	if model == "" {
		model = "accounts/fireworks/models/llama-v3p3-70b-instruct"
	}
	return OpenAICompatConfig{
		ProvType: ProviderFireworks,
		BaseURL:  "https://api.fireworks.ai/inference/v1",
		APIKey:   apiKey,
		Model:    model,
	}
}

// CloudflarePreset builds the config for Cloudflare Workers AI.
// The OpenAI-compatible endpoint is under the account's AI gateway.
func CloudflarePreset(apiKey, accountID, model string) OpenAICompatConfig {
	if model == "" {
		model = "@cf/meta/llama-3.1-8b-instruct"
	}
	return OpenAICompatConfig{
		ProvType: ProviderCloudflare,
		BaseURL:  fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/ai/v1", accountID),
		APIKey:   apiKey,
		Model:    model,
	}
}

// OpenAICompatProvider -------------------------------------------------

// OpenAICompatProvider implements Provider for any OpenAI-compatible API.
type OpenAICompatProvider struct {
	cfg    OpenAICompatConfig
	client *http.Client
}

// NewOpenAICompatProvider creates a provider from the given config.
func NewOpenAICompatProvider(cfg OpenAICompatConfig) *OpenAICompatProvider {
	return &OpenAICompatProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *OpenAICompatProvider) ProviderType() ProviderType { return p.cfg.ProvType }

// Chat sends one turn to the OpenAI-compatible completions endpoint.
func (p *OpenAICompatProvider) Chat(ctx context.Context, system string, messages []Message, tools []ToolDef) (*ChatResponse, error) {
	// ----- build messages -----
	reqMsgs := []map[string]interface{}{
		{"role": "system", "content": system},
	}

	for _, msg := range messages {
		switch msg.Role {
		case RoleUser:
			reqMsgs = append(reqMsgs, map[string]interface{}{
				"role":    "user",
				"content": msg.Content,
			})

		case RoleAssistant:
			if len(msg.ToolCalls) > 0 {
				calls := make([]map[string]interface{}, len(msg.ToolCalls))
				for i, tc := range msg.ToolCalls {
					calls[i] = map[string]interface{}{
						"id":   tc.ID,
						"type": "function",
						"function": map[string]interface{}{
							"name":      tc.Name,
							"arguments": string(tc.Arguments),
						},
					}
				}
				reqMsgs = append(reqMsgs, map[string]interface{}{
					"role":       "assistant",
					"content":    nil,
					"tool_calls": calls,
				})
			} else {
				reqMsgs = append(reqMsgs, map[string]interface{}{
					"role":    "assistant",
					"content": msg.Content,
				})
			}

		case RoleTool:
			reqMsgs = append(reqMsgs, map[string]interface{}{
				"role":         "tool",
				"content":      msg.Content,
				"tool_call_id": msg.ToolCallID,
			})
		}
	}

	// ----- build tools -----
	body := map[string]interface{}{
		"model":      p.cfg.Model,
		"messages":   reqMsgs,
		"max_tokens": 8192,
	}
	if len(tools) > 0 {
		reqTools := make([]map[string]interface{}, len(tools))
		for i, t := range tools {
			reqTools[i] = map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        t.Name,
					"description": t.Description,
					"parameters": map[string]interface{}{
						"type":       "object",
						"properties": t.Properties,
						"required":   t.Required,
					},
				},
			}
		}
		body["tools"] = reqTools
		body["tool_choice"] = "auto"
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// ----- HTTP request -----
	req, err := http.NewRequestWithContext(ctx, "POST", p.cfg.BaseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	for k, v := range p.cfg.ExtraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s API error %d: %s", p.cfg.ProvType, resp.StatusCode, respBody)
	}

	// ----- parse response -----
	var result struct {
		Choices []struct {
			Message struct {
				Content   *string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string          `json:"name"`
						Arguments json.RawMessage `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response from %s", p.cfg.ProvType)
	}

	chatResp := &ChatResponse{
		Usage: TokenUsage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
	}
	if result.Choices[0].Message.Content != nil {
		chatResp.Content = *result.Choices[0].Message.Content
	}
	for _, tc := range result.Choices[0].Message.ToolCalls {
		chatResp.ToolCalls = append(chatResp.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	return chatResp, nil
}
