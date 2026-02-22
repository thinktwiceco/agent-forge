package persistence

import "path/filepath"

// NewPersistence creates and returns a Persistence implementation based on the persistence type.
//
// Parameters:
//   - agentName: The name of the agent (used for generating unique file paths)
//   - persistenceType: The type of persistence ("json", or "" for none)
//   - workingDir: The agent's working directory. When non-empty, conversations are stored at
//     workingDir/data/conversations/{agentName}. When empty, uses data/conversations/{agentName}
//     relative to process CWD.
//
// Returns:
//   - Persistence: The appropriate persistence implementation, or nil if no persistence is configured
func NewPersistence(agentName, persistenceType, workingDir string) Persistence {
	if persistenceType == "" {
		return nil
	}

	switch persistenceType {
	case "json":
		var baseDir string
		if workingDir != "" {
			baseDir = filepath.Join(workingDir, "data", "conversations", agentName)
		} else {
			baseDir = filepath.Join("data", "conversations", agentName)
		}
		return NewJSONPersistence(baseDir)
	default:
		// Unknown persistence type, return nil
		return nil
	}
}
