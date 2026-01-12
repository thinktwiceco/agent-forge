package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"unicode"

	agentforge "github.com/thinktwice/agentForge/src"
)

// embeddingService handles generating embeddings using a local, open-source approach
// Uses TF-IDF-like weighting with word hashing for semantic similarity
type embeddingService struct {
	dimensions int
	vocab      map[string]int // word vocabulary for better embeddings
}

// newEmbeddingService creates a new embedding service
// Uses a simple but effective embedding approach that works entirely locally
func newEmbeddingService() (*embeddingService, error) {
	// Use 384 dimensions to match common embedding models
	dimensions := 384

	return &embeddingService{
		dimensions: dimensions,
		vocab:      make(map[string]int),
	}, nil
}

// generateEmbedding generates an embedding for the given text
func (es *embeddingService) generateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := es.generateEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding generated")
	}
	return embeddings[0], nil
}

// generateEmbeddings generates embeddings for multiple texts
func (es *embeddingService) generateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	for i, text := range texts {
		embedding := es.computeEmbedding(text)
		embeddings[i] = embedding
	}

	return embeddings, nil
}

// computeEmbedding computes an embedding vector for the given text
// Uses a combination of word hashing and TF-IDF-like weighting
func (es *embeddingService) computeEmbedding(text string) []float32 {
	// Tokenize text
	words := tokenize(text)
	if len(words) == 0 {
		// Return zero vector for empty text
		return make([]float32, es.dimensions)
	}

	// Initialize embedding vector
	embedding := make([]float32, es.dimensions)

	// Compute word frequencies for TF weighting
	wordFreq := make(map[string]float32)
	for _, word := range words {
		wordFreq[word]++
	}

	// Generate embedding using word hashing
	for _, word := range words {
		// Create multiple hash features for each word (like feature hashing)
		for dim := 0; dim < es.dimensions; dim++ {
			hash := hashWord(word, dim)
			// Use TF weighting
			tf := wordFreq[word] / float32(len(words))
			// Add weighted hash contribution
			embedding[dim] += float32(hash%1000-500) / 500.0 * tf
		}
	}

	// Normalize the embedding vector
	norm := float32(0)
	for _, v := range embedding {
		norm += v * v
	}
	if norm > 0 {
		norm = float32(math.Sqrt(float64(norm)))
		for i := range embedding {
			embedding[i] /= norm
		}
	}

	return embedding
}

// tokenize tokenizes text into words
func tokenize(text string) []string {
	// Convert to lowercase and split
	text = strings.ToLower(text)

	// Split on whitespace and punctuation
	var words []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				word := current.String()
				if len(word) > 1 { // Filter out single characters
					words = append(words, word)
				}
				current.Reset()
			}
		}
	}

	// Add last word if any
	if current.Len() > 0 {
		word := current.String()
		if len(word) > 1 {
			words = append(words, word)
		}
	}

	return words
}

// hashWord creates a deterministic hash for a word with a dimension offset
func hashWord(word string, dimension int) uint32 {
	hash := uint32(2166136261) // FNV-1a offset basis
	for _, c := range word {
		hash ^= uint32(c)
		hash *= 16777619 // FNV-1a prime
	}
	// Mix in dimension to create different hash per dimension
	hash ^= uint32(dimension)
	hash *= 16777619
	return hash
}

// close cleans up resources (no-op for this implementation)
func (es *embeddingService) close() error {
	return nil
}

// embeddingToJSON converts an embedding vector to JSON for storage
func embeddingToJSON(embedding []float32) (string, error) {
	data, err := json.Marshal(embedding)
	if err != nil {
		return "", fmt.Errorf("failed to marshal embedding: %w", err)
	}
	return string(data), nil
}

// jsonToEmbedding converts JSON string back to embedding vector
func jsonToEmbedding(jsonStr string) ([]float32, error) {
	var embedding []float32
	if err := json.Unmarshal([]byte(jsonStr), &embedding); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embedding: %w", err)
	}
	return embedding, nil
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		agentforge.Warn("Vector length mismatch: %d != %d", len(a), len(b))
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

// sqrt calculates square root (simple implementation)
func sqrt(x float32) float32 {
	// Using Newton's method for square root
	if x == 0 {
		return 0
	}
	guess := x / 2
	for i := 0; i < 10; i++ {
		guess = (guess + x/guess) / 2
	}
	return guess
}










