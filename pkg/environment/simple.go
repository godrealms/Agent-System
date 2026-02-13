package environment

import (
	"fmt"
	"os"
)

type EnvironmentManager struct {
	projectDir string
}

func NewEnvironmentManager(projectDir string) *EnvironmentManager {
	return &EnvironmentManager{projectDir: projectDir}
}

func (em *EnvironmentManager) SetupProject() error {
	if err := os.MkdirAll(em.projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}
	return nil
}
