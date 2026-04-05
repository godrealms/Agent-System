package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"AI-agent/pkg/providers"
)

// AgentType identifies the role of the agent.
type AgentType string

const (
	InitializerAgent AgentType = "initializer"
	CodingAgent      AgentType = "coding"
)

// SessionResult holds the outcome of one agent session.
type SessionResult struct {
	SessionID    string
	Success      bool
	Message      string
	FeaturesDone []string
	CommitHash   string
	Duration     time.Duration
	TokenUsage   providers.TokenUsage
}

// Agent drives a single session using any Provider.
type Agent struct {
	Type       AgentType
	projectDir string
	provider   providers.Provider
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewAgent creates an Agent that uses the given provider.
func NewAgent(agentType AgentType, projectDir string, provider providers.Provider) (*Agent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	return &Agent{
		Type:       agentType,
		projectDir: projectDir,
		provider:   provider,
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

// Run executes a full agent session and returns the result.
func (a *Agent) Run() (*SessionResult, error) {
	defer a.cancel()

	startTime := time.Now()
	sessionID := fmt.Sprintf("session-%d", startTime.Unix())

	log.Printf("[%s] Starting %s session (provider: %s)", sessionID, a.Type, a.provider.ProviderType())

	var (
		featuresDone []string
		commitHash   string
		totalUsage   providers.TokenUsage
		runErr       error
	)

	switch a.Type {
	case InitializerAgent:
		commitHash, totalUsage, runErr = a.runInitializer(sessionID)
	case CodingAgent:
		featuresDone, commitHash, totalUsage, runErr = a.runCoding(sessionID)
	default:
		runErr = fmt.Errorf("unknown agent type: %s", a.Type)
	}

	if runErr != nil {
		return &SessionResult{
			SessionID: sessionID,
			Success:   false,
			Message:   runErr.Error(),
			Duration:  time.Since(startTime),
		}, runErr
	}

	return &SessionResult{
		SessionID:    sessionID,
		Success:      true,
		Message:      "Session completed successfully",
		FeaturesDone: featuresDone,
		CommitHash:   commitHash,
		Duration:     time.Since(startTime),
		TokenUsage:   totalUsage,
	}, nil
}

// Close cancels the session context.
func (a *Agent) Close() { a.cancel() }

// ---------------------------------------------------------------------------
// Initializer agent
// ---------------------------------------------------------------------------

func (a *Agent) runInitializer(sessionID string) (string, providers.TokenUsage, error) {
	system := fmt.Sprintf(`You are a project initializer agent. Set up a new software project in: %s

Use the available tools to:
1. Create a README.md describing the project
2. Set up the project directory structure
3. Create an init.sh script to start the development server

Be concise — create only essential files.`, a.projectDir)

	_, usage, err := a.runAgenticLoop(sessionID, system,
		"Set up this project directory with a good structure and initial files.")
	if err != nil {
		return "", usage, err
	}
	return a.tryGitCommit("Initial project setup by AI agent"), usage, nil
}

// ---------------------------------------------------------------------------
// Coding agent
// ---------------------------------------------------------------------------

func (a *Agent) runCoding(sessionID string) ([]string, string, providers.TokenUsage, error) {
	featureList, err := a.readFeatureList()
	if err != nil {
		log.Printf("[%s] Warning: could not read feature list: %v", sessionID, err)
		featureList = "No feature list found. Please explore the project and implement improvements."
	}
	progress, err := a.readProgress()
	if err != nil {
		progress = "No progress file found."
	}

	system := fmt.Sprintf(`You are a coding agent working on a software project at: %s

Each session you must:
1. Read the feature list and progress to understand current state
2. Pick ONE pending feature (highest priority)
3. Implement it fully, including tests where applicable
4. Mark it complete via update_feature_status
5. Record progress via append_progress

Current feature list:
%s

Current progress:
%s

Be focused: one feature per session.`, a.projectDir, featureList, progress)

	_, usage, err := a.runAgenticLoop(sessionID, system,
		"Please implement the next pending feature from the feature list.")
	if err != nil {
		return nil, "", usage, err
	}

	featuresDone := a.detectCompletedFeatures()
	commitHash := ""
	if len(featuresDone) > 0 {
		commitHash = a.tryGitCommit(fmt.Sprintf("feat: implement %s", strings.Join(featuresDone, ", ")))
	}
	return featuresDone, commitHash, usage, nil
}

// ---------------------------------------------------------------------------
// Provider-agnostic agentic loop
// ---------------------------------------------------------------------------

func (a *Agent) runAgenticLoop(sessionID, system, userMessage string) ([]providers.Message, providers.TokenUsage, error) {
	tools := a.buildTools()
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: userMessage},
	}

	var totalUsage providers.TokenUsage
	const maxIterations = 20

	for i := 0; i < maxIterations; i++ {
		log.Printf("[%s] Iteration %d/%d", sessionID, i+1, maxIterations)

		resp, err := a.provider.Chat(a.ctx, system, messages, tools)
		if err != nil {
			return messages, totalUsage, fmt.Errorf("provider chat: %w", err)
		}

		totalUsage.PromptTokens += resp.Usage.PromptTokens
		totalUsage.CompletionTokens += resp.Usage.CompletionTokens
		totalUsage.TotalTokens += resp.Usage.TotalTokens

		// Append assistant turn.
		messages = append(messages, providers.Message{
			Role:      providers.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		if len(resp.ToolCalls) == 0 {
			log.Printf("[%s] Agent done (no tool calls)", sessionID)
			break
		}

		// Execute tools and append results.
		for _, tc := range resp.ToolCalls {
			log.Printf("[%s] Tool: %s", sessionID, tc.Name)
			output, isError := a.dispatchTool(tc.Name, tc.Arguments)
			content := output
			if isError {
				content = "ERROR: " + output
			}
			messages = append(messages, providers.Message{
				Role:       providers.RoleTool,
				Content:    content,
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
			})
		}
	}

	return messages, totalUsage, nil
}

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------

func (a *Agent) buildTools() []providers.ToolDef {
	return []providers.ToolDef{
		{
			Name:        "read_file",
			Description: "Read the contents of a file in the project directory",
			Properties: map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Relative path from project root",
				},
			},
			Required: []string{"path"},
		},
		{
			Name:        "write_file",
			Description: "Write content to a file, creating directories as needed",
			Properties: map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Relative path from project root",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "File content to write",
				},
			},
			Required: []string{"path", "content"},
		},
		{
			Name:        "list_files",
			Description: "List files and directories at a path",
			Properties: map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Relative path from project root (use '.' for root)",
				},
			},
			Required: []string{"path"},
		},
		{
			Name:        "run_command",
			Description: "Run a shell command in the project directory and return combined stdout+stderr",
			Properties: map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "Shell command to execute",
				},
			},
			Required: []string{"command"},
		},
		{
			Name:        "update_feature_status",
			Description: "Mark a feature as complete in feature_list.json",
			Properties: map[string]interface{}{
				"feature_id": map[string]interface{}{
					"type":        "string",
					"description": "The feature ID to mark as complete",
				},
			},
			Required: []string{"feature_id"},
		},
		{
			Name:        "append_progress",
			Description: "Append a timestamped message to the progress tracking file",
			Properties: map[string]interface{}{
				"message": map[string]interface{}{
					"type":        "string",
					"description": "Progress note to append",
				},
			},
			Required: []string{"message"},
		},
	}
}

