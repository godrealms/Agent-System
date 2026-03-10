package environment

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// EnvironmentManager sets up and manages the project directory and git repo
type EnvironmentManager struct {
	projectDir string
}

// NewEnvironmentManager creates a manager for the given directory
func NewEnvironmentManager(projectDir string) *EnvironmentManager {
	return &EnvironmentManager{projectDir: projectDir}
}

// SetupProject creates the project directory and initialises a git repo
func (em *EnvironmentManager) SetupProject() error {
	if err := os.MkdirAll(em.projectDir, 0755); err != nil {
		return fmt.Errorf("create project directory: %w", err)
	}

	if err := em.initGit(); err != nil {
		return fmt.Errorf("init git: %w", err)
	}

	if err := em.createProgressFile(); err != nil {
		return fmt.Errorf("create progress file: %w", err)
	}

	return nil
}

// initGit runs git init if .git does not already exist
func (em *EnvironmentManager) initGit() error {
	gitDir := filepath.Join(em.projectDir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return nil // already initialised
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = em.projectDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %w\n%s", err, out)
	}

	// Set default author so commits work without global git config
	for _, args := range [][]string{
		{"config", "user.email", "agent@claude.ai"},
		{"config", "user.name", "Claude Agent"},
	} {
		c := exec.Command("git", args...)
		c.Dir = em.projectDir
		if out, err := c.CombinedOutput(); err != nil {
			return fmt.Errorf("git config: %w\n%s", err, out)
		}
	}

	return nil
}

// createProgressFile creates claude-progress.txt with an initial entry
func (em *EnvironmentManager) createProgressFile() error {
	progressFile := filepath.Join(em.projectDir, "claude-progress.txt")
	if _, err := os.Stat(progressFile); err == nil {
		return nil // already exists
	}

	content := fmt.Sprintf("Project initialized at %s\n", time.Now().Format(time.RFC3339))
	return os.WriteFile(progressFile, []byte(content), 0644)
}

// IsGitRepo returns true if the project directory is inside a git repository
func (em *EnvironmentManager) IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = em.projectDir
	return cmd.Run() == nil
}
