package llms

// Base URLs for LLM providers
const DEEPSEEK_BASE_URL = "https://api.deepseek.com/v1"
const TOGETHERAI_BASE_URL = "https://api.together.xyz/v1"
const OPENAI_BASE_URL = "https://api.openai.com/v1"

// Environment variable names for API keys
const (
	DeepSeekAPIKeyEnvVar   = "AF_DEEPSEEK_API_KEY"
	TogetherAIAPIKeyEnvVar = "AF_TOGETHERAI_API_KEY"
	OpenAIAPIKeyEnvVar     = "AF_OPENAI_API_KEY"
)

const TOGETHERAI_Llama323BInstructTurbo = "meta-llama/Llama-3.2-3B-Instruct-Turbo"
const TOGETHERAI_OPENAIGPTOSS120B = "openai/gpt-oss-120b"
const TOGETHERAI_Qwen257BInstructTurbo = "Qwen/Qwen2.5-7B-Instruct-Turbo"
const TOGETHERAI_Llama3170BInstructTurbo = "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo"

const OPENAI_GPT5O = "gpt-5"
const OPENAI_GPT5_1 = "gpt-5.1"
const OPENAI_GPT5_2 = "gpt-5.2"

const DEEPSEEK_CHAT = "deepseek-chat"
const DEEPSEEK_REASONING = "deepseek-reasoning"

var DefaultModel = map[string]string{
	"openai":     OPENAI_GPT5_1,
	"deepseek":   DEEPSEEK_CHAT,
	"togetherai": TOGETHERAI_Llama3170BInstructTurbo,
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
