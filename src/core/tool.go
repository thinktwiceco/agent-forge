package core

import (
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/llms"
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
	Items       map[string]interface{} // For array types, defines the schema of array items
	Validator   func(value any) error  // Optional custom validation
}

// Tool is a universal tool implementation that satisfies both llms.Tool and agentforge.Discoverable interfaces
type Tool struct {
	Name                string
	Description         string
	AdvanceDesc         string // Public field - use AdvanceDescription() method to access via interface
	TroubleshootingInfo string // Public field - use Troubleshooting() method to access via interface
	Parameters          []Parameter
	Handler             func(agentContext map[string]any, args map[string]any) llms.ToolReturn
	Hooks               Hooks // Optional external validation hooks
}

// GetName returns the name of the tool (implements llms.Tool)
func (t *Tool) GetName() string {
	return t.Name
}

// BasicDescription returns a short one-line description of the tool (implements agentforge.Discoverable)
func (t *Tool) BasicDescription() string {
	return t.Description
}

// AdvanceDescription returns detailed information about the tool's capabilities (implements agentforge.Discoverable)
func (t *Tool) AdvanceDescription() string {
	return t.AdvanceDesc
}

// Troubleshooting returns information about common issues and debugging tips (implements agentforge.Discoverable)
func (t *Tool) Troubleshooting() string {
	return t.TroubleshootingInfo
}

// GetHooks returns the hooks interface for external validation (can be nil)
func (t *Tool) GetHooks() Hooks {
	return t.Hooks
}

// SetHooks sets the hooks interface for external validation
func (t *Tool) SetHooks(hooks Hooks) {
	t.Hooks = hooks
}

// GetFunctionDefinition returns the function definition for LLM API calls (implements llms.Tool)
func (t *Tool) GetFunctionDefinition() llms.FunctionDefinition {
	properties := make(map[string]llms.FunctionObjectParameter)
	var required []string

	for _, param := range t.Parameters {
		prop := llms.FunctionObjectParameter{
			Type_:       param.Type,
			Description: param.Description,
			Name:        param.Name,
		}
		// For array types, include items schema if provided
		if param.Type == "array" && param.Items != nil {
			prop.Items = param.Items
		}
		properties[param.Name] = prop
		if param.Required {
			required = append(required, param.Name)
		}
	}

	return llms.FunctionDefinition{
		Name:        t.Name,
		Description: t.Description,
		Parameters: llms.FunctionParameters{
			Type_:      "object",
			Properties: properties,
			Required:   required,
		},
	}
}

// Call executes the tool with validation (implements llms.Tool)
//
// Tool Usage Guidelines:
//   - Tools receive agentContext as a map[string]any for read access
//   - Tools can read any field from the context map directly
//   - Tools that need to modify context should:
//     1. Use RehydrateContext() to get the struct
//     2. Use helper methods (e.g., SetLastSubagentMessage()) to modify fields
//     3. Update the context map with changes (e.g., ctx["lastSubagentMessage"] = value)
//   - Changes to mutable fields (LastSubagentMessage, PluginFields) will be synced back
//     to the struct automatically after tool execution
//   - SessionStorage is shared by reference - modifications persist automatically
//
// Parameters:
//   - agentContext: The agent context map containing agent state and configuration
//   - args: The tool arguments as a map
//
// Returns:
//   - llms.ToolReturn: The result of tool execution
func (t *Tool) Call(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	// Validate arguments
	validated, err := t.validateAndExtractArgs(args)
	if err != nil {
		return err
	}

	// Call handler with validated args
	return t.Handler(agentContext, validated)
}

// validateAndExtractArgs validates arguments and extracts them with proper types
func (t *Tool) validateAndExtractArgs(args map[string]any) (map[string]any, llms.ToolReturn) {
	validated := make(map[string]any)

	for _, param := range t.Parameters {
		value, exists := args[param.Name]

		// Check if required parameter is missing
		if param.Required && !exists {
			return nil, NewErrorResponse(fmt.Sprintf("missing required parameter: %s. Hint: Use the expand tool with subject_name='%s' to get more information about how to use this tool", param.Name, t.Name))
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
