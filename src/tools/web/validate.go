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
		"open_session":  true,
		"navigate":      true,
		"click":         true,
		"fill":          true,
		"fill_secret":   true,
		"get_content":   true,
		"get_snapshot":  true,
		"save_content":  true,
		"fetch":         true,
		"web_search":    true,
		"upload_file":   true,
		"refresh":       true,
		"list_sessions": true,
		"close_session": true,
	}
	if !validActions[action] {
		return fmt.Errorf("invalid action: %s. Must be one of: open_session, navigate, click, fill, fill_secret, get_content, get_snapshot, save_content, fetch, web_search, upload_file, refresh, list_sessions, close_session", action)
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
