package api

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Endpoint represents a single API endpoint configuration
type Endpoint struct {
	Name          string // Unique identifier for the endpoint
	URL           string // URL template with {param} and ${ENV_VAR} placeholders
	Method        string // HTTP method (GET, POST, PUT, DELETE, PATCH)
	Description   string // Brief description of the endpoint
	Payload       string // Human-readable description of body parameters
	QueryParams   string // Human-readable description of query parameters
	URLParameters string // Human-readable description of URL path parameters
	ContentType   string // Request body encoding: "" or "json" (default) | "form" (application/x-www-form-urlencoded)
}

// ServiceConfig holds the configuration for a single API service
type ServiceConfig struct {
	ServiceName        string // Display name for prompts (optional; falls back to config key)
	ServiceDescription string // Human-readable description of the service (optional)
	Headers            map[string]string
	BasicAuth          string // if set, resolved at call time and sent as Authorization: Basic <base64>
	Endpoints          []Endpoint
}

// ServiceConfigJSON is the JSON schema for a <service>.json config file.
type ServiceConfigJSON struct {
	ServiceName        string               `json:"serviceName,omitempty"`
	ServiceDescription string               `json:"serviceDescription,omitempty"`
	Headers            map[string]string    `json:"headers"`
	BasicAuth          string               `json:"basic_auth,omitempty"`
	Endpoints          []EndpointConfigJSON `json:"endpoints"`
}

// EndpointConfigJSON is the JSON schema for one endpoint inside a service config file.
type EndpointConfigJSON struct {
	Name          string `json:"name"`
	URL           string `json:"url"`
	Method        string `json:"method"`
	Description   string `json:"description"`
	Payload       string `json:"payload,omitempty"`
	QueryParams   string `json:"query_params,omitempty"`
	URLParameters string `json:"url_parameters,omitempty"`
	ContentType   string `json:"content_type,omitempty"`
}

// ParseServiceConfig parses raw JSON bytes into a ServiceConfig.
func ParseServiceConfig(data []byte) (ServiceConfig, error) {
	var svcJSON ServiceConfigJSON
	if err := json.Unmarshal(data, &svcJSON); err != nil {
		return ServiceConfig{}, err
	}
	endpoints := make([]Endpoint, len(svcJSON.Endpoints))
	for i, e := range svcJSON.Endpoints {
		endpoints[i] = Endpoint(e)
	}
	headers := svcJSON.Headers
	if headers == nil {
		headers = map[string]string{}
	}
	return ServiceConfig{
		ServiceName:        svcJSON.ServiceName,
		ServiceDescription: svcJSON.ServiceDescription,
		Headers:            headers,
		BasicAuth:          svcJSON.BasicAuth,
		Endpoints:          endpoints,
	}, nil
}

// Api represents the unified API tool with multiple services
type Api struct {
	name          string
	services      map[string]ServiceConfig
	repositoryDir string // directory for remotely installed service configs; empty means no repository
	workingDir    string // base path for resolving relative file paths in resolvers (e.g. resolve_to_base64)
}

// apiResponse represents the response from an API call
type apiResponse struct {
	Service    string
	Endpoint   string
	Method     string
	URL        string
	StatusCode int
	Body       string
	Headers    map[string]string
	Error      string
}

func (r *apiResponse) String() string {
	var b strings.Builder
	b.WriteString("API Response\n")
	fmt.Fprintf(&b, "Service: %s\n", r.Service)
	fmt.Fprintf(&b, "Endpoint: %s\n", r.Endpoint)
	fmt.Fprintf(&b, "Method: %s\n", r.Method)
	fmt.Fprintf(&b, "URL: %s\n", r.URL)
	fmt.Fprintf(&b, "Status: %d\n", r.StatusCode)
	if r.Error != "" {
		fmt.Fprintf(&b, "Error: %s\n", r.Error)
	}
	if len(r.Headers) > 0 {
		b.WriteString("\nResponse Headers:\n")
		for k, v := range r.Headers {
			fmt.Fprintf(&b, "  %s: %s\n", k, v)
		}
	}
	fmt.Fprintf(&b, "\nResponse Body:\n%s\n", r.Body)
	return b.String()
}
