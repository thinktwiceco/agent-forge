package brain

import (
	"testing"
	"time"
)

func TestDeriveTitleAndDescriptionFromSummary(t *testing.T) {
	title, desc := deriveTitleAndDescriptionFromSummary("First line title\n\nSecond paragraph body goes here and should be trimmed.")
	if title != "First line title" {
		t.Errorf("title: got %q", title)
	}
	if desc == "" {
		t.Error("expected description")
	}
}

func TestEffectiveRecallTime(t *testing.T) {
	n := &Node{
		Metadata:  map[string]any{"last_access": "2026-04-01T12:00:00Z"},
		UpdatedAt: time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC),
	}
	got, err := effectiveRecallTime(n)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Year() != 2026 || got.Month() != 4 || got.Day() != 1 {
		t.Errorf("expected last_access date, got %v", got)
	}

	// Missing last_access must error.
	bad := &Node{Metadata: map[string]any{}}
	if _, err := effectiveRecallTime(bad); err == nil {
		t.Error("expected error for missing last_access")
	}

	// Malformed last_access must error.
	bad2 := &Node{Metadata: map[string]any{"last_access": "not-a-date"}}
	if _, err := effectiveRecallTime(bad2); err == nil {
		t.Error("expected error for malformed last_access")
	}
}

func TestTruncateRunes(t *testing.T) {
	if truncateRunes("hello", 3) != "hel..." {
		t.Errorf("got %q", truncateRunes("hello", 3))
	}
}

func TestNormalizeTopicNames(t *testing.T) {
	got := normalizeTopicNames([]string{" Science ", "science", "Deep   Work"})
	if len(got) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(got))
	}
	if got[0] != "deep work" || got[1] != "science" {
		t.Fatalf("unexpected normalized topics: %+v", got)
	}
}
