package tools

import (
	"fmt"

	"github.com/thinktwice/agentForge/src/llms"
)

// Parameter defines a tool parameter with validation
type Parameter struct {
	Name        string
	Type        string // "string", "number", "boolean", "object", "array"
	Description string
	Required    bool
	Validator   func(value any) error // Optional custom validation
}

// BaseTool provides common functionality for tool implementations
type BaseTool struct {
	name               string
	description        string
	advanceDescription string
	troubleshooting    string
	parameters         []Parameter
}

// NewBaseTool creates a new base tool
func NewBaseTool(name, description, advanceDescription, troubleshooting string, params []Parameter) *BaseTool {
	return &BaseTool{
		name:               name,
		description:        description,
		advanceDescription: advanceDescription,
		troubleshooting:    troubleshooting,
		parameters:         params,
	}
}

func (b *BaseTool) GetName() string {
	return b.name
}

// BasicDescription returns a short one-line description of the tool.
// This implements the agentforge.Discoverable interface.
func (b *BaseTool) BasicDescription() string {
	return b.description
}

// AdvanceDescription returns detailed information about the tool's
// capabilities, parameters, usage patterns, and behavior.
// This implements the agentforge.Discoverable interface.
func (b *BaseTool) AdvanceDescription() string {
	return b.advanceDescription
}

// Troubleshooting returns information about common issues, debugging tips,
// validation errors, and configuration guidance for this tool.
// This implements the agentforge.Discoverable interface.
func (b *BaseTool) Troubleshooting() string {
	return b.troubleshooting
}

func (b *BaseTool) GetFunctionDefinition() llms.FunctionDefinition {
	properties := make(map[string]llms.FunctionObjectParameter)
	var required []string

	for _, param := range b.parameters {
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
		Name:        b.name,
		Description: b.description,
		Parameters: llms.FunctionParameters{
			Type_:      "object",
			Properties: properties,
			Required:   required,
		},
	}
}

// ValidateAndExtractArgs validates arguments and extracts them with proper types
func (b *BaseTool) ValidateAndExtractArgs(args map[string]any) (map[string]any, llms.ToolReturn) {
	validated := make(map[string]any)

	for _, param := range b.parameters {
		value, exists := args[param.Name]

		// Check if required parameter is missing
		if param.Required && !exists {
			return nil, NewErrorResponse(fmt.Sprintf("missing required parameter: %s", param.Name))
		}

		if exists {
			// Type validation
			if err := b.validateType(value, param.Type); err != nil {
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

func (b *BaseTool) validateType(value any, expectedType string) error {
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

// Helper functions for creating responses

// NewSuccessResponse creates a successful tool response
func NewSuccessResponse(data string) llms.ToolReturn {
	return &ToolResponse{
		success: true,
		error:   "",
		data:    data,
	}
}

// NewErrorResponse creates an error response without data
func NewErrorResponse(errorMsg string) llms.ToolReturn {
	return &ToolResponse{
		success: false,
		error:   errorMsg,
		data:    "",
	}
}

// NewFailureResponse creates an error response with partial data
func NewFailureResponse(errorMsg, data string) llms.ToolReturn {
	return &ToolResponse{
		success: false,
		error:   errorMsg,
		data:    data,
	}
}

// SimpleTool wraps a simple function as a tool
type SimpleTool struct {
	*BaseTool
	handler func(args map[string]any) llms.ToolReturn
}

// NewSimpleTool creates a tool from a handler function
func NewSimpleTool(name, description, advanceDescription, troubleshooting string, params []Parameter, handler func(map[string]any) llms.ToolReturn) llms.Tool {
	return &SimpleTool{
		BaseTool: NewBaseTool(name, description, advanceDescription, troubleshooting, params),
		handler:  handler,
	}
}

func (s *SimpleTool) Call(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	// Validate arguments
	validated, err := s.ValidateAndExtractArgs(args)
	if err != nil {
		return err
	}

	// Call handler with validated args
	return s.handler(validated)
}

// ContextAwareTool wraps a function that needs agent context
type ContextAwareTool struct {
	*BaseTool
	handler func(agentContext map[string]any, args map[string]any) llms.ToolReturn
}

// NewContextAwareTool creates a tool that receives agent context
func NewContextAwareTool(name, description, advanceDescription, troubleshooting string, params []Parameter, handler func(map[string]any, map[string]any) llms.ToolReturn) llms.Tool {
	return &ContextAwareTool{
		BaseTool: NewBaseTool(name, description, advanceDescription, troubleshooting, params),
		handler:  handler,
	}
}

func (c *ContextAwareTool) Call(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	validated, err := c.ValidateAndExtractArgs(args)
	if err != nil {
		return err
	}

	return c.handler(agentContext, validated)
}
