package knowledge

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/plugins/registry"
)

const (
	PLUGIN_NAME = "knowledge"
)

// KnowledgePlugin provides a knowledge graph for storing and retrieving user information
type KnowledgePlugin struct {
	db      *sql.DB
	dir     string
	querier *GraphQuerier
}

// NewKnowledgePlugin creates a new knowledge graph plugin
func NewKnowledgePlugin(workingDir string) *KnowledgePlugin {
	dir := filepath.Join(workingDir, "knowledge")
	return &KnowledgePlugin{dir: dir}
}

// Name implements core.Plugin
func (p *KnowledgePlugin) Name() string {
	return PLUGIN_NAME
}

// Hooks implements core.HookProvider
func (p *KnowledgePlugin) Hooks() map[core.Event]core.AgentHookFn {
	return map[core.Event]core.AgentHookFn{
		core.EventAgentInitialized: agents.OnAgentInitializedHook(p.onInit),
		core.EventToolExecution:    agents.OnToolExecutionHook(p.onToolExecution),
	}
}

// onToolExecution appends a reminder to tool responses to prompt knowledge retention
func (p *KnowledgePlugin) onToolExecution(a *agents.Agent, toolResult *llms.ToolResult) error {
	// Skip if tool execution failed
	if !toolResult.Success {
		return nil
	}

	// Append reminder to the tool result
	if toolResult.Result != "" {
		toolResult.Result += "\n\n"
	}
	toolResult.Result += "[Knowledge Reminder]: is this worth saving or just transactional? If you are not sure, ask your human!"

	return nil
}

// onInit initializes the plugin when the agent is initialized
func (p *KnowledgePlugin) onInit(a *agents.Agent) error {
	agentforge.Debug("🧠 [Knowledge Plugin] Initializing...")

	// Open database connection
	if err := p.openDB(); err != nil {
		return fmt.Errorf("failed to open knowledge database: %w", err)
	}

	// Ensure schema exists
	if err := p.ensureSchema(); err != nil {
		return fmt.Errorf("failed to ensure knowledge schema: %w", err)
	}

	p.querier = NewGraphQuerier(p.db)

	// Now that DB is initialized, append the categories section to system prompt
	categoriesSection := p.buildCategoriesSection()
	if categoriesSection != "" {
		a.AppendSystemPrompt(categoriesSection)
	}

	agentforge.Debug("🧠 [Knowledge Plugin] Initialized successfully at %s", p.dir)
	return nil
}

// SystemPrompt implements core.PromptProvider
func (p *KnowledgePlugin) SystemPrompt() string {
	return `[KNOWLEDGE GRAPH SYSTEM]
Type: Hierarchical Graph
Purpose: Proactive persistent memory

[EXTEND PERSONALITY]
As an assistant, you are curious about the user.
Make sure to save information that is relevant to the user.

[SCHEMA]
Nodes:
  CATEGORY: Top-level. field:name=content field:title=optional
  SUBCATEGORY: Recursive child. field:name=content field:title=optional
  FACT: Knowledge unit. field:body=content field:title=short_id
  DOCUMENT: File ref. field:path=content field:name=title

Edges:
  has_category:    root -> Category
  has_subcategory: [Category|Subcategory] -> Subcategory
  has_fact:        [Category|Subcategory] -> Fact
  has_document:    [Category|Fact] -> Document
  is_relevant_to:  [Any] bidirectional [Any]

[TOOL USAGE - RETRIEVAL]
1. DISCOVERY: Call out_nodes with empty node="" to list all top-level categories from the graph root.
2. TRAVERSAL: Call out_nodes(node) on any node. Returns light nodes (id, type, title, edge_type) — navigate by following edges.
3. EXPANSION: Call get_node_content(node) to read full content of a Fact or any node.
4. BACKTRACK: Call in_nodes(node) to see what nodes point to a given node (parents, cross-refs).
5. SEARCH: Use find for text search when graph descent is insufficient.

[TOOL USAGE - WRITE]
Node creation follows: add_node(parent, edge, type, name, content)
  parent="" targets the graph root.
  name is the short label surfaced during traversal (stored as metadata.title).
  content holds the full body; defaults to name when omitted.

Common patterns:
  New category:    add_node(parent="",          edge="has_category",    type="Category",    name="Work")
  New subcategory: add_node(parent="Work",       edge="has_subcategory", type="Subcategory", name="Projects")
  New fact:        add_node(parent="Projects",   edge="has_fact",        type="Fact",        name="Short label", content="Full fact body")
  Attach document: add_node(parent="<node>",     edge="has_document",    type="Document",    name="file.pdf",    content="/abs/path")
  Cross-reference: link_relevant(node_a, node_b) — bidirectional is_relevant_to on existing nodes
  Delete:          delete_node(identifier) — cascade deletes node and all descendants

[CONSTRAINTS]
- SILENT_EXECUTION: Execute tools without conversational narration (e.g., do not say "I am checking..."). Formulate responses as innate knowledge.
- PROACTIVE_RETENTION: Automatically store user preferences, goals, context, findings, and corrections post-response.
`
}

