package llms

import (
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/shared"
)

type FunctionObjectParameter struct {
	Type_       string `json:"type"`
	Description string `json:"description,omitempty"`
	Name        string `json:"name"`
}

type FunctionParameters struct {
	Type_      string                             `json:"type"`
	Properties map[string]FunctionObjectParameter `json:"properties"`
	Required   []string                           `json:"required,omitempty"`
}

type FunctionDefinition struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Parameters  FunctionParameters `json:"parameters"`
}

func ToOpenAITool(tool Tool) openai.ChatCompletionToolUnionParam {
	fnDef := tool.GetFunctionDefinition()

	return openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
		Name:        fnDef.Name,
		Description: param.Opt[string]{Value: fnDef.Description},
		Parameters: openai.FunctionParameters{
			"type":       fnDef.Parameters.Type_,
			"properties": fnDef.Parameters.Properties,
			"required":   fnDef.Parameters.Required,
		},
	})

}

// LLMEngine is the public interface for LLM engines with channel-based streaming.
//
// This interface provides the ChatStream method for streaming responses.
// The LLM engine is a pure API communication layer that converts messages and tools
// to the appropriate API format and streams responses back.
type LLMEngine interface {
	// ChatStream sends messages with optional tools and returns a ResponseCh for streaming responses.
	//
	// The engine converts the messages and tools to the appropriate API format,
	// makes the API call, and streams responses through the returned channel.
	//
	// Parameters:
	//   - messages: The messages to send
	//   - tools: Optional tools available for this request (can be nil or empty)
	//
	// Returns:
	//   - *responseCh: responseCh instance with channels for streaming
	ChatStream(messages []UnifiedMessage, tools []Tool) *responseCh
}

// Tool is an interface for tools that can be used by agents.
//
// This interface is compatible with the agent package's AgentTool interface
// but is defined locally to maintain package self-containment.
// External code can implement this interface to provide tools to agents.
type Tool interface {
	// GetName returns the name of the tool.
	GetName() string

	// Call executes the tool with the given context and arguments.
	Call(agentContext map[string]any, args map[string]any) toolReturn

	// GetFunctionDefinition returns the function definition of the tool.
	GetFunctionDefinition() FunctionDefinition
}

// ToolReturn represents the return value from a tool execution.
type ToolReturn interface {
	Success() bool
	Error() string
	Data() string
}

// toolReturn is an alias for ToolReturn to maintain backward compatibility.
type toolReturn = ToolReturn
