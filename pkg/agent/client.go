package agent

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIClient wraps the OpenAI API client
type OpenAIClient struct {
	client *openai.Client
	model  string
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(apiKey string) *OpenAIClient {
	config := openai.DefaultConfig(apiKey)
	client := openai.NewClientWithConfig(config)

	return &OpenAIClient{
		client: client,
		model:  openai.GPT4TurboPreview, // Using latest model
	}
}

// ChatCompletion sends a chat completion request
func (c *OpenAIClient) ChatCompletion(ctx context.Context, prompt string) (string, error) {
	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		Temperature: 0.7,
	})

	if err != nil {
		return "", fmt.Errorf("chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from API")
	}

	return resp.Choices[0].Message.Content, nil
}

// SetModel sets the model to use
func (c *OpenAIClient) SetModel(model string) {
	c.model = model
}
