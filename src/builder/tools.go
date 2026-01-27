package builder

import (
	"fmt"

	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/tools/fs"
	"github.com/thinktwice/agentForge/src/tools/git"
	"github.com/thinktwice/agentForge/src/tools/vector"
	"github.com/thinktwice/agentForge/src/tools/web"
)

type Tool string

const (
	FILE_SYSTEM_TOOL Tool = "fs"
	WEB_BROWSER_TOOL Tool = "web"
	VECTOR_DB_TOOL   Tool = "vector"
	GIT_TOOL         Tool = "git"
)

func (t Tool) getTool(
	workingDir string,
	vectorDB core.VectorDB,
	embeddingGenerator core.EmbeddingGenerator,
) (llms.Tool, error) {
	switch t {
	case FILE_SYSTEM_TOOL:
		return fs.NewFsTool(workingDir), nil
	case WEB_BROWSER_TOOL:
		return web.NewWebTool(workingDir), nil
	case VECTOR_DB_TOOL:
		if vectorDB == nil || embeddingGenerator == nil {
			return nil, fmt.Errorf("vectorDB and embeddingGenerator are required for vector DB tool")
		}
		return vector.NewVectorTool(vectorDB, embeddingGenerator), nil
	case GIT_TOOL:
		return git.NewGitTool(workingDir), nil
	}
	return nil, fmt.Errorf("invalid tool: %s", t)
}
