package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// knownProviders defines the fixed set of provider API keys surfaced in the UI.
// Group is "llm" or "messaging" to allow the UI to render them in sections.
var knownProviders = []ProviderConfig{
	{EnvKey: "AF_OPENAI_API_KEY", Label: "OpenAI", Group: "llm"},
	{EnvKey: "AF_TOGETHERAI_API_KEY", Label: "TogetherAI", Group: "llm"},
	{EnvKey: "AF_DEEPSEEK_API_KEY", Label: "DeepSeek", Group: "llm"},
	{EnvKey: "AF_BRAVE_API_KEY", Label: "Brave Search", Group: "llm"},
	{EnvKey: "INSTAGRAM_ACCESS_TOKEN", Label: "Instagram", Group: "messaging"},
	{EnvKey: "TELEGRAM_BOT_TOKEN", Label: "Telegram", Group: "messaging"},
}

// maskToken returns a masked representation showing only the last 4 characters.
func maskToken(val string) string {
	if len(val) <= 4 {
		return strings.Repeat("*", len(val))
	}
	return "****" + val[len(val)-4:]
}

// handleGetProviders returns the known provider keys with masked current values.
func (s *Server) handleGetProviders(c *gin.Context) {
	result := make([]ProviderConfig, 0, len(knownProviders))
	for _, p := range knownProviders {
		val := os.Getenv(p.EnvKey)
		entry := ProviderConfig{
			EnvKey: p.EnvKey,
			Label:  p.Label,
			Group:  p.Group,
			IsSet:  val != "",
		}
		if val != "" {
			entry.MaskedValue = maskToken(val)
		}
		result = append(result, entry)
	}
	c.JSON(http.StatusOK, result)
}

// handleUpdateProviders writes updated provider tokens to .env and applies them
// to the current process immediately. The agent must be reloaded separately for
// the new keys to take effect in the LLM engines.
func (s *Server) handleUpdateProviders(c *gin.Context) {
	var req UpdateProvidersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate keys against the known set to prevent arbitrary env writes.
	allowed := make(map[string]bool, len(knownProviders))
	for _, p := range knownProviders {
		allowed[p.EnvKey] = true
	}
	for key := range req.Providers {
		if !allowed[key] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown provider key: %s", key)})
			return
		}
	}

	envPath := filepath.Join(s.appDir, ".env")
	if err := patchDotEnv(envPath, req.Providers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Apply to current process immediately so a reload picks up the new values.
	for key, val := range req.Providers {
		if val == "" {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, val)
		}
	}

	c.Status(http.StatusNoContent)
}

// patchDotEnv reads the .env file (creating it if absent), replaces or adds
// the given key=value pairs, and writes the file back. Existing lines for
// other keys are preserved verbatim.
func patchDotEnv(path string, updates map[string]string) error {
	existing := make(map[string]string)
	var order []string

	// Read existing file if present
	if data, err := os.ReadFile(path); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			existing[key] = parts[1]
			order = append(order, key)
		}
	}

	// Apply updates
	for key, val := range updates {
		if _, exists := existing[key]; !exists {
			order = append(order, key)
		}
		existing[key] = val
	}

	// Write back
	var sb strings.Builder
	seen := make(map[string]bool)
	for _, key := range order {
		if seen[key] {
			continue
		}
		seen[key] = true
		val := existing[key]
		if val == "" {
			// Write as commented-out placeholder to preserve the key name
			sb.WriteString(fmt.Sprintf("# %s=\n", key))
		} else {
			sb.WriteString(fmt.Sprintf("%s=%s\n", key, val))
		}
	}

	return os.WriteFile(path, []byte(sb.String()), 0600)
}
