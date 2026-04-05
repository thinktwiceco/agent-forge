package agents

import (
	"testing"

	"github.com/thinktwiceco/agent-forge/src/history"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

func TestInjectSystemPrompt_NoProvider_AddsSystem(t *testing.T) {
	a := &Agent{
		config:       &AgentConfig{AgentName: "t", SystemPrompt: "base"},
		systemPrompt: "built base",
	}
	hm := history.NewConversationHistory()
	a.injectSystemPrompt(hm)
	msgs := hm.Messages()
	if len(msgs) != 1 || msgs[0].Role() != llms.MessageRoleSystem {
		t.Fatalf("want one system message, got %#v", msgs)
	}
	if got := msgs[0].Content(); got != "built base" {
		t.Errorf("content = %q, want built base", got)
	}
}

func TestInjectSystemPrompt_WithProvider_PrependsShortTermBlock(t *testing.T) {
	a := &Agent{
		config:       &AgentConfig{AgentName: "t", SystemPrompt: "base"},
		systemPrompt: "built base",
		memoryPrefixProvider: func() string {
			return "line1\nline2"
		},
	}
	hm := history.NewConversationHistory()
	a.injectSystemPrompt(hm)
	msgs := hm.Messages()
	if len(msgs) != 1 {
		t.Fatalf("want 1 message, got %d", len(msgs))
	}
	got := msgs[0].Content()
	want := "[SHORT_TERM_MEMORY]\nline1\nline2\n\nbuilt base"
	if got != want {
		t.Errorf("content mismatch\n got: %q\nwant: %q", got, want)
	}
}

func TestInjectSystemPrompt_ReplaceFirstSystemWhenLoaded(t *testing.T) {
	a := &Agent{
		config:       &AgentConfig{AgentName: "t"},
		systemPrompt: "fresh base",
		memoryPrefixProvider: func() string {
			return "mem"
		},
	}
	hm := history.NewConversationHistory()
	hm.SetMessages([]*llms.UnifiedMessage{
		llms.SystemMessage("stale"),
		llms.UserMessage("hi"),
	})
	a.injectSystemPrompt(hm)
	msgs := hm.Messages()
	if len(msgs) != 2 {
		t.Fatalf("want 2 messages, got %d", len(msgs))
	}
	want := "[SHORT_TERM_MEMORY]\nmem\n\nfresh base"
	if msgs[0].Content() != want {
		t.Errorf("system = %q, want %q", msgs[0].Content(), want)
	}
	if msgs[1].Content() != "hi" {
		t.Errorf("user message lost: %q", msgs[1].Content())
	}
}

func TestInjectSystemPrompt_EmptyPrefixSkipsWrapper(t *testing.T) {
	a := &Agent{
		config:       &AgentConfig{AgentName: "t"},
		systemPrompt: "only",
		memoryPrefixProvider: func() string {
			return "   "
		},
	}
	hm := history.NewConversationHistory()
	a.injectSystemPrompt(hm)
	if hm.Messages()[0].Content() != "only" {
		t.Errorf("got %q", hm.Messages()[0].Content())
	}
}
