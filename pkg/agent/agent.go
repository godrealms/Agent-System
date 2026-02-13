package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"AI-agent/internal/config"
)

// AgentType represents the type of agent
type AgentType string

const (
	InitializerAgent AgentType = "initializer"
	CodingAgent      AgentType = "coding"
)

// Agent represents a long-running agent instance
type Agent struct {
	Type       AgentType
	Config     *config.Config
	Client     *OpenAIClient
	Context    context.Context
	CancelFunc context.CancelFunc
}

// SessionResult represents the result of an agent session
type SessionResult struct {
	SessionID     string
	Success       bool
	Message       string
	FeaturesDone  []string
	FeaturesAdded []string
	CommitHash    string
	Duration      time.Duration
}

// NewAgent creates a new agent instance
func NewAgent(agentType AgentType, cfg *config.Config) (*Agent, error) {
	if cfg.OpenAIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	client := NewOpenAIClient(cfg.OpenAIKey)

	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(cfg.SessionTimeout)*time.Minute)

	return &Agent{
		Type:       agentType,
		Config:     cfg,
		Client:     client,
		Context:    ctx,
		CancelFunc: cancel,
	}, nil
}

// Run executes the agent session
func (a *Agent) Run() (*SessionResult, error) {
	defer a.CancelFunc()

	startTime := time.Now()
	sessionID := fmt.Sprintf("session-%d", startTime.Unix())

	log.Printf("[%s] Starting %s agent session", sessionID, a.Type)

	var result *SessionResult
	var err error

	switch a.Type {
	case InitializerAgent:
		result, err = a.runInitializer()
	case CodingAgent:
		result, err = a.runCodingAgent()
	default:
		return nil, fmt.Errorf("unknown agent type: %s", a.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("agent session failed: %w", err)
	}

	result.SessionID = sessionID
	result.Duration = time.Since(startTime)

	log.Printf("[%s] Session completed in %v", sessionID, result.Duration)
	return result, nil
}

// runInitializer handles the first-time project setup
func (a *Agent) runInitializer() (*SessionResult, error) {
	prompt := a.getInitializerPrompt()

	response, err := a.Client.ChatCompletion(a.Context, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get initializer response: %w", err)
	}

	// Parse the response and extract setup information
	featuresAdded := a.parseFeaturesFromResponse(response)

	result := &SessionResult{
		Success:       true,
		Message:       response,
		FeaturesAdded: featuresAdded,
	}

	return result, nil
}

// runCodingAgent handles incremental development sessions
func (a *Agent) runCodingAgent() (*SessionResult, error) {
	// Get current project state
	state, err := a.getProjectState()
	if err != nil {
		return nil, fmt.Errorf("failed to get project state: %w", err)
	}

	prompt := a.getCodingAgentPrompt(state)

	response, err := a.Client.ChatCompletion(a.Context, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get coding response: %w", err)
	}

	// Parse the response and extract completed features
	featuresDone := a.parseCompletedFeatures(response)
	commitHash := a.extractCommitHash(response)

	result := &SessionResult{
		Success:      true,
		Message:      response,
		FeaturesDone: featuresDone,
		CommitHash:   commitHash,
	}

	return result, nil
}

// getInitializerPrompt returns the prompt for the initializer agent
func (a *Agent) getInitializerPrompt() string {
	return fmt.Sprintf(`You are the Initializer Agent responsible for setting up a new project environment.

TASK: Set up the development environment for a web application based on the user's requirements.

STEPS:
1. Analyze the project requirements and create a comprehensive feature list
2. Set up the initial project structure and files
3. Create an init.sh script for starting the development server
4. Initialize git repository and make initial commit
5. Create claude-progress.txt for tracking work progress
6. Write feature_list.json with all required features marked as failing

PROJECT DIRECTORY: %s

REQUIREMENTS:
- Create a clean, professional project structure
- Document all setup steps in the progress file
- Make sure all features are clearly defined and testable
- Use JSON format for feature list to prevent accidental modifications
- Include comprehensive setup instructions

EXPECTED OUTPUT FORMAT:
1. Project structure created
2. init.sh script written
3. Git initialized with initial commit
4. claude-progress.txt created with setup summary
5. feature_list.json created with all features
6. Ready for coding agent to start implementation`, a.Config.ProjectDir)
}

// getCodingAgentPrompt returns the prompt for the coding agent
func (a *Agent) getCodingAgentPrompt(state *ProjectState) string {
	return fmt.Sprintf(`You are the Coding Agent responsible for incremental development.

CURRENT PROJECT STATE:
%s

TASK: Work on implementing one feature at a time, leaving the codebase in a clean, mergeable state.

MANDATORY STEPS FOR EACH SESSION:
1. Run pwd to confirm working directory
2. Read git logs and progress files to understand recent work
3. Read feature_list.json and choose highest priority incomplete feature
4. Implement ONLY that feature completely
5. Test the feature thoroughly end-to-end
6. Commit changes with descriptive message
7. Update progress file with what was accomplished
8. Verify the app still works overall

RULES:
- Work on exactly ONE feature per session
- Leave code in production-ready state
- All tests must pass before committing
- Never remove or modify existing tests
- Always verify existing functionality still works
- Make small, focused commits
- Document everything clearly

EXPECTED OUTPUT FORMAT:
1. Feature implementation completed
2. Tests passed
3. Code committed with descriptive message
4. Progress file updated
5. Ready for next session`, state.String())
}

// Close cleans up the agent resources
func (a *Agent) Close() {
	a.CancelFunc()
}
