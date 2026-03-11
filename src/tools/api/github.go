package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	githubRepo    = "thinktwiceco/agent-forge"
	githubBaseURL = "https://api.github.com"
)

type ghContentsEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type"` // "file" or "dir"
	SHA      string `json:"sha"`
	Content  string `json:"content"`  // base64, only when fetching a single file
	Encoding string `json:"encoding"` // "base64"
}

func githubGet(path string, out any) error {
	url := fmt.Sprintf("%s/%s", githubBaseURL, path)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	return json.Unmarshal(body, out)
}

// listRemoteApiConfigs returns the names (without .json extension) of all
// service config files available in the remote repository/api_configs/ folder.
func listRemoteApiConfigs() ([]string, error) {
	var entries []ghContentsEntry
	path := fmt.Sprintf("repos/%s/contents/repository/api_configs", githubRepo)
	if err := githubGet(path, &entries); err != nil {
		return nil, fmt.Errorf("failed to list remote api configs: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.Type != "file" || !strings.HasSuffix(e.Name, ".json") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name, ".json"))
	}
	return names, nil
}

// fetchAndInstallApiConfig downloads a single service config JSON from the
// remote repository/api_configs/<name>.json, writes it to destDir, parses it,
// and returns the resulting ServiceConfig ready to hot-load.
func fetchAndInstallApiConfig(name, destDir string) (ServiceConfig, error) {
	var entry ghContentsEntry
	path := fmt.Sprintf("repos/%s/contents/repository/api_configs/%s.json", githubRepo, name)
	if err := githubGet(path, &entry); err != nil {
		return ServiceConfig{}, fmt.Errorf("failed to fetch api config %q: %w", name, err)
	}

	data, err := decodeGHContent(entry.Content, entry.Encoding)
	if err != nil {
		return ServiceConfig{}, fmt.Errorf("decode api config %q: %w", name, err)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return ServiceConfig{}, fmt.Errorf("mkdir %s: %w", destDir, err)
	}

	destPath := filepath.Join(destDir, name+".json")
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return ServiceConfig{}, fmt.Errorf("write %s: %w", destPath, err)
	}

	svc, err := ParseServiceConfig(data)
	if err != nil {
		return ServiceConfig{}, fmt.Errorf("parse api config %q: %w", name, err)
	}
	return svc, nil
}

func decodeGHContent(content, encoding string) ([]byte, error) {
	if encoding != "base64" {
		return []byte(content), nil
	}
	clean := strings.ReplaceAll(content, "\n", "")
	return base64.StdEncoding.DecodeString(clean)
}
