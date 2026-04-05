package brain

import (
	"fmt"
	"strings"
	"time"
)

// GraphResult represents the result of a graph traversal query
type GraphResult struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// LightNodeWithEdge is a minimal neighbor representation that includes the connecting edge type.
type LightNodeWithEdge struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	EdgeType string `json:"edge_type"`
}

// NodeNeighborsResult is returned by out_nodes and in_nodes tools.
type NodeNeighborsResult struct {
	Node      LightNode           `json:"node"`
	Neighbors []LightNodeWithEdge `json:"neighbors"`
	Count     int                 `json:"count"`
}

// Node represents a brain node in the graph
type Node struct {
	ID                 string         `json:"id"`
	Type               string         `json:"type"`
	Content            string         `json:"content"`
	Title              string         `json:"title,omitempty"`
	Description        string         `json:"description,omitempty"`
	DistillationReason string         `json:"distillation_reason,omitempty"`
	SearchText         string         `json:"search_text,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// Edge represents a relationship between two nodes
type Edge struct {
	ID           string         `json:"id"`
	FromNodeID   string         `json:"from_node_id"`
	ToNodeID     string         `json:"to_node_id"`
	RelationType string         `json:"relation_type"`
	Weight       float64        `json:"weight"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// Type represents a node or edge type definition
type Type struct {
	ID          string         `json:"id"`
	Category    string         `json:"category"` // "node_type" or "edge_type"
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// ScoredNode represents a node with a relevance score
type ScoredNode struct {
	Node  Node    `json:"node"`
	Score float64 `json:"score"`
}

// LightNode is a minimal node representation used during graph traversal.
// It lets the agent browse available nodes by title before deciding which to explore fully.
type LightNode struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

// ConversationRecallItem is returned by recall_recent_conversations and recall_older_conversation.
type ConversationRecallItem struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Topics      []string `json:"topics"`
}

// ConversationTopicItem is returned by get_conversation_topics.
type ConversationTopicItem struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	ConversationCount int    `json:"conversation_count"`
}

// GetTitle returns the title column, then metadata.title, then a truncated content snippet.
func (n *Node) GetTitle() string {
	if strings.TrimSpace(n.Title) != "" {
		return strings.TrimSpace(n.Title)
	}
	if n.Metadata != nil {
		if title, ok := n.Metadata["title"].(string); ok && title != "" {
			return title
		}
	}
	if len(n.Content) > 80 {
		return n.Content[:80] + "..."
	}
	return n.Content
}

// toLightNode converts a Node to a LightNode.
func toLightNode(n Node) LightNode {
	return LightNode{ID: n.ID, Type: n.Type, Title: n.GetTitle()}
}

// Validate checks that a conversation node has all required fields populated.
// Non-conversation nodes always pass. Returns a descriptive error listing missing fields.
func (n *Node) Validate() error {
	if n.Type != "conversation" {
		return nil
	}
	var missing []string
	if strings.TrimSpace(n.Title) == "" {
		missing = append(missing, "title")
	}
	if strings.TrimSpace(n.Description) == "" {
		missing = append(missing, "description")
	}
	if strings.TrimSpace(n.DistillationReason) == "" {
		missing = append(missing, "distillation_reason")
	}
	if n.Metadata == nil {
		missing = append(missing, "metadata")
	}
	if len(missing) > 0 {
		return fmt.Errorf("conversation node missing required fields: %s", strings.Join(missing, ", "))
	}
	return nil
}
