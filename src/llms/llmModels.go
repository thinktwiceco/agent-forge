package llms

// ModelInfo contains static metadata about a model.
// Zero-value means unknown/unsupported for that field.
type ModelInfo struct {
	Name              string
	Provider          string
	ContextWindow     int // max input tokens; 0 = unknown
	MaxOutputTokens   int // max completion tokens; 0 = unknown
	SupportsReasoning bool
	SupportsVision    bool
	InputPricePer1M   float64 // USD per 1M input tokens
	OutputPricePer1M  float64 // USD per 1M output tokens
}

// Base URLs for LLM providers
const DEEPSEEK_BASE_URL = "https://api.deepseek.com/v1"
const TOGETHERAI_BASE_URL = "https://api.together.xyz/v1"
const OPENAI_BASE_URL = "https://api.openai.com/v1"

// Environment variable names for API keys
const DeepSeekAPIKeyEnvVar = "AF_DEEPSEEK_API_KEY"
const TogetherAIAPIKeyEnvVar = "AF_TOGETHERAI_API_KEY"
const OpenAIAPIKeyEnvVar = "AF_OPENAI_API_KEY"

// Models for TogetherAI
const TOGETHERAI_Llama323BInstructTurbo = "meta-llama/Llama-3.2-3B-Instruct-Turbo"
const TOGETHERAI_OPENAIGPTOSS120B = "openai/gpt-oss-120b"
const TOGETHERAI_Qwen257BInstructTurbo = "Qwen/Qwen2.5-7B-Instruct-Turbo"
const TOGETHERAI_Llama3170BInstructTurbo = "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo"
const TOGETHERAI_Qwen3Coder480B = "Qwen/Qwen3-Coder-480B-A35B-Instruct-FP8"
const TOGETHERAI_ZaiGLM47 = "zai-org/GLM-4.7"
const TOGETHERAI_KimiK25 = "moonshotai/Kimi-K2.5"

// Models for OpenAI
const OPENAI_GPT5O = "gpt-5"
const OPENAI_GPT5_1 = "gpt-5.1"
const OPENAI_GPT5_2 = "gpt-5.2"

// Models for DeepSeek
const DEEPSEEK_CHAT = "deepseek-chat"
const DEEPSEEK_REASONING = "deepseek-chat" // Using deepseek-chat as reasoning model (deepseek-reasoning doesn't exist in API)

var DefaultModel = map[string]string{
	"openai":     OPENAI_GPT5_1,
	"deepseek":   DEEPSEEK_CHAT,
	"togetherai": TOGETHERAI_Llama3170BInstructTurbo,
}

var DefaultCheapModel = map[string]string{
	"openai":     OPENAI_GPT5O,
	"deepseek":   DEEPSEEK_CHAT,
	"togetherai": TOGETHERAI_Llama323BInstructTurbo,
}

var DefaultReasoningModel = map[string]string{
	"openai":     OPENAI_GPT5_2,
	"deepseek":   DEEPSEEK_REASONING,
	"togetherai": TOGETHERAI_Llama3170BInstructTurbo,
}

var DefaultFastModel = map[string]string{
	"openai":     OPENAI_GPT5O,
	"deepseek":   DEEPSEEK_CHAT,
	"togetherai": TOGETHERAI_Llama323BInstructTurbo,
}

var DefaultBaseURL = map[string]string{
	"openai":     OPENAI_BASE_URL,
	"deepseek":   DEEPSEEK_BASE_URL,
	"togetherai": TOGETHERAI_BASE_URL,
}

var ProviderAPIKeyEnvVar = map[string]string{
	"openai":     OpenAIAPIKeyEnvVar,
	"deepseek":   DeepSeekAPIKeyEnvVar,
	"togetherai": TogetherAIAPIKeyEnvVar,
}

// KnownModels maps model name strings to their static metadata.
// Models not present return a zero-value ModelInfo (all fields unknown).
var KnownModels = map[string]ModelInfo{
	OPENAI_GPT5O: {
		Name: OPENAI_GPT5O, Provider: "openai",
		ContextWindow: 128_000, MaxOutputTokens: 16_384,
		SupportsVision: true,
	},
	OPENAI_GPT5_1: {
		Name: OPENAI_GPT5_1, Provider: "openai",
		ContextWindow: 128_000, MaxOutputTokens: 16_384,
		SupportsVision: true,
	},
	OPENAI_GPT5_2: {
		Name: OPENAI_GPT5_2, Provider: "openai",
		ContextWindow: 128_000, MaxOutputTokens: 16_384,
		SupportsReasoning: true, SupportsVision: true,
	},
	DEEPSEEK_CHAT: {
		Name: DEEPSEEK_CHAT, Provider: "deepseek",
		ContextWindow: 64_000, MaxOutputTokens: 8_000,
		InputPricePer1M: 0.27, OutputPricePer1M: 1.10,
	},
	TOGETHERAI_Llama3170BInstructTurbo: {
		Name: TOGETHERAI_Llama3170BInstructTurbo, Provider: "togetherai",
		ContextWindow: 128_000, MaxOutputTokens: 4_096,
	},
	TOGETHERAI_Llama323BInstructTurbo: {
		Name: TOGETHERAI_Llama323BInstructTurbo, Provider: "togetherai",
		ContextWindow: 128_000, MaxOutputTokens: 4_096,
	},
	TOGETHERAI_Qwen257BInstructTurbo: {
		Name: TOGETHERAI_Qwen257BInstructTurbo, Provider: "togetherai",
		ContextWindow: 32_000, MaxOutputTokens: 4_096,
	},
	TOGETHERAI_Qwen3Coder480B: {
		Name: TOGETHERAI_Qwen3Coder480B, Provider: "togetherai",
		ContextWindow: 131_072, MaxOutputTokens: 8_192,
		SupportsReasoning: true,
	},
	TOGETHERAI_OPENAIGPTOSS120B: {
		Name: TOGETHERAI_OPENAIGPTOSS120B, Provider: "togetherai",
		ContextWindow: 128_000, MaxOutputTokens: 16_384,
	},
	TOGETHERAI_ZaiGLM47: {
		Name: TOGETHERAI_ZaiGLM47, Provider: "togetherai",
		ContextWindow: 128_000, MaxOutputTokens: 8_192,
	},
	TOGETHERAI_KimiK25: {
		Name: TOGETHERAI_KimiK25, Provider: "togetherai",
		ContextWindow: 128_000, MaxOutputTokens: 8_192,
	},
}
