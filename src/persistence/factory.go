package persistence

import (
	"fmt"
	"math/rand"
	"time"
)

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
		// Generate unique file path
		uniqueID := fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63())
		filePath := fmt.Sprintf("./history/%s-%s.json", agentName, uniqueID)
		return NewJSONPersistence(filePath)
	default:
		// Unknown persistence type, return nil
		return nil
	}
}
