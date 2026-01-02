package core

import "github.com/thinktwice/agentForge/src/llms"

// ToolResponse implements llms.ToolReturn interface
type ToolResponse struct {
	success   bool
	error     string
	data      string
	ephemeral bool
}

func (t *ToolResponse) Success() bool {
	return t.success
}

func (t *ToolResponse) Error() string {
	return t.error
}

func (t *ToolResponse) Data() string {
	return t.data
}

func (t *ToolResponse) Ephemeral() bool {
	return t.ephemeral
}

// NewSuccessResponse creates a successful tool response
func NewSuccessResponse(data string) llms.ToolReturn {
	return &ToolResponse{
		success:   true,
		error:     "",
		data:      data,
		ephemeral: false,
	}
}

// NewErrorResponse creates an error response without data
func NewErrorResponse(errorMsg string) llms.ToolReturn {
	return &ToolResponse{
		success:   false,
		error:     errorMsg,
		data:      "",
		ephemeral: false,
	}
}

// NewFailureResponse creates an error response with partial data
func NewFailureResponse(errorMsg, data string) llms.ToolReturn {
	return &ToolResponse{
		success:   false,
		error:     errorMsg,
		data:      data,
		ephemeral: false,
	}
}
