package brain

// recategorizePendingOnly assigns concrete topic labels to conversation nodes that
// are only linked to the default [pending] topic after distillation left them there
// (empty topics from the distiller or explicit fallback in updateConversationNodeSummary).

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

type recategorizeTopicsResult struct {
	Topics []string `json:"topics"`
}

func dreamedAtFromNode(n *Node) time.Time {
	if n == nil || n.Metadata == nil {
		return time.Now()
	}
	s, ok := n.Metadata["dreamed_at"].(string)
	if !ok || strings.TrimSpace(s) == "" {
		return time.Now()
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Now()
	}
	return t
}

func conversationConvIDFromNode(n *Node) string {
	if n == nil || n.Metadata == nil {
		return ""
	}
	s, ok := n.Metadata["conv_id"].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

// onlyTopicIsPending reports whether the conversation is linked solely to the
// default pending topic (the case we want to recategorize).
func (p *BrainPlugin) onlyTopicIsPending(conversationNodeID string) (bool, error) {
	topics, err := p.getTopicsForConversationNodeID(conversationNodeID)
	if err != nil {
		return false, err
	}
	if len(topics) != 1 {
		return false, nil
	}
	return normalizeTopicName(topics[0].GetTitle()) == defaultConversationTopic, nil
}

func topicsWithoutPending(topics []string) []string {
	topics = normalizeTopicNames(topics)
	var out []string
	for _, t := range topics {
		if t == defaultConversationTopic {
			continue
		}
		out = append(out, t)
	}
	return normalizeTopicNames(out)
}

// recategorizePendingOnly lists conversations under the [pending] topic, keeps those
// that are dreamed and only on [pending], and reassigns topics via LLM.
func (d *DreamingRunner) recategorizePendingOnly(ctx context.Context) error {
	if d.plugin == nil || d.plugin.db == nil || d.llmEngine == nil {
		return nil
	}

	nodes, err := d.plugin.getConversationsForTopic(defaultConversationTopic)
	if err != nil {
		if strings.Contains(err.Error(), "topic not found") {
			agentforge.Debug("🧠 [Dreaming] Pending recategorize: no [pending] topic node yet")
			return nil
		}
		return fmt.Errorf("list conversations under pending: %w", err)
	}
	if len(nodes) == 0 {
		return nil
	}

	var work []Node
	for i := range nodes {
		n := &nodes[i]
		if d.plugin.isOmniaNuncNode(n.ID) {
			continue
		}
		convID := conversationConvIDFromNode(n)
		if convID == "" {
			continue
		}
		if strings.HasPrefix(convID, syntheticMemoryConvIDPrefix) {
			continue
		}
		if n.Metadata == nil {
			continue
		}
		if _, ok := n.Metadata["dreamed_at"].(string); !ok {
			continue
		}
		ok, err := d.plugin.onlyTopicIsPending(n.ID)
		if err != nil {
			agentforge.Debug("🧠 [Dreaming] Pending recategorize: topics for %s: %v", convID, err)
			continue
		}
		if !ok {
			continue
		}
		if strings.TrimSpace(n.Title) == "" || strings.TrimSpace(n.Description) == "" || strings.TrimSpace(n.DistillationReason) == "" {
			continue
		}
		work = append(work, *n)
	}

	if len(work) == 0 {
		return nil
	}

	agentforge.Debug("🧠 [Dreaming] Pending recategorize: %d conversation(s)", len(work))
	for i := range work {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		n := &work[i]
		convID := conversationConvIDFromNode(n)
		rawTopics, err := d.callRecategorizeTopicsLLM(ctx, n)
		if err != nil {
			agentforge.Debug("🧠 [Dreaming] Pending recategorize LLM failed for %s: %v", convID, err)
			continue
		}
		topics := topicsWithoutPending(rawTopics)
		if len(topics) == 0 {
			agentforge.Debug("🧠 [Dreaming] Pending recategorize: no substantive topics for %s", convID)
			continue
		}
		dreamedAt := dreamedAtFromNode(n)
		summary := d.plugin.readSummaryFromFile(convID, dreamedAt)
		if err := d.plugin.updateConversationNodeSummary(convID, summary, topics, n.Title, n.Description, n.DistillationReason, dreamedAt); err != nil {
			agentforge.Debug("🧠 [Dreaming] Pending recategorize: could not update node for %s: %v", convID, err)
			continue
		}
		agentforge.Debug("🧠 [Dreaming] Pending recategorize: %s → topics %v", convID, topics)
	}
	return nil
}

func (d *DreamingRunner) callRecategorizeTopicsLLM(ctx context.Context, n *Node) ([]string, error) {
	currentTopicsLine := d.rulesSectionCurrentTopicsLine()
	systemMsg := llms.SystemMessage(`You assign topic labels for a conversation already stored in long-term memory.
The conversation was only under the placeholder topic [pending]. Assign 1–5 substantive topics so it can be filed correctly.

Return strict JSON only:
{"topics":["topic one","topic two"]}

Rules:
` + currentTopicsLine + `- Prefer labels from CURRENT TOPICS (exact spelling, lowercase). New concise lowercase labels only if nothing fits.
- Topics must be stable and specific enough to group similar future sessions (e.g. work, health, project-x).
- Do not use "pending" or empty strings.
- If you cannot pick any label, return {"topics":[]}`)

	dreamedAt := dreamedAtFromNode(n)
	convID := conversationConvIDFromNode(n)
	summary := d.plugin.readSummaryFromFile(convID, dreamedAt)
	userMsg := llms.UserMessage(fmt.Sprintf(
		"TITLE:\n%s\n\nDESCRIPTION:\n%s\n\nDISTILLATION REASON:\n%s\n\nSUMMARY (memory body):\n%s",
		strings.TrimSpace(n.Title),
		strings.TrimSpace(n.Description),
		strings.TrimSpace(n.DistillationReason),
		strings.TrimSpace(summary),
	))

	responseCh := d.llmEngine.ChatStream([]*llms.UnifiedMessage{systemMsg, userMsg}, nil)

	var fullContent string
	for chunk := range responseCh.Start() {
		if chunk.Status == llms.StatusError {
			return nil, fmt.Errorf("llm stream error: %s", chunk.Content)
		}
		if len(chunk.FullContent) > len(fullContent) {
			fullContent = chunk.FullContent
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	fullContent = strings.TrimSpace(fullContent)
	fullContent = strings.TrimPrefix(fullContent, "```json")
	fullContent = strings.TrimPrefix(fullContent, "```")
	fullContent = strings.TrimSuffix(fullContent, "```")
	fullContent = strings.TrimSpace(fullContent)

	var out recategorizeTopicsResult
	if err := json.Unmarshal([]byte(fullContent), &out); err != nil {
		return nil, fmt.Errorf("parse recategorize JSON: %w", err)
	}
	return normalizeTopicNames(out.Topics), nil
}
