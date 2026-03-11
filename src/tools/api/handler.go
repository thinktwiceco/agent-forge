package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// resolveString expands ${VAR} placeholders via os.Getenv.
// Returns an error if any referenced variable is empty or unset.
func resolveString(s string) (string, error) {
	var missingVar string
	expanded := os.Expand(s, func(key string) string {
		val := os.Getenv(key)
		if val == "" && missingVar == "" {
			missingVar = key
		}
		return val
	})
	if missingVar != "" {
		return "", fmt.Errorf("references env var %q which is empty or unset", missingVar)
	}
	return expanded, nil
}

// resolveHeaders expands ${VAR} placeholders in all header values.
func resolveHeaders(headers map[string]string) (map[string]string, error) {
	resolved := make(map[string]string, len(headers))
	for k, v := range headers {
		expanded, err := resolveString(v)
		if err != nil {
			return nil, fmt.Errorf("header %q: %w", k, err)
		}
		resolved[k] = expanded
	}
	return resolved, nil
}

func (a *Api) handler(_ map[string]any, args map[string]any) llms.ToolReturn {
	action, ok := args["action"].(string)
	if !ok || action == "" {
		return core.NewErrorResponse("action parameter is required")
	}

	// Repository actions do not require a service name.
	switch action {
	case "list_api_configs":
		return a.handleListApiConfigs()
	case "install_api_config":
		return a.handleInstallApiConfig(args)
	}

	// All other actions require a valid service.
	serviceName, ok := args["service"].(string)
	if !ok || serviceName == "" {
		return core.NewErrorResponse("service parameter is required for this action")
	}
	svc, exists := a.findService(serviceName)
	if !exists {
		return core.NewErrorResponse(fmt.Sprintf("unknown service: %s. Available: %s",
			serviceName, strings.Join(a.getServiceNames(), ", ")))
	}

	switch action {
	case "show_apis":
		return a.handleShowApis(svc, serviceName)
	case "show_api":
		endpointName, _ := args["endpoint"].(string)
		return a.handleShowApi(svc, serviceName, endpointName)
	default:
		return a.handleCall(svc, serviceName, action, args)
	}
}

func (a *Api) handleListApiConfigs() llms.ToolReturn {
	if a.repositoryDir == "" {
		return core.NewErrorResponse("repository not configured for this tool instance")
	}
	names, err := listRemoteApiConfigs()
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to list remote api configs: %v", err))
	}
	if len(names) == 0 {
		return core.NewEphemeralResponse("No API configs found in the remote repository.")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Found %d API config(s) available in thinktwiceco/agent-forge repository:\n\n", len(names))
	for _, n := range names {
		fmt.Fprintf(&b, "  - %s\n", n)
	}
	b.WriteString("\nUse install_api_config with configName=<name> to install any of the above.")
	return core.NewEphemeralResponse(b.String())
}

func (a *Api) handleInstallApiConfig(args map[string]any) llms.ToolReturn {
	if a.repositoryDir == "" {
		return core.NewErrorResponse("repository not configured for this tool instance")
	}
	name, ok := args["configName"].(string)
	if !ok || name == "" {
		return core.NewErrorResponse("configName parameter is required for install_api_config")
	}
	agentforge.Info("API Tool: Installing repository config '%s' to %s", name, a.repositoryDir)
	svc, err := fetchAndInstallApiConfig(name, a.repositoryDir)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to install api config '%s': %v", name, err))
	}
	a.services[name] = svc
	return core.NewEphemeralResponse(fmt.Sprintf(
		"API config '%s' installed successfully (%d endpoint(s)). It is now available — use show_apis with service=%s to explore it.",
		name, len(svc.Endpoints), name,
	))
}

func (a *Api) handleShowApis(svc ServiceConfig, serviceName string) llms.ToolReturn {
	var b strings.Builder
	fmt.Fprintf(&b, "Service: %s\n", serviceName)
	b.WriteString("Endpoints:\n")
	for _, ep := range svc.Endpoints {
		fmt.Fprintf(&b, "  - %s: %s\n", ep.Name, ep.Description)
	}
	return core.NewSuccessResponse(b.String())
}

