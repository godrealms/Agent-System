package agent

import (
	"fmt"
	"strings"
)

// ProjectState represents the current state of the project
type ProjectState struct {
	WorkingDirectory  string
	GitStatus         string
	RecentCommits     []string
	ProgressSummary   string
	PendingFeatures   []string
	CompletedFeatures []string
}

// String returns a formatted string representation of the project state
func (ps *ProjectState) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Working Directory: %s\n", ps.WorkingDirectory))
	sb.WriteString(fmt.Sprintf("Git Status: %s\n", ps.GitStatus))
	sb.WriteString("Recent Commits:\n")
	for _, commit := range ps.RecentCommits {
		sb.WriteString(fmt.Sprintf("  - %s\n", commit))
	}
	sb.WriteString(fmt.Sprintf("Progress Summary: %s\n", ps.ProgressSummary))
	sb.WriteString("Pending Features:\n")
	for _, feature := range ps.PendingFeatures {
		sb.WriteString(fmt.Sprintf("  - %s\n", feature))
	}
	sb.WriteString("Completed Features:\n")
	for _, feature := range ps.CompletedFeatures {
		sb.WriteString(fmt.Sprintf("  - %s\n", feature))
	}

	return sb.String()
}

// parseFeaturesFromResponse extracts feature names from the initializer response
func (a *Agent) parseFeaturesFromResponse(response string) []string {
	// Simple parsing - in practice, you'd want more sophisticated parsing
	features := []string{}

	// Look for feature mentions in the response
	// This is a simplified implementation
	if strings.Contains(response, "feature") || strings.Contains(response, "Feature") {
		features = append(features, "Initial project setup features")
	}

	return features
}

// parseCompletedFeatures extracts completed feature names from coding response
func (a *Agent) parseCompletedFeatures(response string) []string {
	features := []string{}

	// Look for completion indicators
	if strings.Contains(response, "completed") || strings.Contains(response, "done") {
		features = append(features, "Feature implementation session")
	}

	return features
}

// extractCommitHash extracts git commit hash from response
func (a *Agent) extractCommitHash(response string) string {
	// Simple extraction - look for commit hash pattern
	// In practice, you'd parse the actual git output
	return "commit-hash-placeholder"
}

// getProjectState retrieves the current project state
func (a *Agent) getProjectState() (*ProjectState, error) {
	// This would integrate with the environment package
	// For now, return a placeholder state
	return &ProjectState{
		WorkingDirectory:  a.Config.ProjectDir,
		GitStatus:         "Clean",
		RecentCommits:     []string{"Initial commit"},
		ProgressSummary:   "Project initialized, ready for feature development",
		PendingFeatures:   []string{"Implement core functionality"},
		CompletedFeatures: []string{},
	}, nil
}
