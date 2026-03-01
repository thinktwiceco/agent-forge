package knowledge

import (
	"database/sql"
	"fmt"
	"path/filepath"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/plugins/registry"
)

const (
	PLUGIN_NAME = "knowledge"
)

// KnowledgePlugin provides a knowledge graph for storing and retrieving user information
type KnowledgePlugin struct {
	db           *sql.DB
	dir          string
	vectorDB     core.VectorDB
	embeddingGen core.EmbeddingGenerator
	querier      *GraphQuerier
	explorer     *KnowledgeExplorer
}

// NewKnowledgePlugin creates a new knowledge graph plugin
func NewKnowledgePlugin(workingDir string, vectorDB core.VectorDB, embeddingGen core.EmbeddingGenerator) *KnowledgePlugin {
	dir := filepath.Join(workingDir, "knowledge")
	return &KnowledgePlugin{
		dir:          dir,
		vectorDB:     vectorDB,
		embeddingGen: embeddingGen,
	}
}

// Name implements core.Plugin
func (p *KnowledgePlugin) Name() string {
	return PLUGIN_NAME
}

// Hooks implements core.HookProvider
func (p *KnowledgePlugin) Hooks() map[core.Event]core.AgentHookFn {
	return map[core.Event]core.AgentHookFn{
		core.EventAgentInitialized: agents.OnAgentInitializedHook(p.onInit),
	}
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

	if p.vectorDB == nil || p.embeddingGen == nil {
		agentforge.Debug("🧠 [Knowledge Plugin] Semantic search not available (missing vector DB or embedding generator)")
	} else {
		agentforge.Debug("🧠 [Knowledge Plugin] Semantic search enabled")
	}

	// Initialize querier and explorer
	p.querier = NewGraphQuerier(p.db)
	p.explorer = NewKnowledgeExplorer(p.querier)

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
	semanticInfo := ""
	if p.vectorDB != nil && p.embeddingGen != nil {
		semanticInfo = " with semantic search"
	}

	return fmt.Sprintf(`<KNOWLEDGE_GRAPH_SYSTEM>
<METADATA>
Type: Hierarchical Graph%s
Purpose: Proactive persistent memory.
</METADATA>

<SCHEMA>
Nodes:
  CATEGORY: Top-level. field:name=content field:title=optional
  SUBCATEGORY: Recursive child. field:name=content field:title=optional
  FACT: Knowledge unit. field:body=content field:title=short_id
  DOCUMENT: File ref. field:path=content field:name=title

Edges:
  has_subcategory: [Category|Subcategory] -> Subcategory
  has_fact: [Category|Subcategory] -> Fact
  has_document: [Category|Fact] -> Document
  is_relevant_to: [Any] <-> [Any]
</SCHEMA>

<TOOL_USAGE_EXPANSION>
<RETRIEVAL>
1. DISCOVERY: Use 'get_categories' to identify top-level nodes.
2. TRAVERSAL: Use 'explore_category' or 'explore_subcategory'. Yields LIGHT NODES (titles only for brevity).
3. EXPANSION: Use 'explore_fact' to retrieve full content of facts discovered in step 2. You MUST expand facts to read their content.
4. RECURSION: Use 'find' for semantic querying when graph descent is insufficient.
</RETRIEVAL>

<STORAGE_AND_RELATIONSHIPS>
1. CLASSIFICATION: When acquiring new knowledge, critically evaluate the correct Category and Subcategory. 
2. STRUCTURE_CREATION: Use 'add_category' or 'add_subcategory' if the required classification hierarchy does not exist.
3. INGESTION: Use 'remember' to store the Fact. You MUST provide a short 'title' for optimal light-node retrieval during traversal.
4. CROSS_REFERENCING: Use 'link_relevant' to connect the new node to other Categories, Subcategories, or Facts to build horizontal relationships.
5. FILE_ATTACHMENT: Use 'attach_document' to bind file paths to related nodes.
</STORAGE_AND_RELATIONSHIPS>
</TOOL_USAGE_EXPANSION>

<CONSTRAINTS>
- SILENT_EXECUTION: Execute tools without conversational narration (e.g., do not say "I am checking..."). Formulate responses as innate knowledge.
- PROACTIVE_RETENTION: Automatically store user preferences, goals, context, findings, and corrections post-response.
</CONSTRAINTS>
</KNOWLEDGE_GRAPH_SYSTEM>
`, semanticInfo)
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
	maxCategories := 15
	if len(categories) > maxCategories {
		categories = categories[:maxCategories]
	}

	// Build the categories list
	var categoryList string
	categoryList = "\n<CURRENT_CATEGORIES>\n<INFO>Top-level knowledge organization nodes</INFO>\n"
	for _, cat := range categories {
		categoryList += fmt.Sprintf("  - %s\n", cat.Content)
	}

	if len(categories) >= maxCategories {
		categoryList += fmt.Sprintf("  ... [%d additional categories muted]\n", len(categories)-maxCategories)
	}

	categoryList += "\n<INSTRUCTION>Evaluate these categories when classifying new knowledge for STORAGE_AND_RELATIONSHIPS operations.</INSTRUCTION>\n</CURRENT_CATEGORIES>\n"

	return categoryList
}

