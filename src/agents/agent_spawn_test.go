package agents

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/mocks"
	"github.com/thinktwiceco/agent-forge/src/tools/expand"
	"github.com/thinktwiceco/agent-forge/src/tools/meta"
)

func TestDrainSubagentResponse_FromDeltas(t *testing.T) {
	ch := make(chan core.ExtendedChunkResponse, 2)
	ch <- core.ExtendedChunkResponse{Content: "hello ", Status: llms.StatusStreaming}
	ch <- core.ExtendedChunkResponse{Delta: "world", Status: llms.StatusStreaming}
	close(ch)

	got, err := drainSubagentResponse(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello world" {
		t.Fatalf("got %q, want %q", got, "hello world")
	}
}

func TestDrainSubagentResponse_PrefersFullContent(t *testing.T) {
	ch := make(chan core.ExtendedChunkResponse, 1)
	ch <- core.ExtendedChunkResponse{FullContent: "complete answer", Status: llms.StatusCompleted}
	close(ch)

	got, err := drainSubagentResponse(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "complete answer" {
		t.Fatalf("got %q", got)
	}
}

func TestDrainSubagentResponse_EmptyIsError(t *testing.T) {
	ch := make(chan core.ExtendedChunkResponse)
	close(ch)

	_, err := drainSubagentResponse(ch)
	if err == nil {
		t.Fatal("expected error for empty subagent response")
	}
}

func TestFilterToolsForSubagent(t *testing.T) {
	tools := []llms.Tool{
		meta.NewMetaTool(),
		expand.NewExpandTool(),
		meta.NewMetaTool(),
	}
	filtered := filterToolsForSubagent(tools)
	if len(filtered) != 0 {
		t.Fatalf("expected reserved tools to be removed, got %d", len(filtered))
	}
}

func TestDedupeToolsByName(t *testing.T) {
	tools := []llms.Tool{meta.NewMetaTool(), meta.NewMetaTool(), expand.NewExpandTool()}
	deduped := dedupeToolsByName(tools)
	if len(deduped) != 2 {
		t.Fatalf("got %d tools, want 2", len(deduped))
	}
}

func findSpawnTool(t *testing.T, agent *Agent) llms.Tool {
	t.Helper()
	for _, tool := range agent.tools {
		if tool.GetName() == "spawn_subagent" {
			return tool
		}
	}
	t.Fatal("spawn_subagent tool not found")
	return nil
}

func spawnAgentContext(agent *Agent, chatID string) map[string]any {
	return map[string]any{
		"tools":  agent.tools,
		"chatId": chatID,
	}
}

func TestSpawnSubagentFactory_ReturnsSpawnIDAck(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	mockLLM.Responses = []string{"42 is the answer"}

	parent, err := NewBuilder(mockLLM, "parent").WithSpawnSubagent().Build()
	if err != nil {
		t.Fatalf("build parent: %v", err)
	}
	defer parent.Stop()

	spawnTool := findSpawnTool(t, parent)
	result := spawnTool.Call(spawnAgentContext(parent, "test-conv"), map[string]any{
		"prompt": "compute",
		"tools":  []any{},
	})
	if !result.Success() {
		t.Fatalf("spawn failed: %s", result.Error())
	}
	data := result.Data()
	if !strings.Contains(data, "spawn_id:") {
		t.Fatalf("got %q, want immediate spawn_id ack", data)
	}
	if strings.Contains(data, "42 is the answer") {
		t.Fatalf("tool result should not contain subagent answer synchronously: %q", data)
	}
}

func TestSpawnSubagentFactory_RequiresChatID(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	mockLLM.Responses = []string{"ok"}

	parent, err := NewBuilder(mockLLM, "parent").WithSpawnSubagent().Build()
	if err != nil {
		t.Fatalf("build parent: %v", err)
	}
	defer parent.Stop()

	spawnTool := findSpawnTool(t, parent)
	result := spawnTool.Call(map[string]any{"tools": parent.tools}, map[string]any{
		"prompt": "task",
		"tools":  []any{},
	})
	if result.Success() {
		t.Fatal("expected error when chatId is missing")
	}
}

func TestSpawnSubagentFactory_DedupesReservedTools(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	mockLLM.Responses = []string{"ok"}

	parent, err := NewBuilder(mockLLM, "parent").WithSpawnSubagent().Build()
	if err != nil {
		t.Fatalf("build parent: %v", err)
	}
	defer parent.Stop()

	spawnTool := findSpawnTool(t, parent)
	result := spawnTool.Call(spawnAgentContext(parent, "test-conv"), map[string]any{
		"prompt": "say ok",
		"tools":  []any{"meta", "expand", "todo_handler", "meta"},
	})
	if !result.Success() {
		t.Fatalf("spawn failed: %s", result.Error())
	}
}

func TestSpawnSubagentFactory_ReturnsImmediately(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	mockLLM.Responses = []string{"done"}
	mockLLM.ResponseDelay = 2 * time.Second

	parent, err := NewBuilder(mockLLM, "parent").WithSpawnSubagent().Build()
	if err != nil {
		t.Fatalf("build parent: %v", err)
	}
	defer parent.Stop()

	spawnTool := findSpawnTool(t, parent)

	start := time.Now()
	result := spawnTool.Call(spawnAgentContext(parent, "test-conv"), map[string]any{
		"prompt": "task",
		"tools":  []any{},
	})
	elapsed := time.Since(start)

	if !result.Success() {
		t.Fatalf("spawn failed: %s", result.Error())
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("spawn blocked for %v, want immediate return", elapsed)
	}
	if !strings.Contains(result.Data(), "spawn_id:") {
		t.Fatalf("got %q, want spawn_id ack", result.Data())
	}
}

func TestSpawnSubagentFactory_SubmitsResultOnCompletion(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	mockLLM.Responses = []string{"subagent answer", "presented to user"}

	parent, err := NewBuilder(mockLLM, "parent").WithSpawnSubagent().Build()
	if err != nil {
		t.Fatalf("build parent: %v", err)
	}
	defer parent.Stop()

	done := make(chan struct{}, 1)
	parent.SetTurnCompleteRouter(func(chatID, _ string) {
		if chatID == "test-conv" {
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})

	spawnTool := findSpawnTool(t, parent)
	result := spawnTool.Call(spawnAgentContext(parent, "test-conv"), map[string]any{
		"prompt": "task",
		"tools":  []any{},
	})
	if !result.Success() {
		t.Fatalf("spawn failed: %s", result.Error())
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for subagent completion turn")
	}

	if len(mockLLM.RecordedCalls) < 2 {
		t.Fatalf("expected subagent + parent follow-up LLM calls, got %d", len(mockLLM.RecordedCalls))
	}
	parentMsg := formatMessages(mockLLM.RecordedCalls[1])
	if !strings.Contains(parentMsg, "subagent answer") {
		t.Fatalf("parent turn message %q missing subagent result body", parentMsg)
	}
	if !strings.Contains(parentMsg, "task_type: subagent_result") {
		t.Fatalf("parent turn message %q missing subagent_result header", parentMsg)
	}
}

func formatMessages(msgs []*llms.UnifiedMessage) string {
	var sb strings.Builder
	for _, m := range msgs {
		sb.WriteString(m.Content())
	}
	return sb.String()
}

func TestSpawnSubagentFactory_SubmitsErrorOnFailure(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	mockLLM.Responses = []string{""}

	parent, err := NewBuilder(mockLLM, "parent").WithSpawnSubagent().Build()
	if err != nil {
		t.Fatalf("build parent: %v", err)
	}
	defer parent.Stop()

	done := make(chan struct{}, 1)
	parent.SetTurnCompleteRouter(func(chatID, _ string) {
		if chatID == "test-conv" {
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})

	spawnTool := findSpawnTool(t, parent)
	result := spawnTool.Call(spawnAgentContext(parent, "test-conv"), map[string]any{
		"prompt": "task",
		"tools":  []any{},
	})
	if !result.Success() {
		t.Fatalf("spawn failed: %s", result.Error())
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for subagent error completion turn")
	}

	if len(mockLLM.RecordedCalls) < 2 {
		t.Fatalf("expected subagent + parent follow-up LLM calls, got %d", len(mockLLM.RecordedCalls))
	}
	parentMsg := formatMessages(mockLLM.RecordedCalls[1])
	if !strings.Contains(parentMsg, "status: error") {
		t.Fatalf("parent turn message %q missing error status", parentMsg)
	}
}

func TestSpawnSubagentFactory_StopCancelsInFlight(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	mockLLM.Responses = []string{"late answer"}
	mockLLM.ResponseDelay = 3 * time.Second

	parent, err := NewBuilder(mockLLM, "parent").WithSpawnSubagent().Build()
	if err != nil {
		t.Fatalf("build parent: %v", err)
	}

	var mu sync.Mutex
	completions := 0
	parent.SetTurnCompleteRouter(func(chatID, _ string) {
		if chatID == "test-conv" {
			mu.Lock()
			completions++
			mu.Unlock()
		}
	})

	spawnTool := findSpawnTool(t, parent)
	result := spawnTool.Call(spawnAgentContext(parent, "test-conv"), map[string]any{
		"prompt": "task",
		"tools":  []any{},
	})
	if !result.Success() {
		t.Fatalf("spawn failed: %s", result.Error())
	}

	parent.Stop()
	time.Sleep(4 * time.Second)

	mu.Lock()
	defer mu.Unlock()
	if completions > 0 {
		t.Fatalf("expected no completion turn after Stop(), got %d", completions)
	}
}

func TestInitSystemTools_DedupesMetaExpand(t *testing.T) {
	mockLLM := mocks.NewMockLLMEngine()
	agent := NewAgent(&AgentConfig{
		AgentName: "t",
		LLMEngine: mockLLM,
		Tools:     []llms.Tool{meta.NewMetaTool(), expand.NewExpandTool()},
	})
	defer agent.Stop()

	names := map[string]int{}
	for _, tool := range agent.tools {
		names[tool.GetName()]++
	}
	for name, count := range names {
		if count > 1 {
			t.Fatalf("duplicate tool %q (%d times)", name, count)
		}
	}
}
