package api

import (
	"fmt"
	"strings"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// ServiceProvider is implemented by the API tool. It lets callers (e.g. sub-agent
// constructors) discover which services are configured without reflection.
type ServiceProvider interface {
	llms.Tool
	// ServiceNames returns the names of all configured API services.
	ServiceNames() []string
}

// apiToolWrapper wraps *core.Tool and exposes the service registry.
type apiToolWrapper struct {
	*core.Tool
	a *Api
}

// ServiceNames returns the configured service names for this API tool.
func (w *apiToolWrapper) ServiceNames() []string {
	return w.a.getServiceNames()
}

// NewApiTool creates a unified API tool from a map of service configurations.
// name is the tool name visible to the agent (typically "api").
// services maps service names to their endpoint + header configurations.
// repositoryDir is the local directory for remotely installed configs (empty = no repository).
// workingDir is the base path for resolving relative file paths in resolvers (e.g. resolve_to_base64).
// The returned value also implements ServiceProvider for service discovery.
func NewApiTool(name string, services map[string]ServiceConfig, repositoryDir, workingDir string) ServiceProvider {
	a := &Api{name: name, services: services, repositoryDir: repositoryDir, workingDir: workingDir}
	t := &core.Tool{
		Name:                name,
		Description:         a.generateDescription(),
		AdvanceDesc:         a.generateAdvancedDescription(),
		TroubleshootingInfo: a.generateTroubleshootingInfo(),
		Parameters:          a.generateParameters(),
		Handler:             a.handler,
	}
	return &apiToolWrapper{Tool: t, a: a}
}

func (a *Api) generateDescription() string {
	names := a.getServiceNames()
	desc := "HTTP API client for configured services"
	if len(names) > 0 {
		hasAnyDesc := false
		var parts []string
		for _, key := range names {
			svc := a.services[key]
			if svc.ServiceDescription != "" {
				hasAnyDesc = true
				if svc.ServiceName != "" {
					parts = append(parts, fmt.Sprintf("%s (%s): %s", key, svc.ServiceName, svc.ServiceDescription))
				} else {
					parts = append(parts, fmt.Sprintf("%s: %s", key, svc.ServiceDescription))
				}
			} else {
				parts = append(parts, key)
			}
		}
		if hasAnyDesc {
			desc += " — " + strings.Join(parts, "; ")
		} else {
			desc += fmt.Sprintf(" (%s)", strings.Join(names, ", "))
		}
	}
	desc += ". Use show_apis to list endpoints, show_api for details, call an endpoint by name, list_api_configs to browse the repository, or install_api_config to add a new service."
	return desc
}

func (a *Api) generateAdvancedDescription() string {
	var b strings.Builder
	b.WriteString("API Tool — unified HTTP client\n\n")
	b.WriteString("Available services:\n")
	for _, key := range a.getServiceNames() {
		svc := a.services[key]
		display := key
		if svc.ServiceName != "" {
			display = fmt.Sprintf("%s (%s)", key, svc.ServiceName)
		}
		if svc.ServiceDescription != "" {
			fmt.Fprintf(&b, "  %s: %s (%d endpoints)\n", display, svc.ServiceDescription, len(svc.Endpoints))
		} else {
			fmt.Fprintf(&b, "  %s (%d endpoints)\n", display, len(svc.Endpoints))
		}
	}
	b.WriteString("\nWorkflow:\n")
	b.WriteString("  1. action=\"show_apis\", service=<name>                                    → list endpoint names + descriptions\n")
	b.WriteString("  2. action=\"show_api\",  service=<name>, endpoint=<name>                   → full parameter docs for one endpoint\n")
	b.WriteString("  3. action=<endpoint>,  service=<name>, [url_params, query_params, body]   → execute the call\n")
	b.WriteString("\nRepository:\n")
	b.WriteString("  4. action=\"list_api_configs\"                                              → list service configs available to install from GitHub\n")
	b.WriteString("  5. action=\"install_api_config\", configName=<name>                        → download and hot-load a service config from GitHub\n")
	return b.String()
}

func (a *Api) generateTroubleshootingInfo() string {
	return `Troubleshooting:
- "unknown service": use one of the service names listed in the tool description
- "unknown endpoint": call show_apis first to see available endpoint names
- "endpoint parameter is required": provide the endpoint field when using show_api
- "URL resolution failed": an ${ENV_VAR} referenced in the endpoint URL is unset
- "basic_auth resolution failed": CLOUDINARY_API_KEY or CLOUDINARY_API_SECRET env vars are unset
- "configName is required": provide configName for install_api_config
- "repository not configured": this tool instance does not support repository actions
- HTTP 4xx: client error — check parameters and authentication
- HTTP 5xx: server error — try again later`
}

func (a *Api) generateParameters() []core.Parameter {
	serviceNames := a.getServiceNames()
	serviceDesc := "The API service to target (required for show_apis, show_api, and endpoint calls)"
	if len(serviceNames) > 0 {
		serviceDesc += fmt.Sprintf(". Available: %s", strings.Join(serviceNames, ", "))
	}
	return []core.Parameter{
		{
			Name:        "action",
			Type:        "string",
			Description: `"show_apis", "show_api", an endpoint name to execute a call, "list_api_configs", or "install_api_config"`,
			Required:    true,
		},
		{
			Name:        "service",
			Type:        "string",
			Description: serviceDesc,
			Required:    false,
		},
		{
			Name:        "endpoint",
			Type:        "string",
			Description: "Endpoint name — required for show_api and when executing a call",
			Required:    false,
		},
		{
			Name:        "configName",
			Type:        "string",
			Description: "Service config name to install (filename without .json from the remote repository/api_configs/ folder) — required for install_api_config",
			Required:    false,
		},
		{
			Name:        "url_params",
			Type:        "object",
			Description: `URL path parameters, e.g. {"media_id": "123"} for a URL containing /{media_id}`,
			Required:    false,
		},
		{
			Name:        "query_params",
			Type:        "object",
			Description: "Query string parameters as a key-value object",
			Required:    false,
		},
		{
			Name:        "body",
			Type:        "object",
			Description: "Request body as a key-value object — serialized to JSON or form-encoded automatically based on the endpoint",
			Required:    false,
		},
	}
}