// ---------------------------------------------------------------------------
// Tool execution
// ---------------------------------------------------------------------------

func (a *Agent) dispatchTool(name string, rawInput json.RawMessage) (string, bool) {
	var input map[string]interface{}
	if err := json.Unmarshal(rawInput, &input); err != nil {
		return fmt.Sprintf("failed to parse tool input: %v", err), true
	}
	switch name {
	case "read_file":
		return a.toolReadFile(input)
	case "write_file":
		return a.toolWriteFile(input)
	case "list_files":
		return a.toolListFiles(input)
	case "run_command":
		return a.toolRunCommand(input)
	case "update_feature_status":
		return a.toolUpdateFeatureStatus(input)
	case "append_progress":
		return a.toolAppendProgress(input)
	default:
		return fmt.Sprintf("unknown tool: %s", name), true
	}
}

func (a *Agent) toolReadFile(input map[string]interface{}) (string, bool) {
	path, _ := input["path"].(string)
	data, err := os.ReadFile(filepath.Join(a.projectDir, filepath.Clean(path)))
	if err != nil {
		return fmt.Sprintf("error reading file: %v", err), true
	}
	return string(data), false
}

func (a *Agent) toolWriteFile(input map[string]interface{}) (string, bool) {
	path, _ := input["path"].(string)
	content, _ := input["content"].(string)
	absPath := filepath.Join(a.projectDir, filepath.Clean(path))
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Sprintf("error creating directories: %v", err), true
	}
	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return fmt.Sprintf("error writing file: %v", err), true
	}
	return fmt.Sprintf("wrote %d bytes to %s", len(content), path), false
}

