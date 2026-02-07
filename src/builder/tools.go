package builder

import (
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
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
	// Common configs
	Root string `yaml:"root,omitempty"`
	// Postgres-specific configs
	PostgresURL    string   `yaml:"postgresURL,omitempty"`
	Mode           string   `yaml:"mode,omitempty"`
	AllowedTables  []string `yaml:"allowedTables,omitempty"`
	AllowedSchemas []string `yaml:"allowedSchemas,omitempty"`
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
)

func (t *Tool) getTool(
	workingDir string,
	vectorDB core.VectorDB,
	embeddingGenerator core.EmbeddingGenerator,
) (llms.Tool, error) {
	switch t.Name {
	case FILE_SYSTEM_TOOL:
		root := t.Root
		if root == "" {
			root = workingDir // fallback for backwards compatibility
		}
		if root == "" {
			return nil, fmt.Errorf("root directory is required for fs tool")
		}
		return fs.NewFsTool(root), nil
	case WEB_BROWSER_TOOL:
		root := t.Root
		if root == "" {
			root = workingDir // fallback for backwards compatibility
		}
		if root == "" {
			return nil, fmt.Errorf("root directory is required for web tool")
		}
		return web.NewWebTool(root), nil
	case VECTOR_DB_TOOL:
		if vectorDB == nil || embeddingGenerator == nil {
			return nil, fmt.Errorf("vectorDB and embeddingGenerator are required for vector DB tool")
		}
		return vector.NewVectorTool(vectorDB, embeddingGenerator), nil
	case GIT_TOOL:
		root := t.Root
		if root == "" {
			root = workingDir // fallback for backwards compatibility
		}
		if root == "" {
			return nil, fmt.Errorf("root directory is required for git tool")
		}
		return git.NewGitTool(root), nil
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
	}
	return nil, fmt.Errorf("invalid tool: %s", t.Name)
}
