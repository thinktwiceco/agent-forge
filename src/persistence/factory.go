package persistence

import "path/filepath"

// NewPersistence creates and returns a Persistence implementation based on the persistence type
// Parameters:
//   - agentName: The name of the agent (used for generating unique file paths)
//   - persistenceType: The type of persistence ("json", or "" for none)
//
// Returns:
//   - Persistence: The appropriate persistence implementation, or nil if no persistence is configured
func NewPersistence(agentName, persistenceType string) Persistence {
	if persistenceType == "" {
		return nil
	}

	switch persistenceType {
	case "json":
		// Create base directory for this agent's conversations
		baseDir := filepath.Join("data", "conversations", agentName)
		return NewJSONPersistence(baseDir)
	default:
		// Unknown persistence type, return nil
		return nil
	}
}
