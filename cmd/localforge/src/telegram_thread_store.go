package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

func telegramThreadMapPath(cm *ConfigManager) string {
	cfg := cm.GetConfig()
	base := "data"
	if cfg.Agent.WorkingDir != "" {
		base = filepath.Join(cfg.Agent.WorkingDir, "data")
	}
	return filepath.Join(base, "telegram_thread_map.json")
}

// TelegramThreadStore maps Telegram chat IDs to Agent Forge conversation IDs
// (chatIds used for JSON history). When no override exists, Resolve uses the
// legacy id webhook-telegram-{chatID}.
type TelegramThreadStore struct {
	mu   sync.Mutex
	path string
	m    map[string]string
}

// NewTelegramThreadStore loads persisted mappings from path (best effort).
func NewTelegramThreadStore(path string) *TelegramThreadStore {
	s := &TelegramThreadStore{
		path: path,
		m:    make(map[string]string),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	if err := json.Unmarshal(data, &s.m); err != nil {
		s.m = make(map[string]string)
	}
	return s
}

// ResolveConversationID returns the active conversation id for a Telegram chat.
func (s *TelegramThreadStore) ResolveConversationID(telegramChatID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.m[telegramChatID]; ok && id != "" {
		return id
	}
	return "webhook-telegram-" + telegramChatID
}

// NewSession assigns a new conversation id for this Telegram chat and persists.
func (s *TelegramThreadStore) NewSession(telegramChatID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	newID := uuid.NewString()
	s.m[telegramChatID] = newID
	s.persistLocked()
	return newID
}

// KnownChatIDs returns all Telegram chat IDs that have ever had a session.
func (s *TelegramThreadStore) KnownChatIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.m))
	for chatID := range s.m {
		ids = append(ids, chatID)
	}
	return ids
}

func (s *TelegramThreadStore) persistLocked() {
	if s.path == "" {
		return
	}
	dir := filepath.Dir(s.path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return
		}
	}
	data, err := json.MarshalIndent(s.m, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, data, 0o600)
}
