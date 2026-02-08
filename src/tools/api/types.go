package api

import (
	"fmt"
	"strings"
)

// EndpointValidationParams contains the parameters to validate for an endpoint
type EndpointValidationParams struct {
	URLParams   map[string]any
	QueryParams map[string]any
	Body        string
}

// EndpointValidator is a function that validates endpoint parameters
type EndpointValidator func(params EndpointValidationParams) error

// Endpoint represents an API endpoint configuration
type Endpoint struct {
	Name          string            // Unique identifier for the endpoint
	URL           string            // URL template with {param} placeholders
	Method        string            // HTTP method (GET, POST, PUT, DELETE, PATCH)
	Description   string            // Brief description of the endpoint
	Payload       string            // Description of body parameters
	QueryParams   string            // Description of query parameters
	URLParameters string            // Description of URL path parameters
	Validate      EndpointValidator // Optional validation function for parameters
}

// Api represents an API tool with configured endpoints
type Api struct {
	name      string
	endpoints []Endpoint
	onApiCall func(url string, headers map[string]string, body string) (map[string]string, error)
}

// apiResponse represents the response from an API call
type apiResponse struct {
	Endpoint   string
	Method     string
	URL        string
	StatusCode int
	Body       string
	Headers    map[string]string
	Error      string
}

// String formats the API response as a string
func (r *apiResponse) String() string {
	var builder strings.Builder

	builder.WriteString("API Response\n")
	builder.WriteString(fmt.Sprintf("Endpoint: %s\n", r.Endpoint))
	builder.WriteString(fmt.Sprintf("Method: %s\n", r.Method))
	builder.WriteString(fmt.Sprintf("URL: %s\n", r.URL))
	builder.WriteString(fmt.Sprintf("Status: %d\n", r.StatusCode))

	if r.Error != "" {
		builder.WriteString(fmt.Sprintf("Error: %s\n", r.Error))
	}

	if len(r.Headers) > 0 {
		builder.WriteString("\nResponse Headers:\n")
		for key, value := range r.Headers {
			builder.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
	}

	builder.WriteString(fmt.Sprintf("\nResponse Body:\n%s\n", r.Body))

	return builder.String()
}
