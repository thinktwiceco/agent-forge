package llms

import (
	"context"
	"fmt"

	agentforge "github.com/thinktwice/agentForge/src"
)

type OpenAILLMBuilder struct {
	ApiKey   string
	Model    string
	BaseURL  string
	Provider string
	Ctx      context.Context
}

func NewOpenAILLMBuilder(provider string) *OpenAILLMBuilder {
	if provider != "openai" && provider != "deepseek" && provider != "togetherai" {
		panic(fmt.Sprintf("Invalid provider: %s", provider))
	}
	return &OpenAILLMBuilder{
		Provider: provider,
		Ctx:      context.Background(),
	}
}

func (b *OpenAILLMBuilder) validate() {
	c, err := agentforge.NewConfig()
	if err != nil {
		agentforge.Error(fmt.Sprintf("Failed to load config: %v", err))
	}

	if b.Provider == "" {
		agentforge.Error("Provider is required")
	}

	if b.ApiKey == "" {
		if b.Provider == "openai" {
			b.ApiKey = c.AFOpenAIAPIKey
		} else if b.Provider == "deepseek" {
			b.ApiKey = c.AFDeepSeekAPIKey
		} else if b.Provider == "togetherai" {
			b.ApiKey = c.AFTogetherAIAPIKey
		}
	}

	if b.ApiKey == "" {
		agentforge.Warn("No API key found for provider: %s", b.Provider)
	}

	if b.Ctx == nil {
		b.Ctx = context.Background()
	}

	if b.BaseURL == "" {
		canidateBaseURL, ok := DefaultBaseURL[b.Provider]
		if ok {
			b.BaseURL = canidateBaseURL
		} else {
			panic(fmt.Sprintf("No default base URL found for provider: %s", b.Provider))
		}
	}

	if b.Model == "" {
		canidateModel, ok := DefaultModel[b.Provider]
		if ok {
			b.Model = canidateModel
		} else {
			panic(fmt.Sprintf("No default model found for provider: %s", b.Provider))
		}
	}

	agentforge.Info("LLM builder validated: %+v", b)
	agentforge.Info("LLM builder validated: %+v", b.Provider)
	agentforge.Info("LLM builder validated: %+d", len(b.ApiKey))
	agentforge.Info("LLM builder validated: %+v", b.Model)
	agentforge.Info("LLM builder validated: %+v", b.BaseURL)
	agentforge.Info("LLM builder validated: %+v", b.Ctx)
}

func (b *OpenAILLMBuilder) SetProvider(p string) *OpenAILLMBuilder {
	b.Provider = p
	return b
}

func (b *OpenAILLMBuilder) SetApiKey(apiKey string) *OpenAILLMBuilder {
	b.ApiKey = apiKey
	return b
}

func (b *OpenAILLMBuilder) SetModel(model string) *OpenAILLMBuilder {
	b.Model = model
	return b
}

func (b *OpenAILLMBuilder) SetBaseURL(baseURL string) *OpenAILLMBuilder {
	b.BaseURL = baseURL
	return b
}

func (b *OpenAILLMBuilder) SetCtx(ctx context.Context) *OpenAILLMBuilder {
	b.Ctx = ctx
	return b
}

func (b *OpenAILLMBuilder) Build() (LLMEngine, error) {

	b.validate()

	return newOpenAILLM(b.Ctx, b.BaseURL, b.Model, b.ApiKey), nil
}
