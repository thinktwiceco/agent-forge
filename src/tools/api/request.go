package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// executeRequest executes an HTTP request and returns a formatted response
func (a *Api) executeRequest(endpoint *Endpoint, method, url string, headers map[string]string, body string) llms.ToolReturn {
	// 1. Create HTTP client with timeout
	client := &http.Client{Timeout: 30 * time.Second}

	// 2. Create request
	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to create request: %v", err))
	}

	// 3. Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set default Content-Type for requests with body
	if body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// 4. Execute request
	resp, err := client.Do(req)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to execute request: %v", err))
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log error but don't fail the request
			_ = closeErr
		}
	}()

	// 5. Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to read response body: %v", err))
	}

	// 6. Format response
	response := &apiResponse{
		Endpoint:   endpoint.Name,
		Method:     method,
		URL:        url,
		StatusCode: resp.StatusCode,
		Body:       string(respBody),
		Headers:    convertHeaders(resp.Header),
	}

	// 7. Return success or error based on status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return core.NewSuccessResponse(response.String())
	}

	response.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	return core.NewFailureResponse(
		response.Error,
		response.String(),
	)
}

// convertHeaders converts http.Header to map[string]string
func convertHeaders(headers http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range headers {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}
