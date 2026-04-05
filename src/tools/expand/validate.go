package expand

import (
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// validateSubjectType ensures that the subject_type is "tool".
func validateSubjectType(subjectType string) error {
	if subjectType != "tool" {
		return fmt.Errorf("invalid subject_type '%s'. Must be 'tool'", subjectType)
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
