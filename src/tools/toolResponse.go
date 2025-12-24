package tools

type ToolResponse struct {
	success bool
	error   string
	data    string
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
