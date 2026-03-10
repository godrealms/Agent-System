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

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AgentType identifies the role of the agent
type AgentType string

const (
	InitializerAgent AgentType = "initializer"
	CodingAgent      AgentType = "coding"
)

// SessionResult holds the outcome of one agent session
type SessionResult struct {
	SessionID    string
	Success      bool
	Message      string
	FeaturesDone []string
	CommitHash   string
	Duration     time.Duration
	TokenUsage   TokenUsage
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Agent wraps a Claude client with project context
type Agent struct {
	Type       AgentType
	projectDir string
	client     anthropic.Client
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewAgent creates an agent; ANTHROPIC_API_KEY must be set
func NewAgent(agentType AgentType, projectDir string) (*Agent, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	return &Agent{
		Type:       agentType,
		projectDir: projectDir,
		client:     client,
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

// Run executes a full agent session and returns the result
func (a *Agent) Run() (*SessionResult, error) {
	defer a.cancel()

	startTime := time.Now()
	sessionID := fmt.Sprintf("session-%d", startTime.Unix())

	log.Printf("[%s] Starting %s session", sessionID, a.Type)

	var (
		featuresDone []string
		commitHash   string
		totalUsage   TokenUsage
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

// Close cancels the agent context
func (a *Agent) Close() {
	a.cancel()
}

// ---------------------------------------------------------------------------
// Initializer agent
// ---------------------------------------------------------------------------

func (a *Agent) runInitializer(sessionID string) (string, TokenUsage, error) {
	systemPrompt := fmt.Sprintf(`You are a project initializer agent. Your task is to set up a new software project in the directory: %s

Use the available tools to initialize the project:
1. Create a README.md describing the project
2. Set up the project directory structure
3. Create an init.sh script to start the development server

Be concise and create only essential files.`, a.projectDir)

	_, usage, err := a.runAgenticLoop(sessionID, systemPrompt, "Set up this project directory with a good structure and initial files.")
	if err != nil {
		return "", usage, err
	}

	commitHash := a.tryGitCommit("Initial project setup by AI agent")
	return commitHash, usage, nil
}

// ---------------------------------------------------------------------------
// Coding agent
// ---------------------------------------------------------------------------

func (a *Agent) runCoding(sessionID string) ([]string, string, TokenUsage, error) {
	featureList, err := a.readFeatureList()
	if err != nil {
		log.Printf("[%s] Warning: could not read feature list: %v", sessionID, err)
		featureList = "No feature list found. Please explore the project and implement improvements."
	}

	progress, err := a.readProgress()
	if err != nil {
		progress = "No progress file found."
	}

	systemPrompt := fmt.Sprintf(`You are a coding agent working on a software project located at: %s

Your job is to implement features from the feature list. For each session:
1. Read the feature list and progress file to understand the current state
2. Pick ONE pending feature to implement (the highest priority one)
3. Implement it fully, including tests if applicable
4. Update the feature status in feature_list.json to mark it as complete
5. Record your progress using append_progress

Current feature list:
%s

Current progress:
%s

Be focused: implement one feature at a time.`, a.projectDir, featureList, progress)

	_, usage, err := a.runAgenticLoop(sessionID, systemPrompt, "Please implement the next pending feature from the feature list.")
	if err != nil {
		return nil, "", usage, err
	}

	featuresDone := a.detectCompletedFeatures()

	commitHash := ""
	if len(featuresDone) > 0 {
		msg := fmt.Sprintf("feat: implement %s", strings.Join(featuresDone, ", "))
		commitHash = a.tryGitCommit(msg)
	}

	return featuresDone, commitHash, usage, nil
}

// ---------------------------------------------------------------------------
// Agentic loop with tool use
// ---------------------------------------------------------------------------

func (a *Agent) runAgenticLoop(sessionID, system, userMessage string) ([]anthropic.MessageParam, TokenUsage, error) {
	tools := a.buildTools()
	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)),
	}

	var totalUsage TokenUsage
	maxIterations := 20

	// adaptive thinking (recommended for Opus 4.6)
	adaptiveParam := anthropic.NewThinkingConfigAdaptiveParam()
	thinking := anthropic.ThinkingConfigParamUnion{OfAdaptive: &adaptiveParam}

	for i := 0; i < maxIterations; i++ {
		log.Printf("[%s] Iteration %d/%d", sessionID, i+1, maxIterations)

		stream := a.client.Messages.NewStreaming(a.ctx, anthropic.MessageNewParams{
			Model:     anthropic.ModelClaudeOpus4_6,
			MaxTokens: 8192,
			System: []anthropic.TextBlockParam{
				{Text: system},
			},
			Tools:    tools,
			Messages: messages,
			Thinking: thinking,
		})

		// Accumulate the full streamed response
		var response anthropic.Message
		for stream.Next() {
			if err := response.Accumulate(stream.Current()); err != nil {
				return messages, totalUsage, fmt.Errorf("accumulate error: %w", err)
			}
		}
		if err := stream.Err(); err != nil {
			return messages, totalUsage, fmt.Errorf("streaming error: %w", err)
		}

		totalUsage.PromptTokens += int(response.Usage.InputTokens)
		totalUsage.CompletionTokens += int(response.Usage.OutputTokens)
		totalUsage.TotalTokens += int(response.Usage.InputTokens) + int(response.Usage.OutputTokens)

		// Convert response content blocks to param blocks
		contentParams := make([]anthropic.ContentBlockParamUnion, len(response.Content))
		for j, b := range response.Content {
			contentParams[j] = b.ToParam()
		}
		messages = append(messages, anthropic.NewAssistantMessage(contentParams...))

		log.Printf("[%s] Stop reason: %s", sessionID, response.StopReason)

		if response.StopReason == anthropic.StopReasonEndTurn {
			break
		}

		if response.StopReason == anthropic.StopReasonToolUse {
			toolResults := a.executeTools(sessionID, response.Content)
			messages = append(messages, anthropic.NewUserMessage(toolResults...))
			continue
		}

		// max_tokens or stop_sequence — exit
		break
	}

	return messages, totalUsage, nil
}

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------

func (a *Agent) buildTools() []anthropic.ToolUnionParam {
	tool := func(name, desc string, props map[string]interface{}, required []string) anthropic.ToolUnionParam {
		t := anthropic.ToolParam{
			Name:        name,
			Description: anthropic.String(desc),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: props,
				Required:   required,
			},
		}
		return anthropic.ToolUnionParam{OfTool: &t}
	}

	return []anthropic.ToolUnionParam{
		tool("read_file", "Read the contents of a file in the project directory",
			map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Relative path from project root",
				},
			}, []string{"path"}),

		tool("write_file", "Write content to a file, creating directories as needed",
			map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Relative path from project root",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "File content to write",
				},
			}, []string{"path", "content"}),

		tool("list_files", "List files and directories at the given path",
			map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Relative path from project root (use '.' for root)",
				},
			}, []string{"path"}),

		tool("run_command", "Run a shell command in the project directory and return combined stdout+stderr",
			map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "Shell command to execute",
				},
			}, []string{"command"}),

		tool("update_feature_status", "Mark a feature as complete in feature_list.json",
			map[string]interface{}{
				"feature_id": map[string]interface{}{
					"type":        "string",
					"description": "The feature ID to mark as complete",
				},
			}, []string{"feature_id"}),

		tool("append_progress", "Append a timestamped message to the progress tracking file",
			map[string]interface{}{
				"message": map[string]interface{}{
					"type":        "string",
					"description": "Progress note to append",
				},
			}, []string{"message"}),
	}
}

