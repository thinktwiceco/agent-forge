package brain

import (
	"fmt"
	"strings"
)

// ForgetResult summarizes a forget operation (graph + optional files).
type ForgetResult struct {
	Scope        string   `json:"scope"`
	Target       string   `json:"target"`
	DeletedNodes int      `json:"deleted_nodes"`
	RemovedFiles []string `json:"removed_files"`
}

// conversationIDFromNode returns the session id used for transcript and persistence filenames.
func conversationIDFromNode(n *Node) string {
	if n == nil || n.Metadata == nil {
		return ""
	}
	if id, ok := n.Metadata["conv_id"].(string); ok {
		id = strings.TrimSpace(id)
		if id != "" {
			return id
		}
	}
	return ""
}

// Forget removes long-term memory for a topic or a single conversation: deletes matching
// distilled markdown and raw conversation JSON, then removes graph nodes via forgetCascade.
func (p *BrainPlugin) Forget(scope, target string) (ForgetResult, error) {
	scope = strings.TrimSpace(strings.ToLower(scope))
	target = strings.TrimSpace(target)
	if target == "" {
		return ForgetResult{}, fmt.Errorf("target is required")
	}
	if p.workingDir == "" {
		return ForgetResult{}, fmt.Errorf("working directory not configured")
	}
	switch scope {
	case "topic":
		return p.forgetTopic(target)
	case "conversation":
		return p.forgetConversation(target)
	default:
		return ForgetResult{}, fmt.Errorf(`scope must be "topic" or "conversation"`)
	}
}

func (p *BrainPlugin) resolveTopicNodeForForget(target string) (*Node, error) {
	topicNode, err := p.getTopicNodeByName(target)
	if err == nil {
		return topicNode, nil
	}
	n, err := p.getNode(target)
	if err != nil {
		return nil, fmt.Errorf("topic not found: %s", target)
	}
	if n.Type != nodeTypeTopic || p.isOmniaNuncNode(n.ID) {
		return nil, fmt.Errorf("not a topic node: %s", target)
	}
	return n, nil
}

func (p *BrainPlugin) resolveConversationNodeForForget(target string) (*Node, error) {
	conv, err := p.getConversationNodeByConvID(target)
	if err == nil {
		return conv, nil
	}
	n, err := p.getNode(target)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %s", target)
	}
	if n.Type != nodeTypeConversation || p.isOmniaNuncNode(n.ID) {
		return nil, fmt.Errorf("not a conversation node: %s", target)
	}
	return n, nil
}

func (p *BrainPlugin) forgetTopic(target string) (ForgetResult, error) {
	topicNode, err := p.resolveTopicNodeForForget(target)
	if err != nil {
		return ForgetResult{}, err
	}

	convs, err := p.getConversationsForTopicID(topicNode.ID)
	if err != nil {
		return ForgetResult{}, err
	}

	var allRemoved []string
	for i := range convs {
		cid := conversationIDFromNode(&convs[i])
		if cid == "" {
			continue
		}
		r, err := RemoveConversationArtifacts(p.workingDir, p.dir, cid)
		if err != nil {
			return ForgetResult{}, err
		}
		allRemoved = append(allRemoved, r...)
	}

	deleted, err := p.forgetCascade(topicNode.ID)
	if err != nil {
		return ForgetResult{}, err
	}

	return ForgetResult{
		Scope:        "topic",
		Target:       target,
		DeletedNodes: deleted,
		RemovedFiles: allRemoved,
	}, nil
}

func (p *BrainPlugin) forgetConversation(target string) (ForgetResult, error) {
	conv, err := p.resolveConversationNodeForForget(target)
	if err != nil {
		return ForgetResult{}, err
	}

	var allRemoved []string
	if cid := conversationIDFromNode(conv); cid != "" {
		allRemoved, err = RemoveConversationArtifacts(p.workingDir, p.dir, cid)
		if err != nil {
			return ForgetResult{}, err
		}
	}

	deleted, err := p.forgetCascade(conv.ID)
	if err != nil {
		return ForgetResult{}, err
	}

	return ForgetResult{
		Scope:        "conversation",
		Target:       target,
		DeletedNodes: deleted,
		RemovedFiles: allRemoved,
	}, nil
}
