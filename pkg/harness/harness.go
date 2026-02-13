package harness

import (
	"fmt"
	"log"
	"time"

	"AI-agent/internal/config"
	"AI-agent/pkg/agent"
	"AI-agent/pkg/environment"
	"AI-agent/pkg/features"
	"AI-agent/pkg/monitoring"
	"AI-agent/pkg/testing"
)

// Harness orchestrates the long-running agent workflow
type Harness struct {
	config         *config.Config
	envManager     *environment.EnvironmentManager
	featureManager *features.FeatureManager
	testManager    *testing.TestManager
	monitor        *monitoring.Monitor
}

// NewHarness creates a new harness instance
func NewHarness(cfg *config.Config) *Harness {
	return &Harness{
		config:         cfg,
		envManager:     environment.NewEnvironmentManager(cfg.ProjectDir),
		featureManager: features.NewFeatureManager(cfg.FeatureFile),
		testManager:    testing.NewTestManager(cfg.ProjectDir),
		monitor:        monitoring.NewMonitor(cfg),
	}
}

// StartMonitoring starts the monitoring dashboard
func (h *Harness) StartMonitoring() error {
	if err := h.monitor.Start(); err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	log.Printf("Monitoring dashboard available at http://localhost:%d", h.config.MonitoringPort)

	// Keep running until interrupted
	select {}
}

// InitializeProject sets up a new project environment
func (h *Harness) InitializeProject(projectType string) error {
	log.Println("Initializing new project...")

	// Start monitoring
	if err := h.monitor.Start(); err != nil {
		log.Printf("Warning: Failed to start monitoring: %v", err)
	}
	defer h.monitor.Stop()

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

// RunSession executes a single agent session with monitoring and proper error handling
func (h *Harness) RunSession() (*agent.SessionResult, error) {
	log.Println("Starting agent session...")

	// Start monitoring if not already running
	if err := h.monitor.Start(); err != nil {
		log.Printf("Warning: Failed to start monitoring: %v", err)
		// Don't fail the session just for monitoring issues
	}

	// Determine agent type based on project state
	agentType := agent.CodingAgent // Simplified for now

	// Create agent with proper error handling
	agt, err := agent.NewAgent(agentType, h.config.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}
	defer func() {
		agt.Close() // Close without error checking since Close doesn't return error
	}()

	// Run agent session
	result, err := agt.Run()
	if err != nil {
		// Record failed session with proper error details
		h.monitor.RecordSession(
			"unknown-session", // result might be nil on error
			false,
			0,
			nil,
			0,
			err,
		)
		return nil, fmt.Errorf("agent session failed: %w", err)
	}

	// Record successful session
	h.monitor.RecordSession(
		result.SessionID,
		result.Success,
		result.Duration,
		result.FeaturesDone,
		result.TokenUsage.TotalTokens,
		nil,
	)

	// Post-session processing with error handling
	logMessage := fmt.Sprintf("🟢 Session %s completed", result.SessionID)
	if len(result.FeaturesDone) > 0 {
		logMessage += fmt.Sprintf(" with %d features", len(result.FeaturesDone))
	}
	if result.CommitHash != "" {
		logMessage += fmt.Sprintf(" (commit: %s)", result.CommitHash)
	}
	log.Println(logMessage)

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
			continue
		}

		results = append(results, result)
		time.Sleep(time.Second * 2) // Brief pause between sessions
	}

	return results, nil
}

// GenerateReport creates a comprehensive project report
func (h *Harness) GenerateReport() (string, error) {
	metrics := h.monitor.GetMetrics()

	report := fmt.Sprintf(`
# Claude Agent Project Report

Generated: %s

## Session Statistics
- Total Sessions: %d
- Successful Sessions: %d
- Failed Sessions: %d
- Average Duration: %v
- Total Tokens Used: %d

## Feature Progress
- Completed Features: %d
- Pending Features: %d

## Recent Sessions
`,
		time.Now().Format("2006-01-02 15:04:05"),
		metrics.TotalSessions,
		metrics.SuccessfulSessions,
		metrics.FailedSessions,
		metrics.AvgSessionDuration,
		metrics.TotalTokensUsed,
		metrics.CompletedFeatures,
		metrics.PendingFeatures)

	// Add recent session details
	for i, session := range metrics.SessionHistory {
		if i >= 5 { // Show last 5 sessions
			break
		}
		report += fmt.Sprintf("- Session %s: %s (%v)\n",
			session.SessionID,
			map[bool]string{true: "SUCCESS", false: "FAILED"}[session.Success],
			session.Duration)
	}

	return report, nil
}

// Cleanup performs cleanup operations
func (h *Harness) Cleanup() error {
	if err := h.monitor.Stop(); err != nil {
		log.Printf("Warning: Failed to stop monitoring: %v", err)
	}
	log.Println("Cleanup completed")
	return nil
}
