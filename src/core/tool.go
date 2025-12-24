package core

import (
	"fmt"

	"github.com/thinktwice/agentForge/src/llms"
)

// Hooks provides external validation hooks for tool execution
type Hooks interface {
	IsSafePath(path string) bool
	IsSafeCommand(cmd string) bool
}

// Parameter defines a tool parameter with validation
type Parameter struct {
	Name        string
	Type        string // "string", "number", "boolean", "object", "array"
	Description string
	Required    bool
	Validator   func(value any) error // Optional custom validation
}

// Tool is a universal tool implementation that satisfies both llms.Tool and agentforge.Discoverable interfaces
type Tool struct {
	name               string
	description        string
	advanceDescription string
	troubleshooting    string
	parameters         []Parameter
	handler            func(agentContext map[string]any, args map[string]any) llms.ToolReturn
	hooks              Hooks // Optional external validation hooks
}

// NewTool creates a new universal tool
func NewTool(
	name string,
	description string,
	advanceDescription string,
	troubleshooting string,
	params []Parameter,
	handler func(agentContext map[string]any, args map[string]any) llms.ToolReturn,
) llms.Tool {
	return &Tool{
		name:               name,
		description:        description,
		advanceDescription: advanceDescription,
		troubleshooting:    troubleshooting,
		parameters:         params,
		handler:            handler,
		hooks:              nil,
	}
}

// GetName returns the name of the tool (implements llms.Tool)
func (t *Tool) GetName() string {
	return t.name
}

// BasicDescription returns a short one-line description of the tool (implements agentforge.Discoverable)
func (t *Tool) BasicDescription() string {
	return t.description
}

// AdvanceDescription returns detailed information about the tool's capabilities (implements agentforge.Discoverable)
func (t *Tool) AdvanceDescription() string {
	return t.advanceDescription
}

// Troubleshooting returns information about common issues and debugging tips (implements agentforge.Discoverable)
func (t *Tool) Troubleshooting() string {
	return t.troubleshooting
}

// GetHooks returns the hooks interface for external validation (can be nil)
func (t *Tool) GetHooks() Hooks {
	return t.hooks
}

// SetHooks sets the hooks interface for external validation
func (t *Tool) SetHooks(hooks Hooks) {
	t.hooks = hooks
}

// GetFunctionDefinition returns the function definition for LLM API calls (implements llms.Tool)
func (t *Tool) GetFunctionDefinition() llms.FunctionDefinition {
	properties := make(map[string]llms.FunctionObjectParameter)
	var required []string

	for _, param := range t.parameters {
		properties[param.Name] = llms.FunctionObjectParameter{
			Type_:       param.Type,
			Description: param.Description,
			Name:        param.Name,
		}
		if param.Required {
			required = append(required, param.Name)
		}
	}

	return llms.FunctionDefinition{
		Name:        t.name,
		Description: t.description,
		Parameters: llms.FunctionParameters{
			Type_:      "object",
			Properties: properties,
			Required:   required,
		},
	}
}

// Call executes the tool with validation (implements llms.Tool)
func (t *Tool) Call(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	// Validate arguments
	validated, err := t.validateAndExtractArgs(args)
	if err != nil {
		return err
	}

	// Call handler with validated args
	return t.handler(agentContext, validated)
}

// validateAndExtractArgs validates arguments and extracts them with proper types
func (t *Tool) validateAndExtractArgs(args map[string]any) (map[string]any, llms.ToolReturn) {
	validated := make(map[string]any)

	for _, param := range t.parameters {
		value, exists := args[param.Name]

		// Check if required parameter is missing
		if param.Required && !exists {
			return nil, NewErrorResponse(fmt.Sprintf("missing required parameter: %s", param.Name))
		}

		if exists {
			// Type validation
			if err := t.validateType(value, param.Type); err != nil {
				return nil, NewErrorResponse(fmt.Sprintf("invalid type for %s: %v", param.Name, err))
			}

			// Custom validation
			if param.Validator != nil {
				if err := param.Validator(value); err != nil {
					return nil, NewErrorResponse(fmt.Sprintf("validation failed for %s: %v", param.Name, err))
				}
			}

			validated[param.Name] = value
		}
	}

	return validated, nil
}

// validateType checks if a value matches the expected type
func (t *Tool) validateType(value any, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "number":
		switch value.(type) {
		case float64, int, int64, int32, float32:
			// Valid number types
		default:
			return fmt.Errorf("expected number, got %T", value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("expected object, got %T", value)
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return fmt.Errorf("expected array, got %T", value)
		}
	}
	return nil
}
