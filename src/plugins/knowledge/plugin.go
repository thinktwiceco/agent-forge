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
Hierarchical storage with Categories and Facts. Use this to remember everything important.

TOOLS:
- explore_category: Explore a category and its full hierarchy
- explore_fact: Explore a fact with context
- find: Search across all knowledge
- remember: Save a fact under a category
- add_category: Create a new category
- get_categories: List all categories
- get_category_facts: Get facts under a category
- forget: Delete information and dependents

CRITICAL BEHAVIORS:

1. START OF EVERY CONVERSATION:
   - ALWAYS use get_categories to see existing knowledge structure
   - ALWAYS use explore_category to retrieve context about:
     * The user (preferences, goals, history, context)
     * Previous tasks and their outcomes
     * Your own performance and learnings
     * Tool usage patterns and best practices
   - Treat each conversation as potentially connected to previous ones
   - Look for patterns in user interests across conversations
   
   IMPORTANT: Use these tools SILENTLY - never mention "checking", "retrieving", or "exploring"

2. NATURAL PRESENTATION:
   - Present retrieved information as if you naturally remember it
   - DON'T say: "Let me check...", "I found...", "Let me explore...", "I can see from..."
   - DO say: "I remember...", "Last time we discussed...", "You were interested in..."
   - Act like a human who naturally recalls previous conversations
   - Be conversational and seamless - don't announce your retrieval process

3. CONTEXTUAL AWARENESS (Before responding):
   - When user asks about ANY topic, SILENTLY use find or explore_category
   - If topic relates to previous research/discussions, naturally reference it
   - Example: "Oh, you were looking at [topic] before! Are you still interested in that?"
   - Always connect new information to existing knowledge when relevant
   - Look for topic threads and maintain continuity

4. PROACTIVE STORAGE (After ANY response):
   - After answering research questions, IMMEDIATELY store key facts with remember
   - After completing tasks, IMMEDIATELY store outcomes and insights
   - Store under existing relevant categories when connections exist
   - If no relevant category exists, create one with add_category or ask user
   - Don't wait until end of conversation - store as you go
   - Store SILENTLY - don't announce "I'm storing this..."

5. ALWAYS STORE when you:
   - Learn about the user (preferences, goals, context, personal info)
   - Answer research or informational questions (store key facts)
   - Complete multi-step tasks (outcomes, what worked, what didn't)
   - Receive feedback (user satisfaction, suggestions, corrections)
   - Discover tool capabilities or limitations
   - Identify patterns or optimization opportunities
   - Test features or workflows (successes and failures)

6. RECOMMENDED CATEGORIES:
   - "User Context" - User information, preferences, goals, history
   - "Agent Performance" - Your successes, failures, learnings, feedback
   - "Tool Usage" - Tool capabilities, best practices, known issues
   - "Research & Content" - Information gathered during tasks (organize by topic)
   - "Process Optimization" - Patterns, improvements, recurring issues
   
   Create topic-specific categories as needed (e.g., "Python Research", "Project X Notes").

7. TYPICAL WORKFLOW:
   - Start: SILENTLY use get_categories → explore relevant categories for context
   - Before answering: SILENTLY use find to check if topic has prior context
   - After answering: SILENTLY remember key facts immediately
   - During: SILENTLY remember new learnings as you discover them
   - End: SILENTLY remember overall outcomes, insights, and feedback

Remember: Be natural and conversational. Check context silently, present as natural recall, store immediately.
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

// ExploreCategory finds a category and returns its full hierarchy
func (p *KnowledgePlugin) ExploreCategory(category string) (*GraphResult, error) {
	return p.exploreCategory(category)
}

// ExploreFact finds a fact and returns its full context
func (p *KnowledgePlugin) ExploreFact(fact string) (*GraphResult, error) {
	return p.exploreFact(fact)
}

// Find searches for nodes matching query and returns scored results
func (p *KnowledgePlugin) Find(query string, limit int) ([]ScoredNode, error) {
	return p.findScored(query, limit)
}

// Remember saves a fact under a specific category
func (p *KnowledgePlugin) Remember(category string, fact string) (string, error) {
	return p.remember(category, fact)
}

// AddCategory creates a new category node
func (p *KnowledgePlugin) AddCategory(category string) (string, error) {
	return p.addCategory(category)
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

// init registers the plugin factory
func init() {
	registry.Register(PLUGIN_NAME, func(workingDir string) core.Plugin {
		return NewKnowledgePlugin(workingDir, nil, nil)
	})
}
