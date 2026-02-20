package queue

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Message is a queued item containing variable metadata headers, a body, and a chat session ID.
//
// Headers are free-form key-value pairs. Common headers include:
//   - "sender": who sent the message (e.g. "user", an agent name)
//   - "timestamp": auto-injected RFC3339 UTC timestamp
//
// ChatId routes the message to the correct conversation session on the agent.
// An empty ChatId starts a new conversation.
type Message struct {
	Headers map[string]string
	Body    string
	ChatId  string
}

// Format renders the message as a header block followed by the body, separated by a blank line.
//
// Headers are emitted in sorted key order for deterministic output.
//
// Example:
//
//	sender: user
//	timestamp: 2026-02-19T10:30:00Z
//
//	Ciao!
func (m *Message) Format() string {
	return FormatHeaders(m.Body, m.Headers)
}

// FormatHeaders builds the enriched message string from an arbitrary header map and a body.
// A "timestamp" header is auto-injected (RFC3339 UTC) if not already present in headers.
// Headers are emitted in sorted key order so the output is deterministic.
//
// This helper is useful when you want header enrichment without going through the queue
// buffer (e.g. directly inside an HTTP handler).
func FormatHeaders(body string, headers map[string]string) string {
	merged := make(map[string]string, len(headers)+1)
	for k, v := range headers {
		merged[k] = v
	}
	if _, ok := merged["timestamp"]; !ok {
		merged["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	}

	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&sb, "%s: %s\n", k, merged[k])
	}
	sb.WriteString("\n")
	sb.WriteString(body)

	return sb.String()
}
