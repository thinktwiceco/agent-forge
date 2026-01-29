package core

import "github.com/thinktwiceco/agent-forge/src/llms"

// ToolResponse implements llms.ToolReturn interface
type ToolResponse struct {
	success   bool
	error     string
	data      string
	ephemeral bool
	cleanup   func() // Optional cleanup function
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

func (t *ToolResponse) Cleanup() func() {
	return t.cleanup
}

func NewEphemeralResponse(data string) llms.ToolReturn {
	return &ToolResponse{
		success:   true,
		error:     "",
		data:      data,
		ephemeral: true,
	}
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

func NewSuccessEphemeralResponse(data string) llms.ToolReturn {
	return &ToolResponse{
		success:   true,
		error:     "",
		data:      data,
		ephemeral: true,
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
