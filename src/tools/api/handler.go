package api

import (
	"fmt"
	"os"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// resolveHeaders returns a copy of headers with ${VAR} placeholders expanded via os.Getenv.
func resolveHeaders(headers map[string]string) map[string]string {
	resolved := make(map[string]string, len(headers))
	for k, v := range headers {
		resolved[k] = os.Expand(v, os.Getenv)
	}
	return resolved
}

// handler is the main tool handler function
func (a *Api) handler(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	// 1. Extract and validate endpoint name
	endpointName, ok := args["endpoint"].(string)
	if !ok {
		return core.NewErrorResponse("endpoint parameter is required and must be a string")
	}

	endpoint := a.findEndpoint(endpointName)
	if endpoint == nil {
		return core.NewErrorResponse(fmt.Sprintf("endpoint not found: %s", endpointName))
	}

	agentforge.Info("API Tool: Calling endpoint '%s' (%s %s)", endpoint.Name, endpoint.Method, endpoint.URL)

	// 2. Extract parameters for validation
	urlParams := map[string]any{}
	if up, ok := args["url_params"].(map[string]any); ok {
		urlParams = up
	}

	queryParams := map[string]any{}
	if qp, ok := args["query_params"].(map[string]any); ok {
		queryParams = qp
	}

	body := ""
	if bodyVal, ok := args["body"]; ok {
		body = bodyVal.(string)
	}

	// 3. Run endpoint-specific validation if configured
	if endpoint.Validate != nil {
		validationParams := EndpointValidationParams{
			URLParams:   urlParams,
			QueryParams: queryParams,
			Body:        body,
		}
		if err := endpoint.Validate(validationParams); err != nil {
			return core.NewErrorResponse(fmt.Sprintf("parameter validation failed: %v", err))
		}
	}

	// 4. Build complete URL with URL parameters
	url, err := a.buildURL(endpoint, args)
	if err != nil {
		return core.NewErrorResponse(err.Error())
	}

	// 5. Add query parameters
	url = a.addQueryParams(url, args["query_params"])

	// 6. Resolve headers, expanding ${ENV_VAR} placeholders
	headers := resolveHeaders(a.headers)

	// 7. Make HTTP request
	return a.executeRequest(endpoint, endpoint.Method, url, headers, body)
}
