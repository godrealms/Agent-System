package environment

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"AI-agent/internal/config"
)

// EnvironmentManager handles project environment operations
type EnvironmentManager struct {
	config *config.Config
}

// NewEnvironmentManager creates a new environment manager
func NewEnvironmentManager(cfg *config.Config) *EnvironmentManager {
	return &EnvironmentManager{
		config: cfg,
	}
}

// SetupProject initializes a new project environment
func (em *EnvironmentManager) SetupProject() error {
	log.Println("Setting up project environment...")

	// Convert project dir to absolute path
	absProjectDir, err := filepath.Abs(em.config.ProjectDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Update config with absolute path
	em.config.ProjectDir = absProjectDir
	em.config.ProgressFile = filepath.Join(absProjectDir, "claude-progress.txt")
	em.config.FeatureFile = filepath.Join(absProjectDir, "feature_list.json")

	// Create project directory if it doesn't exist
	if err := os.MkdirAll(em.config.ProjectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Store original directory
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	log.Printf("Original directory: %s", originalDir)
	log.Printf("Target project directory: %s", em.config.ProjectDir)

	// Change to project directory
	if err := os.Chdir(em.config.ProjectDir); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

	// Verify we're in the right directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory after chdir: %w", err)
	}
	log.Printf("Current directory after chdir: %s", currentDir)

	// Initialize git repository
	if em.config.GitEnabled {
		if err := em.initGit(); err != nil {
			return fmt.Errorf("failed to initialize git: %w", err)
		}
	}

	// Create initial files
	if err := em.createInitialFiles(); err != nil {
		return fmt.Errorf("failed to create initial files: %w", err)
	}

	// Make initial commit
	if em.config.GitEnabled {
		if err := em.makeInitialCommit(); err != nil {
			return fmt.Errorf("failed to make initial commit: %w", err)
		}
	}

	// Change back to original directory
	if err := os.Chdir(originalDir); err != nil {
		log.Printf("Warning: Failed to change back to original directory: %v", err)
	}

	log.Println("Project environment setup completed")
	return nil
}

// initGit initializes a git repository
func (em *EnvironmentManager) initGit() error {
	cmd := exec.Command("git", "init")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git init failed: %s, %w", string(output), err)
	}

	// Configure git user (if not already configured)
	cmd = exec.Command("git", "config", "user.name", "Claude Agent")
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: Could not set git user name: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "claude@agent.local")
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: Could not set git user email: %v", err)
	}

	return nil
}

// createInitialFiles creates the essential initial files
func (em *EnvironmentManager) createInitialFiles() error {
	// Create init.sh script
	if err := em.createInitScript(); err != nil {
		return fmt.Errorf("failed to create init script: %w", err)
	}

	// Create progress tracking file
	if err := em.createProgressFile(); err != nil {
		return fmt.Errorf("failed to create progress file: %w", err)
	}

	// Create README
	if err := em.createReadme(); err != nil {
		return fmt.Errorf("failed to create README: %w", err)
	}

	return nil
}

// createInitScript creates the initialization script
func (em *EnvironmentManager) createInitScript() error {
	scriptContent := []byte(`#!/bin/bash
# Auto-generated initialization script

echo "Starting development environment setup..."

# Install dependencies (example for Node.js project)
# npm install

# Start development server
# npm run dev

echo "Development server started!"
echo "Visit http://localhost:3000 to view the application"
`)

	// Use absolute path for the script
	scriptPath := filepath.Join(em.config.ProjectDir, "init.sh")

	// Ensure parent directory exists using absolute path
	parentDir := filepath.Dir(scriptPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
	}

	if err := os.WriteFile(scriptPath, scriptContent, 0755); err != nil {
		return fmt.Errorf("failed to write init.sh: %w", err)
	}

	return nil
}

// createProgressFile creates the progress tracking file
func (em *EnvironmentManager) createProgressFile() error {
	progressContent := fmt.Sprintf(`# Claude Agent Progress Tracking
Started: %s

## Session History
- Initial project setup completed

## Current Status
Project initialized and ready for feature development.

## Next Steps
1. Review feature list in feature_list.json
2. Start implementing highest priority features
3. Test each feature thoroughly before committing
`,
		time.Now().Format("2006-01-02 15:04:05"))

	// Ensure directory exists using absolute path
	if err := os.MkdirAll(filepath.Dir(em.config.ProgressFile), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(em.config.ProgressFile, []byte(progressContent), 0644)
}

// createReadme creates a basic README file
func (em *EnvironmentManager) createReadme() error {
	readmeContent := `# Project README

This project is being developed by Claude Agents using incremental development approach.

## Setup
Run ./init.sh to start the development environment.

## Development Process
- Features are implemented one at a time
- Each session focuses on a single feature
- All changes are thoroughly tested before committing
- Progress is tracked in claude-progress.txt

## Project Structure
- feature_list.json: Complete list of features and their status
- claude-progress.txt: Session-by-session progress tracking
- init.sh: Development environment initialization script
`

	// Use absolute path for README
	readmePath := filepath.Join(em.config.ProjectDir, "README.md")

	// Ensure directory exists using absolute path
	if err := os.MkdirAll(filepath.Dir(readmePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(readmePath, []byte(readmeContent), 0644)
}

// makeInitialCommit makes the initial git commit
func (em *EnvironmentManager) makeInitialCommit() error {
	// Add all files
	cmd := exec.Command("git", "add", ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %s, %w", string(output), err)
	}

	// Commit
	cmd = exec.Command("git", "commit", "-m", "Initial project setup by Claude Agent")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %s, %w", string(output), err)
	}

	return nil
}

// GetCurrentDirectory returns the current working directory
func (em *EnvironmentManager) GetCurrentDirectory() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	return dir, nil
}

// GetGitStatus returns the current git status
func (em *EnvironmentManager) GetGitStatus() (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return "Not a git repository", nil
	}

	if len(output) == 0 {
		return "Clean", nil
	}

	return "Modified files present", nil
}

// GetRecentCommits returns recent git commit messages
func (em *EnvironmentManager) GetRecentCommits(count int) ([]string, error) {
	cmd := exec.Command("git", "log", "--oneline", fmt.Sprintf("-%d", count))
	output, err := cmd.Output()
	if err != nil {
		return []string{}, nil // Return empty slice if not a git repo
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var commits []string
	for _, line := range lines {
		if line != "" {
			commits = append(commits, line)
		}
	}

	return commits, nil
}

// AppendToProgressFile adds content to the progress file
func (em *EnvironmentManager) AppendToProgressFile(content string) error {
	file, err := os.OpenFile(em.config.ProgressFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open progress file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	if _, err := writer.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to progress file: %w", err)
	}

	return writer.Flush()
}

// RunCommand executes a shell command in the project directory
func (em *EnvironmentManager) RunCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = em.config.ProjectDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command failed: %s, %w", string(output), err)
	}

	return string(output), nil
}
