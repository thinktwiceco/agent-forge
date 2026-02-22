package system

// VectorAgentTemplate defines the system agent template for vector database operations.
//
// This agent handles vector database tasks including indexing documents, performing
// semantic search, and deleting documents. It specializes in vector operations
// and provides guidance on how to structure vector database requests.

// CreateVectorAgentTemplate creates the template for vector database operations agent.
func CreateVectorAgentTemplate() *SystemAgentTemplate {
	template, err := LoadAgentTemplateFromMarkdown("vector", AgentNameSystemVector, TraceVector)
	if err != nil {
		panic(err)
	}
	return template
}
