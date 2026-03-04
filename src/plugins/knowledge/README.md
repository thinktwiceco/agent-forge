# Knowledge Plugin

A knowledge graph plugin that combines SQLite graph storage with semantic search to remember user information, preferences, and context with intelligent retrieval.

## Overview

The knowledge plugin provides a hybrid storage system that combines:
- **Graph Database**: SQLite-based graph storage for structured relationships
- **Semantic Search**: Optional vector database integration for intelligent fuzzy matching
- **Graph Traversal**: Recursive CTE queries for discovering related information
- **Clean API**: Eight focused methods for graph exploration and management

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   Knowledge Plugin API                       │
├─────────────────────────────────────────────────────────────┤
│  Exploration: explore_category | explore_fact | find        │
│  CRUD: remember | add_category | get_categories |           │
│        get_category_facts                                    │
│  Management: forget                                          │
└───────────┬─────────────────────────────────┬───────────────┘
            │                                 │
   ┌────────▼────────┐               ┌───────▼──────────┐
   │  Graph Storage  │               │  Semantic Search │
   │   (SQLite)      │               │  (Vector DB)     │
   ├─────────────────┤               ├──────────────────┤
   │ • Nodes table   │               │ • Embeddings     │
   │ • Edges table   │               │ • Similarity     │
   │ • Indexes       │               │ • Optional       │
   └─────────────────┘               └──────────────────┘
```

## Database Schema

### Nodes Table
Stores knowledge in two types: Categories and Facts:

```sql
CREATE TABLE knowledge_nodes (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,              -- "Category" or "Fact"
    content TEXT NOT NULL,           -- The actual information
    embedding_id TEXT,               -- Reference to vector DB entry
    metadata TEXT,                   -- JSON metadata
    created_at DATETIME,
    updated_at DATETIME
);
```

### Edges Table
Stores relationships between nodes:

```sql
CREATE TABLE knowledge_edges (
    id TEXT PRIMARY KEY,
    from_node_id TEXT NOT NULL,
    to_node_id TEXT NOT NULL,
    relation_type TEXT NOT NULL,     -- "has_category" or "has_fact"
    weight REAL DEFAULT 1.0,
    metadata TEXT,
    created_at DATETIME
);
```

### Omnia Nunc Node
A special root node ("Omnia Nunc Node") that connects all top-level categories, forming the entry point of the knowledge graph.

## API Methods

The plugin exposes eight focused methods organized into three categories:

### Exploration Methods

#### ExploreCategory(category: string)
Explore a category and retrieve its full hierarchy.

**Returns**: `*GraphResult` with all child categories and facts

**Example**:
```go
result, err := plugin.ExploreCategory("Programming")
// Returns the category node, all sub-categories, and all facts
```

#### ExploreFact(fact: string)
Explore a fact and retrieve its full context.

**Returns**: `*GraphResult` with the fact, related facts, and parent categories

**Example**:
```go
result, err := plugin.ExploreFact("User prefers dark mode")
// Returns the fact and its surrounding context
```

#### Find(query: string, limit: int)
Search for nodes using semantic search (if available) or text search.

**Returns**: `[]ScoredNode` with relevance scores

**Example**:
```go
results, err := plugin.Find("programming preferences", 10)
// Returns top 10 matching nodes with scores
```

### CRUD Methods

#### Remember(category: string, fact: string)
Save a fact under a specific category.

**Returns**: Fact node ID (string)

**Example**:
```go
factID, err := plugin.Remember("Preferences", "User prefers dark mode")
// Creates a new Fact node linked to the Preferences category
```

#### AddCategory(category: string)
Create a new category node.

**Returns**: Category node ID (string)

**Example**:
```go
categoryID, err := plugin.AddCategory("Programming")
// Creates a new Category node linked to Omnia Nunc Node
```

#### GetCategories()
Retrieve all Category nodes.

**Returns**: `[]Node`

**Example**:
```go
categories, err := plugin.GetCategories()
// Returns all categories in the knowledge graph
```

#### GetCategoryFacts(category: string)
Get all Fact nodes directly connected to a category.

**Returns**: `[]Node`

**Example**:
```go
facts, err := plugin.GetCategoryFacts("Programming")
// Returns all facts under the Programming category
```

### Management Methods

#### Forget(identifier: string)
Delete a node and all its dependents (cascade delete).

**Returns**: Count of deleted nodes (int)

**Example**:
```go
deletedCount, err := plugin.Forget("some-node-id")
// Deletes the node and all child nodes
```

## Tool Interface

The plugin exposes eight tools for agent use:

### explore_category
Explore a category and get its full hierarchy.

**Parameters**:
- `category` (string, required): Category name to explore

**Response**:
```json
{
  "nodes": [...],
  "edges": [...]
}
```

### explore_fact
Explore a fact and get its context.

**Parameters**:
- `fact` (string, required): Fact content to explore

**Response**:
```json
{
  "nodes": [...],
  "edges": [...]
}
```

### find
Search for nodes with semantic/text search.

**Parameters**:
- `query` (string, required): Search query
- `limit` (number, optional): Max results (default: 10)

**Response**:
```json
[
  {
    "node": {...},
    "score": 0.95
  }
]
```

### remember
Save a fact under a category.

**Parameters**:
- `category` (string, required): Category name
- `fact` (string, required): Fact content

**Response**:
```json
{
  "fact_id": "uuid",
  "category": "Programming",
  "fact": "User knows Go"
}
```

### add_category
Create a new category.

**Parameters**:
- `category` (string, required): Category name

**Response**:
```json
{
  "category_id": "uuid",
  "category": "Programming"
}
```

### get_categories
List all categories.

**Parameters**: None

**Response**:
```json
{
  "categories": [...],
  "count": 5
}
```

### get_category_facts
Get facts under a category.

**Parameters**:
- `category` (string, required): Category name

**Response**:
```json
{
  "category": "Programming",
  "facts": [...],
  "count": 3
}
```

### forget
Delete a node and its dependents.

**Parameters**:
- `identifier` (string, required): Node ID or content

**Response**:
```json
{
  "deleted_count": 5,
  "identifier": "uuid"
}
```

## Usage Examples

### Basic Workflow

```go
// 1. Create a category
categoryID, err := plugin.AddCategory("Personal Information")