// buildCategoriesSection creates the CURRENT CATEGORIES section for the system prompt
// This is called after database initialization to dynamically add category information
func (p *KnowledgePlugin) buildCategoriesSection() string {
	// Retrieve top-level categories, handling errors gracefully
	categories, err := p.getTopLevelCategories()
	if err != nil {
		agentforge.Debug("🧠 [Knowledge Plugin] Warning: failed to retrieve top-level categories: %v", err)
		return ""
	}

	if len(categories) == 0 {
		return ""
	}

	agentforge.Debug("🧠 [Knowledge Plugin] Found %d top-level categories", len(categories))

	// Limit to first 15 categories to avoid bloating the prompt
	const maxCategories = 15
	total := len(categories)
	if total > maxCategories {
		categories = categories[:maxCategories]
	}

	// Build the categories list
	var categoryList string
	categoryList = "\n[CURRENT CATEGORIES]\nTop-level knowledge organization nodes:\n"
	for _, cat := range categories {
		categoryList += fmt.Sprintf("  - %s\n", cat.Content)
	}

	if total > maxCategories {
		categoryList += fmt.Sprintf("  ... [%d additional categories not shown — call out_nodes(\"\") to see all]\n", total-maxCategories)
	}

	categoryList += "\nInstruction: Evaluate these categories when classifying new knowledge for storage operations.\n"

	return categoryList
}

// Public API Methods for Knowledge Graph Abstraction

// resolveNode resolves an identifier to a Node.
// An empty identifier resolves to the graph root (omnia-nunc-root).
// Otherwise tries exact ID match, then content/title LIKE search.
func (p *KnowledgePlugin) resolveNode(identifier string) (*Node, error) {
	if identifier == "" {
		return p.getNode(omniaNuncNodeID)
	}
	node, err := p.getNode(identifier)
	if err == nil {
		return node, nil
	}
	ctx := context.Background()
	pattern := "%" + identifier + "%"
	query := `
		SELECT id, type, content, embedding_id, metadata, created_at, updated_at
		FROM knowledge_nodes
		WHERE id = ? OR content LIKE ?
		  OR (metadata IS NOT NULL AND json_valid(metadata) AND json_extract(metadata, '$.title') LIKE ?)
		ORDER BY created_at DESC
		LIMIT 1
	`
	nodes, err := p.queryNodes(ctx, query, identifier, pattern, pattern)
	if err != nil || len(nodes) == 0 {
		return nil, fmt.Errorf("node not found: %s", identifier)
	}
	return &nodes[0], nil
}

