package main

import (
	"path/filepath"
	"testing"
)

func TestTelegramThreadStoreResolveAndPersist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "telegram_thread_map.json")

	s := NewTelegramThreadStore(path)
	chat := "12345"
	if got := s.ResolveConversationID(chat); got != "webhook-telegram-12345" {
		t.Fatalf("default id: %s", got)
	}

	s.NewSession(chat)
	if got := s.ResolveConversationID(chat); got == "webhook-telegram-12345" || got == "" {
		t.Fatalf("expected new uuid, got %q", got)
	}

	// Reload from disk
	s2 := NewTelegramThreadStore(path)
	if s2.ResolveConversationID(chat) != s.ResolveConversationID(chat) {
		t.Fatal("persisted map mismatch after reload")
	}
}

func TestTelegramThreadStoreEmptyPathNoPanic(t *testing.T) {
	s := NewTelegramThreadStore("")
	s.NewSession("1")
	_ = s.ResolveConversationID("1")
}
