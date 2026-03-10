package config

import (
	"os"
	"path/filepath"
	"strconv"
)

// Config holds the configuration for the long-running agent system
type Config struct {
	// API Keys
	AnthropicKey string

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

	// Monitoring
	MonitoringEnabled bool
	MonitoringPort    int

	// Performance
	MaxConcurrentSessions int
	CacheEnabled          bool
	CacheSizeMB           int
	MemoryLimitMB         int
	GCThreshold           float64
	BatchProcessing       bool
	BatchSize             int

	// Plugins
	PluginDir string
}

// LoadConfig loads configuration from environment variables and defaults
func LoadConfig(projectDir string) *Config {
	return &Config{
		AnthropicKey:          os.Getenv("ANTHROPIC_API_KEY"),
		ProjectDir:            projectDir,
		MaxContextSize:        200000, // Claude Opus 4.6 context window
		SessionTimeout:        30,     // 30 minutes
		GitEnabled:            true,
		GitRemote:             "",
		ProgressFile:          filepath.Join(projectDir, "claude-progress.txt"),
		FeatureFile:           filepath.Join(projectDir, "feature_list.json"),
		TestEnabled:           true,
		BrowserTool:           "puppeteer",
		MonitoringEnabled:     true,
		MonitoringPort:        8080,
		MaxConcurrentSessions: getIntEnv("MAX_CONCURRENT_SESSIONS", 1),
		CacheEnabled:          getBoolEnv("CACHE_ENABLED", true),
		CacheSizeMB:           getIntEnv("CACHE_SIZE_MB", 100),
		MemoryLimitMB:         getIntEnv("MEMORY_LIMIT_MB", 512),
		GCThreshold:           getFloatEnv("GC_THRESHOLD", 0.8),
		BatchProcessing:       getBoolEnv("BATCH_PROCESSING", false),
		BatchSize:             getIntEnv("BATCH_SIZE", 10),
		PluginDir:             filepath.Join(projectDir, "plugins"),
	}
}

// Helper functions for environment variables
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true"
	}
	return defaultValue
}

func getFloatEnv(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}
