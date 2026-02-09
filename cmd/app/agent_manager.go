package main

import (
	"fmt"
	"sync"

	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/builder"
)

type AgentManager struct {
	mu        sync.RWMutex
	agent     *agents.Agent
	configMgr *ConfigManager
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
	am.mu.Unlock()
	return nil
}

func (am *AgentManager) buildAgent() (*agents.Agent, error) {
	configPath := am.configMgr.ConfigPath()

	agentBuilder, err := builder.NewAgentBuilderFromConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("create agent builder: %w", err)
	}

	// Vector components are optional - only build if configured
	vectorBuilder, err := builder.NewVectorBuilderFromConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("create vector builder: %w", err)
	}

	// Only build and set vector components if they're configured
	if err := vectorBuilder.Build(); err == nil {
		agentBuilder.SetVectorDB(vectorBuilder.GetVectorDB())
		agentBuilder.SetEmbeddingGenerator(vectorBuilder.GetEmbeddingGenerator())
	}
	// Silently ignore vector build errors - vector DB is optional

	agent, err := agentBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("build agent: %w", err)
	}

	return agent, nil
}