func (a *Api) handleShowApi(svc ServiceConfig, serviceName, endpointName string) llms.ToolReturn {
	if endpointName == "" {
		return core.NewErrorResponse("endpoint parameter is required for show_api")
	}
	ep := a.findEndpoint(svc, endpointName)
	if ep == nil {
		names := make([]string, len(svc.Endpoints))
		for i, e := range svc.Endpoints {
			names[i] = e.Name
		}
		return core.NewErrorResponse(fmt.Sprintf("unknown endpoint: %s. Available: %s",
			endpointName, strings.Join(names, ", ")))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Service: %s\n", serviceName)
	fmt.Fprintf(&b, "Endpoint: %s\n", ep.Name)
	fmt.Fprintf(&b, "Method: %s %s\n", ep.Method, ep.URL)
	fmt.Fprintf(&b, "Description: %s\n", ep.Description)
	if ep.URLParameters != "" {
		fmt.Fprintf(&b, "URL Parameters:\n  %s\n",
			strings.ReplaceAll(strings.TrimSpace(ep.URLParameters), "\n", "\n  "))
	}
	if ep.QueryParams != "" {
		fmt.Fprintf(&b, "Query Parameters:\n  %s\n",
			strings.ReplaceAll(strings.TrimSpace(ep.QueryParams), "\n", "\n  "))
	}
	if ep.Payload != "" {
		fmt.Fprintf(&b, "Body:\n  %s\n",
			strings.ReplaceAll(strings.TrimSpace(ep.Payload), "\n", "\n  "))
	}
	return core.NewSuccessResponse(b.String())
}

func (a *Api) handleCall(svc ServiceConfig, serviceName, endpointName string, args map[string]any) llms.ToolReturn {
	ep := a.findEndpoint(svc, endpointName)
	if ep == nil {
		names := make([]string, len(svc.Endpoints))
		for i, e := range svc.Endpoints {
			names[i] = e.Name
		}
		return core.NewErrorResponse(fmt.Sprintf("unknown endpoint: %s. Available: %s",
			endpointName, strings.Join(names, ", ")))
	}

	agentforge.Info("API Tool: Calling %s/%s (%s)", serviceName, ep.Name, ep.Method)

	// Build URL: replace {param} placeholders then expand ${ENV_VAR}
	rawURL, err := a.buildURL(ep, args)
	if err != nil {
		return core.NewErrorResponse(err.Error())
	}
	resolvedURL, err := resolveString(rawURL)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("URL resolution failed: %v", err))
	}

	// Add query params
	resolvedURL = a.addQueryParams(resolvedURL, args["query_params"])

	// Resolve headers
	headers, err := resolveHeaders(svc.Headers)
	if err != nil {
		return core.NewErrorResponse(err.Error())
	}

	// Handle basic_auth: resolve and base64-encode into Authorization header
	if svc.BasicAuth != "" {
		resolved, err := resolveString(svc.BasicAuth)
		if err != nil {
			return core.NewErrorResponse(fmt.Sprintf("basic_auth resolution failed: %v", err))
		}
		headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(resolved))
	}

	// Serialize body based on endpoint content type
	body := ""
	if bodyVal, ok := args["body"]; ok && bodyVal != nil {
		if bodyMap, ok := bodyVal.(map[string]any); ok {
			if err := resolveBodyArgs(bodyMap, a.workingDir); err != nil {
				return core.NewErrorResponse(fmt.Sprintf("resolver error: %v", err))
			}
		}
		if ep.ContentType == "form" {
			if bodyMap, ok := bodyVal.(map[string]any); ok {
				values := url.Values{}
				for k, v := range bodyMap {
					values.Set(k, fmt.Sprint(v))
				}
				body = values.Encode()
			}
		} else {
			bodyBytes, err := json.Marshal(bodyVal)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to serialize body: %v", err))
			}
			body = string(bodyBytes)
		}
	}

	return a.executeRequest(ep, ep.Method, resolvedURL, headers, body)
}
