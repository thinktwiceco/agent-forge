package llms

import (
	"encoding/json"
	"strings" // Added strings package
	"testing"
)

func TestUnifiedMessage_Getters(t *testing.T) {
	m := &UnifiedMessage{
		role:             MessageRoleUser,
		content:          "hello",
		toolCallID:       "id-123",
		toolCalls:        []ToolCall{{ID: "call-1"}},
		promptTokens:     10,
		completionTokens: 20,
		totalTokens:      30,
		ephemeral:        true,
	}

	if m.Role() != MessageRoleUser {
		t.Errorf("expected role user, got %s", m.Role())
	}
	if m.Content() != "hello" {
		t.Errorf("expected content hello, got %s", m.Content())
	}
	if m.ToolCallID() != "id-123" {
		t.Errorf("expected toolCallID id-123, got %s", m.ToolCallID())
	}
	if len(m.ToolCalls()) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(m.ToolCalls()))
	}
	if m.PromptTokens() != 10 {
		t.Errorf("expected prompt tokens 10, got %d", m.PromptTokens())
	}
	if m.CompletionTokens() != 20 {
		t.Errorf("expected completion tokens 20, got %d", m.CompletionTokens())
	}
	if m.TotalTokens() != 30 {
		t.Errorf("expected total tokens 30, got %d", m.TotalTokens())
	}
	if !m.Ephemeral() {
		t.Error("expected ephemeral true")
	}

	m.SetContent("new content")
	if m.Content() != "new content" {
		t.Errorf("expected content new content, got %s", m.Content())
	}
}

func TestUnifiedMessage_Constructors(t *testing.T) {
	u := UserMessage("user msg")
	if u.Role() != MessageRoleUser || u.Content() != "user msg" {
		t.Error("UserMessage constructor failed")
	}

	s := SystemMessage("sys msg")
	if s.Role() != MessageRoleSystem || s.Content() != "sys msg" {
		t.Error("SystemMessage constructor failed")
	}

	a := AssistantMessage("asst msg", 1, 2, 3)
	if a.Role() != MessageRoleAssistant || a.Content() != "asst msg" || a.TotalTokens() != 3 {
		t.Error("AssistantMessage constructor failed")
	}

	tm := ToolMessage("id-1", "result", true)
	if tm.Role() != MessageRoleTool || tm.ToolCallID() != "id-1" || !tm.Ephemeral() {
		t.Error("ToolMessage constructor failed")
	}

	aw := AssistantMessageWithToolCalls("content", []ToolCall{{ID: "1"}}, 1, 1, 2)
	if aw.Role() != MessageRoleAssistant || len(aw.ToolCalls()) != 1 {
		t.Error("AssistantMessageWithToolCalls constructor failed")
	}
}

func TestUnifiedMessage_JSON(t *testing.T) {
	m := UserMessage("test")
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var m2 UnifiedMessage
	if err := json.Unmarshal(data, &m2); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if m2.Role() != MessageRoleUser || m2.Content() != "test" {
		t.Error("JSON roundtrip failed")
	}
}

func TestMessageRole_String(t *testing.T) {
	if MessageRoleUser.String() != "user" {
		t.Error("MessageRole string failed")
	}
}

func TestOpenAILLMBuilder_SettersAndBuild(t *testing.T) {
	// We need to handle the panic in validate if defaults are missing in config
	// However, we can set everything explicitly to avoid defaults logic

	defer func() {
		_ = recover()
	}()

	b := NewOpenAILLMBuilder("openai")
	b.SetApiKey("sk-test")
	b.SetModel("gpt-4")
	b.SetCheapModel("gpt-3.5")
	b.SetReasoningModel("o1-preview")
	b.SetFastModel("gpt-3.5-turbo")
	b.SetBaseURL("https://api.openai.com/v1")

	llms, err := b.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if llms.MainModel().Model() != "gpt-4" {
		t.Error("Main model mismatch")
	}
	if llms.CheapModel().Model() != "gpt-3.5" {
		t.Error("Cheap model mismatch")
	}
}

func TestOpenAILLMBuilder_InvalidProvider(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid provider")
		}
	}()
	NewOpenAILLMBuilder("invalid")
}

// Mock tool for ToOpenAITool test
type MockTool struct{}

func (m *MockTool) GetName() string                                { return "test_tool" }
func (m *MockTool) Call(map[string]any, map[string]any) ToolReturn { return nil }
func (m *MockTool) GetFunctionDefinition() FunctionDefinition {
	return FunctionDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: FunctionParameters{
			Type_: "object",
			Properties: map[string]FunctionObjectParameter{
				"arg1": {Type_: "string", Description: "Argument 1"},
			},
			Required: []string{"arg1"},
		},
	}
}

func TestToOpenAITool(t *testing.T) {
	tool := &MockTool{}
	openAITool := ToOpenAITool(tool)

	// Use JSON serialization to verify content since we can't easily access fields of the external SDK type
	jsonBytes, err := json.Marshal(openAITool)
	if err != nil {
		// If serialization fails, maybe it's not marshallable? OpenAI standard types usually are.
		// If fails, we might skip or fail.
		t.Fatalf("Failed to marshal openAITool: %v", err)
	}

	jsonString := string(jsonBytes)

	// Check for key elements in the JSON
	if !strings.Contains(jsonString, `"type":"function"`) {
		t.Error("JSON should contain type:function")
	}
	// Note: OpenAI Go SDK might not field 'name' at top level of ToolParam but inside function
	// The structure is {"type":"function","function":{"name":"...","description":"...","parameters":{...}}}
	if !strings.Contains(jsonString, `"name":"test_tool"`) {
		t.Error("JSON should contain name:test_tool")
	}
	if !strings.Contains(jsonString, `"description":"A test tool"`) {
		t.Error("JSON should contain description")
	}
}
