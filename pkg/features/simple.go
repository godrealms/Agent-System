package features

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Feature represents a single project feature
type Feature struct {
	ID          string    `json:"id"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Steps       []string  `json:"steps,omitempty"`
	Passes      bool      `json:"passes"`
	Priority    int       `json:"priority"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

// FeatureList is the top-level JSON structure
type FeatureList struct {
	ProjectType string    `json:"project_type"`
	CreatedAt   time.Time `json:"created_at"`
	Features    []Feature `json:"features"`
}

// FeatureManager manages the feature list JSON file
type FeatureManager struct {
	featureFile string
}

// NewFeatureManager creates a manager for the given feature file path
func NewFeatureManager(featureFile string) *FeatureManager {
	return &FeatureManager{featureFile: featureFile}
}

// CreateInitialFeatureList generates and saves a feature list for the given project type
func (fm *FeatureManager) CreateInitialFeatureList(projectType string) (*FeatureList, error) {
	fl := &FeatureList{
		ProjectType: projectType,
		CreatedAt:   time.Now(),
		Features:    fm.defaultFeatures(projectType),
	}

	data, err := json.MarshalIndent(fl, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal feature list: %w", err)
	}
	if err := os.WriteFile(fm.featureFile, data, 0644); err != nil {
		return nil, fmt.Errorf("write feature list: %w", err)
	}
	return fl, nil
}

// Load reads and parses the feature list from disk
func (fm *FeatureManager) Load() (*FeatureList, error) {
	data, err := os.ReadFile(fm.featureFile)
	if err != nil {
		return nil, fmt.Errorf("read feature file: %w", err)
	}
	var fl FeatureList
	if err := json.Unmarshal(data, &fl); err != nil {
		return nil, fmt.Errorf("parse feature file: %w", err)
	}
	return &fl, nil
}

// PendingFeatures returns features that have not yet passed
func (fm *FeatureManager) PendingFeatures() ([]Feature, error) {
	fl, err := fm.Load()
	if err != nil {
		return nil, err
	}
	var pending []Feature
	for _, f := range fl.Features {
		if !f.Passes {
			pending = append(pending, f)
		}
	}
	return pending, nil
}

// MarkComplete marks a feature as done and saves the file
func (fm *FeatureManager) MarkComplete(id string) error {
	fl, err := fm.Load()
	if err != nil {
		return err
	}
	found := false
	for i, f := range fl.Features {
		if f.ID == id {
			fl.Features[i].Passes = true
			fl.Features[i].CompletedAt = time.Now()
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("feature %q not found", id)
	}
	data, err := json.MarshalIndent(fl, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return os.WriteFile(fm.featureFile, data, 0644)
}

// defaultFeatures returns a sensible starting list for known project types
func (fm *FeatureManager) defaultFeatures(projectType string) []Feature {
	switch projectType {
	case "web-chat-app":
		return []Feature{
			{ID: "chat-001", Category: "core", Description: "Create basic HTML/CSS chat UI", Priority: 1},
			{ID: "chat-002", Category: "core", Description: "Implement WebSocket message sending and receiving", Priority: 2},
			{ID: "chat-003", Category: "core", Description: "Add message history display with timestamps", Priority: 3},
			{ID: "chat-004", Category: "ux", Description: "Add user nickname support", Priority: 4},
			{ID: "chat-005", Category: "ux", Description: "Show online user count", Priority: 5},
		}
	case "api-server":
		return []Feature{
			{ID: "api-001", Category: "core", Description: "Set up HTTP router with health-check endpoint", Priority: 1},
			{ID: "api-002", Category: "core", Description: "Implement CRUD endpoints for main resource", Priority: 2},
			{ID: "api-003", Category: "core", Description: "Add request validation and error responses", Priority: 3},
			{ID: "api-004", Category: "auth", Description: "Implement JWT authentication middleware", Priority: 4},
			{ID: "api-005", Category: "ops", Description: "Add structured logging and metrics endpoint", Priority: 5},
		}
	default:
		return []Feature{
			{ID: "feat-001", Category: "core", Description: "Explore project structure and create initial README", Priority: 1},
			{ID: "feat-002", Category: "core", Description: "Implement core business logic", Priority: 2},
			{ID: "feat-003", Category: "test", Description: "Add unit tests for core logic", Priority: 3},
		}
	}
}
