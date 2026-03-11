package builder

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/tools/api"
	"github.com/thinktwiceco/agent-forge/src/tools/fs"
	"github.com/thinktwiceco/agent-forge/src/tools/git"
	"github.com/thinktwiceco/agent-forge/src/tools/instagram"
	"github.com/thinktwiceco/agent-forge/src/tools/postgres"
	"github.com/thinktwiceco/agent-forge/src/tools/update"
	"github.com/thinktwiceco/agent-forge/src/tools/web"
	"gopkg.in/yaml.v3"
)

// Tool represents a tool configuration entry in config.yaml
type Tool struct {
	Name string `yaml:"name"`
	// Postgres-specific configs
	PostgresURL    string   `yaml:"postgresURL,omitempty"`
	Mode           string   `yaml:"mode,omitempty"`
	AllowedTables  []string `yaml:"allowedTables,omitempty"`
	AllowedSchemas []string `yaml:"allowedSchemas,omitempty"`
	// API tool: path to folder containing one JSON file per service (relative to working_dir or absolute)
	ConfigFolder string `yaml:"config_folder,omitempty"`
}

// UnmarshalYAML implements custom unmarshaling to support both string and object formats
func (t *Tool) UnmarshalYAML(value *yaml.Node) error {
	var toolName string
	if err := value.Decode(&toolName); err == nil {
		t.Name = toolName
		return nil
	}
	type toolAlias Tool
	var tmp toolAlias
	if err := value.Decode(&tmp); err != nil {
		return err
	}
	*t = Tool(tmp)
	return nil
}

const (
	FILE_SYSTEM_TOOL = "fs"
	WEB_BROWSER_TOOL = "web"
	GIT_TOOL         = "git"
	POSTGRES_TOOL    = "postgres"
	API_TOOL         = "api"
	INSTAGRAM_TOOL   = "instagram"
	UPDATE_TOOL      = "update"
)

func (t *Tool) getTool(
	workingDir string,
	vectorDB core.VectorDB,
	embeddingGenerator core.EmbeddingGenerator,
) (llms.Tool, error) {
	switch t.Name {
	case FILE_SYSTEM_TOOL:
		if workingDir == "" {
			return nil, fmt.Errorf("working_dir is required for fs tool")
		}
		return fs.NewFsTool(workingDir), nil
	case WEB_BROWSER_TOOL:
		if workingDir == "" {
			return nil, fmt.Errorf("working_dir is required for web tool")
		}
		return web.NewWebTool(filepath.Join(workingDir, "web")), nil
	case GIT_TOOL:
		if workingDir == "" {
			return nil, fmt.Errorf("working_dir is required for git tool")
		}
		return git.NewGitTool(filepath.Join(workingDir, "repos")), nil
	case POSTGRES_TOOL:
		if t.PostgresURL == "" {
			return nil, fmt.Errorf("postgresURL is required for postgres tool")
		}
		if t.Mode == "" {
			return nil, fmt.Errorf("mode is required for postgres tool")
		}
		if t.Mode != "read" && t.Mode != "write" {
			return nil, fmt.Errorf("mode must be 'read' or 'write', got: %s", t.Mode)
		}
		if len(t.AllowedTables) == 0 {
			return nil, fmt.Errorf("allowedTables must contain at least one table")
		}
		return postgres.NewPostgresTool(
			t.PostgresURL,
			t.Mode,
			t.AllowedTables,
			t.AllowedSchemas,
		), nil
	case INSTAGRAM_TOOL:
		token := os.Getenv("INSTAGRAM_ACCESS_TOKEN")
		return instagram.NewInstagramTool(map[string]string{
			"Authorization": "Bearer " + token,
			"Content-Type":  "application/json",
		}), nil
	case API_TOOL:
		if t.ConfigFolder == "" {
			return nil, fmt.Errorf("config_folder is required for api tool (path to folder containing <service>.json files)")
		}
		folderPath := t.ConfigFolder
		if !filepath.IsAbs(folderPath) {
			folderPath = filepath.Join(workingDir, folderPath)
		}
		repositoryDir := filepath.Join(workingDir, "api_config")
		_ = os.MkdirAll(repositoryDir, 0755)

		services := make(map[string]api.ServiceConfig)
		for _, dir := range []string{folderPath, repositoryDir} {
			entries, err := os.ReadDir(dir)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, fmt.Errorf("failed to read api config folder %q: %w", dir, err)
			}
			for _, entry := range entries {
				if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
					continue
				}
				svcName := entry.Name()[:len(entry.Name())-len(".json")]
				data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
				if err != nil {
					return nil, fmt.Errorf("failed to read api config %q: %w", entry.Name(), err)
				}
				svc, err := api.ParseServiceConfig(data)
				if err != nil {
					return nil, fmt.Errorf("failed to parse api config %q: %w", entry.Name(), err)
				}
				services[svcName] = svc
			}
		}
		return api.NewApiTool("api", services, repositoryDir, workingDir), nil
	case UPDATE_TOOL:
		if workingDir == "" {
			return nil, fmt.Errorf("working_dir is required for update tool")
		}
		return update.NewUpdateTool(workingDir), nil
	}
	return nil, fmt.Errorf("invalid tool: %s", t.Name)
}
