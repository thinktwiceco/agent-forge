package vector

import (
	"strings"
)

const (
	// maxTokensPerChunk is the maximum number of tokens per chunk.
	// Using 7000 to leave margin below the 8192 token limit for text-embedding-3-small
	maxTokensPerChunk = 7000

	// charsPerToken is a conservative estimate: ~4 characters per token for English text
	charsPerToken = 4

	// maxCharsPerChunk is the maximum characters per chunk based on token limit
	maxCharsPerChunk = maxTokensPerChunk * charsPerToken
)

// chunkText splits text into chunks that are within the token limit.
// It attempts to split at sentence boundaries when possible.
func chunkText(text string) []string {
	if len(text) <= maxCharsPerChunk {
		return []string{text}
	}

	var chunks []string
	remaining := text

	for len(remaining) > 0 {
		if len(remaining) <= maxCharsPerChunk {
			chunks = append(chunks, remaining)
			break
		}

		// Try to find a good split point (sentence boundary)
		chunk := remaining[:maxCharsPerChunk]

		// Look for sentence endings within the last 20% of the chunk
		searchStart := int(float64(len(chunk)) * 0.8)
		bestSplit := len(chunk)

		// Look for sentence endings (. ! ? followed by space or newline)
		for i := searchStart; i < len(chunk); i++ {
			char := chunk[i]
			// Check if we're at a sentence boundary
			if char == '.' || char == '!' || char == '?' {
				// Check if followed by space/newline or if we're at the end
				if i+1 >= len(chunk) {
					bestSplit = i + 1
					break
				}
				nextChar := chunk[i+1]
				if nextChar == ' ' || nextChar == '\n' || nextChar == '\r' {
					bestSplit = i + 1
					break
				}
			}
		}

		// If no sentence boundary found, try paragraph boundary (double newline)
		if bestSplit == len(chunk) {
			lastNewline := strings.LastIndex(chunk[:bestSplit], "\n\n")
			if lastNewline > searchStart {
				bestSplit = lastNewline + 2
			} else {
				// Try single newline
				lastNewline = strings.LastIndex(chunk[:bestSplit], "\n")
				if lastNewline > searchStart {
					bestSplit = lastNewline + 1
				}
			}
		}

		chunks = append(chunks, remaining[:bestSplit])
		remaining = strings.TrimSpace(remaining[bestSplit:])
	}

	return chunks
}
