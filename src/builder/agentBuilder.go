package builder

import (
	"fmt"

	"github.com/thinktwice/agentForge/src/agents"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

type AgentBuilder struct {
	name               string
	systemPrompt       string
	tools              []Tool
	Subagents          map[LLM]Subagent
	plugins            []Plugin
	llmEngine          llms.LLMEngine
	vectorDB           core.VectorDB
	embeddingGenerator core.EmbeddingGenerator
	workingDir         string
	persistence        string
}

func NewAgentBuilder(name string, persistence string) *AgentBuilder {
	if persistence != "" && persistence != "json" {
		panic("invalid persistence type: " + persistence)
	}
	return &AgentBuilder{
		name:        name,
		persistence: persistence,
	}
}

func (b *AgentBuilder) SetSystemPrompt(prompt string) *AgentBuilder {
	b.systemPrompt = prompt
	return b
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
		b.Subagents = make(map[LLM]Subagent)
	}
	b.Subagents[llm] = subagent
	return b
}

func (b *AgentBuilder) AddPlugin(plugin Plugin) *AgentBuilder {
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

func (b *AgentBuilder) buildSubagents() ([]*core.SubAgent, error) {
	subagents := []*core.SubAgent{}
	for llm, subagent := range b.Subagents {
		llmEngine, err := b.createLLMEngine(llm)
		if err != nil {
			return nil, err
		}
		subagent, err := subagent.getSubagent(llmEngine, b.vectorDB, b.embeddingGenerator, b.workingDir)
		if err != nil {
			return nil, err
		}
		subagents = append(subagents, subagent)
	}
	return subagents, nil
}

func (b *AgentBuilder) buildTools() ([]llms.Tool, error) {
	tools := []llms.Tool{}
	for _, tool := range b.tools {
		tool, err := tool.getTool(b.workingDir, b.vectorDB, b.embeddingGenerator)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

func (b *AgentBuilder) buildPlugins() ([]core.Plugin, error) {
	plugins := []core.Plugin{}
	for _, plugin := range b.plugins {
		plugin, err := plugin.getPlugin()
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, plugin)
	}
	return plugins, nil
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
		LLMEngine:   b.llmEngine,
		AgentName:   b.name,
		Description: "Main Agent",
		Tone:        agents.ToneKeepItShort,
		Trace:       fmt.Sprintf("%s-trace", b.name),
		CanExpand:   true,
		MainAgent:   true,
		Persistence: b.persistence,
		SubAgents:   subagents,
		Tools:       tools,
		Plugins:     plugins,
	}

	agent := agents.NewAgent(agentConfig)

	return agent, nil
}
