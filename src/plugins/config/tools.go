package config

import (
	"fmt"
	"strings"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

const ConfigToolName = "config"

const mutationsUnavailable = "Config mutations not available (no ConfigWriter configured)"

// Writer is the optional runtime hook for persisting config changes (Localforge sets this).
var Writer core.ConfigWriter

// SetWriter registers the ConfigWriter implementation (call before building the agent).
func SetWriter(w core.ConfigWriter) {
	Writer = w
}

var validToolNames = map[string]struct{}{
	"fs":         {},
	"web":        {},
	"git":        {},
	"postgres":   {},
	"api":        {},
	"instagram":  {},
	"update":     {},
	"telegram":   {},
}

var reservedPluginNames = map[string]struct{}{
	"brain":  {},
	"skills": {},
	"config": {},
}

func newConfigTool() llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name:        ConfigToolName,
		Description: "Read agent configuration reference documentation and mutate config.yaml (tools, plugins, heartbeat, dreaming).",
		AdvanceDesc: `[ACTIONS — read]
- get_config_reference: Return full configuration reference markdown
- get_tools_reference: Return the Tools Configuration section only

[ACTIONS — mutate] (Localforge only; triggers agent reload)
- add_tool: Add a tool entry. Required: name. Optional tool params as additional keys (headless, port, config_folder, postgresURL, mode, allowedTables, allowedSchemas)
- remove_tool: Remove a tool by name. Required: name
- add_plugin: Append a plugin name. Required: name
- remove_plugin: Remove a plugin from the YAML list. Required: name (cannot remove brain, skills, or config defaults)
- set_heartbeat: Set agent.heartbeat.every. Required: every (e.g. "30m", "0m" to disable)
- set_dream: Set dreaming. Required: status (on|off). Optional: time (HH:MM local)`,
		TroubleshootingInfo: `Troubleshooting:
- Mutations fail outside Localforge or when config_tool: false removed the plugin
- Use get_tools_reference before add_tool to see valid tool names and parameters
- After mutations the agent reloads; changes apply on the next turn`,
		Parameters: []core.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Required:    true,
				Description: "One of: get_config_reference, get_tools_reference, add_tool, remove_tool, add_plugin, remove_plugin, set_heartbeat, set_dream",
			},
			{
				Name:        "name",
				Type:        "string",
				Required:    false,
				Description: "Tool or plugin name (required for add_tool, remove_tool, add_plugin, remove_plugin)",
			},
			{
				Name:        "every",
				Type:        "string",
				Required:    false,
				Description: "Heartbeat interval for set_heartbeat (e.g. 30m)",
			},
			{
				Name:        "status",
				Type:        "string",
				Required:    false,
				Description: "Dreaming on or off for set_dream",
			},
			{
				Name:        "time",
				Type:        "string",
				Required:    false,
				Description: "Dream wall time HH:MM for set_dream",
			},
		},
		Handler: handleConfigTool,
	})
}

func handleConfigTool(_ map[string]any, args map[string]any) llms.ToolReturn {
	action, _ := args["action"].(string)
	action = strings.TrimSpace(action)
	if action == "" {
		return core.NewErrorResponse("action parameter is required")
	}

	switch action {
	case "get_config_reference":
		return handleGetConfigReference()
	case "get_tools_reference":
		return handleGetToolsReference()
	case "add_tool":
		return handleAddTool(args)
	case "remove_tool":
		return handleRemoveTool(args)
	case "add_plugin":
		return handleAddPlugin(args)
	case "remove_plugin":
		return handleRemovePlugin(args)
	case "set_heartbeat":
		return handleSetHeartbeat(args)
	case "set_dream":
		return handleSetDream(args)
	default:
		return core.NewErrorResponse(fmt.Sprintf("unknown action %q", action))
	}
}

func handleGetConfigReference() llms.ToolReturn {
	content, err := ReferenceContent()
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to load config reference: %v", err))
	}
	return core.NewSuccessResponse(content)
}

func handleGetToolsReference() llms.ToolReturn {
	content, err := ReferenceContent()
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to load config reference: %v", err))
	}
	section := ExtractToolsReference(content)
	if section == "" {
		return core.NewErrorResponse("tools configuration section not found in reference")
	}
	return core.NewSuccessResponse(section)
}

func requireWriter() (core.ConfigWriter, bool) {
	return Writer, Writer != nil
}

