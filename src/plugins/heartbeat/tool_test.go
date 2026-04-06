package heartbeat

import (
	"encoding/json"
	"strings"
	"testing"
)

func invoke(m *HeartbeatManager, action string, extra map[string]any) (string, bool) {
	tool := newHeartbeatManagerTool(m)
	args := map[string]any{"action": action}
	for k, v := range extra {
		args[k] = v
	}
	result := tool.Call(nil, args)
	if result.Success() {
		return result.Data(), false
	}
	return result.Error(), true
}

func TestTool_AddInstruction_OK(t *testing.T) {
	m := NewHeartbeatManager()
	data, isErr := invoke(m, "add_instruction", map[string]any{
		"title":       "Daily check",
		"instruction": "Review the logs.",
	})
	if isErr {
		t.Fatalf("expected success, got error: %s", data)
	}
	if !strings.Contains(data, "Daily check") {
		t.Fatalf("unexpected response: %s", data)
	}
}

func TestTool_AddInstruction_EmptyTitle(t *testing.T) {
	m := NewHeartbeatManager()
	_, isErr := invoke(m, "add_instruction", map[string]any{
		"title":       "",
		"instruction": "body",
	})
	if !isErr {
		t.Fatal("expected error for empty title")
	}
}

func TestTool_RemoveInstruction_OK(t *testing.T) {
	m := NewHeartbeatManager()
	_ = m.AddInstruction("Temp", "body")
	data, isErr := invoke(m, "remove_instruction", map[string]any{"title": "Temp"})
	if isErr {
		t.Fatalf("expected success, got error: %s", data)
	}
}

func TestTool_RemoveInstruction_NotFound(t *testing.T) {
	m := NewHeartbeatManager()
	_, isErr := invoke(m, "remove_instruction", map[string]any{"title": "Ghost"})
	if !isErr {
		t.Fatal("expected error for unknown title")
	}
}

func TestTool_ListInstructions_Empty(t *testing.T) {
	m := NewHeartbeatManager()
	data, isErr := invoke(m, "list_instructions", nil)
	if isErr {
		t.Fatalf("expected success, got error: %s", data)
	}
	var titles []string
	if err := json.Unmarshal([]byte(data), &titles); err != nil {
		t.Fatalf("response is not valid JSON: %v — data: %s", err, data)
	}
	if len(titles) != 0 {
		t.Fatalf("expected empty array, got %v", titles)
	}
}

func TestTool_ListInstructions_WithItems(t *testing.T) {
	m := NewHeartbeatManager()
	_ = m.AddInstruction("Beta", "b")
	_ = m.AddInstruction("Alpha", "a")
	data, isErr := invoke(m, "list_instructions", nil)
	if isErr {
		t.Fatalf("expected success, got error: %s", data)
	}
	var titles []string
	if err := json.Unmarshal([]byte(data), &titles); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(titles) != 2 || titles[0] != "Alpha" || titles[1] != "Beta" {
		t.Fatalf("unexpected titles: %v", titles)
	}
}

func TestTool_UnknownAction(t *testing.T) {
	m := NewHeartbeatManager()
	_, isErr := invoke(m, "fly_to_moon", nil)
	if !isErr {
		t.Fatal("expected error for unknown action")
	}
}

func TestTool_MissingAction(t *testing.T) {
	m := NewHeartbeatManager()
	tool := newHeartbeatManagerTool(m)
	result := tool.Call(nil, map[string]any{})
	if result.Success() {
		t.Fatal("expected error when action is missing")
	}
}
