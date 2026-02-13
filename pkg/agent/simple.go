package agent

import (
	"context"
	"fmt"
	"time"
)

type AgentType string

const (
	InitializerAgent AgentType = "initializer"
	CodingAgent      AgentType = "coding"
)

type Agent struct {
	Type       AgentType
	Context    context.Context
	CancelFunc context.CancelFunc
}

type SessionResult struct {
	SessionID     string
	Success       bool
	Message       string
	FeaturesDone  []string
	FeaturesAdded []string
	CommitHash    string
	Duration      time.Duration
	TokenUsage    TokenUsage
}

type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

func NewAgent(agentType AgentType, projectDir string) (*Agent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)

	return &Agent{
		Type:       agentType,
		Context:    ctx,
		CancelFunc: cancel,
	}, nil
}

func (a *Agent) Run() (*SessionResult, error) {
	defer a.CancelFunc()

	startTime := time.Now()
	sessionID := fmt.Sprintf("session-%d", startTime.Unix())

	// Simulate agent work
	time.Sleep(2 * time.Second)

	result := &SessionResult{
		SessionID:    sessionID,
		Success:      true,
		Message:      "Session completed successfully",
		FeaturesDone: []string{"feature-1", "feature-2"},
		CommitHash:   "abc123",
		Duration:     time.Since(startTime),
		TokenUsage:   TokenUsage{TotalTokens: 1000},
	}

	return result, nil
}

func (a *Agent) Close() {
	a.CancelFunc()
}
