package builder

// ─── Application Configuration Layer ─────────────────────────────────────────
//
// This package owns the middle configuration layer: the agent's identity,
// behaviour, and capability set as declared in config.yaml.
//
// Responsibility split:
//
//	builder.Config        — direct YAML→Go mapping; source of truth on disk
//	builder.AgentFactory  — validates the config, resolves model strings into
//	                        LLMEngine instances, instantiates tools and plugins,
//	                        then produces an agents.AgentConfig ready for
//	                        agents.NewAgent().
//
// What belongs here (application concerns):
//   - Agent name, system prompt, model selection, working directory, persistence
//   - Tool list with per-tool settings (mode, DB URL, allowed tables…)
//   - Plugin list by name (resolved via registry at build time)
//   - Vector-storage config (optional)
//
// What does NOT belong here (agent-layer concerns):
//   - Plugin hook wiring, tool injection, system-prompt fragments from plugins
//     (all done inside agents.agentInit.go at EventAgentInitialization)
//   - Executor configuration (max iterations, tracer)
//   - Context window strategy, truncation, history management

import (
	"fmt"
	"os"

	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/plugins/brain"
	"github.com/thinktwiceco/agent-forge/src/plugins/heartbeat"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Agent struct {
		Name         string                     `yaml:"name"`
		SystemPrompt string                     `yaml:"system_prompt"`
		Model        string                     `yaml:"model"`
		WorkingDir   string                     `yaml:"working_dir"`
		Persistence  string                     `yaml:"persistence"`
		Tools        []Tool                     `yaml:"tools"`
		Plugins      []string                   `yaml:"plugins"`
		Heartbeat    *heartbeat.HeartbeatConfig `yaml:"heartbeat,omitempty"`
		// Brain controls whether the brain plugin loads. Omitting the field (nil)
		// or setting it to true enables brain. Set to false to opt out:
		//   agent:
		//     brain: false
		Brain *bool `yaml:"brain,omitempty"`
		// BrainPlugin configures scheduled dreaming (dreamTime cron). Only used when brain loads.
		// Example: brain_plugin: { dream: off, dreamTime: "03:00" }
		BrainPlugin *brain.PluginConfig `yaml:"brain_plugin,omitempty"`
		// SpawnSubagent enables the built-in spawn_subagent tool (ephemeral child agent).
		SpawnSubagent bool `yaml:"spawn_subagent,omitempty"`
	} `yaml:"agent"`
	VectorStorage *VectorStorageConfig `yaml:"vector-storage,omitempty"`
}

type AgentFactory struct {
	name               string
	systemPrompt       string
	tools              []Tool
	plugins            []string
	heartbeatYAML      *heartbeat.HeartbeatConfig
	llmEngine          llms.LLMEngine
	workingDir         string
	persistence        string
	vectorDB           core.VectorDB
	embeddingGenerator core.EmbeddingGenerator
	// brainDisabled suppresses the automatic brain plugin injection.
	// Set via brain: false in YAML; false by default (brain loads automatically).
	brainDisabled    bool
	brainPluginCfg   *brain.PluginConfig
	modelName        string
	canSpawnSubagent bool
}

// Public API methods

func NewAgentFactory(name string, persistence string) (*AgentFactory, error) {
	if persistence != "" && persistence != "json" {
		return nil, fmt.Errorf("invalid persistence type: %s", persistence)
	}
	return &AgentFactory{
		name:        name,
		persistence: persistence,
	}, nil
}

func NewAgentFactoryFromConfig(configPath string) (*AgentFactory, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return newAgentFactoryFromConfigStruct(cfg)
}

// NewAgentFactoryFromConfigStruct creates an agent builder from an already-loaded Config struct.
// This is useful when the config has been pre-processed (e.g., environment variable interpolation).
func NewAgentFactoryFromConfigStruct(cfg Config) (*AgentFactory, error) {
	return newAgentFactoryFromConfigStruct(cfg)
}

func newAgentFactoryFromConfigStruct(cfg Config) (*AgentFactory, error) {
	b, err := NewAgentFactory(cfg.Agent.Name, cfg.Agent.Persistence)
	if err != nil {
		return nil, err
	}
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
	for _, plugin := range cfg.Agent.Plugins {
		b.AddPlugin(plugin)
	}
	b.heartbeatYAML = cfg.Agent.Heartbeat
	// brain: false in YAML opts out of the automatic brain plugin injection.
	b.brainDisabled = cfg.Agent.Brain != nil && !*cfg.Agent.Brain
	b.brainPluginCfg = cfg.Agent.BrainPlugin
	b.canSpawnSubagent = cfg.Agent.SpawnSubagent

	return b, nil
}