// 2. Remember facts under the category
plugin.Remember("Personal Information", "Lives in San Francisco")
plugin.Remember("Personal Information", "Works as a software engineer")

// 3. Explore the category
result, err := plugin.ExploreCategory("Personal Information")
// Returns all facts under this category

// 4. Search across all knowledge
matches, err := plugin.Find("San Francisco", 10)

// 5. Get all categories
categories, err := plugin.GetCategories()

// 6. Get specific category facts
facts, err := plugin.GetCategoryFacts("Personal Information")

// 7. Forget specific information
deletedCount, err := plugin.Forget("some-fact-id")
```

### Agent Usage Example

```javascript
// Agent learns something new
{
  "tool": "add_category",
  "arguments": {
    "category": "User Preferences"
  }
}

{
  "tool": "remember",
  "arguments": {
    "category": "User Preferences",
    "fact": "Prefers dark mode in IDE"
  }
}

// Later, agent explores what it knows
{
  "tool": "explore_category",
  "arguments": {
    "category": "User Preferences"
  }
}

// Or searches for specific information
{
  "tool": "find",
  "arguments": {
    "query": "IDE preferences",
    "limit": 5
  }
}
```

## Node and Edge Types

### Node Types
- **Category**: Organizational nodes for grouping knowledge
- **Fact**: Actual information/knowledge nodes

### Edge Types
- **has_category**: Links Category to Category
- **has_fact**: Links Category to Fact, or Fact to Fact

### Validation Rules
- `has_category` edges can only connect two Category nodes
- `has_fact` edges connect Category to Fact, or Fact to Fact
- Facts cannot have Category children
- Top-level categories are automatically linked to Omnia Nunc Node

## Features

### 1. Semantic Search (Optional)

When vector DB and embedding generator are available:
- Automatic embedding generation for all nodes
- Fuzzy matching based on semantic similarity
- Ranked results by relevance score

### 2. Graph Traversal

- **Cycle Prevention**: Path tracking prevents infinite loops
- **Depth Control**: Configurable traversal depth
- **Bidirectional**: Follows edges in both directions

### 3. Cascade Delete

The `forget` method performs cascade deletion:
1. Identifies all dependent nodes recursively
2. Deletes in reverse order (leaves first)
3. SQLite foreign keys automatically remove edges
4. Protects system nodes (Omnia Nunc Node)

### 4. Metadata Support

Store rich metadata with nodes:
- `confidence`: Reliability score (0.0-1.0)
- `source`: Information origin
- `tags`: Categorization tags
- Custom fields as needed

### 5. Knowledge Retention Hook

The plugin automatically appends a reminder to all successful tool executions:
- Prompts the agent to consider saving information
- Encourages proactive knowledge retention
- Only applies to successful tool calls
- Message: `[Reminder]: is this worth saving or just transactional? If you are not sure, ask your human!`

This hook helps agents develop a habit of persisting useful information in the knowledge graph.

## Best Practices

### 1. Organize with Categories

Create a clear category structure:
```go
plugin.AddCategory("Personal")
plugin.AddCategory("Work")
plugin.AddCategory("Projects")
```

### 2. Use Meaningful Names

Choose clear, descriptive category and fact names:
```go
// Good
plugin.Remember("Skills", "Proficient in Go programming")

