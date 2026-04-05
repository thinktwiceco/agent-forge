package core

import (
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/llms"
)

// AgentContext holds static agent context information that is built once
// at agent instantiation. This avoids rebuilding the context on every tool execution.
//
// Field Mutability:
//   - Immutable fields (set at initialization, never change): AgentName, Trace, Model, Tools, ResponseCh
//   - Mutable fields (can change during tool execution): SessionStorage, PluginFields
type AgentContext struct {
	// AgentName is the name of the agent (immutable)
	AgentName string
	// Trace is optional trace information (e.g., "thinking", "response") (immutable)
	Trace string
	// Model is the name of the LLM model being used (immutable)
	Model string
	// Tools is the list of tools available to the agent (immutable)
	Tools []llms.Tool
	// WorkingDir is the agent's working directory, accessible to tools (immutable)
	WorkingDir string
	// ResponseCh is the response channel for the current chat session (immutable per session)
	ResponseCh *ResponseCh
	// SessionStorage is a map that persists state across tool calls (mutable, shared by reference)
	SessionStorage map[string]any
	// PluginFields is a map for plugins to store custom context data that persists across tool calls (mutable)
	PluginFields map[string]any
}

// Snapshot returns a new AgentContext for a single ChatStream call.
// Immutable fields are copied by value; mutable fields (SessionStorage, PluginFields)
// start fresh so concurrent calls cannot bleed state into each other.
func (ac *AgentContext) Snapshot() *AgentContext {
	return &AgentContext{
		AgentName:  ac.AgentName,
		Trace:      ac.Trace,
		Model:      ac.Model,
		Tools:      ac.Tools,
		WorkingDir: ac.WorkingDir,
		// mutable fields start empty for each call
		SessionStorage: make(map[string]any),
		PluginFields:   make(map[string]any),
	}
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
	chatId := ""
	if responseCh != nil {
		chatId = responseCh.GetChatId()
	}
	context["chatId"] = chatId
	context["tools"] = ac.Tools
	context["workingDir"] = ac.WorkingDir

	// SessionStorage should always be initialized (never nil)
	// This ensures the same map reference is shared across all tool calls
	if ac.SessionStorage == nil {
		ac.SessionStorage = make(map[string]any)
	}
	context["sessionStorage"] = ac.SessionStorage

	// Include plugin fields for extensibility
	// Plugins can add custom fields that persist across tool calls
	if ac.PluginFields != nil {
		context["pluginFields"] = ac.PluginFields
	} else {
		context["pluginFields"] = make(map[string]any)
	}

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
	if err := ac.validateWorkingDir(contextMap); err != nil {
		return nil, err
	}
	if err := ac.validateTools(contextMap); err != nil {
		return nil, err
	}
	if err := ac.validateSessionStorage(contextMap); err != nil {
		return nil, err
	}
	if err := ac.validatePluginFields(contextMap); err != nil {
		return nil, err
	}

	if responseCh, ok := contextMap["responseCh"].(*ResponseCh); ok {
		ac.ResponseCh = responseCh
	} else if responseCh, exists := contextMap["responseCh"]; exists {
		return nil, fmt.Errorf("responseCh must be a *ResponseCh, got %T", responseCh)
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

// validateWorkingDir extracts and validates the workingDir field from the context map.
func (ac *AgentContext) validateWorkingDir(contextMap map[string]any) error {
	if workingDir, ok := contextMap["workingDir"].(string); ok {
		ac.WorkingDir = workingDir
	} else if workingDir, exists := contextMap["workingDir"]; exists {
		return fmt.Errorf("workingDir must be a string, got %T", workingDir)
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

	out := make([]llms.Tool, 0, len(toolsSlice))
	for i, toolRaw := range toolsSlice {
		tool, ok := toolRaw.(llms.Tool)
		if !ok {
			return fmt.Errorf("tools[%d] must implement llms.Tool, got %T", i, toolRaw)
		}
		out = append(out, tool)
	}
	ac.Tools = out
	return nil
}

// validateSessionStorage extracts and validates the sessionStorage field from the context map.
func (ac *AgentContext) validateSessionStorage(contextMap map[string]any) error {
	sessionStorageRaw, exists := contextMap["sessionStorage"]
	if !exists {
		// SessionStorage is optional, initialize empty if not present
		ac.SessionStorage = make(map[string]any)
		return nil
	}

	if sessionStorage, ok := sessionStorageRaw.(map[string]any); ok {
		ac.SessionStorage = sessionStorage
		return nil
	}

	return fmt.Errorf("sessionStorage must be map[string]any, got %T", sessionStorageRaw)
}

// GetSessionStorage returns the session storage map
func (ac *AgentContext) GetSessionStorage() map[string]any {
	if ac.SessionStorage == nil {
		ac.SessionStorage = make(map[string]any)
	}
	return ac.SessionStorage
}

// SetSessionStorage sets the session storage map
func (ac *AgentContext) SetSessionStorage(sessionStorage map[string]any) {
	ac.SessionStorage = sessionStorage
}

// SyncFromMap syncs mutable fields from the context map back to the struct.
// This ensures changes made by tools persist across tool calls.
//
// Mutable fields that are synced:
//   - PluginFields: Merged from context map (if present)
//
// Immutable fields are not synced as they should never change after initialization.
//
// Parameters:
//   - contextMap: The context map that may have been modified during tool execution
//
// Returns:
//   - error: An error if sync fails (should not happen in normal operation)
func (ac *AgentContext) SyncFromMap(contextMap map[string]any) error {
	// Sync PluginFields
	if pluginFields, ok := contextMap["pluginFields"].(map[string]any); ok {
		if ac.PluginFields == nil {
			ac.PluginFields = make(map[string]any)
		}
		// Merge plugin fields from context map
		for k, v := range pluginFields {
			ac.PluginFields[k] = v
		}
	}

	// SessionStorage is already shared by reference, no sync needed
	// Changes to SessionStorage in the map are automatically reflected in the struct

	return nil
}

// validatePluginFields extracts and validates the pluginFields field from the context map.
func (ac *AgentContext) validatePluginFields(contextMap map[string]any) error {
	pluginFieldsRaw, exists := contextMap["pluginFields"]
	if !exists {
		// PluginFields is optional, initialize empty if not present
		ac.PluginFields = make(map[string]any)
		return nil
	}

	if pluginFields, ok := pluginFieldsRaw.(map[string]any); ok {
		ac.PluginFields = pluginFields
		return nil
	}

	return fmt.Errorf("pluginFields must be map[string]any, got %T", pluginFieldsRaw)
}
