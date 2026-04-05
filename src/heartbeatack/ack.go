package heartbeatack

import "strings"

// HeartbeatOK is the acknowledgment token the agent replies with when
// a heartbeat tick requires no visible output.
const HeartbeatOK = "HEARTBEAT_OK"

// HeartbeatTickHeader is the task_type value queued for periodic heartbeat turns.
const HeartbeatTickHeader = "heartbeat_tick"

// IsHeartbeatTickUserContent reports whether formatted user content is a heartbeat tick
// (headers include task_type: heartbeat_tick).
func IsHeartbeatTickUserContent(content string) bool {
	return strings.Contains(content, "task_type: "+HeartbeatTickHeader)
}

// ShouldSuppressAckReply is true when raw is an ack-only reply per stripAck rules.
func ShouldSuppressAckReply(raw string, maxAckChars int) bool {
	_, suppress := stripAck(raw, maxAckChars)
	return suppress
}

// stripAck returns ("", true) when raw is an ack-only reply: the assembled
// message contains HEARTBEAT_OK at the start or end and the remaining text
// is at most maxAckChars characters. Otherwise it returns (trimmed, false).
func stripAck(raw string, maxAckChars int) (string, bool) {
	t := strings.TrimSpace(raw)
	if !strings.Contains(t, HeartbeatOK) {
		return t, false
	}
	// Token at start
	after := strings.TrimSpace(strings.TrimPrefix(t, HeartbeatOK))
	if len(after) <= maxAckChars {
		return "", true
	}
	// Token at end
	before := strings.TrimSpace(strings.TrimSuffix(t, HeartbeatOK))
	if len(before) <= maxAckChars {
		return "", true
	}
	// Token in the middle — not a pure ack, leave as-is
	return t, false
}
