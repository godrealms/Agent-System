package providers

// Factory creates the correct Provider based on Config.

import (
	"fmt"

	"AI-agent/internal/config"
)

// New creates the Provider specified by cfg.DefaultProvider.
// Returns an error if the required API key is missing.
func New(cfg *config.Config) (Provider, error) {
	switch ProviderType(cfg.DefaultProvider) {

	case ProviderClaude, "":
		if cfg.AnthropicKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY is required for provider %q", ProviderClaude)
		}
		return NewAnthropicProvider(cfg.AnthropicKey, cfg.AnthropicModel), nil

	case ProviderOpenAI:
		if cfg.OpenAIKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY is required for provider %q", ProviderOpenAI)
		}
		return NewOpenAICompatProvider(OpenAIPreset(cfg.OpenAIKey, cfg.OpenAIModel)), nil

	case ProviderKimi:
		if cfg.KimiKey == "" {
			return nil, fmt.Errorf("KIMI_API_KEY is required for provider %q", ProviderKimi)
		}
		return NewOpenAICompatProvider(KimiPreset(cfg.KimiKey, cfg.KimiModel)), nil

	case ProviderMiniMax:
		if cfg.MiniMaxKey == "" {
			return nil, fmt.Errorf("MINIMAX_API_KEY is required for provider %q", ProviderMiniMax)
		}
		return NewOpenAICompatProvider(MiniMaxPreset(cfg.MiniMaxKey, cfg.MiniMaxModel)), nil

	case ProviderOpenRouter:
		if cfg.OpenRouterKey == "" {
			return nil, fmt.Errorf("OPENROUTER_API_KEY is required for provider %q", ProviderOpenRouter)
		}
		return NewOpenAICompatProvider(OpenRouterPreset(cfg.OpenRouterKey, cfg.OpenRouterModel)), nil

	case ProviderGroq:
		if cfg.GroqKey == "" {
			return nil, fmt.Errorf("GROQ_API_KEY is required for provider %q", ProviderGroq)
		}
		return NewOpenAICompatProvider(GroqPreset(cfg.GroqKey, cfg.GroqModel)), nil

	case ProviderGemini:
		if cfg.GeminiKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY is required for provider %q", ProviderGemini)
		}
		return NewGeminiProvider(cfg.GeminiKey, cfg.GeminiModel), nil

	case ProviderHuggingFace:
		if cfg.HuggingFaceKey == "" {
			return nil, fmt.Errorf("HUGGINGFACE_API_KEY is required for provider %q", ProviderHuggingFace)
		}
		return NewOpenAICompatProvider(HuggingFacePreset(cfg.HuggingFaceKey, cfg.HuggingFaceModel)), nil

	case ProviderFireworks:
		if cfg.FireworksKey == "" {
			return nil, fmt.Errorf("FIREWORKS_API_KEY is required for provider %q", ProviderFireworks)
		}
		return NewOpenAICompatProvider(FireworksPreset(cfg.FireworksKey, cfg.FireworksModel)), nil

	case ProviderCloudflare:
		if cfg.CloudflareKey == "" || cfg.CloudflareAccountID == "" {
			return nil, fmt.Errorf("CLOUDFLARE_API_KEY and CLOUDFLARE_ACCOUNT_ID are required for provider %q", ProviderCloudflare)
		}
		return NewOpenAICompatProvider(CloudflarePreset(cfg.CloudflareKey, cfg.CloudflareAccountID, cfg.CloudflareModel)), nil

	default:
		return nil, fmt.Errorf("unknown provider %q — valid values: claude openai kimi minimax openrouter groq gemini huggingface fireworks cloudflare", cfg.DefaultProvider)
	}
}
