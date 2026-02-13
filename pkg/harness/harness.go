package harness

import (
	"fmt"
	"log"
	"time"

	"AI-agent/internal/config"
	"AI-agent/pkg/agent"
	"AI-agent/pkg/environment"
	"AI-agent/pkg/features"
	"AI-agent/pkg/testing"
)

// Harness orchestrates the long-running agent workflow
type Harness struct {
	config         *config.Config
	envManager     *environment.EnvironmentManager
	featureManager *features.FeatureManager
	testManager    *testing.TestManager
}

// NewHarness creates a new harness instance
func NewHarness(cfg *config.Config) *Harness {
	return &Harness{
		config:         cfg,
		envManager:     environment.NewEnvironmentManager(cfg),
		featureManager: features.NewFeatureManager(cfg.FeatureFile),
		testManager:    testing.NewTestManager(cfg),
	}
}

// InitializeProject sets up a new project environment
func (h *Harness) InitializeProject(projectType string) error {
	log.Println("Initializing new project...")

	// Setup environment
	if err := h.envManager.SetupProject(); err != nil {
		return fmt.Errorf("failed to setup environment: %w", err)
	}

	// Create initial feature list
	if _, err := h.featureManager.CreateInitialFeatureList(projectType); err != nil {
		return fmt.Errorf("failed to create feature list: %w", err)
	}

	log.Println("Project initialization completed successfully")
	return nil
}

// RunSession executes a single agent session
func (h *Harness) RunSession() (*agent.SessionResult, error) {
	log.Println("Starting agent session...")

	// Determine agent type based on project state
	agentType := h.determineAgentType()

	// Create agent
	agt, err := agent.NewAgent(agentType, h.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}
	defer agt.Close()

	// Run agent session
	result, err := agt.Run()
	if err != nil {
		return nil, fmt.Errorf("agent session failed: %w", err)
	}

	// Post-session processing
	if err := h.processSessionResult(result); err != nil {
		return nil, fmt.Errorf("failed to process session result: %w", err)
	}

	return result, nil
}

// RunMultipleSessions executes multiple agent sessions
func (h *Harness) RunMultipleSessions(count int) ([]*agent.SessionResult, error) {
	var results []*agent.SessionResult

	for i := 0; i < count; i++ {
		log.Printf("Starting session %d of %d", i+1, count)

		result, err := h.RunSession()
		if err != nil {
			log.Printf("Session %d failed: %v", i+1, err)
			// Continue with next session instead of stopping completely
			continue
		}

		results = append(results, result)

		// Check if project is complete
		if h.isProjectComplete() {
			log.Println("Project completed! Stopping sessions.")
			break
		}

		// Brief pause between sessions
		time.Sleep(time.Second * 5)
	}

	return results, nil
}

// determineAgentType decides which type of agent to use
func (h *Harness) determineAgentType() agent.AgentType {
	// Check if this is the first run by checking if feature list exists
	_, err := h.featureManager.LoadFeatures()
	if err != nil || h.isFirstRun() {
		return agent.InitializerAgent
	}

	return agent.CodingAgent
}

// isFirstRun checks if this is the initial setup
func (h *Harness) isFirstRun() bool {
	// Check if essential files exist
	// This is a simplified check - in practice you'd want more robust detection
	_, err := h.featureManager.LoadFeatures()
	return err != nil
}

// processSessionResult handles post-session processing
func (h *Harness) processSessionResult(result *agent.SessionResult) error {
	// Update progress file
	progressEntry := fmt.Sprintf(`
## Session %s
Date: %s
Duration: %v
Status: %s
Features Completed: %v
Commit Hash: %s

%s
`,
		result.SessionID,
		time.Now().Format("2006-01-02 15:04:05"),
		result.Duration,
		map[bool]string{true: "SUCCESS", false: "FAILED"}[result.Success],
		result.FeaturesDone,
		result.CommitHash,
		result.Message)

	if err := h.envManager.AppendToProgressFile(progressEntry); err != nil {
		log.Printf("Warning: Failed to update progress file: %v", err)
	}

	// Update feature statuses
	for _, featureID := range result.FeaturesDone {
		if err := h.featureManager.MarkFeatureComplete(featureID); err != nil {
			log.Printf("Warning: Failed to mark feature %s as complete: %v", featureID, err)
		}
	}

	// Run validation tests
	if result.Success && len(result.FeaturesDone) > 0 {
		if err := h.runValidationTests(result.FeaturesDone); err != nil {
			log.Printf("Warning: Validation tests failed: %v", err)
		}
	}

	return nil
}

// runValidationTests executes tests for completed features
func (h *Harness) runValidationTests(featureIDs []string) error {
	for _, featureID := range featureIDs {
		feature, err := h.featureManager.GetFeatureByID(featureID)
		if err != nil {
			continue
		}

		result, err := h.testManager.ValidateFeature(featureID, feature.Steps)
		if err != nil {
			log.Printf("Failed to validate feature %s: %v", featureID, err)
			continue
		}

		if !result.Passed {
			log.Printf("Feature %s validation failed: %s", featureID, result.Error)
		} else {
			log.Printf("Feature %s validated successfully", featureID)
		}
	}

	return nil
}

// isProjectComplete checks if all features have been implemented
func (h *Harness) isProjectComplete() bool {
	pending, err := h.featureManager.GetPendingFeatures()
	if err != nil {
		log.Printf("Error checking project completion: %v", err)
		return false
	}

	return len(pending) == 0
}

// GetProjectStatus returns current project status
func (h *Harness) GetProjectStatus() (map[string]interface{}, error) {
	stats, err := h.featureManager.GetFeatureStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get feature stats: %w", err)
	}

	gitStatus, err := h.envManager.GetGitStatus()
	if err != nil {
		gitStatus = "Unknown"
	}

	commits, err := h.envManager.GetRecentCommits(10)
	if err != nil {
		commits = []string{}
	}

	status := map[string]interface{}{
		"features":       stats,
		"git_status":     gitStatus,
		"recent_commits": commits,
		"complete":       h.isProjectComplete(),
	}

	return status, nil
}

// GenerateReport creates a comprehensive project report
func (h *Harness) GenerateReport() (string, error) {
	status, err := h.GetProjectStatus()
	if err != nil {
		return "", fmt.Errorf("failed to get project status: %w", err)
	}

	var report string
	report += "# Claude Agent Project Report\n\n"
	report += fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// Feature statistics
	if features, ok := status["features"].(map[string]interface{}); ok {
		report += "## Feature Statistics\n\n"
		report += fmt.Sprintf("- Total Features: %v\n", features["total_features"])
		report += fmt.Sprintf("- Completed: %v\n", features["completed"])
		report += fmt.Sprintf("- Pending: %v\n", features["pending"])
		report += fmt.Sprintf("- Completion Rate: %.1f%%\n\n", features["completion_rate"])
	}

	// Git status
	report += fmt.Sprintf("## Environment Status\n\n")
	report += fmt.Sprintf("Git Status: %v\n", status["git_status"])

	// Recent commits
	if commits, ok := status["recent_commits"].([]string); ok && len(commits) > 0 {
		report += "\nRecent Commits:\n"
		for _, commit := range commits {
			report += fmt.Sprintf("- %s\n", commit)
		}
	}

	report += fmt.Sprintf("\nProject Complete: %v\n", status["complete"])

	return report, nil
}

// Cleanup performs cleanup operations
func (h *Harness) Cleanup() error {
	// In a real implementation, this might:
	// - Close database connections
	// - Clean up temporary files
	// - Stop running services
	// - Generate final reports

	log.Println("Cleanup completed")
	return nil
}
