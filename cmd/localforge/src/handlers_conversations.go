package main

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thinktwiceco/agent-forge/src/persistence"
)

func (s *Server) handleListConversations(c *gin.Context) {
	cfg := s.configMgr.GetConfig()
	if cfg.Agent.Persistence != "json" {
		c.JSON(http.StatusOK, []ConversationSummary{})
		return
	}

	baseDir := filepath.Join("data", "conversations", s.agentMgr.GetAgentName())
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, []ConversationSummary{})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read conversations"})
		return
	}

	type item struct {
		summary ConversationSummary
		modTime time.Time
	}

	var items []item
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		id := entry.Name()[:len(entry.Name())-len(".json")]
		items = append(items, item{
			summary: ConversationSummary{
				ID:        id,
				UpdatedAt: info.ModTime().UTC().Format(time.RFC3339),
			},
			modTime: info.ModTime(),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].modTime.After(items[j].modTime)
	})

	summaries := make([]ConversationSummary, 0, len(items))
	for _, it := range items {
		summaries = append(summaries, it.summary)
	}

	c.JSON(http.StatusOK, summaries)
}

func (s *Server) handleGetConversation(c *gin.Context) {
	chatID := c.Param("id")
	if chatID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation id is required"})
		return
	}

	cfg := s.configMgr.GetConfig()
	if cfg.Agent.Persistence != "json" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "persistence is disabled"})
		return
	}

	baseDir := filepath.Join("data", "conversations", s.agentMgr.GetAgentName())
	store := persistence.NewJSONPersistence(baseDir)
	history := store.GetHistory(chatID, 0, 0)
	c.JSON(http.StatusOK, history)
}

func (s *Server) handleDeleteConversation(c *gin.Context) {
	chatID := c.Param("id")
	if chatID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation id is required"})
		return
	}

	cfg := s.configMgr.GetConfig()
	if cfg.Agent.Persistence != "json" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "persistence is disabled"})
		return
	}

	baseDir := filepath.Join("data", "conversations", s.agentMgr.GetAgentName())
	path := filepath.Join(baseDir, chatID+".json")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete conversation"})
		return
	}

	c.Status(http.StatusNoContent)
}
