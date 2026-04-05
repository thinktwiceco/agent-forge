package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleGetConfig returns the full agent configuration including plugins.
func (s *Server) handleGetConfig(c *gin.Context) {
	cfg := s.configMgr.GetConfig()

	tools := make([]ToolConfigResponse, 0, len(cfg.Agent.Tools))
	for _, tool := range cfg.Agent.Tools {
		var postgresURL, mode string
		if v, ok := tool.Params["postgresURL"].(string); ok {
			postgresURL = v
		}
		if v, ok := tool.Params["mode"].(string); ok {
			mode = v
		}

		var allowedTables, allowedSchemas []string
		if tables, ok := tool.Params["allowedTables"].([]any); ok {
			for _, table := range tables {
				if s, ok := table.(string); ok {
					allowedTables = append(allowedTables, s)
				}
			}
		}
		if schemas, ok := tool.Params["allowedSchemas"].([]any); ok {
			for _, schema := range schemas {
				if s, ok := schema.(string); ok {
					allowedSchemas = append(allowedSchemas, s)
				}
			}
		}

		var headless *bool
		if h, ok := tool.Params["headless"].(bool); ok {
			headless = &h
		}

		tools = append(tools, ToolConfigResponse{
			Name:           tool.Name,
			PostgresURL:    postgresURL,
			Mode:           mode,
			AllowedTables:  allowedTables,
			AllowedSchemas: allowedSchemas,
			Headless:       headless,
		})
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
