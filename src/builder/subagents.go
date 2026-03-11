package builder

import (
	"fmt"
	"strings"

	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/agents/system"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/plugins/knowledge"
	"github.com/thinktwiceco/agent-forge/src/tools/fs"
)

type Subagent string

const (
	GIT_AGENT       Subagent = "git"
	REASONING_AGENT Subagent = "reasoning"
	CODING_AGENT    Subagent = "coding"
	OS_AGENT        Subagent = "os"
	WEB_AGENT       Subagent = "web"
	VISION_AGENT    Subagent = "vision"
	KNOWLEDGE_AGENT Subagent = "knowledge"
)

func (s Subagent) getSubagent(
	llmEngine llms.LLMEngine,
	vectorDB core.VectorDB,
	embeddingGenerator core.EmbeddingGenerator,
	workingDir string,
	extraTools ...llms.Tool,
) (core.SubAgent, error) {
	switch s {
	case GIT_AGENT:
		return agents.GitAgent(llmEngine, workingDir), nil
	case REASONING_AGENT:
		return agents.ReasoningAgent(llmEngine), nil
	case CODING_AGENT:
		return agents.CodingAgent(llmEngine, workingDir), nil
	case OS_AGENT:
		return agents.OsAgent(llmEngine, workingDir), nil
	case WEB_AGENT:
		return agents.WebAgent(llmEngine, workingDir, extraTools...), nil
	case VISION_AGENT:
		return agents.VisionAgent(llmEngine, workingDir), nil
	case KNOWLEDGE_AGENT:
		return newKnowledgeAgent(llmEngine, workingDir, vectorDB, embeddingGenerator), nil
	}
	return nil, fmt.Errorf("invalid subagent: %s", s)
}

// knowledgeDetailsAbout returns per-operation details for the knowledge subagent.
func knowledgeDetailsAbout(item string) string {
	switch strings.ToLower(strings.TrimSpace(item)) {
	case "remember", "add_node":
		return `Store a fact or node. Use add_node(parent, edge, type, name, content).
For facts: edge=has_fact, type=Fact. Traverse with out_nodes first to find category.
Common edges: has_category, has_subcategory, has_fact, has_document.`
	case "find":
		return `Search nodes by query. Params: query (required), limit (optional, default 10).
Returns matching nodes with id, type, title. Use for semantic search across the graph.`
	case "out_nodes":
		return `List outgoing neighbors of a node. Pass node="" for root — discovers top-level categories.
Returns id, type, title, edge_type for each neighbor. Start here to traverse the graph.`
	case "in_nodes":
		return `List incoming neighbors (nodes that point TO this node).
Params: node (required). Use to find parents and cross-references.`
	case "get_node_content":
		return `Retrieve full content and metadata of a node by id or name.
Params: node (required). Returns full body for Fact, Category, Subcategory, or Document.`
	case "link_relevant":
		return `Link two nodes with is_relevant_to edge. Params: node_a, node_b (required).
Use for cross-referencing related facts across categories.`
	case "delete_node", "forget":
		return `Remove a node from the graph. Params: node (required).
Use delete_node to remove any node; "forget" is the conceptual action for Facts.`
	default:
		return fmt.Sprintf("Nothing to add about %s", item)
	}
}

// newKnowledgeAgent constructs a knowledge subagent. Defined here (not in the agents
// package) to avoid an import cycle: knowledge/plugin.go imports agents, so agents
// cannot import knowledge.
func newKnowledgeAgent(
	llmEngine llms.LLMEngine,
	workingDir string,
	vectorDB core.VectorDB,
	embeddingGen core.EmbeddingGenerator,
) core.SubAgent {
	template := system.CreateKnowledgeAgentTemplate()
	config := agents.AgentConfig{
		LLMEngine:          llmEngine,
		AgentName:          template.Name,
		Trace:              template.Trace,
		SystemPrompt:       template.SystemPrompt(),
		Description:        template.Description(),
		AdvanceDescription: template.AdvanceDescription(),
		Troubleshooting:    template.Troubleshooting(),
		Tone:               system.ToneSystemAgent,
		WorkingDir:         workingDir,
		Tools:              []llms.Tool{fs.NewFsTool(workingDir)},
		Plugins:            []core.Plugin{knowledge.NewKnowledgePlugin(workingDir)},
		DetailsAboutFunc:   knowledgeDetailsAbout,
	}
	agent := agents.NewAgent(&config)
	return agent.AgentAsSubAgent()
}
