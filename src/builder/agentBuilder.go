package builder

import (
	"fmt"
	"os"

	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Agent struct {
		Name         string              `yaml:"name"`
		SystemPrompt string              `yaml:"system_prompt"`
		Model        string              `yaml:"model"`
		WorkingDir   string              `yaml:"working_dir"`
		Persistence  string              `yaml:"persistence"`
		Tools        []Tool              `yaml:"tools"`
		Subagents    map[Subagent]string `yaml:"subagents"`
		Plugins      []string            `yaml:"plugins"`
	} `yaml:"agent"`
	VectorStorage *VectorStorageConfig `yaml:"vector-storage,omitempty"`
}

type AgentBuilder struct {
	name               string
	systemPrompt       string
	tools              []Tool
	Subagents          map[Subagent]LLM
	plugins            []string
	llmEngine          llms.LLMEngine
	workingDir         string
	persistence        string
	vectorDB           core.VectorDB
	embeddingGenerator core.EmbeddingGenerator
}

// Public API methods

func NewAgentBuilder(name string, persistence string) *AgentBuilder {
	if persistence != "" && persistence != "json" {
		panic("invalid persistence type: " + persistence)
	}
	return &AgentBuilder{
		name:        name,
		persistence: persistence,
	}
}

func NewAgentBuilderFromConfig(configPath string) (*AgentBuilder, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return newAgentBuilderFromConfigStruct(cfg)
}

// NewAgentBuilderFromConfigStruct creates an agent builder from an already-loaded Config struct.
// This is useful when the config has been pre-processed (e.g., environment variable interpolation).
func NewAgentBuilderFromConfigStruct(cfg Config) (*AgentBuilder, error) {
	return newAgentBuilderFromConfigStruct(cfg)
}

func newAgentBuilderFromConfigStruct(cfg Config) (*AgentBuilder, error) {
	b := NewAgentBuilder(cfg.Agent.Name, cfg.Agent.Persistence)
	if cfg.Agent.SystemPrompt != "" {
		b.SetSystemPrompt(cfg.Agent.SystemPrompt)
	}
	if cfg.Agent.Model != "" {
		b.SetModel(cfg.Agent.Model)
	}
	if cfg.Agent.WorkingDir != "" {
		b.SetWorkingDir(cfg.Agent.WorkingDir)
	}
	if len(cfg.Agent.Tools) > 0 {
		b.AddTools(cfg.Agent.Tools...)
	}
	for sub, model := range cfg.Agent.Subagents {
		b.AddSubagent(sub, model)
	}
	for _, plugin := range cfg.Agent.Plugins {
		b.AddPlugin(plugin)
	}

	return b, nil
}

func (b *AgentBuilder) SetSystemPrompt(prompt string) *AgentBuilder {
	b.systemPrompt = prompt
	return b
}

func (b *AgentBuilder) GetName() string {
	return b.name
}

func (b *AgentBuilder) AddTools(tools ...Tool) *AgentBuilder {
	b.tools = append(b.tools, tools...)
	return b
}

func (b *AgentBuilder) SetModel(model string) *AgentBuilder {
	llm := fromString(model)
	llm.model()
	llmEngine, err := b.createLLMEngine(llm)
	if err != nil {
		panic(err)
	}
	b.llmEngine = llmEngine
	return b
}

func (b *AgentBuilder) AddSubagent(subagent Subagent, model string) *AgentBuilder {
	llm := fromString(model)
	llm.model()
	llm.provider()
	if b.Subagents == nil {
		b.Subagents = make(map[Subagent]LLM)
	}
	b.Subagents[subagent] = llm
	return b
}

func (b *AgentBuilder) AddPlugin(plugin string) *AgentBuilder {
	b.plugins = append(b.plugins, plugin)
	return b
}

func (b *AgentBuilder) SetVectorDB(vectorDB core.VectorDB) *AgentBuilder {
	b.vectorDB = vectorDB
	return b
}

func (b *AgentBuilder) SetEmbeddingGenerator(embeddingGenerator core.EmbeddingGenerator) *AgentBuilder {
	b.embeddingGenerator = embeddingGenerator
	return b
}

func (b *AgentBuilder) SetWorkingDir(workingDir string) *AgentBuilder {
	b.workingDir = workingDir
	return b
}

func (b *AgentBuilder) Build() (*agents.Agent, error) {
	err := b.validate()
	if err != nil {
		return nil, err
	}

	subagents, err := b.buildSubagents()

	if err != nil {
		return nil, err
	}

	tools, err := b.buildTools()
	if err != nil {
		return nil, err
	}

	plugins, err := b.buildPlugins()
	if err != nil {
		return nil, err
	}

	agentConfig := &agents.AgentConfig{
		LLMEngine:    b.llmEngine,
		AgentName:    b.name,
		Description:  "Main Agent",
		Tone:         agents.ToneKeepItShort,
		Trace:        fmt.Sprintf("%s-trace", b.name),
		CanExpand:    true,
		MainAgent:    true,
		Persistence:  b.persistence,
		WorkingDir:   b.workingDir,
		SubAgents:    subagents,
		Tools:        tools,
		Plugins:      plugins,
		SystemPrompt: b.systemPrompt,
	}

	agent := agents.NewAgent(agentConfig)

	return agent, nil
}

// Private helper methods

func (b *AgentBuilder) validate() error {
	if b.name == "" {
		return fmt.Errorf("`name` is required to build an agent")
	}

	if b.llmEngine == nil {
		return fmt.Errorf("`llmEngine` is required to build an agent")
	}

	return nil
}

func (b *AgentBuilder) createLLMEngine(l LLM) (llms.LLMEngine, error) {
	llmBuilder := llms.NewOpenAILLMBuilder(l.provider())
	llmBuilder.SetModel(l.model())
	mmEngine, err := llmBuilder.Build()
	if err != nil {
		return nil, err
	}
	return mmEngine.MainModel(), nil
}

func (b *AgentBuilder) buildSubagents() ([]core.SubAgent, error) {
	subagents := []core.SubAgent{}
	for subagent, llm := range b.Subagents {
		llmEngine, err := b.createLLMEngine(llm)
		if err != nil {
			return nil, err
		}

		subagentInstance, err := subagent.getSubagent(llmEngine, b.vectorDB, b.embeddingGenerator, b.workingDir)

		if err != nil {
			return nil, err
		}
		subagents = append(subagents, subagentInstance)
	}
	return subagents, nil
}

func (b *AgentBuilder) buildTools() ([]llms.Tool, error) {
	tools := []llms.Tool{}
	for _, tool := range b.tools {
		t, err := tool.getTool(b.workingDir, b.vectorDB, b.embeddingGenerator)
		if err != nil {
			return nil, err
		}
		tools = append(tools, t)
	}
	return tools, nil
}

func (b *AgentBuilder) buildPlugins() ([]core.Plugin, error) {
	plugins := []core.Plugin{}
	for _, plugin := range b.plugins {
		plugin, err := getPlugin(plugin, b.workingDir)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, plugin)
	}
	return plugins, nil
}
