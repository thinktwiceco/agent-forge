package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// Tools implements core.ToolProvider.
//
// Tool categories:
//
//	Memory — topic-organized session LTM in graph + optional short-term MEMORY.md:
//	  get_conversation_topics, recall_recent_conversations, recall_older_conversation,
//	  retrieve_conversation (ephemeral full text),
//	  memory_read_short_term, memory_patch_short_term, save_short_term_memory, forget
//
//	Distillation:
//	  dream (on-demand dreaming)
//
//	Search:
//	  find
func (p *BrainPlugin) Tools() []llms.Tool {
	return []llms.Tool{
		// Memory / recall tools
		p.newGetConversationTopicsTool(),
		p.newRecallRecentConversationsTool(),
		p.newRecallOlderConversationTool(),
		p.newRetrieveConversationTool(),
		p.newMemoryReadShortTermTool(),
		p.newMemoryPatchShortTermTool(),
		p.newSaveShortTermMemoryTool(),
		p.newForgetTool(),
		p.newDreamTool(),
		// Search
		p.newFindTool(),
	}
}

// ─── Memory tools ─────────────────────────────────────────────────────────────
//
// Session long-term memory is stored as conversation nodes in brain.db (distilled
// summary in content). Optional brain/MEMORY.md: quick daily facts, easy to read and rewrite.

func (p *BrainPlugin) newRecallRecentConversationsTool() llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name: "recall_recent_conversations",
		Description: "List conversation memory from the graph for the rolling past 7 days (local time). " +
			"Each row includes id, title, and description. Optionally filter by topic name from get_conversation_topics.",
		Parameters: []core.Parameter{
			{
				Name:        "topic",
				Type:        "string",
				Description: "Optional topic name to filter by.",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			now := time.Now()
			start := now.AddDate(0, 0, -7)
			topic, _ := args["topic"].(string)
			items, err := p.listConversationsInTimeRange(start, now, topic)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to list conversations: %v", err))
			}
			b, _ := json.Marshal(items)
			return core.NewSuccessResponse(string(b))
		},
	})
}

func (p *BrainPlugin) newRecallOlderConversationTool() llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name: "recall_older_conversation",
		Description: "List conversation memory nodes whose activity date falls between fromDate and toDate inclusive. " +
			"Dates are calendar days in local time (YYYY-MM-DD). Optionally filter by topic name from get_conversation_topics.",
		Parameters: []core.Parameter{
			{
				Name:        "fromDate",
				Type:        "string",
				Description: "Start date inclusive, format YYYY-MM-DD (local).",
				Required:    true,
			},
			{
				Name:        "toDate",
				Type:        "string",
				Description: "End date inclusive, format YYYY-MM-DD (local).",
				Required:    true,
			},
			{
				Name:        "topic",
				Type:        "string",
				Description: "Optional topic name to filter by.",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			fromStr, _ := args["fromDate"].(string)
			toStr, _ := args["toDate"].(string)
			topic, _ := args["topic"].(string)
			if fromStr == "" || toStr == "" {
				return core.NewErrorResponse("fromDate and toDate are required (YYYY-MM-DD)")
			}
			loc := time.Local
			fromDay, err := time.ParseInLocation("2006-01-02", fromStr, loc)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("invalid fromDate: %v", err))
			}
			toDay, err := time.ParseInLocation("2006-01-02", toStr, loc)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("invalid toDate: %v", err))
			}
			if toDay.Before(fromDay) {
				return core.NewErrorResponse("toDate must be on or after fromDate")
			}
			start := startOfDayLocal(fromDay)
			end := endOfDayLocal(toDay)
			items, err := p.listConversationsInTimeRange(start, end, topic)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to list conversations: %v", err))
			}
			b, _ := json.Marshal(items)
			return core.NewSuccessResponse(string(b))
		},
	})
}