func handleAddTool(args map[string]any) llms.ToolReturn {
	w, ok := requireWriter()
	if !ok {
		return core.NewErrorResponse(mutationsUnavailable)
	}
	name, ok := args["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		return core.NewErrorResponse("name parameter is required for add_tool")
	}
	name = strings.TrimSpace(name)
	if _, ok := validToolNames[name]; !ok {
		return core.NewErrorResponse(fmt.Sprintf("invalid tool name %q; use get_tools_reference for valid tools", name))
	}
	params := toolParamsFromArgs(args)
	if err := w.AddTool(name, params); err != nil {
		return core.NewErrorResponse(err.Error())
	}
	return core.NewSuccessResponse(fmt.Sprintf("Added tool %q. Agent reloaded.", name))
}

func handleRemoveTool(args map[string]any) llms.ToolReturn {
	w, ok := requireWriter()
	if !ok {
		return core.NewErrorResponse(mutationsUnavailable)
	}
	name, ok := args["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		return core.NewErrorResponse("name parameter is required for remove_tool")
	}
	if err := w.RemoveTool(strings.TrimSpace(name)); err != nil {
		return core.NewErrorResponse(err.Error())
	}
	return core.NewSuccessResponse(fmt.Sprintf("Removed tool %q. Agent reloaded.", strings.TrimSpace(name)))
}

func handleAddPlugin(args map[string]any) llms.ToolReturn {
	w, ok := requireWriter()
	if !ok {
		return core.NewErrorResponse(mutationsUnavailable)
	}
	name, ok := args["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		return core.NewErrorResponse("name parameter is required for add_plugin")
	}
	name = strings.TrimSpace(name)
	if _, ok := reservedPluginNames[name]; ok {
		return core.NewErrorResponse(fmt.Sprintf("plugin %q is loaded by default and cannot be added via add_plugin", name))
	}
	if err := w.AddPlugin(name); err != nil {
		return core.NewErrorResponse(err.Error())
	}
	return core.NewSuccessResponse(fmt.Sprintf("Added plugin %q. Agent reloaded.", name))
}

func handleRemovePlugin(args map[string]any) llms.ToolReturn {
	w, ok := requireWriter()
	if !ok {
		return core.NewErrorResponse(mutationsUnavailable)
	}
	name, ok := args["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		return core.NewErrorResponse("name parameter is required for remove_plugin")
	}
	name = strings.TrimSpace(name)
	if _, ok := reservedPluginNames[name]; ok {
		return core.NewErrorResponse(fmt.Sprintf("plugin %q is a default plugin and cannot be removed from config", name))
	}
	if err := w.RemovePlugin(name); err != nil {
		return core.NewErrorResponse(err.Error())
	}
	return core.NewSuccessResponse(fmt.Sprintf("Removed plugin %q. Agent reloaded.", name))
}

func handleSetHeartbeat(args map[string]any) llms.ToolReturn {
	w, ok := requireWriter()
	if !ok {
		return core.NewErrorResponse(mutationsUnavailable)
	}
	every, ok := args["every"].(string)
	if !ok || strings.TrimSpace(every) == "" {
		return core.NewErrorResponse("every parameter is required for set_heartbeat")
	}
	if err := w.SetHeartbeat(strings.TrimSpace(every)); err != nil {
		return core.NewErrorResponse(err.Error())
	}
	return core.NewSuccessResponse(fmt.Sprintf("Set heartbeat every to %q. Agent reloaded.", strings.TrimSpace(every)))
}

func handleSetDream(args map[string]any) llms.ToolReturn {
	w, ok := requireWriter()
	if !ok {
		return core.NewErrorResponse(mutationsUnavailable)
	}
	status, ok := args["status"].(string)
	if !ok || strings.TrimSpace(status) == "" {
		return core.NewErrorResponse("status parameter is required for set_dream (on or off)")
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "on" && status != "off" {
		return core.NewErrorResponse("status must be on or off")
	}
	timeVal, _ := args["time"].(string)
	timeVal = strings.TrimSpace(timeVal)
	if err := w.SetDream(status, timeVal); err != nil {
		return core.NewErrorResponse(err.Error())
	}
	msg := fmt.Sprintf("Set dream to %q.", status)
	if timeVal != "" {
		msg = fmt.Sprintf("Set dream to %q at %q.", status, timeVal)
	}
	return core.NewSuccessResponse(msg + " Agent reloaded.")
}

var toolParamKeys = map[string]struct{}{
	"headless":       {},
	"port":           {},
	"config_folder":  {},
	"postgresURL":    {},
	"mode":           {},
	"allowedTables":  {},
	"allowedSchemas": {},
}

func toolParamsFromArgs(args map[string]any) map[string]any {
	params := make(map[string]any)
	for key, val := range args {
		if key == "action" || key == "name" {
			continue
		}
		if _, ok := toolParamKeys[key]; ok {
			params[key] = val
		}
	}
	return params
}
