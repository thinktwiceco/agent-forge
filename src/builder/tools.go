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
	"github.com/thinktwiceco/agent-forge/src/tools/telegram"
	"github.com/thinktwiceco/agent-forge/src/tools/update"
	"github.com/thinktwiceco/agent-forge/src/tools/web"
	"gopkg.in/yaml.v3"
)

// Tool represents a tool configuration entry in config.yaml
type Tool struct {
	Name   string
	Params map[string]any
}

// UnmarshalYAML implements custom unmarshaling to support both string and object formats
func (t *Tool) UnmarshalYAML(value *yaml.Node) error {
	var toolName string
	if err := value.Decode(&toolName); err == nil {
		t.Name = toolName
		return nil
	}

	var raw map[string]any
	if err := value.Decode(&raw); err != nil {
		return err
	}

	nameObj, ok := raw["name"]
	if !ok {
		return fmt.Errorf("tool definition missing 'name' field")
	}

	nameStr, ok := nameObj.(string)
	if !ok {
		return fmt.Errorf("tool 'name' must be a string")
	}

	t.Name = nameStr
	delete(raw, "name")

	if len(raw) > 0 {
		t.Params = raw
	}

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
	TELEGRAM_TOOL    = "telegram"
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
		var headless *bool
		if h, ok := t.Params["headless"].(bool); ok {
			headless = &h
		}
		return web.NewWebTool(filepath.Join(workingDir, "web"), headless), nil
	case GIT_TOOL:
		if workingDir == "" {
			return nil, fmt.Errorf("working_dir is required for git tool")
		}
		return git.NewGitTool(filepath.Join(workingDir, "repos")), nil
	case POSTGRES_TOOL:
		postgresURL, _ := t.Params["postgresURL"].(string)
		mode, _ := t.Params["mode"].(string)

		var allowedTables, allowedSchemas []string
		if tables, ok := t.Params["allowedTables"].([]any); ok {
			for _, table := range tables {
				if s, ok := table.(string); ok {
					allowedTables = append(allowedTables, s)
				}
			}
		}
		if schemas, ok := t.Params["allowedSchemas"].([]any); ok {
			for _, schema := range schemas {
				if s, ok := schema.(string); ok {
					allowedSchemas = append(allowedSchemas, s)
				}
			}
		}

		if postgresURL == "" {
			return nil, fmt.Errorf("postgresURL is required for postgres tool")
		}
		if mode == "" {
			return nil, fmt.Errorf("mode is required for postgres tool")
		}
		if mode != "read" && mode != "write" {
			return nil, fmt.Errorf("mode must be 'read' or 'write', got: %s", mode)
		}
		if len(allowedTables) == 0 {
			return nil, fmt.Errorf("allowedTables must contain at least one table")
		}
		return postgres.NewPostgresTool(postgresURL, mode, allowedTables, allowedSchemas), nil
	case INSTAGRAM_TOOL:
		token := os.Getenv("INSTAGRAM_ACCESS_TOKEN")
		return instagram.NewInstagramTool(map[string]string{
			"Authorization": "Bearer " + token,
			"Content-Type":  "application/json",
		}), nil
	case API_TOOL:
		configFolder, _ := t.Params["config_folder"].(string)
		if configFolder == "" {
			return nil, fmt.Errorf("config_folder is required for api tool (path to folder containing <service>.json files)")
		}
		folderPath := configFolder
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
	case TELEGRAM_TOOL:
		port, _ := t.Params["port"].(string)
		return telegram.NewTelegramTool(port), nil
	}
	return nil, fmt.Errorf("invalid tool: %s", t.Name)
}
