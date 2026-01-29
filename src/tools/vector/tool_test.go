package vector

import (
	"errors"
	"testing"

	"github.com/thinktwiceco/agent-forge/src/core"
)

// MockVectorDB
type mockVectorDB struct {
	storedDocs map[string]core.DocumentSummary
	indexErr   error
	searchErr  error
	deleteErr  error
}

func (m *mockVectorDB) Index(embedding []float32, text string, metadata map[string]any, model string) (string, error) {
	if m.indexErr != nil {
		return "", m.indexErr
	}
	id := "doc-123"
	if m.storedDocs == nil {
		m.storedDocs = make(map[string]core.DocumentSummary)
	}
	m.storedDocs[id] = core.DocumentSummary{
		DocumentID: id,
		Text:       text,
		Metadata:   metadata,
	}
	return id, nil
}

func (m *mockVectorDB) Search(emb []float32, topK int, filters map[string]any) ([]core.SearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return []core.SearchResult{
		{DocumentID: "doc-1", Text: "match", Score: 0.9},
	}, nil
}

func (m *mockVectorDB) ListDocuments(opts core.ListOptions) ([]core.DocumentSummary, int, error) {
	return []core.DocumentSummary{
		{DocumentID: "doc-1", Text: "match"},
	}, 1, nil
}

func (m *mockVectorDB) Delete(id string) error {
	return m.deleteErr
}

// MockEmbeddingGenerator
type mockEmbedder struct {
	err error
}

func (m *mockEmbedder) GenerateEmbedding(text string) ([]float32, string, error) {
	if m.err != nil {
		return nil, "", m.err
	}
	return []float32{0.1, 0.2, 0.3}, "mock-model", nil
}
func (m *mockEmbedder) ModelName() string { return "mock-embedal" }

func TestVectorTool_Handlers(t *testing.T) {
	mockDB := &mockVectorDB{}
	mockEmb := &mockEmbedder{}
	tool := NewVectorTool(mockDB, mockEmb)

	// 1. Index
	t.Run("index_success", func(t *testing.T) {
		args := map[string]any{
			"action": "index",
			"text":   "hello world",
		}
		result := tool.Call(nil, args)
		if !result.Success() {
			t.Errorf("Index failed: %s", result.Error())
		}
		if result.Data() != "Document indexed successfully with ID: doc-123" {
			t.Errorf("Unexpected output: %s", result.Data())
		}
	})

	// 2. Search
	t.Run("search_success", func(t *testing.T) {
		args := map[string]any{
			"action": "search",
			"query":  "hello",
		}
		result := tool.Call(nil, args)
		if !result.Success() {
			t.Errorf("Search failed: %s", result.Error())
		}
	})

	// 3. Delete
	t.Run("delete_success", func(t *testing.T) {
		args := map[string]any{
			"action":      "delete",
			"document_id": "doc-123",
		}
		result := tool.Call(nil, args)
		if !result.Success() {
			t.Errorf("Delete failed: %s", result.Error())
		}
	})

	// 4. Missing Action
	t.Run("missing_action", func(t *testing.T) {
		args := map[string]any{}
		// Call directly or via Tool.Call validator?
		// Tool logic validates params before Handler, but Handler also checks.
		// Since "action" is required in params, validation fails first.
		result := tool.Call(nil, args)
		if result.Success() {
			t.Error("Expected failure for missing action")
		}
	})

	// 5. Index Failure (Embedding)
	t.Run("index_embed_fail", func(t *testing.T) {
		badEmb := &mockEmbedder{err: errors.New("embed fail")}
		toolBad := NewVectorTool(mockDB, badEmb)
		args := map[string]any{
			"action": "index",
			"text":   "fail",
		}
		result := toolBad.Call(nil, args)
		if result.Success() {
			t.Error("Expected index failure due to embedding error")
		}
	})
}
