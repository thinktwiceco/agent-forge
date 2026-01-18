package web

import (
	"fmt"
)

// validateAction ensures that the action is one of the valid web browser actions.
func validateAction(value any) error {
	action, ok := value.(string)
	if !ok {
		return fmt.Errorf("action must be a string")
	}
	validActions := map[string]bool{
		"navigate":     true,
		"click":        true,
		"type":         true,
		"screenshot":   true,
		"get_content":  true,
		"save_content": true,
		"wait":         true,
		"back":         true,
		"forward":      true,
		"evaluate":     true,
		"close":        true,
	}
	if !validActions[action] {
		return fmt.Errorf("invalid action: %s. Must be one of: navigate, click, type, screenshot, get_content, save_content, wait, back, forward, evaluate, close", action)
	}
	return nil
}

// validateSelector ensures that the CSS selector is not empty.
func validateSelector(selector string) error {
	if selector == "" {
		return fmt.Errorf("selector cannot be empty")
	}
	return nil
}