func (a *Agent) toolListFiles(input map[string]interface{}) (string, bool) {
	path, _ := input["path"].(string)
	entries, err := os.ReadDir(filepath.Join(a.projectDir, filepath.Clean(path)))
	if err != nil {
		return fmt.Sprintf("error listing directory: %v", err), true
	}
	var lines []string
	for _, e := range entries {
		kind := "file"
		if e.IsDir() {
			kind = "dir"
		}
		lines = append(lines, fmt.Sprintf("[%s] %s", kind, e.Name()))
	}
	return strings.Join(lines, "\n"), false
}

func (a *Agent) toolRunCommand(input map[string]interface{}) (string, bool) {
	command, _ := input["command"].(string)
	cmd := exec.CommandContext(a.ctx, "sh", "-c", command)
	cmd.Dir = a.projectDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("command failed (%v):\n%s", err, out), true
	}
	return string(out), false
}

func (a *Agent) toolUpdateFeatureStatus(input map[string]interface{}) (string, bool) {
	featureID, _ := input["feature_id"].(string)
	featureFile := filepath.Join(a.projectDir, "feature_list.json")
	data, err := os.ReadFile(featureFile)
	if err != nil {
		return fmt.Sprintf("could not read feature_list.json: %v", err), true
	}
	var fl map[string]interface{}
	if err := json.Unmarshal(data, &fl); err != nil {
		return fmt.Sprintf("could not parse feature_list.json: %v", err), true
	}
	features, _ := fl["features"].([]interface{})
	found := false
	for _, f := range features {
		feature, ok := f.(map[string]interface{})
		if ok && feature["id"] == featureID {
			feature["passes"] = true
			feature["completed_at"] = time.Now().Format(time.RFC3339)
			found = true
		}
	}
	if !found {
		return fmt.Sprintf("feature %q not found", featureID), true
	}
	updated, err := json.MarshalIndent(fl, "", "  ")
	if err != nil {
		return fmt.Sprintf("marshal: %v", err), true
	}
	if err := os.WriteFile(featureFile, updated, 0644); err != nil {
		return fmt.Sprintf("write: %v", err), true
	}
	return fmt.Sprintf("feature %q marked as complete", featureID), false
}

func (a *Agent) toolAppendProgress(input map[string]interface{}) (string, bool) {
	message, _ := input["message"].(string)
	progressFile := filepath.Join(a.projectDir, "claude-progress.txt")
	f, err := os.OpenFile(progressFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Sprintf("open progress file: %v", err), true
	}
	defer f.Close()
	line := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), message)
	if _, err := f.WriteString(line); err != nil {
		return fmt.Sprintf("write progress: %v", err), true
	}
	return "progress recorded", false
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (a *Agent) readFeatureList() (string, error) {
	data, err := os.ReadFile(filepath.Join(a.projectDir, "feature_list.json"))
	return string(data), err
}

func (a *Agent) readProgress() (string, error) {
	data, err := os.ReadFile(filepath.Join(a.projectDir, "claude-progress.txt"))
	return string(data), err
}

func (a *Agent) detectCompletedFeatures() []string {
	data, err := os.ReadFile(filepath.Join(a.projectDir, "feature_list.json"))
	if err != nil {
		return nil
	}
	var fl map[string]interface{}
	if err := json.Unmarshal(data, &fl); err != nil {
		return nil
	}
	var done []string
	for _, f := range fl["features"].([]interface{}) {
		feature, ok := f.(map[string]interface{})
		if ok {
			if passes, _ := feature["passes"].(bool); passes {
				if id, _ := feature["id"].(string); id != "" {
					done = append(done, id)
				}
			}
		}
	}
	return done
}

func (a *Agent) tryGitCommit(message string) string {
	stageCmd := exec.Command("git", "add", "-A")
	stageCmd.Dir = a.projectDir
	if err := stageCmd.Run(); err != nil {
		log.Printf("git add failed: %v", err)
		return ""
	}
	commitCmd := exec.Command("git", "commit", "-m", message)
	commitCmd.Dir = a.projectDir
	if out, err := commitCmd.CombinedOutput(); err != nil {
		log.Printf("git commit failed: %v\n%s", err, out)
		return ""
	}
	hashCmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	hashCmd.Dir = a.projectDir
	hashOut, err := hashCmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(hashOut))
}
