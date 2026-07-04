package persistence

import (
	"testing"

	"github.com/thinktwiceco/agent-forge/src/llms"
)

func TestTailSliceRecentPage(t *testing.T) {
	messages := make([]*llms.UnifiedMessage, 120)

	page, total, hasMore := tailSlice(messages, 50, 0)
	if total != 120 {
		t.Fatalf("expected total 120, got %d", total)
	}
	if len(page) != 50 {
		t.Fatalf("expected 50 messages, got %d", len(page))
	}
	if !hasMore {
		t.Fatal("expected hasMore true")
	}
}

func TestTailSliceOlderPage(t *testing.T) {
	messages := make([]*llms.UnifiedMessage, 120)
	page, total, hasMore := tailSlice(messages, 50, 50)
	if total != 120 || len(page) != 50 {
		t.Fatalf("expected 50 messages on older page, got %d (total=%d)", len(page), total)
	}
	if !hasMore {
		t.Fatal("expected hasMore true for middle page")
	}
}

func TestTailSliceOldestPage(t *testing.T) {
	messages := make([]*llms.UnifiedMessage, 120)
	page, _, hasMore := tailSlice(messages, 50, 100)
	if len(page) != 20 {
		t.Fatalf("expected 20 messages on oldest page, got %d", len(page))
	}
	if hasMore {
		t.Fatal("expected hasMore false on oldest page")
	}
}

func TestTailSliceSmallConversation(t *testing.T) {
	messages := make([]*llms.UnifiedMessage, 1)
	page, total, hasMore := tailSlice(messages, 50, 0)
	if total != 1 || len(page) != 1 || hasMore {
		t.Fatalf("unexpected small conversation result: len=%d total=%d hasMore=%v", len(page), total, hasMore)
	}
}
