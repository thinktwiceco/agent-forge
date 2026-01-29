package agents

import (
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/tools/vector"
)

// VectorAgentTemplate defines the system agent template for vector database operations.
//
// This agent handles vector database tasks including indexing documents, performing
// semantic search, and deleting documents. It specializes in vector operations
// and provides guidance on how to structure vector database requests.

func createVectorAgentTemplate() *SystemAgentTemplate {
	template, err := NewSystemAgentTemplate(AgentNameSystemVector, TraceVector)
	if err != nil {
		panic(err)
	}

	// Build system prompt with structured components
	template.AddSystemPrompt(
		`You are a vector database specialist agent. Your role is to handle vector database
operations including indexing documents, performing semantic search, and managing documents
in the vector database.

When given a task, analyze what vector operations are needed:
- What documents need to be indexed?
- What semantic searches need to be performed?
- What documents need to be deleted?
- What metadata should be associated with documents?
- What filters should be applied to searches?`,
		// Steps
		[]string{
			"Identify what vector database operations are needed for the task",
			"Determine the appropriate action (index, search, or delete)",
			"Extract or prepare the required parameters for the operation",
			"Execute the vector database operation using the vector_db tool",
			"Format and return the results clearly",
		},
		// Output format
		`
You will perform vector database operations using the vector_db tool. When executing operations:
- Always specify the action parameter (index, search, or delete)
- For index: provide text and optional metadata/document_id
- For search: provide query and optional top_k/filters
- For delete: provide document_id
- Return only operation results (document IDs, similarity scores, etc.)
- Report errors only when operations fail`,
		// Examples
		[]string{`
'user': Index a document with the text "Machine learning is a subset of artificial intelligence"

'assistant':
[Uses vector_db tool with action="index", text="Machine learning is a subset of artificial intelligence"]
Document indexed successfully with ID: 550e8400-e29b-41d4-a716-446655440000`,
			`
'user': Search for documents related to "AI and neural networks"

'assistant':
[Uses vector_db tool with action="search", query="AI and neural networks", top_k=10]
Search Results:
[
  {
    "text": "Machine learning is a subset of artificial intelligence",
    "metadata": {"_embedding_model": "text-embedding-ada-002"},
    "similarity": 0.87
  }
]`,
			`
'user': Delete the document with ID 550e8400-e29b-41d4-a716-446655440000

'assistant':
[Uses vector_db tool with action="delete", document_id="550e8400-e29b-41d4-a716-446655440000"]
Document 550e8400-e29b-41d4-a716-446655440000 deleted successfully`,
			`
'user': Index a document about Python programming with metadata: {"category": "programming", "language": "python"}

'assistant':
[Uses vector_db tool with action="index", text="Python is a high-level programming language", metadata={"category": "programming", "language": "python"}]
Document indexed successfully with ID: 660e8400-e29b-41d4-a716-446655440001`,
			`
'user': Search for programming documents with top 5 results

'assistant':
[Uses vector_db tool with action="search", query="programming", top_k=5, filters={"category": "programming"}]
Search Results:
[
  {
    "text": "Python is a high-level programming language",
    "metadata": {"category": "programming", "language": "python", "_embedding_model": "text-embedding-ada-002"},
    "similarity": 0.92
  }
]`,
		},
		// Critical rules
		[]string{
			`Always specify the action parameter (index, search, or delete)`,
			`For index action: text parameter is required, metadata and document_id are optional`,
			`For search action: query parameter is required, top_k and filters are optional`,
			`For delete action: document_id parameter is required`,
			`Return only operation results without commentary`,
			`Include document IDs, similarity scores, and metadata in responses when relevant`,
			`Report errors concisely when operations fail`,
			`Use semantic search capabilities - queries don't need exact matches`,
		},
	)

	// Build description with structured components
	template.AddDescription(
		// Incipit
		`Handles vector database operations: index documents, perform semantic search, and delete documents. Automatically generates embeddings.`,
		// Examples
		[]string{
			`✅ Use for: Indexing documents, semantic search queries, managing document metadata`,
			`❌ Don't use: File system operations (use OS or Coding agent instead)`,
		},
	)

	// Add advanced description
	template.AddAdvanceDescription(`
Advanced Details:
- Purpose: Handles vector database operations including indexing, semantic search, and document management
- Tools Available:
  * vector_db: Vector database operations (index, search, delete) with automatic embedding generation
- Capabilities:
  * Index documents with text content and optional metadata
  * Perform semantic search using natural language queries
  * Delete documents by document ID
  * Filter search results using metadata filters
  * Control number of search results with top_k parameter
  * Automatically generate embeddings using the configured embedding generator
  * Track embedding model used in document metadata
- Vector Operations:
  * index: Stores documents with embeddings for semantic search
    - Required: text (document content)
    - Optional: metadata (key-value pairs), document_id (auto-generated if not provided)
  * search: Finds semantically similar documents
    - Required: query (search text)
    - Optional: top_k (number of results, default: 10), filters (metadata filters for exact match)
  * delete: Removes documents from the database
    - Required: document_id
- Embeddings:
  * Embeddings are automatically generated using the configured embedding generator
  * The embedding model name is stored in metadata as "_embedding_model"
  * Semantic search finds documents based on meaning, not exact text matches
- Metadata:
  * Can include any key-value pairs for document organization
  * Used for filtering search results with exact match
  * Automatically includes "_embedding_model" field
- Search Behavior:
  * Returns results ordered by similarity score (highest first)
  * Similarity scores indicate how closely documents match the query
  * Filters use exact match on metadata key-value pairs
- Integration: Automatically available as a sub-agent when vector database operations are needed`)

	// Add troubleshooting information
	template.AddTroubleshooting(`
Troubleshooting:
- "action parameter is required": Always specify action as "index", "search", or "delete"
- "text parameter is required for index": Provide the document text content when indexing
- "query parameter is required for search": Provide the search query text when searching
- "document_id parameter is required for delete": Provide the document ID when deleting
- "failed to generate embedding": Check that the embedding generator is properly configured
- "failed to index document": Verify the vector database connection and configuration
- "failed to search": Ensure the vector database is accessible and contains indexed documents
- "no results found": Try adjusting top_k, removing filters, or using different query terms
- Common mistakes:
  * Forgetting to specify the action parameter
  * Missing required parameters for the chosen action
  * Using incorrect parameter names (e.g., "text" instead of "query" for search)
  * Expecting exact text matches in search (semantic search finds similar meanings)
  * Not providing document_id when trying to delete
- Best practices:
  * Always specify the action parameter first
  * Include relevant metadata when indexing for better filtering
  * Use descriptive queries for semantic search
  * Store document IDs when indexing for later deletion
  * Use filters to narrow search results when needed
  * Understand that semantic search finds similar meanings, not exact matches`)

	return template
}

// VectorAgent creates a vector database operations agent with vector_db tool.
//
// Parameters:
//   - llmEngine: The LLM engine to use for this agent
//   - vectorDB: The vector database implementation
//   - embeddingGenerator: The embedding generator for creating embeddings
//
// Returns:
//   - *core.SubAgent: The Vector agent as a sub-agent
func VectorAgent(llmEngine llms.LLMEngine, vectorDB core.VectorDB, embeddingGenerator core.EmbeddingGenerator) core.SubAgent {
	vectorTemplate := createVectorAgentTemplate()
	vectorConfig := vectorTemplate.ToAgentConfig(llmEngine)

	// Add vector_db tool
	vectorTool := vector.NewVectorTool(vectorDB, embeddingGenerator)
	vectorConfig.Tools = []llms.Tool{vectorTool}

	vector := NewAgent(&vectorConfig)
	return vector.AgentAsSubAgent()
}
