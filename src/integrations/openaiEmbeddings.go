package integrations

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	agentforge "github.com/thinktwice/agentForge/src"
)

// OpenAIEmbeddingGenerator implements the EmbeddingGenerator interface using OpenAI's embedding API.
type OpenAIEmbeddingGenerator struct {
	client openai.Client
	model  string
	apiKey string
}

// NewOpenAIEmbeddingGenerator creates a new OpenAI embedding generator.
//
// Parameters:
//   - apiKey: OpenAI API key (empty string will try to load from config)
//   - model: Embedding model name (e.g., "text-embedding-3-small", "text-embedding-ada-002")
//
// Returns:
//   - *OpenAIEmbeddingGenerator: A new embedding generator instance
func NewOpenAIEmbeddingGenerator(apiKey, model string) (*OpenAIEmbeddingGenerator, error) {
	if apiKey == "" {
		// Try to get from config
		config, err := agentforge.NewConfig()
		if err == nil {
			apiKey = config.AFOpenAIAPIKey
		}
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI API key is required")
		}
	}

	if model == "" {
		model = "text-embedding-3-small" // Default model
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	return &OpenAIEmbeddingGenerator{
		client: client,
		model:  model,
		apiKey: apiKey,
	}, nil
}

// GenerateEmbedding generates an embedding vector for the given text.
// Implements core.EmbeddingGenerator interface.
func (g *OpenAIEmbeddingGenerator) GenerateEmbedding(text string) ([]float32, string, error) {
	ctx := context.Background()

	params := openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: []string{text},
		},
		Model: openai.EmbeddingModel(g.model),
	}

	resp, err := g.client.Embeddings.New(ctx, params)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, "", fmt.Errorf("no embedding data returned")
	}

	// Convert to []float32
	embeddingData := resp.Data[0].Embedding
	embedding := make([]float32, len(embeddingData))
	for i, v := range embeddingData {
		embedding[i] = float32(v)
	}

	return embedding, g.model, nil
}