func (p *BrainPlugin) newGetConversationTopicsTool() llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name:        "get_conversation_topics",
		Description: "List available long-term memory topics. Returns topic id, name, and conversation_count.",
		Parameters:  []core.Parameter{},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			items, err := p.listConversationTopics()
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to list topics: %v", err))
			}
			b, _ := json.Marshal(items)
			return core.NewSuccessResponse(string(b))
		},
	})
}

func (p *BrainPlugin) newRetrieveConversationTool() llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name: "retrieve_conversation",
		Description: "Return the distilled summary for a conversation graph node id, plus distillation_reason when present. " +
			"Result is ephemeral: it is not persisted verbatim in chat history. Use after get_conversation_topics or recall_* to pick an id.",
		Parameters: []core.Parameter{
			{
				Name:        "id",
				Type:        "string",
				Description: "Conversation graph id from recall_recent_conversations or recall_older_conversation.",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			id, _ := args["id"].(string)
			if id == "" {
				return core.NewErrorResponse("id is required")
			}
			node, err := p.getNode(id)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("node not found: %v", err))
			}
			if node.Type != nodeTypeConversation || p.isOmniaNuncNode(node.ID) {
				return core.NewErrorResponse("id must be a conversation session node")
			}
			if err := node.Validate(); err != nil {
				return core.NewErrorResponse(fmt.Sprintf("conversation not yet distilled: %v", err))
			}
			out := p.formatRetrieveConversation(node)
			return core.NewSuccessEphemeralResponse(out)
		},
	})
}

// newMemoryReadShortTermTool reads brain/MEMORY.md (short-term daily working notes).
func (p *BrainPlugin) newMemoryReadShortTermTool() llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name:        "memory_read_short_term",
		Description: "Read brain/MEMORY.md: short, easy-to-update daily working facts (preferences, reminders, current context). For graph-backed session memory use get_conversation_topics / recall_* / find.",
		Parameters:  []core.Parameter{},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			if p.workingDir == "" {
				return core.NewErrorResponse("working directory not configured")
			}
			data, err := p.readMemoryMDRaw()
			if os.IsNotExist(err) {
				return core.NewSuccessResponse(
					"No brain/MEMORY.md yet (optional daily working notes file). " +
						"Topic and conversation summaries live in the brain graph — use get_conversation_topics, recall_recent_conversations, recall_older_conversation, find, or retrieve_conversation.",
				)
			}
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to read MEMORY.md: %v", err))
			}
			return core.NewSuccessResponse(string(data))
		},
	})
}

// newMemoryPatchShortTermTool rewrites MEMORY.md (short-term daily working notes).
// The agent should read current content first, merge, then patch.
func (p *BrainPlugin) newMemoryPatchShortTermTool() llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name: "memory_patch_short_term",
		Description: "Rewrite brain/MEMORY.md with new content (daily working notes: brief, easy to scan). " +
			"Read with memory_read_short_term first, merge edits, then pass the full updated text here.",
		Parameters: []core.Parameter{
			{
				Name:        "content",
				Type:        "string",
				Description: "Complete new body for MEMORY.md (replaces the file).",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			content, ok := args["content"].(string)
			if !ok || content == "" {
				return core.NewErrorResponse("content parameter is required")
			}
			if p.workingDir == "" {
				return core.NewErrorResponse("working directory not configured")
			}
			if err := p.writeMemoryMDFull(content); err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to write MEMORY.md: %v", err))
			}
			return core.NewSuccessResponse("MEMORY.md updated.")
		},
	})
}

