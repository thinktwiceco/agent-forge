package heartbeat

import (
	"testing"
)

func TestAddInstruction_Basic(t *testing.T) {
	m := NewHeartbeatManager()
	if err := m.AddInstruction("Check logs", "Review error logs daily."); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	titles := m.ListInstructions()
	if len(titles) != 1 || titles[0] != "Check logs" {
		t.Fatalf("expected [Check logs], got %v", titles)
	}
}

func TestAddInstruction_EmptyTitle(t *testing.T) {
	m := NewHeartbeatManager()
	if err := m.AddInstruction("", "body"); err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestAddInstruction_Overwrite(t *testing.T) {
	m := NewHeartbeatManager()
	_ = m.AddInstruction("Title", "first")
	_ = m.AddInstruction("Title", "second")
	titles := m.ListInstructions()
	if len(titles) != 1 {
		t.Fatalf("expected 1 entry after overwrite, got %d", len(titles))
	}
}

func TestRemoveInstruction_Exists(t *testing.T) {
	m := NewHeartbeatManager()
	_ = m.AddInstruction("Task A", "do A")
	if err := m.RemoveInstruction("Task A"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.ListInstructions()) != 0 {
		t.Fatal("expected empty list after removal")
	}
}

func TestRemoveInstruction_NotFound(t *testing.T) {
	m := NewHeartbeatManager()
	if err := m.RemoveInstruction("Missing"); err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestListInstructions_Sorted(t *testing.T) {
	m := NewHeartbeatManager()
	_ = m.AddInstruction("Zebra", "z")
	_ = m.AddInstruction("Alpha", "a")
	_ = m.AddInstruction("Mango", "m")
	got := m.ListInstructions()
	want := []string{"Alpha", "Mango", "Zebra"}
	for i, v := range want {
		if got[i] != v {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestListInstructions_Empty(t *testing.T) {
	m := NewHeartbeatManager()
	if titles := m.ListInstructions(); len(titles) != 0 {
		t.Fatalf("expected empty list, got %v", titles)
	}
}

func TestRenderInstructions_Format(t *testing.T) {
	m := NewHeartbeatManager()
	_ = m.AddInstruction("Daily", "Check inbox.")
	out := m.renderInstructions()
	if out != "## Daily\nCheck inbox.\n\n" {
		t.Fatalf("unexpected render output: %q", out)
	}
}

func TestRenderInstructions_Empty(t *testing.T) {
	m := NewHeartbeatManager()
	if out := m.renderInstructions(); out != "" {
		t.Fatalf("expected empty render, got %q", out)
	}
}
