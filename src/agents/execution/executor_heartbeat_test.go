package execution

import (
	"testing"

	"github.com/thinktwiceco/agent-forge/src/llms"
)

func TestLastUserContent(t *testing.T) {
	msgs := []*llms.UnifiedMessage{
		llms.UserMessage("first"),
		llms.AssistantMessage("mid", 0, 0, 0),
		llms.UserMessage("task_type: heartbeat_tick\n\nbody"),
	}
	if got := lastUserContent(msgs); got != "task_type: heartbeat_tick\n\nbody" {
		t.Fatalf("last user: %q", got)
	}
	if lastUserContent([]*llms.UnifiedMessage{llms.AssistantMessage("only", 0, 0, 0)}) != "" {
		t.Fatal("expected empty when no user")
	}
}
