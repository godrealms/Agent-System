package config

import (
	"os"
	"path/filepath"
)

// Config holds the configuration for the long-running agent system
type Config struct {
	// API Keys
	OpenAIKey string

	// Project settings
	ProjectDir     string
	MaxContextSize int
	SessionTimeout int // in minutes

	// Git settings
	GitEnabled bool
	GitRemote  string

	// Progress tracking
	ProgressFile string
	FeatureFile  string

	// Testing
	TestEnabled bool
	BrowserTool string // puppeteer, selenium, etc.
}

// LoadConfig loads configuration from environment variables and defaults
func LoadConfig(projectDir string) *Config {
	return &Config{
		OpenAIKey:      os.Getenv("OPENAI_API_KEY"),
		ProjectDir:     projectDir,
		MaxContextSize: 128000, // Claude Opus context window
		SessionTimeout: 30,     // 30 minutes
		GitEnabled:     true,
		GitRemote:      "",
		ProgressFile:   filepath.Join(projectDir, "claude-progress.txt"),
		FeatureFile:    filepath.Join(projectDir, "feature_list.json"),
		TestEnabled:    true,
		BrowserTool:    "puppeteer",
	}
}
