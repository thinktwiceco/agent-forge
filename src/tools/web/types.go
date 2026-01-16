package web

import "fmt"

// NavigateResponse represents a navigation operation result.
type NavigateResponse struct {
	Operation string
	URL       string
	Title     string
	Success   bool
	Error     string
}

// String formats the navigate response as a string.
func (r *NavigateResponse) String() string {
	result := fmt.Sprintf(`Web Browser Operation: Navigate
URL: %s
Status: %s
`, r.URL, map[bool]string{true: "Success", false: "Failed"}[r.Success])

	if r.Title != "" {
		result += fmt.Sprintf("Page Title: %s\n", r.Title)
	}

	if r.Error != "" {
		result += fmt.Sprintf("Error: %s\n", r.Error)
	}

	return result
}

// ClickResponse represents a click operation result.
type ClickResponse struct {
	Operation string
	Selector  string
	Success   bool
	Error     string
}

// String formats the click response as a string.
func (r *ClickResponse) String() string {
	result := fmt.Sprintf(`Web Browser Operation: Click
Selector: %s
Status: %s
`, r.Selector, map[bool]string{true: "Success", false: "Failed"}[r.Success])

	if r.Error != "" {
		result += fmt.Sprintf("Error: %s\n", r.Error)
	}

	return result
}

// TypeResponse represents a type operation result.
type TypeResponse struct {
	Operation string
	Selector  string
	Success   bool
	Error     string
}

// String formats the type response as a string.
func (r *TypeResponse) String() string {
	result := fmt.Sprintf(`Web Browser Operation: Type
Selector: %s
Status: %s
`, r.Selector, map[bool]string{true: "Success", false: "Failed"}[r.Success])

	if r.Error != "" {
		result += fmt.Sprintf("Error: %s\n", r.Error)
	}

	return result
}

// ScreenshotResponse represents a screenshot operation result.
type ScreenshotResponse struct {
	Operation string
	Path      string
	Size      int64
	Success   bool
	Error     string
}

// String formats the screenshot response as a string.
func (r *ScreenshotResponse) String() string {
	result := fmt.Sprintf(`Web Browser Operation: Screenshot
Path: %s
Size: %d bytes
Status: %s
`, r.Path, r.Size, map[bool]string{true: "Success", false: "Failed"}[r.Success])

	if r.Error != "" {
		result += fmt.Sprintf("Error: %s\n", r.Error)
	}

	return result
}

// ContentResponse represents a get_content operation result.
type ContentResponse struct {
	Operation string
	Type      string
	Content   string
	Success   bool
	Error     string
}

// String formats the content response as a string.
func (r *ContentResponse) String() string {
	result := fmt.Sprintf(`Web Browser Operation: Get Content
Type: %s
Status: %s
`, r.Type, map[bool]string{true: "Success", false: "Failed"}[r.Success])

	if r.Content != "" {
		// Truncate content if too long for display
		content := r.Content
		if len(content) > 500 {
			content = content[:500] + "... (truncated)"
		}
		result += fmt.Sprintf("\nContent:\n%s\n", content)
	}

	if r.Error != "" {
		result += fmt.Sprintf("Error: %s\n", r.Error)
	}

	return result
}

// WaitResponse represents a wait operation result.
type WaitResponse struct {
	Operation string
	Selector  string
	Timeout   int
	Waited    float64
	Success   bool
	Error     string
}

// String formats the wait response as a string.
func (r *WaitResponse) String() string {
	result := fmt.Sprintf(`Web Browser Operation: Wait
Selector: %s
Timeout: %d seconds
Waited: %.2f seconds
Status: %s
`, r.Selector, r.Timeout, r.Waited, map[bool]string{true: "Success", false: "Failed"}[r.Success])

	if r.Error != "" {
		result += fmt.Sprintf("Error: %s\n", r.Error)
	}

	return result
}

// HistoryResponse represents a browser history navigation result.
type HistoryResponse struct {
	Operation string
	URL       string
	Success   bool
	Error     string
}

// String formats the history response as a string.
func (r *HistoryResponse) String() string {
	result := fmt.Sprintf(`Web Browser Operation: %s
URL: %s
Status: %s
`, r.Operation, r.URL, map[bool]string{true: "Success", false: "Failed"}[r.Success])

	if r.Error != "" {
		result += fmt.Sprintf("Error: %s\n", r.Error)
	}

	return result
}

// EvaluateResponse represents a JavaScript evaluation result.
type EvaluateResponse struct {
	Operation string
	Result    string
	Success   bool
	Error     string
}

// String formats the evaluate response as a string.
func (r *EvaluateResponse) String() string {
	result := fmt.Sprintf(`Web Browser Operation: Evaluate JavaScript
Status: %s
`, map[bool]string{true: "Success", false: "Failed"}[r.Success])

	if r.Result != "" {
		// Truncate result if too long
		resultStr := r.Result
		if len(resultStr) > 500 {
			resultStr = resultStr[:500] + "... (truncated)"
		}
		result += fmt.Sprintf("\nResult:\n%s\n", resultStr)
	}

	if r.Error != "" {
		result += fmt.Sprintf("Error: %s\n", r.Error)
	}

	return result
}

