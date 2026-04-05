package prompts

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed files/main/*.md
var promptsFS embed.FS

// LoadMainPrompt reads a main prompt file and returns its content.
// Name is the filename without .md (e.g., "default", "main-agent", "tone-keep-it-short").
func LoadMainPrompt(name string) (string, error) {
	path := "files/main/" + name + ".md"
	data, err := promptsFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("load main prompt %s: %w", name, err)
	}
	return strings.TrimSpace(string(data)), nil
}
