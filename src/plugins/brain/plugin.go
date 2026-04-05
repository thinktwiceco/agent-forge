package brain

import (
	"context"
	"fmt"
	"time"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/plugins/registry"
)

const (
	PLUGIN_NAME = "brain"
)

// Name implements core.Plugin
func (p *BrainPlugin) Name() string {
	return PLUGIN_NAME
}

// Hooks implements core.HookProvider
func (p *BrainPlugin) Hooks() map[core.Event]core.AgentHookFn {
	return map[core.Event]core.AgentHookFn{
		// EventAgentInitialized: open the SQLite DB and seed the schema.
		core.EventAgentInitialized: agents.OnAgentInitializedHook(p.onInit),
		// EventToolExecution: append a memory reminder after non-brain tools run.
		core.EventToolExecution: agents.OnToolExecutionHook(p.onToolExecution),
		// EventContextBuild: periodic flush hint when conversation grows long.
		core.EventContextBuild: agents.OnContextBuildHook(p.onContextBuild),
	}
}

// brainToolNames is the set of tools provided by this plugin.
// The post-tool memory reminder is suppressed when one of these just ran
// (no need to remind the agent to save when it just used a memory tool).
var brainToolNames = map[string]bool{
	// Memory / recall + optional MEMORY.md (daily working notes)
	"get_conversation_topics":     true,
	"recall_recent_conversations": true,
	"recall_older_conversation":   true,
	"retrieve_conversation":       true,
	"memory_read_short_term":      true,
	"memory_patch_short_term":     true,
	"save_short_term_memory":      true,
	"forget":                      true,
	"dream":                       true,
	// Search
	"find": true,
}

// onToolExecution appends a reminder to tool responses to prompt brain retention,
// but suppresses the reminder when the brain is already being consulted.
func (p *BrainPlugin) onToolExecution(a *agents.Agent, toolResult *llms.ToolResult) error {
	if !toolResult.Success {
		return nil
	}
	if brainToolNames[toolResult.ToolName] {
		return nil
	}
	if toolResult.Result != "" {
		toolResult.Result += "\n\n"
	}
	toolResult.Result += "[Brain]: If something is worth remembering across sessions, use save_short_term_memory(topic, fact) — only for important facts; if unsure, ask the user."
	return nil
}

// onInit initializes the plugin when the agent is initialized
func (p *BrainPlugin) onInit(a *agents.Agent) error {
	agentforge.Debug("🧠 [Brain Plugin] Initializing...")

	if err := p.openDB(); err != nil {
		return fmt.Errorf("failed to open brain database: %w", err)
	}

	if err := p.ensureSchema(); err != nil {
		return fmt.Errorf("failed to ensure brain schema: %w", err)
	}

	p.querier = NewGraphQuerier(p.db)

	a.SetMemoryPrefixProvider(func() string {
		return p.readMemoryMDForInjection()
	})

	agentforge.Debug("🧠 [Brain Plugin] Initialized successfully at %s", p.dir)

	// Scheduled dreaming: daily at brain_plugin.dreamTime (see startScheduledDreaming). On-demand via dream tool.
	p.startScheduledDreaming()

	return nil
}

// SystemPrompt implements core.PromptProvider.
//
// Session long-term memory lives in brain.db as a topic -> conversation graph.
// Optional brain/MEMORY.md: quick daily working notes (easy read/write). Dreaming writes brain/persistence/...md for distilled transcripts and may rewrite MEMORY.md; optional promotion adds at most one graph node per run.
func (p *BrainPlugin) SystemPrompt() string {
	return `[MEMORY]
- brain/MEMORY.md is already injected as [SHORT_TERM_MEMORY]; use memory_read_short_term only to refresh it mid-turn.
- For recall, use get_conversation_topics to narrow scope, recall_recent_conversations or recall_older_conversation to list candidates, retrieve_conversation(id) for details, and find(query) for keyword search.
- Use save_short_term_memory(topic, fact) sparingly for one important daily fact. For broader MEMORY.md edits, read first and then use memory_patch_short_term.
- Use forget(scope, target) only when the user explicitly wants memory removed; it deletes the graph and on-disk persistence for that topic or conversation.
- Use dream to run transcript distillation and the MEMORY.md cleanup pass now when needed; long-term promotion from MEMORY.md is optional and at most one item per run.
- Use memory tools silently and present recalled facts naturally.
- Do not write to brain/persistence/, manually mutate the graph, or invent missing memory.

[INDEXED TOPICS]
- Long-term topic anchors are not listed here; call get_conversation_topics to retrieve them.`
}

const contextFlushThreshold = 40

// onContextBuild is called on each context build cycle and appends a flush hint
// when the turn count crosses the threshold.
func (p *BrainPlugin) onContextBuild(a *agents.Agent, agentContext *core.AgentContext) error {
	storage := agentContext.GetSessionStorage()
	turnsKey := "brain_turn_count"
	count, _ := storage[turnsKey].(int)
	count++
	storage[turnsKey] = count

	if count%contextFlushThreshold == 0 {
		a.AppendSystemPrompt(
			"\n[Brain]: Context is growing. " +
				"If important daily decisions should be persisted, use save_short_term_memory sparingly; for large MEMORY.md edits, memory_read_short_term then memory_patch_short_term.",
		)
	}
	return nil
}

