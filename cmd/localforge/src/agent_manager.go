package main

import (
	"fmt"
	"sync"

	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/builder"
	"github.com/thinktwiceco/agent-forge/src/core"
)

type AgentManager struct {
	mu          sync.RWMutex
	agent       *agents.Agent
	configMgr   *ConfigManager
	chunkRouter func(chatId string, chunk core.ExtendedChunkResponse)
}

func NewAgentManager(configMgr *ConfigManager) (*AgentManager, error) {
	manager := &AgentManager{
		configMgr: configMgr,
	}
	agent, err := manager.buildAgent()
	if err != nil {
		return nil, err
	}
	manager.agent = agent
	return manager, nil
}

func (am *AgentManager) GetAgent() *agents.Agent {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.agent
}

// SetChunkRouter sets the callback used to route background-drain chunks to the push registry.
// The router is preserved across Reload calls so it is applied to newly built agents.
func (am *AgentManager) SetChunkRouter(fn func(chatId string, chunk core.ExtendedChunkResponse)) {
	am.mu.Lock()
	am.chunkRouter = fn
	agent := am.agent
	am.mu.Unlock()
	if agent != nil {
		agent.SetChunkRouter(fn)
	}
}

func (am *AgentManager) GetAgentName() string {
	cfg := am.configMgr.GetConfig()
	if cfg.Agent.Name != "" {
		return cfg.Agent.Name
	}
	return "agent"
}

func (am *AgentManager) Reload() error {
	if err := am.configMgr.Load(); err != nil {
		return err
	}
	agent, err := am.buildAgent()
	if err != nil {
		return err
	}

	am.mu.Lock()
	am.agent = agent
	if am.chunkRouter != nil {
		agent.SetChunkRouter(am.chunkRouter)
	}
	am.mu.Unlock()
	return nil
}

func (am *AgentManager) buildAgent() (*agents.Agent, error) {
	// Use the already-loaded config with interpolated environment variables
	cfg := am.configMgr.GetConfig()

	agentBuilder, err := builder.NewAgentBuilderFromConfigStruct(cfg)
	if err != nil {
		return nil, fmt.Errorf("create agent builder: %w", err)
	}

	// Vector components are optional - only build if configured
	if cfg.VectorStorage != nil {
		vectorBuilder, err := builder.NewVectorBuilderFromConfigStruct(*cfg.VectorStorage)
		if err != nil {
			return nil, fmt.Errorf("create vector builder: %w", err)
		}

		// Only build and set vector components if they're configured
		if err := vectorBuilder.Build(); err == nil {
			agentBuilder.SetVectorDB(vectorBuilder.GetVectorDB())
			agentBuilder.SetEmbeddingGenerator(vectorBuilder.GetEmbeddingGenerator())
		}
		// Silently ignore vector build errors - vector DB is optional
	}

	agent, err := agentBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("build agent: %w", err)
	}

	return agent, nil
}
