package mocks

import (
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// MockTool is a mock implementation of llms.Tool
type MockTool struct {
	Name       string
	Definition llms.FunctionDefinition
	Called     bool
	LastArgs   map[string]any
	Result     llms.ToolReturn
	Calls      []map[string]any
}

func NewMockTool(name string) *MockTool {
	return &MockTool{
		Name: name,
		Definition: llms.FunctionDefinition{
			Name: name,
			Parameters: llms.FunctionParameters{
				Type_:      "object",
				Properties: make(map[string]llms.FunctionObjectParameter),
			},
		},
		Calls: []map[string]any{},
	}
}

func (t *MockTool) GetName() string {
	return t.Name
}

func (t *MockTool) Call(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	t.Called = true
	t.LastArgs = args
	t.Calls = append(t.Calls, args)

	if t.Result != nil {
		return t.Result
	}

	return &MockToolResult{
		SuccessVal: true,
		DataVal:    "mock success",
	}
}

func (t *MockTool) GetFunctionDefinition() llms.FunctionDefinition {
	return t.Definition
}

// MockToolResult implements llms.ToolReturn
type MockToolResult struct {
	SuccessVal   bool
	ErrorVal     string
	DataVal      string
	EphemeralVal bool
	CleanupFunc  func()
}

func (r *MockToolResult) Success() bool   { return r.SuccessVal }
func (r *MockToolResult) Error() string   { return r.ErrorVal }
func (r *MockToolResult) Data() string    { return r.DataVal }
func (r *MockToolResult) Ephemeral() bool { return r.EphemeralVal }
func (r *MockToolResult) Cleanup() func() { return r.CleanupFunc }
