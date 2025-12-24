package llms

import (
	"context"
	"fmt"

	agentforge "github.com/thinktwice/agentForge/src"
)

const (
	// DeepSeekBaseURL is the base URL for DeepSeek's API
	DeepSeekBaseURL = "https://api.deepseek.com/v1"
	// DeepSeekAPIKeyEnvVar is the environment variable name for DeepSeek API key
	DeepSeekAPIKeyEnvVar = "DEEPSEEK_API_KEY"
	// TogetherAIBaseURL is the base URL for TogetherAI's API
	TogetherAIBaseURL = "https://api.together.xyz/v1"
	// TogetherAIAPIKeyEnvVar is the environment variable name for TogetherAI API key
	TogetherAIAPIKeyEnvVar = "TOGETHERAI_API_KEY"
)

// GetDeepSeekLLM creates a new LLM engine instance configured for DeepSeek.
//
// DeepSeek uses an OpenAI-compatible API, so this function creates an
// engine with DeepSeek's base URL and default model.
//
// This function automatically loads the API key from:
// 1. .env file (searches current directory and parent directories)
// 2. os.Getenv("DEEPSEEK_API_KEY")
// 3. Returns error if API key is not found
//
// Parameters:
//   - ctx: Context for cancellation
//   - model: Model name (e.g., "deepseek-chat", defaults to "deepseek-chat" if empty)
//
// Returns:
//   - LLMEngine: LLMEngine instance configured for DeepSeek
//   - error: Error if API key is not found
func GetDeepSeekLLM(ctx context.Context, model string) (LLMEngine, error) {
	// Get API key from .env file or os environment
	apiKey, err := agentforge.GetEnvVar(DeepSeekAPIKeyEnvVar)
	if err != nil {
		return nil, fmt.Errorf("failed to get DeepSeek API key: %w", err)
	}

	// Default to deepseek-chat if no model specified
	if model == "" {
		model = "deepseek-chat"
	}

	// Create engine instance
	engine := newOpenAILLM(ctx, DeepSeekBaseURL, model, apiKey)

	return engine, nil
}

// GetTogetherAILLM creates a new LLM engine instance configured for TogetherAI.
//
// TogetherAI uses an OpenAI-compatible API, so this function creates an
// engine with TogetherAI's base URL and default model.
//
// This function automatically loads the API key from:
// 1. .env file (searches current directory and parent directories)
// 2. os.Getenv("TOGETHER_API_KEY")
// 3. Returns error if API key is not found
//
// Parameters:
//   - ctx: Context for cancellation
//   - model: Model name (e.g., "meta-llama/Llama-3-8b-chat-hf", defaults to "meta-llama/Llama-3-8b-chat-hf" if empty)
//
// Returns:
//   - LLMEngine: LLMEngine instance configured for TogetherAI
//   - error: Error if API key is not found
func GetTogetherAILLM(ctx context.Context, model string) (LLMEngine, error) {
	// Get API key from .env file or os environment
	apiKey, err := agentforge.GetEnvVar(TogetherAIAPIKeyEnvVar)
	if err != nil {
		return nil, fmt.Errorf("failed to get TogetherAI API key: %w", err)
	}

	if model == "" {
		model = Llama323BInstructTurbo
	}

	// Create engine instance
	engine := newOpenAILLM(ctx, TogetherAIBaseURL, model, apiKey)

	return engine, nil
}
