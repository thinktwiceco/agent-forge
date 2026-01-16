package integrations

import "fmt"

// ChromaDBConfig holds configuration parameters for connecting to ChromaDB.
type ChromaDBConfig struct {
	// Host is the ChromaDB server host (e.g., "localhost")
	Host string

	// Port is the ChromaDB server port (e.g., 8000)
	Port int

	// CollectionName is the name of the collection to use for documents
	CollectionName string

	// DefaultTopK is the default number of results to return for search (default: 10)
	DefaultTopK int
}

// Validate checks that all required fields are set and sets defaults.
func (c *ChromaDBConfig) Validate() error {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 8000
	}
	if c.CollectionName == "" {
		c.CollectionName = "default"
	}
	if c.DefaultTopK == 0 {
		c.DefaultTopK = 10
	}
	return nil
}

// MilvusConfig holds configuration parameters for connecting to Milvus.
type MilvusConfig struct {
	// Host is the Milvus server host (e.g., "localhost")
	Host string

	// Port is the Milvus server port (e.g., 19530)
	Port int

	// CollectionName is the name of the collection to use for documents
	CollectionName string

	// DefaultTopK is the default number of results to return for search (default: 10)
	DefaultTopK int

	// VectorDim is the dimension of the vector embeddings (required)
	VectorDim int

	// Username is optional username for authentication
	Username string

	// Password is optional password for authentication
	Password string
}

// Validate checks that all required fields are set and sets defaults.
func (c *MilvusConfig) Validate() error {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 19530
	}
	if c.CollectionName == "" {
		c.CollectionName = "default"
	}
	if c.DefaultTopK == 0 {
		c.DefaultTopK = 10
	}
	if c.VectorDim <= 0 {
		return fmt.Errorf("VectorDim must be greater than 0")
	}
	return nil
}
