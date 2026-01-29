package builder

import (
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

type Subagent string

const (
	GIT_AGENT       Subagent = "git"
	REASONING_AGENT Subagent = "reasoning"
	OS_AGENT        Subagent = "os"
	WEB_AGENT       Subagent = "web"
	VECTOR_DB_AGENT Subagent = "vector"
)

func (s Subagent) getSubagent(
	llmEngine llms.LLMEngine,
	vectorDB core.VectorDB,
	embeddingGenerator core.EmbeddingGenerator,
	workingDir string,
) (core.SubAgent, error) {
	switch s {
	case GIT_AGENT:
		return agents.GitAgent(llmEngine, workingDir), nil
	case REASONING_AGENT:
		return agents.ReasoningAgent(llmEngine), nil
	case OS_AGENT:
		return agents.OsAgent(llmEngine, workingDir), nil
	case WEB_AGENT:
		return agents.WebAgent(llmEngine, workingDir), nil
	case VECTOR_DB_AGENT:
		if vectorDB == nil || embeddingGenerator == nil {
			return nil, fmt.Errorf("vectorDB and embeddingGenerator are required for vector DB subagent")
		}
		return agents.VectorAgent(llmEngine, vectorDB, embeddingGenerator), nil
	}
	return nil, fmt.Errorf("invalid subagent: %s", s)
}
