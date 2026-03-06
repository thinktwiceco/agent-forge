package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type FSNodeInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	IsDir    bool   `json:"is_dir"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"`
}

func (s *Server) handleFSList(c *gin.Context) {
	relPath := c.Query("path")
	if relPath == "" {
		relPath = "."
	}

	agent := s.agentMgr.GetAgent()
	if agent == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Agent not initialized"})
		return
	}

	baseDir := s.configMgr.GetConfig().Agent.WorkingDir
	if baseDir == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Agent working directory not configured"})
		return
	}

	targetPath := filepath.Join(baseDir, relPath)

	// Prevent path traversal. A trailing separator is required so that a
	// directory like /base/work2 is not accepted as a child of /base/work.
	cleanBase := filepath.Clean(baseDir) + string(filepath.Separator)
	cleanTarget := filepath.Clean(targetPath)
	if cleanTarget != filepath.Clean(baseDir) && !strings.HasPrefix(cleanTarget+string(filepath.Separator), cleanBase) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var nodes []FSNodeInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // skip entries we can't stat
		}

		nodePath := filepath.Join(relPath, entry.Name())
		// Normalize for web (forward slashes)
		nodePath = filepath.ToSlash(nodePath)

		nodes = append(nodes, FSNodeInfo{
			Name:     entry.Name(),
			Path:     nodePath,
			IsDir:    entry.IsDir(),
			Size:     info.Size(),
			Modified: info.ModTime().UnixMilli(),
		})
	}

	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

func (s *Server) handleFSRead(c *gin.Context) {
	relPath := c.Query("path")
	if relPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path parameter is required"})
		return
	}

	agent := s.agentMgr.GetAgent()
	if agent == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Agent not initialized"})
		return
	}

	baseDir := s.configMgr.GetConfig().Agent.WorkingDir
	if baseDir == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Agent working directory not configured"})
		return
	}

	targetPath := filepath.Join(baseDir, relPath)

	// Prevent path traversal. A trailing separator is required so that a
	// directory like /base/work2 is not accepted as a child of /base/work.
	cleanBase := filepath.Clean(baseDir) + string(filepath.Separator)
	cleanTarget := filepath.Clean(targetPath)
	if cleanTarget != filepath.Clean(baseDir) && !strings.HasPrefix(cleanTarget+string(filepath.Separator), cleanBase) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is a directory, not a file"})
		return
	}

	file, err := os.Open(targetPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() { _ = file.Close() }()

	// Read first 512 bytes to detect content type
	header := make([]byte, 512)
	n, _ := file.Read(header)
	if _, err := file.Seek(0, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	contentType := http.DetectContentType(header[:n])
	c.Header("Content-Type", contentType)

	if _, err := io.Copy(c.Writer, file); err != nil {
		_ = c.Error(err)
	}
}