// Public API Methods for Knowledge Graph Abstraction

// ExploreCategory finds a category and returns its structured exploration result with light nodes.
func (p *KnowledgePlugin) ExploreCategory(category string) (*CategoryExploreResult, error) {
	if p.explorer == nil {
		return nil, fmt.Errorf("knowledge plugin not fully initialized")
	}
	return p.explorer.ExploreCategory(category)
}

// ExploreSubcategory finds a subcategory and returns its structured exploration result with light nodes.
func (p *KnowledgePlugin) ExploreSubcategory(subcategory string) (*SubcategoryExploreResult, error) {
	if p.explorer == nil {
		return nil, fmt.Errorf("knowledge plugin not fully initialized")
	}
	return p.explorer.ExploreSubcategory(subcategory)
}

// ExploreFact finds a fact and returns its full content plus light-node neighbours.
func (p *KnowledgePlugin) ExploreFact(fact string) (*FactExploreResult, error) {
	if p.explorer == nil {
		return nil, fmt.Errorf("knowledge plugin not fully initialized")
	}
	return p.explorer.ExploreFact(fact)
}

// Find searches for nodes matching query and returns scored results
func (p *KnowledgePlugin) Find(query string, limit int) ([]ScoredNode, error) {
	return p.findScored(query, limit)
}

// Remember saves a fact under a specific category
func (p *KnowledgePlugin) Remember(category string, fact string) (string, error) {
	return p.remember(category, fact, "")
}

// RememberWithTitle saves a fact under a category with an explicit short title.
func (p *KnowledgePlugin) RememberWithTitle(category, title, fact string) (string, error) {
	return p.remember(category, fact, title)
}
func (p *KnowledgePlugin) AddCategory(category string) (string, error) {
	return p.addCategory(category)
}

// AddSubcategory creates a new subcategory node under a parent category or subcategory
func (p *KnowledgePlugin) AddSubcategory(parentIdentifier string, subcategory string) (string, error) {
	return p.addSubcategory(parentIdentifier, subcategory)
}

// GetCategories returns all Category nodes
func (p *KnowledgePlugin) GetCategories() ([]Node, error) {
	return p.getCategories()
}

// GetCategoryFacts returns all Fact nodes directly connected to a category
func (p *KnowledgePlugin) GetCategoryFacts(category string) ([]Node, error) {
	return p.getCategoryFacts(category)
}

// Forget deletes a node and all its dependents (cascade delete)
func (p *KnowledgePlugin) Forget(identifier string) (int, error) {
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
		return NewKnowledgePlugin(workingDir, nil, nil)
	})
}
