package main

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thinktwiceco/agent-forge/src/persistence"
)

func (s *Server) conversationsBaseDir() string {
	cfg := s.configMgr.GetConfig()
	agentName := s.agentMgr.GetAgentName()
	if cfg.Agent.WorkingDir != "" {
		return filepath.Join(cfg.Agent.WorkingDir, "data", "conversations", agentName)
	}
	return filepath.Join("data", "conversations", agentName)
}

func (s *Server) handleListConversations(c *gin.Context) {
	cfg := s.configMgr.GetConfig()
	if cfg.Agent.Persistence != "json" {
		c.JSON(http.StatusOK, []ConversationSummary{})
		return
	}

	baseDir := s.conversationsBaseDir()
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, []ConversationSummary{})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read conversations"})
		return
	}

	titles := persistence.NewJSONConversationMetadata(baseDir).LoadTitlesFromEntries(entries)

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
		id := strings.TrimSuffix(entry.Name(), ".json")
		items = append(items, item{
			summary: ConversationSummary{
				ID:        id,
				Title:     titles[id],
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

	limit := 100
	offset := 0
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}
		limit = parsed
	}
	if raw := c.Query("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
			return
		}
		offset = parsed
	}

	baseDir := s.conversationsBaseDir()
	store := persistence.NewJSONPersistence(baseDir)
	page, total, hasMore := store.GetTailHistory(chatID, limit, offset)

	c.JSON(http.StatusOK, ConversationHistoryResponse{
		Messages: page,
		Total:    total,
		HasMore:  hasMore,
	})
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

	baseDir := s.conversationsBaseDir()
	path := filepath.Join(baseDir, chatID+".json")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete conversation"})
		return
	}

	// Clean up title sidecar; ignore missing-file errors
	_ = persistence.NewJSONConversationMetadata(baseDir).DeleteTitle(chatID)

	c.Status(http.StatusNoContent)
}

func (s *Server) handleRenameConversation(c *gin.Context) {
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

	var req RenameChatRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Title) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}

	baseDir := s.conversationsBaseDir()

	// Verify the conversation exists before setting a title
	if _, err := os.Stat(filepath.Join(baseDir, chatID+".json")); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}

	if err := persistence.NewJSONConversationMetadata(baseDir).SetTitle(chatID, req.Title); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save title"})
		return
	}

	c.Status(http.StatusNoContent)
}
