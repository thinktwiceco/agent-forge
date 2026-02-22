package builder

import (
	"fmt"
	"path/filepath"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/tools/api"
	"github.com/thinktwiceco/agent-forge/src/tools/fs"
	"github.com/thinktwiceco/agent-forge/src/tools/git"
	"github.com/thinktwiceco/agent-forge/src/tools/postgres"
	"github.com/thinktwiceco/agent-forge/src/tools/vector"
	"github.com/thinktwiceco/agent-forge/src/tools/web"
	"gopkg.in/yaml.v3"
)

// Tool represents a tool configuration with its initialization parameters
type Tool struct {
	Name string `yaml:"name"`
	// Postgres-specific configs
	PostgresURL    string   `yaml:"postgresURL,omitempty"`
	Mode           string   `yaml:"mode,omitempty"`
	AllowedTables  []string `yaml:"allowedTables,omitempty"`
	AllowedSchemas []string `yaml:"allowedSchemas,omitempty"`
	// API-specific configs
	Endpoints []struct {
		Name          string `yaml:"name"`
		URL           string `yaml:"url"`
		Method        string `yaml:"method"`
		Description   string `yaml:"description"`
		Payload       string `yaml:"payload,omitempty"`
		QueryParams   string `yaml:"queryParams,omitempty"`
		URLParameters string `yaml:"urlParameters,omitempty"`
		Validator     string `yaml:"validator,omitempty"` // Name of registered validator function
	} `yaml:"endpoints,omitempty"`
	OnApiCallHook string `yaml:"onApiCallHook,omitempty"` // Name of registered hook
}

// UnmarshalYAML implements custom unmarshaling to support both string and object formats
func (t *Tool) UnmarshalYAML(value *yaml.Node) error {
	// Try to unmarshal as string (backwards compatibility)
	var toolName string
	if err := value.Decode(&toolName); err == nil {
		t.Name = toolName
		return nil
	}

	// Unmarshal as object
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
	VECTOR_DB_TOOL   = "vector"
	GIT_TOOL         = "git"
	POSTGRES_TOOL    = "postgres"
	API_TOOL         = "api"
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
	case VECTOR_DB_TOOL:
		if vectorDB == nil || embeddingGenerator == nil {
			return nil, fmt.Errorf("vectorDB and embeddingGenerator are required for vector DB tool")
		}
		return vector.NewVectorTool(vectorDB, embeddingGenerator), nil
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
	case API_TOOL:
		if len(t.Endpoints) == 0 {
			return nil, fmt.Errorf("endpoints are required for api tool")
		}

		// Convert builder endpoints to api.Endpoint
		endpoints := make([]api.Endpoint, len(t.Endpoints))
		for i, e := range t.Endpoints {
			endpoint := api.Endpoint{
				Name:          e.Name,
				URL:           e.URL,
				Method:        e.Method,
				Description:   e.Description,
				Payload:       e.Payload,
				QueryParams:   e.QueryParams,
				URLParameters: e.URLParameters,
			}

			// Attach validator if specified
			if e.Validator != "" {
				validator := api.GetValidator(e.Validator)
				if validator == nil {
					return nil, fmt.Errorf("validator not found: %s for endpoint: %s", e.Validator, e.Name)
				}
				endpoint.Validate = validator
			}

			endpoints[i] = endpoint
		}

		// Get hook from registry or use default
		hook := api.GetHook(t.OnApiCallHook)

		return api.NewApiTool(t.Name, endpoints, hook), nil
	}
	return nil, fmt.Errorf("invalid tool: %s", t.Name)
}
