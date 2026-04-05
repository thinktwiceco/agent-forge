package brain

import (
	"database/sql"
	"encoding/json"
	"strings"
)

// sqlBrainNodeColumns is the canonical column list for brain_nodes (excluding edges).
const sqlBrainNodeColumns = `id, type, content, metadata, title, description, distillation_reason, search_text, created_at, updated_at`

// sqlBrainNodeColumnsN is the same columns with table alias n. for JOIN queries.
const sqlBrainNodeColumnsN = `n.id, n.type, n.content, n.metadata, n.title, n.description, n.distillation_reason, n.search_text, n.created_at, n.updated_at`

// buildSearchText concatenates non-empty fields for FTS and row updates.
func buildSearchText(content, title, description, distillationReason string) string {
	var parts []string
	for _, s := range []string{
		strings.TrimSpace(content),
		strings.TrimSpace(title),
		strings.TrimSpace(description),
		strings.TrimSpace(distillationReason),
	} {
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "\n")
}

func applyNodeMetadataJSON(meta sql.NullString, node *Node) {
	if meta.Valid && meta.String != "" {
		if err := json.Unmarshal([]byte(meta.String), &node.Metadata); err != nil {
			node.Metadata = make(map[string]any)
		}
		return
	}
	node.Metadata = nil
}

// scanBrainNodeFromScanner scans one row with sqlBrainNodeColumns order into Node.
func scanBrainNodeFromScanner(scanner interface {
	Scan(dest ...interface{}) error
}) (Node, error) {
	var node Node
	var meta sql.NullString
	var title, desc, reason, searchText sql.NullString
	var createdAt, updatedAt string
	err := scanner.Scan(
		&node.ID, &node.Type, &node.Content, &meta,
		&title, &desc, &reason, &searchText,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return Node{}, err
	}
	applyNodeMetadataJSON(meta, &node)
	if title.Valid {
		node.Title = title.String
	}
	if desc.Valid {
		node.Description = desc.String
	}
	if reason.Valid {
		node.DistillationReason = reason.String
	}
	if searchText.Valid {
		node.SearchText = searchText.String
	}
	node.CreatedAt = parseSQLiteDatetime(createdAt)
	node.UpdatedAt = parseSQLiteDatetime(updatedAt)
	return node, nil
}
