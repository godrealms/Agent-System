package providers

// Google Gemini provider (native REST API).
// Gemini uses a different conversation format from OpenAI:
//   - roles are "user" / "model" (not "assistant")
//   - tool calls appear as functionCall parts in a model turn
//   - tool results appear as functionResponse parts in a user turn
//   - multiple consecutive tool results share one user turn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models"

// GeminiProvider implements Provider using Google's generateContent REST API.
type GeminiProvider struct {
	apiKey string
	model  string
	client *http.Client
}

// NewGeminiProvider creates a Gemini provider.
// model defaults to "gemini-2.0-flash" when empty.
func NewGeminiProvider(apiKey, model string) *GeminiProvider {
	if model == "" {
		model = "gemini-2.0-flash"
	}
	return &GeminiProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *GeminiProvider) ProviderType() ProviderType { return ProviderGemini }

// Chat sends the conversation to Gemini and returns a ChatResponse.
func (p *GeminiProvider) Chat(ctx context.Context, system string, messages []Message, tools []ToolDef) (*ChatResponse, error) {
	reqBody := p.buildRequest(system, messages, tools)

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiBaseURL, p.model, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("Gemini API error %d: %s", resp.StatusCode, respBody)
	}

	return p.parseResponse(respBody)
}

// ---- internal Gemini request/response types --------------------------

type geminiRequest struct {
	SystemInstruction *geminiContent       `json:"systemInstruction,omitempty"`
	Contents          []geminiContent      `json:"contents"`
	Tools             []geminiToolConfig   `json:"tools,omitempty"`
	GenerationConfig  map[string]any       `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             *string                  `json:"text,omitempty"`
	FunctionCall     *geminiFunctionCall      `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse  `json:"functionResponse,omitempty"`
}

type geminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type geminiFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type geminiToolConfig struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations"`
}

type geminiFunctionDecl struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// buildRequest converts common types to Gemini's request format.
func (p *GeminiProvider) buildRequest(system string, messages []Message, tools []ToolDef) geminiRequest {
	req := geminiRequest{
		GenerationConfig: map[string]any{"maxOutputTokens": 8192},
	}

	// System instruction
	if system != "" {
		text := system
		req.SystemInstruction = &geminiContent{
			Role:  "user",
			Parts: []geminiPart{{Text: &text}},
		}
	}

	// Tools
	if len(tools) > 0 {
		decls := make([]geminiFunctionDecl, len(tools))
		for i, t := range tools {
			decls[i] = geminiFunctionDecl{
				Name:        t.Name,
				Description: t.Description,
				Parameters: map[string]any{
					"type":       "object",
					"properties": t.Properties,
					"required":   t.Required,
				},
			}
		}
		req.Tools = []geminiToolConfig{{FunctionDeclarations: decls}}
	}

	// Messages — convert to Gemini contents.
	// Gemini requires alternating user/model roles.
	// Multiple consecutive tool results share one user turn.
	for i := 0; i < len(messages); {
		msg := messages[i]

		switch msg.Role {
		case RoleUser:
			text := msg.Content
			req.Contents = append(req.Contents, geminiContent{
				Role:  "user",
				Parts: []geminiPart{{Text: &text}},
			})
			i++

		case RoleAssistant:
			var parts []geminiPart
			if msg.Content != "" {
				text := msg.Content
				parts = append(parts, geminiPart{Text: &text})
			}
			for _, tc := range msg.ToolCalls {
				var args map[string]any
				_ = json.Unmarshal(tc.Arguments, &args)
				parts = append(parts, geminiPart{
					FunctionCall: &geminiFunctionCall{Name: tc.Name, Args: args},
				})
			}
			if len(parts) > 0 {
				req.Contents = append(req.Contents, geminiContent{Role: "model", Parts: parts})
			}
			i++

		case RoleTool:
			// Collect consecutive tool results into one user turn.
			var parts []geminiPart
			for i < len(messages) && messages[i].Role == RoleTool {
				m := messages[i]
				parts = append(parts, geminiPart{
					FunctionResponse: &geminiFunctionResponse{
						Name:     m.ToolName,
						Response: map[string]any{"result": m.Content},
					},
				})
				i++
			}
			req.Contents = append(req.Contents, geminiContent{Role: "user", Parts: parts})

		default:
			i++
		}
	}

	return req
}

// parseResponse extracts text and function calls from Gemini's response.
func (p *GeminiProvider) parseResponse(body []byte) (*ChatResponse, error) {
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         *string `json:"text"`
					FunctionCall *struct {
						Name string         `json:"name"`
						Args map[string]any `json:"args"`
					} `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in Gemini response")
	}

	chatResp := &ChatResponse{
		Usage: TokenUsage{
			PromptTokens:     result.UsageMetadata.PromptTokenCount,
			CompletionTokens: result.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      result.UsageMetadata.TotalTokenCount,
		},
	}

	for idx, part := range result.Candidates[0].Content.Parts {
		if part.Text != nil {
			chatResp.Content += *part.Text
		}
		if part.FunctionCall != nil {
			args, _ := json.Marshal(part.FunctionCall.Args)
			chatResp.ToolCalls = append(chatResp.ToolCalls, ToolCall{
				// Gemini doesn't provide tool-call IDs; generate one.
				ID:        fmt.Sprintf("gemini-%d", idx),
				Name:      part.FunctionCall.Name,
				Arguments: args,
			})
		}
	}

	return chatResp, nil
}