// ---------------------------------------------------------------------------
// Tool execution
// ---------------------------------------------------------------------------

func (a *Agent) executeTools(sessionID string, content []anthropic.ContentBlockUnion) []anthropic.ContentBlockParamUnion {
	var results []anthropic.ContentBlockParamUnion

	for _, block := range content {
		toolUse, ok := block.AsAny().(anthropic.ToolUseBlock)
		if !ok {
			continue
		}

		log.Printf("[%s] Tool call: %s", sessionID, toolUse.Name)
		output, isError := a.dispatchTool(toolUse.Name, toolUse.Input)
		results = append(results, anthropic.NewToolResultBlock(toolUse.ID, output, isError))
	}

	return results
}

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
	absPath := filepath.Join(a.projectDir, filepath.Clean(path))

	data, err := os.ReadFile(absPath)
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
	absPath := filepath.Join(a.projectDir, filepath.Clean(path))

	entries, err := os.ReadDir(absPath)
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

	result := string(out)
	if err != nil {
		return fmt.Sprintf("command failed (%v):\n%s", err, result), true
	}
	return result, false
}

func (a *Agent) toolUpdateFeatureStatus(input map[string]interface{}) (string, bool) {
	featureID, _ := input["feature_id"].(string)

	featureFile := filepath.Join(a.projectDir, "feature_list.json")
	data, err := os.ReadFile(featureFile)
	if err != nil {
		return fmt.Sprintf("could not read feature_list.json: %v", err), true
	}

	var featureList map[string]interface{}
	if err := json.Unmarshal(data, &featureList); err != nil {
		return fmt.Sprintf("could not parse feature_list.json: %v", err), true
	}

	features, _ := featureList["features"].([]interface{})
	found := false
	for _, f := range features {
		feature, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		if feature["id"] == featureID {
			feature["passes"] = true
			feature["completed_at"] = time.Now().Format(time.RFC3339)
			found = true
		}
	}

	if !found {
		return fmt.Sprintf("feature %q not found", featureID), true
	}

	updated, err := json.MarshalIndent(featureList, "", "  ")
	if err != nil {
		return fmt.Sprintf("could not marshal feature list: %v", err), true
	}
	if err := os.WriteFile(featureFile, updated, 0644); err != nil {
		return fmt.Sprintf("could not write feature_list.json: %v", err), true
	}

	return fmt.Sprintf("feature %q marked as complete", featureID), false
}

func (a *Agent) toolAppendProgress(input map[string]interface{}) (string, bool) {
	message, _ := input["message"].(string)

	progressFile := filepath.Join(a.projectDir, "claude-progress.txt")
	line := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), message)

	f, err := os.OpenFile(progressFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Sprintf("could not open progress file: %v", err), true
	}
	defer f.Close()

	if _, err := f.WriteString(line); err != nil {
		return fmt.Sprintf("could not write to progress file: %v", err), true
	}
	return "progress recorded", false
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (a *Agent) readFeatureList() (string, error) {
	data, err := os.ReadFile(filepath.Join(a.projectDir, "feature_list.json"))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (a *Agent) readProgress() (string, error) {
	data, err := os.ReadFile(filepath.Join(a.projectDir, "claude-progress.txt"))
	if err != nil {
		return "", err
	}
	return string(data), nil
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

	features, _ := fl["features"].([]interface{})
	var done []string
	for _, f := range features {
		feature, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		if passes, _ := feature["passes"].(bool); passes {
			if id, _ := feature["id"].(string); id != "" {
				done = append(done, id)
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
	out, err := commitCmd.CombinedOutput()
	if err != nil {
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
