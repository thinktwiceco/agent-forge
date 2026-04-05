package scheduler

import (
	"fmt"
	"time"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

const scheduleTool = "schedule"

func newScheduleTool(p *SchedulerPlugin) llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name:        scheduleTool,
		Description: "Schedule a reminder message to be delivered to the agent at a specific future time.",
		AdvanceDesc: `Advanced Details:
- Parameters:
  * message (string, required): The reminder text the agent will receive when the task fires
  * scheduled_at (string, required): RFC3339 datetime when the reminder fires (e.g. 2026-03-05T15:00:00Z)
  * chat_id (string, optional): Resume an existing conversation; empty string starts a new one
- Returns a task_id for reference
- The reminder arrives as an inbox message with headers: sender=scheduler, task_type=agent_reminder, task_id=<id>
- The agent does not need to wait or poll — it will be notified automatically`,
		TroubleshootingInfo: `Troubleshooting:
- "scheduler not initialized" error: The scheduler plugin must be listed in the agent config plugins list
- Invalid scheduled_at: Must be a valid RFC3339 string (e.g. 2026-03-05T15:00:00Z)
- Reminder not received: Ensure the agent process is running at the scheduled time; the poller checks every 5 seconds`,
		Parameters: []core.Parameter{
			{
				Name:        "message",
				Type:        "string",
				Description: "The reminder message to deliver to the agent",
				Required:    true,
			},
			{
				Name:        "scheduled_at",
				Type:        "string",
				Description: "RFC3339 datetime when the reminder fires (e.g. 2026-03-05T15:00:00Z)",
				Required:    true,
			},
			{
				Name:        "chat_id",
				Type:        "string",
				Description: "Optional conversation ID to resume. Empty string starts a new conversation.",
				Required:    false,
			},
		},
		Handler: func(agentCtx map[string]any, args map[string]any) llms.ToolReturn {
			if p.sched == nil {
				return core.NewErrorResponse("scheduler not initialized")
			}

			message, _ := args["message"].(string)
			if message == "" {
				return core.NewErrorResponse("message is required")
			}

			scheduledAtStr, _ := args["scheduled_at"].(string)
			if scheduledAtStr == "" {
				return core.NewErrorResponse("scheduled_at is required")
			}

			scheduledAt, err := time.Parse(time.RFC3339, scheduledAtStr)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("invalid scheduled_at format, expected RFC3339: %v", err))
			}

			chatID, _ := args["chat_id"].(string)
			// Fall back to the current conversation so the reminder is delivered here.
			if chatID == "" {
				chatID, _ = agentCtx["chatId"].(string)
			}

			id, err := p.sched.schedule(TaskTypeAgentReminder, message, chatID, scheduledAt)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to schedule reminder: %v", err))
			}

			return core.NewSuccessResponse(fmt.Sprintf(
				"Reminder scheduled (task_id=%d) for %s", id, scheduledAt.Format(time.RFC3339),
			))
		},
	})
}
