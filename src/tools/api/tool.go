package api

import (
	"fmt"
	"strings"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// NewApiTool creates a new API tool that allows making HTTP requests to configured endpoints.
//
// Parameters:
//   - name: The name of the tool (e.g., "github_api", "stripe_api")
//   - endpoints: List of endpoint configurations
//   - headers: Static headers sent with every request; values support ${ENV_VAR} expansion
//
// The tool provides a generic interface for making API calls with:
//   - Dynamic endpoint discovery (agent sees all available endpoints)
//   - URL parameter substitution (e.g., /users/{user_id})
//   - Query parameter support (e.g., ?limit=10&offset=0)
//   - Request body support (JSON strings)
//   - Custom validation per endpoint
//   - Authentication via declarative headers
func NewApiTool(
	name string,
	endpoints []Endpoint,
	headers map[string]string,
) llms.Tool {
	if headers == nil {
		headers = map[string]string{}
	}

	api := &Api{
		name:      name,
		endpoints: endpoints,
		headers:   headers,
	}

	return &core.Tool{
		Name:                name,
		Description:         api.generateDescription(),
		AdvanceDesc:         api.generateAdvancedDescription(),
		TroubleshootingInfo: api.generateTroubleshootingInfo(),
		DetailsAboutFunc:    api.detailsAbout,
		Parameters:          api.generateParameters(),
		Handler:             api.handler,
	}
}

// generateDescription generates a brief description of the tool
func (a *Api) generateDescription() string {
	endpointNames := a.getEndpointNames()
	return fmt.Sprintf("Make API calls to configured endpoints. Available endpoints: %s",
		strings.Join(endpointNames, ", "))
}

// generateAdvancedDescription generates a high-level overview of the tool and its endpoints.
func (a *Api) generateAdvancedDescription() string {
	var builder strings.Builder

	builder.WriteString("Advanced Details:\n")
	fmt.Fprintf(&builder, "- Tool: %s\n", a.name)
	fmt.Fprintf(&builder, "- Total Endpoints: %d\n\n", len(a.endpoints))

	builder.WriteString("Available Endpoints:\n")
	for i, endpoint := range a.endpoints {
		fmt.Fprintf(&builder, "  %d. %s — %s [%s %s]\n", i+1, endpoint.Name, endpoint.Description, endpoint.Method, endpoint.URL)
	}

	builder.WriteString("\nUse expand tool with details_about=\"<endpoint>\" for full parameter details on any endpoint.\n")
	builder.WriteString("\n- Usage:\n")
	builder.WriteString("  * Specify the endpoint name in the 'endpoint' parameter\n")
	builder.WriteString("  * Provide URL parameters in 'url_params' as an object (e.g., {\"user_id\": \"123\"})\n")
	builder.WriteString("  * Provide query parameters in 'query_params' as an object (e.g., {\"limit\": 10, \"offset\": 0})\n")
	builder.WriteString("  * Provide request body in 'body' as a JSON string\n")
	builder.WriteString("  * Authentication headers are injected automatically from tool configuration\n")

	return builder.String()
}

// detailsAbout returns detailed documentation for a specific endpoint.
func (a *Api) detailsAbout(item string) string {
	for _, endpoint := range a.endpoints {
		if endpoint.Name == item {
			var builder strings.Builder
			fmt.Fprintf(&builder, "Endpoint: %s\n", endpoint.Name)
			fmt.Fprintf(&builder, "Description: %s\n", endpoint.Description)
			fmt.Fprintf(&builder, "Method: %s\n", endpoint.Method)
			fmt.Fprintf(&builder, "URL: %s\n", endpoint.URL)

			if endpoint.URLParameters != "" {
				builder.WriteString("URL Parameters:\n")
				for _, line := range strings.Split(strings.TrimSpace(endpoint.URLParameters), "\n") {
					fmt.Fprintf(&builder, "  %s\n", strings.TrimSpace(line))
				}
			}

			if endpoint.QueryParams != "" {
				builder.WriteString("Query Parameters:\n")
				for _, line := range strings.Split(strings.TrimSpace(endpoint.QueryParams), "\n") {
					fmt.Fprintf(&builder, "  %s\n", strings.TrimSpace(line))
				}
			}

			if endpoint.Payload != "" {
				builder.WriteString("Request Body:\n")
				for _, line := range strings.Split(strings.TrimSpace(endpoint.Payload), "\n") {
					fmt.Fprintf(&builder, "  %s\n", strings.TrimSpace(line))
				}
			}

			return builder.String()
		}
	}
	return fmt.Sprintf("Nothing to add about %s", item)
}

// generateTroubleshootingInfo generates troubleshooting information
func (a *Api) generateTroubleshootingInfo() string {
	return `Troubleshooting:
- "endpoint not found": Ensure you're using one of the available endpoint names listed in the tool description
- "missing required URL parameters": Check that all URL parameters in the URL template (e.g., {user_id}) are provided in url_params
- "parameter validation failed": The endpoint has custom validation rules that were not satisfied. Check the error message for details
- "authentication hook failed": The authentication hook returned an error. This typically means credentials are invalid or expired
- "failed to execute request": Network error or invalid URL. Check network connectivity and URL format
- HTTP 4xx errors: Client error (bad request, unauthorized, not found, etc.). Check your parameters and authentication
- HTTP 5xx errors: Server error. The API endpoint is experiencing issues, try again later`
}

// generateParameters generates the tool parameters definition
func (a *Api) generateParameters() []core.Parameter {
	endpointNames := a.getEndpointNames()

	return []core.Parameter{
		{
			Name:        "endpoint",
			Type:        "string",
			Description: fmt.Sprintf("The endpoint to call. Available: %s", strings.Join(endpointNames, ", ")),
			Required:    true,
			Validator:   a.validateEndpoint,
		},
		{
			Name:        "url_params",
			Type:        "object",
			Description: "URL path parameters (e.g., {\"user_id\": \"123\"} for /users/{user_id})",
			Required:    false,
		},
		{
			Name:        "query_params",
			Type:        "object",
			Description: "Query string parameters (e.g., {\"limit\": 10, \"offset\": 0} for ?limit=10&offset=0)",
			Required:    false,
		},
		{
			Name:        "body",
			Type:        "string",
			Description: "Request body as a JSON string (for POST, PUT, PATCH requests)",
			Required:    false,
		},
	}
}
