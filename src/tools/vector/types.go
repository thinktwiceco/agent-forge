package vector

import "github.com/thinktwiceco/agent-forge/src/core"

// Vector represents the vector tool instance with its dependencies.
type Vector struct {
	vectorDB           core.VectorDB
	embeddingGenerator core.EmbeddingGenerator
}
