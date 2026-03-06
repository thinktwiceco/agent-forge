package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleGetConfig returns the full agent configuration including subagents and plugins.
func (s *Server) handleGetConfig(c *gin.Context) {
	cfg := s.configMgr.GetConfig()

	tools := make([]ToolConfigResponse, 0, len(cfg.Agent.Tools))
	for _, tool := range cfg.Agent.Tools {
		tools = append(tools, ToolConfigResponse{
			Name:           tool.Name,
			PostgresURL:    tool.PostgresURL,
			Mode:           tool.Mode,
			AllowedTables:  tool.AllowedTables,
			AllowedSchemas: tool.AllowedSchemas,
		})
	}

	subagents := make(map[string]string, len(cfg.Agent.Subagents))
	for role, model := range cfg.Agent.Subagents {
		subagents[string(role)] = string(model)
	}

	plugins := cfg.Agent.Plugins
	if plugins == nil {
		plugins = []string{}
	}

	response := AgentConfigResponse{
		Name:         cfg.Agent.Name,
		Model:        cfg.Agent.Model,
		SystemPrompt: cfg.Agent.SystemPrompt,
		WorkingDir:   cfg.Agent.WorkingDir,
		Persistence:  cfg.Agent.Persistence,
		Tools:        tools,
		Subagents:    subagents,
		Plugins:      plugins,
	}
	c.JSON(http.StatusOK, response)
}

// handleUpdateAgentConfig updates top-level agent identity fields and saves to disk.
// Only provided (non-nil) fields are patched; config.yaml is updated via yaml.Node
// so ${VAR} references in other fields are preserved.
func (s *Server) handleUpdateAgentConfig(c *gin.Context) {
	var req UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.Name == nil && req.Model == nil && req.SystemPrompt == nil &&
		req.WorkingDir == nil && req.Persistence == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	if err := s.configMgr.UpdateAgentFields(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.handleGetConfig(c)
}

// handleUpdatePlugins replaces the full plugin list in config.yaml.
func (s *Server) handleUpdatePlugins(c *gin.Context) {
	var req UpdatePluginsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := s.configMgr.UpdatePlugins(req.Plugins); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// handleUpdateSubagents replaces the full subagents map in config.yaml.
func (s *Server) handleUpdateSubagents(c *gin.Context) {
	var req UpdateSubagentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := s.configMgr.UpdateSubagents(req.Subagents); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// handleUpdateToolConfig updates settings for a single named tool.
func (s *Server) handleUpdateToolConfig(c *gin.Context) {
	toolName := c.Param("toolName")
	if toolName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tool name is required"})
		return
	}

	var req UpdateToolConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := s.configMgr.UpdateToolConfig(toolName, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// handleReload rebuilds the agent from the current saved config.
// In-flight requests complete on the old agent; only subsequent requests use the new one.
func (s *Server) handleReload(c *gin.Context) {
	if err := s.agentMgr.Reload(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
