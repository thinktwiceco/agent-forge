package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

var nonAlphanumRe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = nonAlphanumRe.ReplaceAllString(name, "_")
	if name == "" || name == "." {
		name = "file"
	}
	return name
}

func randomHex(n int) string {
	const letters = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (s *Server) uploadDir() string {
	cfg := s.configMgr.GetConfig()
	base := cfg.Agent.WorkingDir
	return filepath.Join(base, "uploaded")
}

func (s *Server) handleUpload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	uploadDir := s.uploadDir()
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create upload directory"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	base := strings.TrimSuffix(sanitizeFilename(file.Filename), filepath.Ext(file.Filename))
	filename := fmt.Sprintf("%s_%s%s", base, randomHex(8), ext)
	dst := filepath.Join(uploadDir, filename)

	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"path": filepath.Join("uploaded", filename)})
}