// resolveNode resolves an identifier to a Node.
// An empty identifier resolves to the graph root (omnia-nunc-root).
// Otherwise tries exact ID match, then content/title LIKE search.
func (p *BrainPlugin) resolveNode(identifier string) (*Node, error) {
	if identifier == "" {
		return p.getNode(omniaNuncNodeID)
	}
	node, err := p.getNode(identifier)
	if err == nil {
		return node, nil
	}
	ctx := context.Background()
	pattern := "%" + identifier + "%"
	query := fmt.Sprintf(`
		SELECT %s
		FROM brain_nodes
		WHERE id = ? OR content LIKE ?
		  OR title LIKE ?
		  OR description LIKE ?
		  OR search_text LIKE ?
		ORDER BY created_at DESC
		LIMIT 1
	`, sqlBrainNodeColumns)
	nodes, err := p.queryNodes(ctx, query, identifier, pattern, pattern, pattern, pattern)
	if err != nil || len(nodes) == 0 {
		return nil, fmt.Errorf("node not found: %s", identifier)
	}
	return &nodes[0], nil
}

// OutNodes returns all outgoing neighbors of a node (edges FROM the node).
// An empty identifier starts from the graph root.
func (p *BrainPlugin) OutNodes(identifier string) (*NodeNeighborsResult, error) {
	if p.querier == nil {
		return nil, fmt.Errorf("brain plugin not fully initialized")
	}
	node, err := p.resolveNode(identifier)
	if err != nil {
		return nil, err
	}
	neighbors, err := p.querier.getOutNodesWithEdge(node.ID)
	if err != nil {
		return nil, err
	}
	if neighbors == nil {
		neighbors = []LightNodeWithEdge{}
	}
	return &NodeNeighborsResult{Node: toLightNode(*node), Neighbors: neighbors, Count: len(neighbors)}, nil
}

// InNodes returns all incoming neighbors of a node (edges TO the node).
func (p *BrainPlugin) InNodes(identifier string) (*NodeNeighborsResult, error) {
	if p.querier == nil {
		return nil, fmt.Errorf("brain plugin not fully initialized")
	}
	node, err := p.resolveNode(identifier)
	if err != nil {
		return nil, err
	}
	neighbors, err := p.querier.getInNodesWithEdge(node.ID)
	if err != nil {
		return nil, err
	}
	if neighbors == nil {
		neighbors = []LightNodeWithEdge{}
	}
	return &NodeNeighborsResult{Node: toLightNode(*node), Neighbors: neighbors, Count: len(neighbors)}, nil
}

// GetNodeContent returns the full Node for the given identifier.
func (p *BrainPlugin) GetNodeContent(identifier string) (*Node, error) {
	if p.querier == nil {
		return nil, fmt.Errorf("brain plugin not fully initialized")
	}
	return p.resolveNode(identifier)
}

// Find searches for nodes matching query and returns scored results
func (p *BrainPlugin) Find(query string, limit int) ([]ScoredNode, error) {
	return p.findScored(query, limit)
}

// AddNode creates a new node and attaches it to a parent via an edge.
func (p *BrainPlugin) AddNode(parentIdentifier, edgeType, nodeType, name, content string) (string, error) {
	return p.addNode(parentIdentifier, edgeType, nodeType, name, content)
}

// DeleteNode deletes a node and all its dependents (cascade delete).
func (p *BrainPlugin) DeleteNode(identifier string) (int, error) {
	return p.forgetCascade(identifier)
}

// SetLLMEngine implements core.LLMEngineAware.
// Called by the agent during EventAgentInitialization before any hooks run.
// The engine is used by DreamingRunner to make distillation calls.
func (p *BrainPlugin) SetLLMEngine(engine llms.LLMEngine) {
	p.llmEngine = engine
}

// startScheduledDreaming runs dreaming daily at brain_plugin.dreamTime (local).
// The dream tool calls runDreaming on demand regardless of brain_plugin.dream. Disabled when dream: off.
func (p *BrainPlugin) startScheduledDreaming() {
	if !p.cfg.DreamingEnabled() {
		agentforge.Debug("🧠 [Dreaming] Scheduled runs disabled (brain_plugin.dream: off); dream tool still works")
		return
	}
	if p.llmEngine == nil {
		agentforge.Debug("🧠 [Dreaming] LLM engine not available; scheduled dreaming disabled")
		return
	}

	go func() {
		ctx := context.Background()
		for {
			next := p.cfg.nextDreamRun(time.Now())
			d := time.Until(next)
			agentforge.Debug("🧠 [Dreaming] Next scheduled run at %s local (in %s)", next.Format(time.RFC3339), d.Round(time.Second))
			timer := time.NewTimer(d)
			<-timer.C
			agentforge.Debug("🧠 [Dreaming] Scheduled run (threshold %s)", DreamingThreshold)
			if err := p.runDreaming(ctx); err != nil {
				agentforge.Debug("🧠 [Dreaming] Run error: %v", err)
			}
		}
	}()
}

// init registers the plugin factory
func init() {
	registry.Register(PLUGIN_NAME, func(workingDir string) core.Plugin {
		return NewBrainPlugin(workingDir)
	})
}
