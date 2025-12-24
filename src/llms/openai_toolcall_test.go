package llms

import (
	"encoding/json"
	"testing"
)

// TestToolCallSerialization tests that ToolCall can be properly serialized and deserialized
func TestToolCallSerialization(t *testing.T) {
	toolCall := ToolCall{
		ID:   "call_abc123",
		Name: "foo",
		Arguments: map[string]any{
			"echo":  "Hello, world!",
			"count": float64(42),
		},
	}

	// Serialize
	jsonBytes, err := json.Marshal(toolCall)
	if err != nil {
		t.Fatalf("Failed to serialize tool call: %v", err)
	}

	// Deserialize
	var deserialized ToolCall
	if err := json.Unmarshal(jsonBytes, &deserialized); err != nil {
		t.Fatalf("Failed to deserialize tool call: %v", err)
	}

	// Verify
	if deserialized.ID != toolCall.ID {
		t.Errorf("ID mismatch: got %s, want %s", deserialized.ID, toolCall.ID)
	}
	if deserialized.Name != toolCall.Name {
		t.Errorf("Name mismatch: got %s, want %s", deserialized.Name, toolCall.Name)
	}
	if deserialized.Arguments["echo"] != "Hello, world!" {
		t.Errorf("Arguments[echo] mismatch: got %v, want %v", deserialized.Arguments["echo"], "Hello, world!")
	}
	if deserialized.Arguments["count"] != float64(42) {
		t.Errorf("Arguments[count] mismatch: got %v, want %v", deserialized.Arguments["count"], float64(42))
	}
}

// TestChunkResponseWithToolCalls tests that ChunkResponse with ToolCalls can be serialized
func TestChunkResponseWithToolCalls(t *testing.T) {
	toolCalls := []ToolCall{
		{
			ID:   "call_1",
			Name: "foo",
			Arguments: map[string]any{
				"echo": "test1",
			},
		},
		{
			ID:   "call_2",
			Name: "bar",
			Arguments: map[string]any{
				"value": float64(123),
			},
		},
	}

	chunk := ChunkResponse{
		Content:     "",
		Delta:       "",
		FullContent: "Some content",
		Status:      StatusToolCall,
		Type:        TypeToolCall,
		ToolCalls:   toolCalls,
	}

	// Serialize
	jsonBytes, err := serializeChunk(chunk)
	if err != nil {
		t.Fatalf("Failed to serialize chunk: %v", err)
	}

	// Deserialize
	var deserialized ChunkResponse
	if err := json.Unmarshal(jsonBytes, &deserialized); err != nil {
		t.Fatalf("Failed to deserialize chunk: %v", err)
	}

	// Verify
	if deserialized.Status != StatusToolCall {
		t.Errorf("Status mismatch: got %s, want %s", deserialized.Status, StatusToolCall)
	}
	if len(deserialized.ToolCalls) != 2 {
		t.Fatalf("ToolCalls length mismatch: got %d, want 2", len(deserialized.ToolCalls))
	}
	if deserialized.ToolCalls[0].Name != "foo" {
		t.Errorf("First tool call name mismatch: got %s, want foo", deserialized.ToolCalls[0].Name)
	}
	if deserialized.ToolCalls[1].Name != "bar" {
		t.Errorf("Second tool call name mismatch: got %s, want bar", deserialized.ToolCalls[1].Name)
	}
}

// TestToolCallAccumulation tests the logic of accumulating tool call deltas
func TestToolCallAccumulation(t *testing.T) {
	// Simulate accumulating tool call data like in streamResponse
	toolCallsMap := make(map[int]*struct {
		ID        string
		Name      string
		Arguments string
	})

	// First delta - ID and name
	idx := 0
	toolCallsMap[idx] = &struct {
		ID        string
		Name      string
		Arguments string
	}{
		ID:   "call_abc123",
		Name: "foo",
	}

	// Second delta - partial arguments
	toolCallsMap[idx].Arguments += `{"echo"`

	// Third delta - more arguments
	toolCallsMap[idx].Arguments += `:"Hello`

	// Fourth delta - complete arguments
	toolCallsMap[idx].Arguments += `, world!"}`

	// Parse final arguments
	var args map[string]any
	if err := json.Unmarshal([]byte(toolCallsMap[idx].Arguments), &args); err != nil {
		t.Fatalf("Failed to parse accumulated arguments: %v", err)
	}

	// Verify
	if args["echo"] != "Hello, world!" {
		t.Errorf("Accumulated arguments mismatch: got %v, want 'Hello, world!'", args["echo"])
	}
}