func (b *AgentFactory) SetSystemPrompt(prompt string) *AgentFactory {
	b.systemPrompt = prompt
	return b
}

func (b *AgentFactory) GetName() string {
	return b.name
}

func (b *AgentFactory) AddTools(tools ...Tool) *AgentFactory {
	b.tools = append(b.tools, tools...)
	return b
}

func (b *AgentFactory) SetModel(model string) *AgentFactory {
	b.modelName = model
	return b
}

func (b *AgentFactory) AddPlugin(plugin string) *AgentFactory {
	b.plugins = append(b.plugins, plugin)
	return b
}

func (b *AgentFactory) SetVectorDB(vectorDB core.VectorDB) *AgentFactory {
	b.vectorDB = vectorDB
	return b
}

func (b *AgentFactory) SetEmbeddingGenerator(embeddingGenerator core.EmbeddingGenerator) *AgentFactory {
	b.embeddingGenerator = embeddingGenerator
	return b
}

func (b *AgentFactory) SetWorkingDir(workingDir string) *AgentFactory {
	b.workingDir = workingDir
	return b
}

func (b *AgentFactory) Build() (*agents.Agent, error) {
	if b.modelName != "" && b.llmEngine == nil {
		llm := fromString(b.modelName)
		llm.model() // Ensure it's valid
		llmEngine, err := b.createLLMEngine(llm)
		if err != nil {
			return nil, fmt.Errorf("failed to create llm engine: %w", err)
		}
		b.llmEngine = llmEngine
	}

	err := b.validate()
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
		LLMEngine:        b.llmEngine,
		AgentName:        b.name,
		Description:      "Main Agent",
		Tone:             agents.ToneKeepItShort,
		Trace:            fmt.Sprintf("%s-trace", b.name),
		CanExpand:        true,
		CanSpawnSubagent: b.canSpawnSubagent,
		MainAgent:        true,
		Persistence:      b.persistence,
		WorkingDir:       b.workingDir,
		Tools:            tools,
		Plugins:          plugins,
		SystemPrompt:     b.systemPrompt,
	}
	for _, p := range plugins {
		if p.Name() == "heartbeat" {
			hb := heartbeat.MergeConfig(b.heartbeatYAML)
			agentConfig.HeartbeatAckMaxChars = hb.AckMaxChars
			break
		}
	}

	agent := agents.NewAgent(agentConfig)

	return agent, nil
}

// Private helper methods

func (b *AgentFactory) validate() error {
	if b.name == "" {
		return fmt.Errorf("`name` is required to build an agent")
	}

	if b.llmEngine == nil {
		return fmt.Errorf("`llmEngine` is required to build an agent")
	}

	return nil
}

func (b *AgentFactory) createLLMEngine(l LLM) (llms.LLMEngine, error) {
	llmBuilder, err := llms.NewOpenAILLMBuilder(l.provider())
	if err != nil {
		return nil, err
	}
	llmBuilder.SetModel(l.model())
	mmEngine, err := llmBuilder.Build()
	if err != nil {
		return nil, err
	}
	return mmEngine.MainModel(), nil
}

func (b *AgentFactory) buildTools() ([]llms.Tool, error) {
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

func (b *AgentFactory) buildPlugins() ([]core.Plugin, error) {
	// Brain is a default plugin: it loads automatically unless disabled.
	// Build an effective name list: start with "brain" (if enabled), then
	// append user-configured plugins skipping any duplicate "brain" entry.
	//
	// This means agents do not need to list brain in their YAML config at all.
	// To disable: set `agent: brain: false` in config.
	effective := []string{}
	if !b.brainDisabled {
		effective = append(effective, "brain")
	}
	// Auto-activate heartbeat when the heartbeat section is configured, so
	// users don't need to list "heartbeat" explicitly under plugins:.
	heartbeatInPlugins := false
	for _, name := range b.plugins {
		if name == "heartbeat" {
			heartbeatInPlugins = true
			break
		}
	}
	if b.heartbeatYAML != nil && !heartbeatInPlugins {
		effective = append(effective, "heartbeat")
	}
	for _, name := range b.plugins {
		if name == "brain" {
			continue // already included by default; skip to avoid double-init
		}
		effective = append(effective, name)
	}

	plugins := []core.Plugin{}
	for _, plugin := range effective {
		if plugin == "heartbeat" {
			cfg := heartbeat.MergeConfig(b.heartbeatYAML)
			plugins = append(plugins, heartbeat.NewHeartbeatPlugin(cfg))
			continue
		}
		if plugin == "brain" {
			plugins = append(plugins, brain.NewBrainPluginWithConfig(b.workingDir, b.brainPluginCfg))
			continue
		}
		p, err := getPlugin(plugin, b.workingDir)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, p)
	}
	return plugins, nil
}
