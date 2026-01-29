package builder

import (
	"fmt"
	"os"

	"github.com/thinktwiceco/agent-forge/src/core"
	"gopkg.in/yaml.v3"
)

// VectorStorageConfig represents the YAML configuration for vector storage
type VectorStorageConfig struct {
	VectorDB       string                 `yaml:"vector_db"`
	EmbeddingModel string                 `yaml:"embedding_model"`
	SQLite         map[string]interface{} `yaml:"sqlite,omitempty"`
	Milvus         map[string]interface{} `yaml:"milvus,omitempty"`
}

// VectorConfig wraps the vector storage configuration
type VectorConfig struct {
	VectorStorage VectorStorageConfig `yaml:"vector-storage"`
}

// VectorBuilder is a builder for creating vector storage components
type VectorBuilder struct {
	vectorDBType       VectorDBType
	embeddingModelType EmbeddingModel
	vectorDB           core.VectorDB
	embeddingGenerator core.EmbeddingGenerator
	dbConfig           map[string]interface{}
}

// Public API methods

// NewVectorBuilder creates a new VectorBuilder instance
func NewVectorBuilder() *VectorBuilder {
	return &VectorBuilder{
		dbConfig: make(map[string]interface{}),
	}
}

// NewVectorBuilderFromConfig creates a VectorBuilder from a YAML configuration file
func NewVectorBuilderFromConfig(configPath string) (*VectorBuilder, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg VectorConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	b := NewVectorBuilder()

	// Set vector database type
	if cfg.VectorStorage.VectorDB != "" {
		b.SetVectorDBType(VectorDBType(cfg.VectorStorage.VectorDB))

		// Set database-specific configuration
		switch b.vectorDBType {
		case SQLITE_VECTOR_DB:
			if cfg.VectorStorage.SQLite != nil {
				b.dbConfig = cfg.VectorStorage.SQLite
			}
		case MILVUS_VECTOR_DB:
			if cfg.VectorStorage.Milvus != nil {
				b.dbConfig = cfg.VectorStorage.Milvus
			}
		}
	}

	// Set embedding model
	if cfg.VectorStorage.EmbeddingModel != "" {
		b.SetEmbeddingModel(cfg.VectorStorage.EmbeddingModel)
	}

	return b, nil
}

// SetVectorDBType sets the vector database type
func (b *VectorBuilder) SetVectorDBType(dbType VectorDBType) *VectorBuilder {
	b.vectorDBType = dbType
	return b
}

// SetEmbeddingModel sets the embedding model
func (b *VectorBuilder) SetEmbeddingModel(model string) *VectorBuilder {
	b.embeddingModelType = EmbeddingModel(model)
	return b
}

// SetSQLitePath sets the SQLite database path
func (b *VectorBuilder) SetSQLitePath(path string) *VectorBuilder {
	if b.dbConfig == nil {
		b.dbConfig = make(map[string]interface{})
	}
	b.dbConfig["db_path"] = path
	return b
}

// SetMilvusEndpoint sets the Milvus endpoint
func (b *VectorBuilder) SetMilvusEndpoint(endpoint string) *VectorBuilder {
	if b.dbConfig == nil {
		b.dbConfig = make(map[string]interface{})
	}
	b.dbConfig["endpoint"] = endpoint
	return b
}

// Build constructs the vector storage components
func (b *VectorBuilder) Build() error {
	if err := b.validate(); err != nil {
		return err
	}

	// Build VectorDB
	vectorDB, err := b.buildVectorDB()
	if err != nil {
		return fmt.Errorf("failed to build vector database: %w", err)
	}
	b.vectorDB = vectorDB

	// Build EmbeddingGenerator
	embeddingGen, err := b.buildEmbeddingGenerator()
	if err != nil {
		return fmt.Errorf("failed to build embedding generator: %w", err)
	}
	b.embeddingGenerator = embeddingGen

	return nil
}

// GetVectorDB returns the built VectorDB instance
func (b *VectorBuilder) GetVectorDB() core.VectorDB {
	return b.vectorDB
}

// GetEmbeddingGenerator returns the built EmbeddingGenerator instance
func (b *VectorBuilder) GetEmbeddingGenerator() core.EmbeddingGenerator {
	return b.embeddingGenerator
}

// Private helper methods

// validate checks that all required fields are set
func (b *VectorBuilder) validate() error {
	if b.vectorDBType == "" {
		return fmt.Errorf("vector database type is required")
	}

	if b.embeddingModelType == "" {
		return fmt.Errorf("embedding model is required")
	}

	return nil
}

// buildVectorDB constructs the VectorDB instance
func (b *VectorBuilder) buildVectorDB() (core.VectorDB, error) {
	return b.vectorDBType.getVectorDB(b.dbConfig)
}

// buildEmbeddingGenerator constructs the EmbeddingGenerator instance
func (b *VectorBuilder) buildEmbeddingGenerator() (core.EmbeddingGenerator, error) {
	return b.embeddingModelType.getEmbeddingGenerator()
}
