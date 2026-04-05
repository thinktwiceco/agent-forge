// ─── Agent Lifecycle Manager ──────────────────────────────────────────────────
//
// AgentManager owns the runtime *agents.Agent and is responsible for building
// and hot-swapping it when the configuration changes.
//
// Reload safety:
//   - GetAgent() acquires a read-lock and returns a pointer to the current agent.
//     Callers (e.g. chat handlers) hold this pointer for the duration of a request.
//   - Reload() builds a brand-new agent, then acquires a write-lock only to swap
//     the pointer. Any request that already called GetAgent() before the swap
//     continues to completion on the old agent — there is no disruption.
//   - The chunkRouter callback is re-applied to the new agent so background-drain
//     push events are routed correctly after a reload.
//
// Reload is triggered explicitly via POST /api/agent/reload. Config mutations
// (PUT /config, PUT /config/providers, etc.) write to disk but do NOT auto-reload,
// giving the operator the chance to batch multiple changes before restarting.

package main

import (
	"fmt"
	"sync"

	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/builder"
	"github.com/thinktwiceco/agent-forge/src/core"
)

type AgentManager struct {
	mu                 sync.RWMutex
	agent              *agents.Agent
	configMgr          *ConfigManager
	chunkRouter        func(chatId string, chunk core.ExtendedChunkResponse)
	turnCompleteRouter func(chatId string, fullContent string)
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

// SetTurnCompleteRouter sets the callback invoked after each background drain turn completes.
// The router is preserved across Reload calls so it is applied to newly built agents.
func (am *AgentManager) SetTurnCompleteRouter(fn func(chatId string, fullContent string)) {
	am.mu.Lock()
	am.turnCompleteRouter = fn
	agent := am.agent
	am.mu.Unlock()
	if agent != nil {
		agent.SetTurnCompleteRouter(fn)
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
	if am.turnCompleteRouter != nil {
		agent.SetTurnCompleteRouter(am.turnCompleteRouter)
	}
	am.mu.Unlock()
	return nil
}

func (am *AgentManager) buildAgent() (*agents.Agent, error) {
	// Use the already-loaded config with interpolated environment variables
	cfg := am.configMgr.GetConfig()

	agentFactory, err := builder.NewAgentFactoryFromConfigStruct(cfg)
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
			agentFactory.SetVectorDB(vectorBuilder.GetVectorDB())
			agentFactory.SetEmbeddingGenerator(vectorBuilder.GetEmbeddingGenerator())
		}
		// Silently ignore vector build errors - vector DB is optional
	}

	agent, err := agentFactory.Build()
	if err != nil {
		return nil, fmt.Errorf("build agent: %w", err)
	}

	return agent, nil
}