// Avoid vague descriptions
plugin.Remember("Stuff", "Does things")
```

### 3. Explore Before Adding

Check existing categories before creating new ones:
```go
categories, _ := plugin.GetCategories()
// Review existing categories before adding duplicates
```

### 4. Search Semantically

Use the `find` method for fuzzy matching:
```go
results, _ := plugin.Find("programming languages", 10)
// Finds related facts even with different wording
```

### 5. Clean Up Outdated Information

Remove obsolete information with cascade delete:
```go
deletedCount, _ := plugin.Forget("outdated-category-id")
// Removes category and all its facts
```

## Performance Considerations

### Indexes

The plugin creates indexes on:
- `knowledge_nodes.type`
- `knowledge_nodes.content`
- `knowledge_edges.from_node_id`
- `knowledge_edges.to_node_id`
- `knowledge_edges.relation_type`

### Scalability

- **Small graphs** (< 1000 nodes): Excellent performance
- **Medium graphs** (1000-10000 nodes): Good performance with indexes
- **Large graphs** (> 10000 nodes): Consider limiting traversal depth

### Optimization Tips

1. Use `GetCategoryFacts` for direct fact retrieval (faster than exploration)
2. Limit search results appropriately
3. Use exploration methods judiciously (they traverse the graph)
4. Regularly clean up outdated information

## Storage Location

Knowledge graphs are stored in:
```
{working_dir}/knowledge/knowledge.db
```

The database uses SQLite with:
- WAL mode for concurrent access
- Foreign keys enabled for referential integrity
- Automatic schema creation on first use

## Integration with Vector DB

To enable semantic search, provide vector DB and embedding generator:

```go
plugin := knowledge.NewKnowledgePlugin(workingDir, vectorDB, embeddingGen)
```

When available, the `find` and `remember` methods automatically use semantic search for better matching.

## Error Handling

The plugin handles errors gracefully:
- **Missing semantic search**: Falls back to text search
- **Invalid identifiers**: Returns clear error messages
- **Non-existent categories**: Prevents orphaned facts
- **System node protection**: Cannot delete Omnia Nunc Node

## Testing

Run the test suite:
```bash
go test ./src/plugins/knowledge/...
```

Tests cover:
- All API methods
- Graph traversal and exploration
- Cascade deletion
- Semantic search (when available)
- Edge validation
- Error conditions

## Migration from Old Interface

If migrating from the old action-based interface:

**Old**:
```json
{
  "action": "save",
  "facts": [{"type": "Fact", "content": "..."}]
}
```

**New**:
```json
{
  "tool": "remember",
  "arguments": {"category": "...", "fact": "..."}
}
```

**Old**:
```json
{
  "action": "find",
  "query": "..."
}
```

**New**:
```json
{
  "tool": "find",
  "arguments": {"query": "...", "limit": 10}
}
```

## See Also

- [Plugin System Documentation](../README.md)
- [Vector DB Integration](../../integrations/sqliteDB.go)
- [Core Interfaces](../../core/interfaces.go)
