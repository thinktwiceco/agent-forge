package expand

import (
	"fmt"

	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// validateSubjectType ensures that the subject_type is either "tool" or "agent".
func validateSubjectType(subjectType string) error {
	if subjectType != "tool" && subjectType != "agent" {
		return fmt.Errorf("invalid subject_type '%s'. Must be either 'tool' or 'agent'", subjectType)
	}
	return nil
}

// validateSubjectTypeAndReturnError validates the subject type and returns an error response if invalid.
// Returns nil if validation passes.
func validateSubjectTypeAndReturnError(subjectType string) llms.ToolReturn {
	if err := validateSubjectType(subjectType); err != nil {
		return core.NewErrorResponse(err.Error())
	}
	return nil
}
