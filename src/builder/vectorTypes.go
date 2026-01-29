package builder

import (
	"fmt"
	"strings"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/integrations"
)

// VectorDBType represents the type of vector database to use
type VectorDBType string

const (
	SQLITE_VECTOR_DB VectorDBType = "sqlite"
	MILVUS_VECTOR_DB VectorDBType = "milvus"
)

// getVectorDB creates a VectorDB instance based on the type and configuration
func (v VectorDBType) getVectorDB(config map[string]interface{}) (core.VectorDB, error) {
	switch v {
	case SQLITE_VECTOR_DB:
		dbPath, ok := config["db_path"].(string)
		if !ok || dbPath == "" {
			return nil, fmt.Errorf("sqlite db_path is required")
		}
		// Default vector dimension for embeddings (text-embedding-3-small uses 1536)
		vectorDim := 1536
		if dim, ok := config["vector_dim"].(int); ok && dim > 0 {
			vectorDim = dim
		}
		sqliteConfig := integrations.SQLiteConfig{
			DBPath:    dbPath,
			VectorDim: vectorDim,
		}
		return integrations.NewSQLiteDB(sqliteConfig)
	case MILVUS_VECTOR_DB:
		// Parse host and port from endpoint or config
		host := "localhost"
		port := 19530

		if h, ok := config["host"].(string); ok && h != "" {
			host = h
		}
		if p, ok := config["port"].(int); ok && p > 0 {
			port = p
		}

		// Default vector dimension
		vectorDim := 1536
		if dim, ok := config["vector_dim"].(int); ok && dim > 0 {
			vectorDim = dim
		}

		milvusConfig := integrations.MilvusConfig{
			Host:      host,
			Port:      port,
			VectorDim: vectorDim,
		}
		return integrations.NewMilvusDB(milvusConfig)
	}
	return nil, fmt.Errorf("invalid vector database type: %s", v)
}

// EmbeddingModel represents an embedding model with provider and model name
// Format: <provider>::<model>
type EmbeddingModel string

// provider extracts the provider from the model string
func (e EmbeddingModel) provider() string {
	segments := strings.Split(string(e), "::")
	if len(segments) != 2 {
		return ""
	}
	return segments[0]
}

// model extracts the model name from the model string
func (e EmbeddingModel) modelName() string {
	segments := strings.Split(string(e), "::")
	if len(segments) != 2 {
		return ""
	}
	return segments[1]
}

// validate checks if the embedding model string is in the correct format
func (e EmbeddingModel) validate() error {
	segments := strings.Split(string(e), "::")
	if len(segments) != 2 {
		return fmt.Errorf("invalid embedding model format. Expected <provider>::<model>, got %s", e)
	}

	provider := segments[0]
	model := segments[1]

	if provider == "" {
		return fmt.Errorf("embedding model provider cannot be empty")
	}
	if model == "" {
		return fmt.Errorf("embedding model name cannot be empty")
	}

	return nil
}

// getEmbeddingGenerator creates an EmbeddingGenerator instance based on the provider
func (e EmbeddingModel) getEmbeddingGenerator() (core.EmbeddingGenerator, error) {
	if err := e.validate(); err != nil {
		return nil, err
	}

	provider := e.provider()
	modelName := e.modelName()

	switch provider {
	case "openai":
		// OpenAI embeddings require API key from environment
		return integrations.NewOpenAIEmbeddingGenerator("", modelName)
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", provider)
	}
}
