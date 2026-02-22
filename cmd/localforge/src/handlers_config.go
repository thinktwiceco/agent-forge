package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleGetConfig(c *gin.Context) {
	cfg := s.configMgr.GetConfig()
	tools := make([]ToolConfigResponse, 0, len(cfg.Agent.Tools))
	for _, tool := range cfg.Agent.Tools {
		tools = append(tools, ToolConfigResponse{
			Name:           tool.Name,
			Root:           tool.Root,
			PostgresURL:    tool.PostgresURL,
			Mode:           tool.Mode,
			AllowedTables:  tool.AllowedTables,
			AllowedSchemas: tool.AllowedSchemas,
		})
	}
	response := AgentConfigResponse{
		Name:         cfg.Agent.Name,
		Model:        cfg.Agent.Model,
		SystemPrompt: cfg.Agent.SystemPrompt,
		WorkingDir:   cfg.Agent.WorkingDir,
		Persistence:  cfg.Agent.Persistence,
		Tools:        tools,
	}
	c.JSON(http.StatusOK, response)
}

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

func (s *Server) handleReload(c *gin.Context) {
	if err := s.agentMgr.Reload(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
