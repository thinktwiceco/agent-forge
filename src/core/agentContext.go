package core

import (
	"fmt"

	"github.com/thinktwice/agentForge/src/llms"
)

// AgentContext holds static agent context information that is built once
// at agent instantiation. This avoids rebuilding the context on every tool execution.
type AgentContext struct {
	// AgentName is the name of the agent
	AgentName string
	// Trace is optional trace information (e.g., "thinking", "response")
	Trace string
	// Model is the name of the LLM model being used
	Model string
	// Tools is the list of tools available to the agent
	Tools []llms.Tool
	// SubAgents is the list of sub-agents available for delegation
	SubAgents []*SubAgent
}

// BuildContext converts the AgentContext struct to a map[string]any and merges
// in the session-specific responseCh parameter.
//
// Parameters:
//   - responseCh: The response channel for the current chat session
//
// Returns:
//   - map[string]any: The complete agent context map ready for tool execution
func (ac *AgentContext) BuildContext(responseCh *ResponseCh) map[string]any {
	context := make(map[string]any)
	context["agentName"] = ac.AgentName
	context["trace"] = ac.Trace
	context["model"] = ac.Model
	context["responseCh"] = responseCh
	context["tools"] = ac.Tools
	context["subAgents"] = ac.SubAgents
	return context
}

// RehydrateContext reconstructs an AgentContext from a map[string]any.
// It performs type assertions to ensure all fields are correctly typed.
//
// Parameters:
//   - contextMap: The map containing agent context data (typically from BuildContext)
//
// Returns:
//   - *AgentContext: A new AgentContext instance with rehydrated data
//   - error: An error if any type assertion fails or required fields are missing
func RehydrateContext(contextMap map[string]any) (*AgentContext, error) {
	ac := &AgentContext{}

	if err := ac.validateAgentName(contextMap); err != nil {
		return nil, err
	}
	if err := ac.validateTrace(contextMap); err != nil {
		return nil, err
	}
	if err := ac.validateModel(contextMap); err != nil {
		return nil, err
	}
	if err := ac.validateTools(contextMap); err != nil {
		return nil, err
	}
	if err := ac.validateSubAgents(contextMap); err != nil {
		return nil, err
	}

	return ac, nil
}

// validateAgentName extracts and validates the agentName field from the context map.
func (ac *AgentContext) validateAgentName(contextMap map[string]any) error {
	if agentName, ok := contextMap["agentName"].(string); ok {
		ac.AgentName = agentName
	} else if agentName, exists := contextMap["agentName"]; exists {
		return fmt.Errorf("agentName must be a string, got %T", agentName)
	}
	return nil
}

// validateTrace extracts and validates the trace field from the context map.
func (ac *AgentContext) validateTrace(contextMap map[string]any) error {
	if trace, ok := contextMap["trace"].(string); ok {
		ac.Trace = trace
	} else if trace, exists := contextMap["trace"]; exists {
		return fmt.Errorf("trace must be a string, got %T", trace)
	}
	return nil
}

// validateModel extracts and validates the model field from the context map.
func (ac *AgentContext) validateModel(contextMap map[string]any) error {
	if model, ok := contextMap["model"].(string); ok {
		ac.Model = model
	} else if model, exists := contextMap["model"]; exists {
		return fmt.Errorf("model must be a string, got %T", model)
	}
	return nil
}

// validateTools extracts and validates the tools field from the context map.
func (ac *AgentContext) validateTools(contextMap map[string]any) error {
	if tools, ok := contextMap["tools"].([]llms.Tool); ok {
		ac.Tools = tools
		return nil
	}

	toolsRaw, exists := contextMap["tools"]
	if !exists {
		return nil
	}

	// Handle case where tools might be []interface{} from JSON unmarshaling
	toolsSlice, ok := toolsRaw.([]interface{})
	if !ok {
		return fmt.Errorf("tools must be []llms.Tool, got %T", toolsRaw)
	}

	tools := make([]llms.Tool, 0, len(toolsSlice))
	for i, toolRaw := range toolsSlice {
		tool, ok := toolRaw.(llms.Tool)
		if !ok {
			return fmt.Errorf("tools[%d] must implement llms.Tool, got %T", i, toolRaw)
		}
		tools = append(tools, tool)
	}
	ac.Tools = tools
	return nil
}

// validateSubAgents extracts and validates the subAgents field from the context map.
func (ac *AgentContext) validateSubAgents(contextMap map[string]any) error {
	subAgentsRaw, exists := contextMap["subAgents"]
	if !exists {
		return nil
	}

	if subAgents, ok := subAgentsRaw.([]*SubAgent); ok {
		ac.SubAgents = subAgents
		return nil
	}

	// Handle case where subAgents might be []interface{} from JSON unmarshaling
	subAgentsSlice, ok := subAgentsRaw.([]interface{})
	if !ok {
		return fmt.Errorf("subAgents must be []*SubAgent or []interface{}, got %T", subAgentsRaw)
	}

	subAgents := make([]*SubAgent, 0, len(subAgentsSlice))
	for i, subAgentRaw := range subAgentsSlice {
		// Try to assert as *SubAgent directly
		// This works if the original value was stored as a pointer
		subAgentPtr, ok := subAgentRaw.(*SubAgent)
		if !ok {
			// If direct assertion fails, check if it implements SubAgent
			// Note: We cannot convert a non-pointer value to *SubAgent
			// In practice, SubAgents should always be stored as pointers
			return fmt.Errorf("subAgents[%d] must be *SubAgent (pointer to SubAgent implementer), got %T", i, subAgentRaw)
		}
		subAgents = append(subAgents, subAgentPtr)
	}
	ac.SubAgents = subAgents
	return nil
}
