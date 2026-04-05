package config

import (
	"os"
	"path/filepath"
	"strconv"
)

// Config holds the configuration for the long-running agent system.
type Config struct {
	// --- Provider selection ---
	// DefaultProvider selects which LLM backend to use.
	// Values: claude | openai | kimi | minimax | openrouter | groq |
	//         gemini | huggingface | fireworks | cloudflare
	// Defaults to "claude" when empty.
	DefaultProvider string

	// --- Anthropic / Claude ---
	AnthropicKey   string
	AnthropicModel string // default: claude-opus-4-6

	// --- OpenAI (ChatGPT) ---
	OpenAIKey   string
	OpenAIModel string // default: gpt-4o

	// --- Kimi (Moonshot AI) ---
	KimiKey   string
	KimiModel string // default: moonshot-v1-8k

	// --- MINIMAX ---
	MiniMaxKey   string
	MiniMaxModel string // default: MiniMax-Text-01

	// --- OpenRouter ---
	OpenRouterKey   string
	OpenRouterModel string // default: openai/gpt-4o

	// --- GroqCloud ---
	GroqKey   string
	GroqModel string // default: llama-3.3-70b-versatile

	// --- Google Gemini ---
	GeminiKey   string
	GeminiModel string // default: gemini-2.0-flash

	// --- Hugging Face Inference Providers ---
	HuggingFaceKey   string
	HuggingFaceModel string // default: meta-llama/Llama-3.3-70B-Instruct

	// --- Fireworks AI ---
	FireworksKey   string
	FireworksModel string // default: accounts/fireworks/models/llama-v3p3-70b-instruct

	// --- Cloudflare Workers AI ---
	CloudflareKey       string
	CloudflareAccountID string
	CloudflareModel     string // default: @cf/meta/llama-3.1-8b-instruct

	// --- Project settings ---
	ProjectDir     string
	MaxContextSize int
	SessionTimeout int // in minutes

	// --- Git settings ---
	GitEnabled bool
	GitRemote  string

	// --- Progress tracking ---
	ProgressFile string
	FeatureFile  string

	// --- Testing ---
	TestEnabled bool
	BrowserTool string

	// --- Monitoring ---
	MonitoringEnabled bool
	MonitoringPort    int

	// --- Performance ---
	MaxConcurrentSessions int
	CacheEnabled          bool
	CacheSizeMB           int
	MemoryLimitMB         int
	GCThreshold           float64
	BatchProcessing       bool
	BatchSize             int

	// --- Plugins ---
	PluginDir string
}

// LoadConfig loads configuration from environment variables with sensible defaults.
func LoadConfig(projectDir string) *Config {
	return &Config{
		// Provider selection
		DefaultProvider: getEnv("LLM_PROVIDER", "claude"),

		// Anthropic
		AnthropicKey:   os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicModel: getEnv("ANTHROPIC_MODEL", "claude-opus-4-6"),

		// OpenAI
		OpenAIKey:   os.Getenv("OPENAI_API_KEY"),
		OpenAIModel: getEnv("OPENAI_MODEL", "gpt-4o"),

		// Kimi
		KimiKey:   os.Getenv("KIMI_API_KEY"),
		KimiModel: getEnv("KIMI_MODEL", "moonshot-v1-8k"),

		// MINIMAX
		MiniMaxKey:   os.Getenv("MINIMAX_API_KEY"),
		MiniMaxModel: getEnv("MINIMAX_MODEL", "MiniMax-Text-01"),

		// OpenRouter
		OpenRouterKey:   os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel: getEnv("OPENROUTER_MODEL", "openai/gpt-4o"),

		// Groq
		GroqKey:   os.Getenv("GROQ_API_KEY"),
		GroqModel: getEnv("GROQ_MODEL", "llama-3.3-70b-versatile"),

		// Gemini
		GeminiKey:   os.Getenv("GEMINI_API_KEY"),
		GeminiModel: getEnv("GEMINI_MODEL", "gemini-2.0-flash"),

		// HuggingFace
		HuggingFaceKey:   os.Getenv("HUGGINGFACE_API_KEY"),
		HuggingFaceModel: getEnv("HUGGINGFACE_MODEL", "meta-llama/Llama-3.3-70B-Instruct"),

		// Fireworks
		FireworksKey:   os.Getenv("FIREWORKS_API_KEY"),
		FireworksModel: getEnv("FIREWORKS_MODEL", "accounts/fireworks/models/llama-v3p3-70b-instruct"),

		// Cloudflare
		CloudflareKey:       os.Getenv("CLOUDFLARE_API_KEY"),
		CloudflareAccountID: os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		CloudflareModel:     getEnv("CLOUDFLARE_MODEL", "@cf/meta/llama-3.1-8b-instruct"),

		// Project
		ProjectDir:     projectDir,
		MaxContextSize: 200000,
		SessionTimeout: 30,
		GitEnabled:     true,
		GitRemote:      "",
		ProgressFile:   filepath.Join(projectDir, "claude-progress.txt"),
		FeatureFile:    filepath.Join(projectDir, "feature_list.json"),

		// Testing
		TestEnabled: true,
		BrowserTool: "puppeteer",

		// Monitoring
		MonitoringEnabled: true,
		MonitoringPort:    8080,

		// Performance
		MaxConcurrentSessions: getIntEnv("MAX_CONCURRENT_SESSIONS", 1),
		CacheEnabled:          getBoolEnv("CACHE_ENABLED", true),
		CacheSizeMB:           getIntEnv("CACHE_SIZE_MB", 100),
		MemoryLimitMB:         getIntEnv("MEMORY_LIMIT_MB", 512),
		GCThreshold:           getFloatEnv("GC_THRESHOLD", 0.8),
		BatchProcessing:       getBoolEnv("BATCH_PROCESSING", false),
		BatchSize:             getIntEnv("BATCH_SIZE", 10),

		// Plugins
		PluginDir: filepath.Join(projectDir, "plugins"),
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

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