// OutNodes returns all outgoing neighbors of a node (edges FROM the node).
// An empty identifier starts from the graph root.
func (p *KnowledgePlugin) OutNodes(identifier string) (*NodeNeighborsResult, error) {
	if p.querier == nil {
		return nil, fmt.Errorf("knowledge plugin not fully initialized")
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
func (p *KnowledgePlugin) InNodes(identifier string) (*NodeNeighborsResult, error) {
	if p.querier == nil {
		return nil, fmt.Errorf("knowledge plugin not fully initialized")
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
func (p *KnowledgePlugin) GetNodeContent(identifier string) (*Node, error) {
	if p.querier == nil {
		return nil, fmt.Errorf("knowledge plugin not fully initialized")
	}
	return p.resolveNode(identifier)
}

// Find searches for nodes matching query and returns scored results
func (p *KnowledgePlugin) Find(query string, limit int) ([]ScoredNode, error) {
	return p.findScored(query, limit)
}

// AddNode creates a new node and attaches it to a parent via an edge.
// parentIdentifier resolves via resolveNode: empty string = graph root.
func (p *KnowledgePlugin) AddNode(parentIdentifier, edgeType, nodeType, name, content string) (string, error) {
	return p.addNode(parentIdentifier, edgeType, nodeType, name, content)
}

// DeleteNode deletes a node and all its dependents (cascade delete).
func (p *KnowledgePlugin) DeleteNode(identifier string) (int, error) {
	return p.forgetCascade(identifier)
}

// saveDocument creates a Document node at filePath and links it to parentID via has_document.
func (p *KnowledgePlugin) saveDocument(parentID, filePath string, extra map[string]any) (string, error) {
	if extra == nil {
		extra = make(map[string]any)
	}
	extra["file_path"] = filePath
	extra["title"] = filepath.Base(filePath)
	docID, err := p.saveNode("Document", filePath, "", extra)
	if err != nil {
		return "", fmt.Errorf("failed to save document node: %w", err)
	}
	_, err = p.saveEdge(parentID, docID, "has_document", 1.0, nil)
	if err != nil {
		_ = p.deleteNode(docID)
		return "", fmt.Errorf("failed to link document: %w", err)
	}
	return docID, nil
}

// AttachDocument resolves a parent node by ID or content and attaches a document to it.
func (p *KnowledgePlugin) AttachDocument(parentIdentifier, filePath string) (string, error) {
	node, err := p.getNode(parentIdentifier)
	if err != nil {
		nodes, err2 := p.findNodesByContent(parentIdentifier, false)
		if err2 != nil || len(nodes) == 0 {
			return "", fmt.Errorf("parent node not found: %s", parentIdentifier)
		}
		node = &nodes[0]
	}
	return p.saveDocument(node.ID, filePath, nil)
}

// LinkRelevant creates bidirectional is_relevant_to edges between two nodes (resolved by ID or content).
func (p *KnowledgePlugin) LinkRelevant(identifierA, identifierB string) (string, string, error) {
	resolve := func(id string) (*Node, error) {
		node, err := p.getNode(id)
		if err == nil {
			return node, nil
		}
		nodes, err := p.findNodesByContent(id, false)
		if err != nil || len(nodes) == 0 {
			return nil, fmt.Errorf("node not found: %s", id)
		}
		return &nodes[0], nil
	}
	nodeA, err := resolve(identifierA)
	if err != nil {
		return "", "", err
	}
	nodeB, err := resolve(identifierB)
	if err != nil {
		return "", "", err
	}
	edgeAB, err := p.saveEdge(nodeA.ID, nodeB.ID, "is_relevant_to", 1.0, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create A→B edge: %w", err)
	}
	edgeBA, err := p.saveEdge(nodeB.ID, nodeA.ID, "is_relevant_to", 1.0, nil)
	if err != nil {
		_ = p.deleteEdge(edgeAB)
		return "", "", fmt.Errorf("failed to create B→A edge: %w", err)
	}
	return edgeAB, edgeBA, nil
}

// init registers the plugin factory
func init() {
	registry.Register(PLUGIN_NAME, func(workingDir string) core.Plugin {
		return NewKnowledgePlugin(workingDir)
	})
}
