package knowledge

import (
	"strings"
)

// chunkText splits text into chunks of approximately chunkSize characters
// with overlap to maintain context between chunks
func chunkText(text string, chunkSize int, overlap int) []string {
	if chunkSize <= 0 {
		chunkSize = 1000 // default chunk size
	}
	if overlap < 0 {
		overlap = 200 // default overlap
	}

	if len(text) <= chunkSize {
		return []string{text}
	}

	var chunks []string
	start := 0

	for start < len(text) {
		end := start + chunkSize
		if end > len(text) {
			end = len(text)
		}

		chunk := text[start:end]

		// Try to break at sentence boundaries if not at the end
		if end < len(text) {
			// Look for sentence endings within the last 100 chars
			searchStart := len(chunk) - 100
			if searchStart < 0 {
				searchStart = 0
			}
			lastPeriod := strings.LastIndex(chunk[searchStart:], ".")
			lastNewline := strings.LastIndex(chunk[searchStart:], "\n")
			
			bestBreak := -1
			if lastPeriod > lastNewline && lastPeriod >= 0 {
				bestBreak = searchStart + lastPeriod + 1
			} else if lastNewline >= 0 {
				bestBreak = searchStart + lastNewline + 1
			}

			if bestBreak > 0 {
				chunk = chunk[:bestBreak]
				end = start + bestBreak
			}
		}

		chunks = append(chunks, strings.TrimSpace(chunk))
		
		// Move start forward, accounting for overlap
		start = end - overlap
		if start < 0 {
			start = 0
		}
		if start >= len(text) {
			break
		}
	}

	return chunks
}










