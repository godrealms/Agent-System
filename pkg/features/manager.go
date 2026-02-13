package features

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// Feature represents a single feature requirement
type Feature struct {
	ID          string   `json:"id"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Steps       []string `json:"steps"`
	Passes      bool     `json:"passes"`
	Priority    int      `json:"priority"` // Lower number = higher priority
	CreatedAt   string   `json:"created_at"`
	CompletedAt string   `json:"completed_at,omitempty"`
}

// FeatureList manages the collection of features
type FeatureList struct {
	Features []Feature `json:"features"`
}

// FeatureManager handles feature list operations
type FeatureManager struct {
	filePath string
}

// NewFeatureManager creates a new feature manager
func NewFeatureManager(filePath string) *FeatureManager {
	return &FeatureManager{
		filePath: filePath,
	}
}

// LoadFeatures loads features from the JSON file
func (fm *FeatureManager) LoadFeatures() (*FeatureList, error) {
	data, err := os.ReadFile(fm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty feature list if file doesn't exist
			return &FeatureList{Features: []Feature{}}, nil
		}
		return nil, fmt.Errorf("failed to read feature file: %w", err)
	}

	var featureList FeatureList
	if err := json.Unmarshal(data, &featureList); err != nil {
		return nil, fmt.Errorf("failed to parse feature file: %w", err)
	}

	return &featureList, nil
}

// SaveFeatures saves features to the JSON file
func (fm *FeatureManager) SaveFeatures(featureList *FeatureList) error {
	data, err := json.MarshalIndent(featureList, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal features: %w", err)
	}

	if err := os.WriteFile(fm.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write feature file: %w", err)
	}

	return nil
}

// CreateInitialFeatureList creates the initial comprehensive feature list
func (fm *FeatureManager) CreateInitialFeatureList(projectType string) (*FeatureList, error) {
	features := fm.generateFeaturesForProject(projectType)

	featureList := &FeatureList{
		Features: features,
	}

	if err := fm.SaveFeatures(featureList); err != nil {
		return nil, fmt.Errorf("failed to save initial feature list: %w", err)
	}

	return featureList, nil
}

// generateFeaturesForProject generates features based on project type
func (fm *FeatureManager) generateFeaturesForProject(projectType string) []Feature {
	switch projectType {
	case "web-chat-app":
		return fm.generateChatAppFeatures()
	case "api-service":
		return fm.generateAPIServiceFeatures()
	default:
		return fm.generateGenericWebAppFeatures()
	}
}

// generateChatAppFeatures generates features for a chat application
func (fm *FeatureManager) generateChatAppFeatures() []Feature {
	return []Feature{
		{
			ID:          "chat-001",
			Category:    "core-functionality",
			Description: "User can open a new chat session",
			Steps: []string{
				"Navigate to main interface",
				"Click the 'New Chat' button",
				"Verify a new conversation is created",
				"Check that chat area shows welcome state",
			},
			Passes:    false,
			Priority:  1,
			CreatedAt: "2024-01-01T00:00:00Z",
		},
		{
			ID:          "chat-002",
			Category:    "core-functionality",
			Description: "User can type and send messages",
			Steps: []string{
				"Focus on message input field",
				"Type a message",
				"Press Enter or click Send button",
				"Verify message appears in chat history",
			},
			Passes:    false,
			Priority:  2,
			CreatedAt: "2024-01-01T00:00:00Z",
		},
		{
			ID:          "chat-003",
			Category:    "ai-integration",
			Description: "AI responds to user messages",
			Steps: []string{
				"Send a message to the AI",
				"Wait for AI response",
				"Verify response is relevant and coherent",
				"Check response formatting",
			},
			Passes:    false,
			Priority:  3,
			CreatedAt: "2024-01-01T00:00:00Z",
		},
		{
			ID:          "chat-004",
			Category:    "ui-ux",
			Description: "Chat history scrolls automatically",
			Steps: []string{
				"Send multiple messages",
				"Verify new messages appear at bottom",
				"Check automatic scrolling behavior",
				"Test manual scroll functionality",
			},
			Passes:    false,
			Priority:  4,
			CreatedAt: "2024-01-01T00:00:00Z",
		},
		{
			ID:          "chat-005",
			Category:    "persistence",
			Description: "Chat conversations are saved and can be resumed",
			Steps: []string{
				"Create a conversation with messages",
				"Refresh the page",
				"Verify conversation history is preserved",
				"Continue the conversation",
			},
			Passes:    false,
			Priority:  5,
			CreatedAt: "2024-01-01T00:00:00Z",
		},
	}
}

// generateAPIServiceFeatures generates features for an API service
func (fm *FeatureManager) generateAPIServiceFeatures() []Feature {
	return []Feature{
		{
			ID:          "api-001",
			Category:    "core-endpoints",
			Description: "Health check endpoint returns 200 OK",
			Steps: []string{
				"Start the API server",
				"Make GET request to /health",
				"Verify response status is 200",
				"Check response body contains status information",
			},
			Passes:    false,
			Priority:  1,
			CreatedAt: "2024-01-01T00:00:00Z",
		},
		// Additional API features would be added here
	}
}

// generateGenericWebAppFeatures generates generic web application features
func (fm *FeatureManager) generateGenericWebAppFeatures() []Feature {
	return []Feature{
		{
			ID:          "web-001",
			Category:    "basic-setup",
			Description: "Application loads without errors",
			Steps: []string{
				"Start development server",
				"Navigate to application URL",
				"Verify no console errors",
				"Check basic page rendering",
			},
			Passes:    false,
			Priority:  1,
			CreatedAt: "2024-01-01T00:00:00Z",
		},
	}
}

// GetPendingFeatures returns features that haven't been completed yet
func (fm *FeatureManager) GetPendingFeatures() ([]Feature, error) {
	featureList, err := fm.LoadFeatures()
	if err != nil {
		return nil, err
	}

	var pending []Feature
	for _, feature := range featureList.Features {
		if !feature.Passes {
			pending = append(pending, feature)
		}
	}

	// Sort by priority
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Priority < pending[j].Priority
	})

	return pending, nil
}

// GetCompletedFeatures returns features that have been completed
func (fm *FeatureManager) GetCompletedFeatures() ([]Feature, error) {
	featureList, err := fm.LoadFeatures()
	if err != nil {
		return nil, err
	}

	var completed []Feature
	for _, feature := range featureList.Features {
		if feature.Passes {
			completed = append(completed, feature)
		}
	}

	return completed, nil
}

// MarkFeatureComplete marks a feature as completed
func (fm *FeatureManager) MarkFeatureComplete(featureID string) error {
	featureList, err := fm.LoadFeatures()
	if err != nil {
		return err
	}

	found := false
	for i := range featureList.Features {
		if featureList.Features[i].ID == featureID {
			featureList.Features[i].Passes = true
			featureList.Features[i].CompletedAt = "2024-01-01T00:00:00Z" // In practice, use current timestamp
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("feature with ID %s not found", featureID)
	}

	return fm.SaveFeatures(featureList)
}

// AddFeature adds a new feature to the list
func (fm *FeatureManager) AddFeature(feature Feature) error {
	featureList, err := fm.LoadFeatures()
	if err != nil {
		return err
	}

	// Check if feature already exists
	for _, existing := range featureList.Features {
		if existing.ID == feature.ID {
			return fmt.Errorf("feature with ID %s already exists", feature.ID)
		}
	}

	featureList.Features = append(featureList.Features, feature)
	return fm.SaveFeatures(featureList)
}

// GetFeatureByID returns a feature by its ID
func (fm *FeatureManager) GetFeatureByID(featureID string) (*Feature, error) {
	featureList, err := fm.LoadFeatures()
	if err != nil {
		return nil, err
	}

	for _, feature := range featureList.Features {
		if feature.ID == featureID {
			return &feature, nil
		}
	}

	return nil, fmt.Errorf("feature with ID %s not found", featureID)
}

// GetFeaturesByCategory returns features filtered by category
func (fm *FeatureManager) GetFeaturesByCategory(category string) ([]Feature, error) {
	featureList, err := fm.LoadFeatures()
	if err != nil {
		return nil, err
	}

	var filtered []Feature
	for _, feature := range featureList.Features {
		if feature.Category == category {
			filtered = append(filtered, feature)
		}
	}

	return filtered, nil
}

// GetFeatureStats returns statistics about feature completion
func (fm *FeatureManager) GetFeatureStats() (map[string]interface{}, error) {
	featureList, err := fm.LoadFeatures()
	if err != nil {
		return nil, err
	}

	total := len(featureList.Features)
	completed := 0
	pending := 0

	categoryStats := make(map[string]int)

	for _, feature := range featureList.Features {
		if feature.Passes {
			completed++
		} else {
			pending++
		}

		categoryStats[feature.Category]++
	}

	stats := map[string]interface{}{
		"total_features":  total,
		"completed":       completed,
		"pending":         pending,
		"completion_rate": float64(completed) / float64(total) * 100,
		"category_counts": categoryStats,
	}

	return stats, nil
}
