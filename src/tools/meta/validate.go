package meta

import (
	"fmt"
)

// validateMethod ensures that the method is one of the valid meta methods.
func validateMethod(value any) error {
	method, ok := value.(string)
	if !ok {
		return fmt.Errorf("method must be a string")
	}
	validMethods := map[string]bool{
		"get_current_model": true,
		"get_agent_name":    true,
		"get_tools":         true,
		"get_subagents":     true,
	}
	if !validMethods[method] {
		return fmt.Errorf("invalid method: %s. Must be one of: get_current_model, get_agent_name, get_tools, get_subagents", method)
	}
	return nil
}
