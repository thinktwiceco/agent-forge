package web

import "fmt"

// navigateResponse represents a navigation operation result.
type navigateResponse struct {
	Operation string
	URL       string
	Title     string
	Success   bool
	Error     string
}

// String formats the navigate response as a string.
func (r *navigateResponse) String() string {
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

// clickResponse represents a click operation result.
type clickResponse struct {
	Operation string
	Selector  string
	Success   bool
	Error     string
}

// String formats the click response as a string.
func (r *clickResponse) String() string {
	result := fmt.Sprintf(`Web Browser Operation: Click
Selector: %s
Status: %s
`, r.Selector, map[bool]string{true: "Success", false: "Failed"}[r.Success])

	if r.Error != "" {
		result += fmt.Sprintf("Error: %s\n", r.Error)
	}

	return result
}

// typeResponse represents a type operation result.
type typeResponse struct {
	Operation string
	Selector  string
	Success   bool
	Error     string
}

// String formats the type response as a string.
func (r *typeResponse) String() string {
	result := fmt.Sprintf(`Web Browser Operation: Type
Selector: %s
Status: %s
`, r.Selector, map[bool]string{true: "Success", false: "Failed"}[r.Success])

	if r.Error != "" {
		result += fmt.Sprintf("Error: %s\n", r.Error)
	}

	return result
}

// screenshotResponse represents a screenshot operation result.
type screenshotResponse struct {
	Operation string
	Path      string
	Size      int64
	Success   bool
	Error     string
}

// String formats the screenshot response as a string.
func (r *screenshotResponse) String() string {
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

// contentResponse represents a get_content operation result.
type contentResponse struct {
	Operation string
	Type      string
	Content   string
	Success   bool
	Error     string
}

// String formats the content response as a string.
func (r *contentResponse) String() string {
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

// waitResponse represents a wait operation result.
type waitResponse struct {
	Operation string
	Selector  string
	Timeout   int
	Waited    float64
	Success   bool
	Error     string
}

// String formats the wait response as a string.
func (r *waitResponse) String() string {
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

// historyResponse represents a browser history navigation result.
type historyResponse struct {
	Operation string
	URL       string
	Success   bool
	Error     string
}

// String formats the history response as a string.
func (r *historyResponse) String() string {
	result := fmt.Sprintf(`Web Browser Operation: %s
URL: %s
Status: %s
`, r.Operation, r.URL, map[bool]string{true: "Success", false: "Failed"}[r.Success])

	if r.Error != "" {
		result += fmt.Sprintf("Error: %s\n", r.Error)
	}

	return result
}

// evaluateResponse represents a JavaScript evaluation result.
type evaluateResponse struct {
	Operation string
	Result    string
	Success   bool
	Error     string
}

// String formats the evaluate response as a string.
func (r *evaluateResponse) String() string {
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

// saveContentResponse represents a save_content operation result.
type saveContentResponse struct {
	Operation string
	Filename  string
	Path      string
	Success   bool
	Error     string
}

// String formats the save_content response as a string.
func (r *saveContentResponse) String() string {
	result := fmt.Sprintf(`Web Browser Operation: Save Content
Filename: %s
Path: %s
Status: %s
`, r.Filename, r.Path, map[bool]string{true: "Success", false: "Failed"}[r.Success])

	if r.Error != "" {
		result += fmt.Sprintf("Error: %s\n", r.Error)
	}

	return result
}
