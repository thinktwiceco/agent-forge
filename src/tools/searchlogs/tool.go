package searchlogs

import (
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/sessionlog"
)

// NewSearchLogsTool creates a tool that regex-searches the current session log file.
func NewSearchLogsTool() llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name: "search_logs",
		Description: "Regex-search the current conversation session log (.logs file). " +
			"Returns matching lines only — never loads the full log into context.",
		AdvanceDesc: `Advanced Details:
- Searches data/conversations/{agentName}/{conversation_id}.logs under the agent working directory
- Defaults to the current conversation when conversation_id is omitted
- Use offset and head_limit to paginate through large result sets
- head_limit defaults to 50 when omitted or 0`,
		TroubleshootingInfo: `Troubleshooting:
- "session log not found": No .logs file yet for this conversation — run at least one turn first
- "invalid regex": Fix the exp pattern syntax
- "missing required parameter: exp": Provide the regex pattern
- "no active conversation": chatId is missing from agent context`,
		Parameters: []core.Parameter{
			{
				Name:        "exp",
				Type:        "string",
				Description: "Regex pattern to search for in the session log",
				Required:    true,
			},
			{
				Name:        "conversation_id",
				Type:        "string",
				Description: "Conversation id to search (defaults to the current session)",
				Required:    false,
			},
			{
				Name:        "offset",
				Type:        "integer",
				Description: "Number of matching lines to skip before returning results (default 0)",
				Required:    false,
			},
			{
				Name:        "head_limit",
				Type:        "integer",
				Description: "Maximum matching lines to return (default 50; 0 uses default)",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			exp, ok := args["exp"].(string)
			if !ok || exp == "" {
				return core.NewErrorResponse("missing required parameter: exp")
			}

			chatId, _ := args["conversation_id"].(string)
			if chatId == "" {
				chatId, _ = agentContext["chatId"].(string)
			}
			if chatId == "" {
				return core.NewErrorResponse("no active conversation: conversation_id is required when chatId is not in context")
			}

			agentName, _ := agentContext["agentName"].(string)
			if agentName == "" {
				return core.NewErrorResponse("agent name is not available in context")
			}

			workingDir, _ := agentContext["workingDir"].(string)

			offset := toInt(args["offset"])
			headLimit := toInt(args["head_limit"])

			path := sessionlog.LogPath(workingDir, agentName, chatId)
			result, err := sessionlog.SearchFile(path, exp, offset, headLimit)
			if err != nil {
				return core.NewErrorResponse(err.Error())
			}
			return core.NewSuccessResponse(result.String())
		},
	})
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}
