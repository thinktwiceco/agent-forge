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

	return fmt.Sprintf(`[KNOWLEDGE STORAGE%s]
Hierarchical graph with Categories, Subcategories, Facts, and Documents. Use proactively to remember everything.

NODE TYPES:
- Category: Top-level organizational grouping. Has a name (content) and an optional title.
- Subcategory: A specific grouping within a Category or another Subcategory (recursive).
- Fact: A piece of knowledge stored under a category or subcategory. Has full content + an optional short title.
- Document: A file system reference (content = absolute path). Title = filename.

EDGE TYPES:
- has_category: Category → Category
- has_subcategory: Category or Subcategory → Subcategory
- has_fact: Category or Subcategory → Fact, or Fact → Fact
- has_document: Category or Fact → Document (file reference)
- is_relevant_to: Bidirectional relevance between any two nodes of any type

TOOLS:
- explore_category: Returns sub-categories, facts (titles only), documents, and relevant nodes
- explore_subcategory: Returns child sub-categories, facts (titles only), documents, and relevant nodes
- explore_fact: Returns full fact content + parent categories, documents, relevant nodes (titles only)
- find: Semantic/text search across all nodes
- remember: Save a fact under a category or subcategory (optional title for navigation)
- attach_document: Attach a file path to a parent node
- link_relevant: Create a bidirectional is_relevant_to edge between any two nodes
- add_category: Create a new category
- add_subcategory: Create a new subcategory under an existing category or subcategory
- get_categories: List all top-level categories
- get_category_facts: Get facts under a category
- forget: Delete a node and all dependents

NAVIGATION MODEL:
  explore_category and explore_subcategory return child titles only — use explore_fact to get full content.
  explore_fact returns neighbour titles only — use explore_fact/explore_category/explore_subcategory to dig deeper.
  This two-phase design keeps responses compact: browse titles first, then read what matters.

CRITICAL BEHAVIORS:

1. START OF EVERY CONVERSATION (SILENTLY):
   - get_categories → explore relevant categories for context about user, history, and prior work.
   - Present retrieved info as natural recall: "I remember...", "Last time..."
   - Never announce "checking", "retrieving", or "exploring".

2. PROACTIVE STORAGE (After ANY response):
   - Store key facts immediately with remember (include a short title).
   - Create categories and subcategories as needed. Store silently — never announce it.

3. ALWAYS STORE:
   - User preferences, goals, context
   - Research findings & task outcomes
   - Feedback and corrections
   - Tool capabilities and known issues
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
	categoryList = "\nCURRENT CATEGORIES:\nTop-level knowledge organization (closest to Omnia Nunc Node):\n"
	for _, cat := range categories {
		categoryList += fmt.Sprintf("  - %s\n", cat.Content)
	}

	if len(categories) >= maxCategories {
		categoryList += fmt.Sprintf("  ... and %d more categories\n", len(categories)-maxCategories)
	}

	categoryList += "\nWhen storing new information, consider organizing it under these existing categories or create new ones as needed.\n"

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