// newSaveShortTermMemoryTool appends one line "topic: fact" to brain/MEMORY.md.
func (p *BrainPlugin) newSaveShortTermMemoryTool() llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name: "save_short_term_memory",
		Description: "Append one line to brain/MEMORY.md as \"topic: fact\". " +
			"Use sparingly for important daily decisions only; MEMORY.md is also injected into context automatically each turn.",
		Parameters: []core.Parameter{
			{
				Name:        "topic",
				Type:        "string",
				Description: "Short label for the fact (single line, no newlines).",
				Required:    true,
			},
			{
				Name:        "fact",
				Type:        "string",
				Description: "The fact to remember (single line, no newlines).",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			if p.workingDir == "" {
				return core.NewErrorResponse("working directory not configured")
			}
			topic, _ := args["topic"].(string)
			fact, _ := args["fact"].(string)
			topic = strings.TrimSpace(topic)
			fact = strings.TrimSpace(fact)
			if topic == "" || fact == "" {
				return core.NewErrorResponse("topic and fact are required")
			}
			if strings.ContainsAny(topic, "\r\n") || strings.ContainsAny(fact, "\r\n") {
				return core.NewErrorResponse("topic and fact must be single-line (no newlines)")
			}
			if err := p.appendShortTermMemoryLineLocked(topic, fact); err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to update MEMORY.md: %v", err))
			}
			return core.NewSuccessResponse("Saved to MEMORY.md.")
		},
	})
}

func (p *BrainPlugin) newForgetTool() llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name: "forget",
		Description: "Permanently remove long-term memory for a topic or one conversation. " +
			"Deletes matching graph nodes plus distilled markdown under brain/persistence/ and raw transcripts under data/conversations/. " +
			"Use scope [topic] to drop every conversation linked to that topic; use [conversation] with a recall id or conv_id.",
		Parameters: []core.Parameter{
			{
				Name:        "scope",
				Type:        "string",
				Description: `Must be "topic" or "conversation".`,
				Required:    true,
			},
			{
				Name:        "target",
				Type:        "string",
				Description: "Topic name or topic id from get_conversation_topics; or conversation graph id / conv_id from recall.",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			scope, _ := args["scope"].(string)
			target, _ := args["target"].(string)
			res, err := p.Forget(scope, target)
			if err != nil {
				return core.NewErrorResponse(err.Error())
			}
			b, _ := json.Marshal(res)
			return core.NewSuccessResponse(string(b))
		},
	})
}

const dreamToolTimeout = 30 * time.Minute

func (p *BrainPlugin) newDreamTool() llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name: "dream",
		Description: "Run memory distillation now (same pipeline as scheduled dreaming): process pending conversation JSON into brain/persistence and update graph nodes, then run a MEMORY.md cleanup pass (may rewrite brain/MEMORY.md and optionally promote at most one durable item into the graph). " +
			"Scheduled runs happen only at brain_plugin.dreamTime; brain_plugin.dream: off disables those only, not this tool. " +
			"Requires a configured model; may process multiple conversations and may take a long time on a large backlog.",
		Parameters: []core.Parameter{},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			ctx, cancel := context.WithTimeout(context.Background(), dreamToolTimeout)
			defer cancel()
			if err := p.runDreaming(ctx); err != nil {
				return core.NewErrorResponse(err.Error())
			}
			return core.NewSuccessResponse("Distillation finished (pending transcripts per usual rules, then MEMORY.md cleanup if the file had content).")
		},
	})
}

// newFindTool creates the find tool
func (p *BrainPlugin) newFindTool() llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name: "find",
		Description: "Full-text search across all brain nodes using BM25 ranking. " +
			"Returns nodes ordered by relevance score (higher = better match). " +
			"Prefer this over graph traversal when you know keywords but not which session to open.",
		Parameters: []core.Parameter{
			{
				Name:        "query",
				Type:        "string",
				Description: "Search query to match against node content",
				Required:    true,
			},
			{
				Name:        "limit",
				Type:        "number",
				Description: "Maximum number of results to return (default: 10)",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			query, ok := args["query"].(string)
			if !ok || query == "" {
				return core.NewErrorResponse("query parameter is required")
			}

			limit := 10
			if l, ok := args["limit"].(float64); ok {
				limit = int(l)
			}

			result, err := p.Find(query, limit)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to search: %v", err))
			}

			resultJSON, _ := json.Marshal(result)
			return core.NewSuccessResponse(string(resultJSON))
		},
	})
}
