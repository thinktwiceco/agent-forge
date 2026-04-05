package llms

import (
	"context"
	"fmt"

	agentforge "github.com/thinktwiceco/agent-forge/src"
)

type OpenAILLMBuilder struct {
	apiKey         string
	model          string
	cheapModel     string
	reasoningModel string
	fastModel      string
	baseURL        string
	provider       string
	ctx            context.Context
}

func NewOpenAILLMBuilder(provider string) (*OpenAILLMBuilder, error) {
	if provider != "openai" && provider != "deepseek" && provider != "togetherai" && provider != "openrouter" {
		return nil, fmt.Errorf("invalid provider: %s", provider)
	}
	return &OpenAILLMBuilder{
		provider: provider,
		ctx:      context.Background(),
	}, nil
}

func (b *OpenAILLMBuilder) validate() error {
	c, err := agentforge.NewConfig()
	if err != nil {
		agentforge.Error("Failed to load config: %v", err)
	}

	if b.provider == "" {
		return fmt.Errorf("provider is required")
	}

	if b.apiKey == "" {
		switch b.provider {
		case "openai":
			b.apiKey = c.AFOpenAIAPIKey
		case "deepseek":
			b.apiKey = c.AFDeepSeekAPIKey
		case "togetherai":
			b.apiKey = c.AFTogetherAIAPIKey
		case "openrouter":
			b.apiKey = c.AFOpenRouterAPIKey
		}
	}

	if b.apiKey == "" {
		agentforge.Warn("No API key found for provider: %s", b.provider)
	}

	if b.ctx == nil {
		b.ctx = context.Background()
	}

	if b.baseURL == "" {
		canidateBaseURL, ok := DefaultBaseURL[b.provider]
		if ok {
			b.baseURL = canidateBaseURL
		} else {
			return fmt.Errorf("no default base URL found for provider: %s", b.provider)
		}
	}

	if b.model == "" {
		canidateModel, ok := DefaultModel[b.provider]
		if ok {
			b.model = canidateModel
		} else {
			return fmt.Errorf("no default model found for provider: %s", b.provider)
		}
	}

	if b.cheapModel == "" {
		canidateModel, ok := DefaultCheapModel[b.provider]
		if ok {
			b.cheapModel = canidateModel
		}
	}

	if b.reasoningModel == "" {
		canidateModel, ok := DefaultReasoningModel[b.provider]
		if ok {
			b.reasoningModel = canidateModel
		}
	}

	if b.fastModel == "" {
		canidateModel, ok := DefaultFastModel[b.provider]
		if ok {
			b.fastModel = canidateModel
		}
	}

	agentforge.Info("LLM builder validated: provider=%+v", b.provider)
	agentforge.Info("LLM builder validated: apiKey length=%+d", len(b.apiKey))
	agentforge.Info("LLM builder validated: model=%+v", b.model)
	agentforge.Info("LLM builder validated: cheapModel=%+v", b.cheapModel)
	agentforge.Info("LLM builder validated: reasoningModel=%+v", b.reasoningModel)
	agentforge.Info("LLM builder validated: fastModel=%+v", b.fastModel)
	agentforge.Info("LLM builder validated: baseURL=%+v", b.baseURL)
	return nil
}

func (b *OpenAILLMBuilder) SetProvider(p string) *OpenAILLMBuilder {
	b.provider = p
	return b
}

func (b *OpenAILLMBuilder) SetApiKey(apiKey string) *OpenAILLMBuilder {
	b.apiKey = apiKey
	return b
}

func (b *OpenAILLMBuilder) SetModel(model string) *OpenAILLMBuilder {
	b.model = model
	return b
}

func (b *OpenAILLMBuilder) SetCheapModel(model string) *OpenAILLMBuilder {
	b.cheapModel = model
	return b
}

func (b *OpenAILLMBuilder) SetReasoningModel(model string) *OpenAILLMBuilder {
	b.reasoningModel = model
	return b
}

func (b *OpenAILLMBuilder) SetFastModel(model string) *OpenAILLMBuilder {
	b.fastModel = model
	return b
}

func (b *OpenAILLMBuilder) SetBaseURL(baseURL string) *OpenAILLMBuilder {
	b.baseURL = baseURL
	return b
}

func (b *OpenAILLMBuilder) SetCtx(ctx context.Context) *OpenAILLMBuilder {
	b.ctx = ctx
	return b
}

// MultiModelLLM holds multiple LLM instances for different use cases
type MultiModelLLM struct {
	mainModel      *openAILLM
	cheapModel     *openAILLM
	reasoningModel *openAILLM
	fastModel      *openAILLM
}

// MainModel returns the main model instance
func (m *MultiModelLLM) MainModel() *openAILLM {
	return m.mainModel
}

// CheapModel returns the cheap model instance (may be nil)
func (m *MultiModelLLM) CheapModel() *openAILLM {
	return m.cheapModel
}

// ReasoningModel returns the reasoning model instance (may be nil)
func (m *MultiModelLLM) ReasoningModel() *openAILLM {
	return m.reasoningModel
}

// FastModel returns the fast model instance (may be nil)
func (m *MultiModelLLM) FastModel() *openAILLM {
	return m.fastModel
}

func (b *OpenAILLMBuilder) Build() (*MultiModelLLM, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}

	multiModel := &MultiModelLLM{}

	// Create main model (always required)
	multiModel.mainModel = newOpenAILLM(b.ctx, b.baseURL, b.model, b.apiKey, b.provider)

	// Create cheap model if set
	if b.cheapModel != "" {
		multiModel.cheapModel = newOpenAILLM(b.ctx, b.baseURL, b.cheapModel, b.apiKey, b.provider)
	}

	// Create reasoning model if set
	if b.reasoningModel != "" {
		multiModel.reasoningModel = newOpenAILLM(b.ctx, b.baseURL, b.reasoningModel, b.apiKey, b.provider)
	}

	// Create fast model if set
	if b.fastModel != "" {
		multiModel.fastModel = newOpenAILLM(b.ctx, b.baseURL, b.fastModel, b.apiKey, b.provider)
	}

	return multiModel, nil
}
