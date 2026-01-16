package vector

import (
	"fmt"
)

// validateAction ensures that the action is one of the valid vector tool actions.
func validateAction(value any) error {
	action, ok := value.(string)
	if !ok {
		return fmt.Errorf("action must be a string")
	}
	validActions := map[string]bool{
		"index":  true,
		"search": true,
		"delete": true,
	}
	if !validActions[action] {
		return fmt.Errorf("invalid action: %s. Must be one of: index, search, delete", action)
	}
	return nil
}
