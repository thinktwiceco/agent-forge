package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type Session struct {
	Username  string
	TokenHash string
	ExpiresAt time.Time
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]Session),
	}
}

func (s *SessionStore) Create(username string, ttl time.Duration) (string, Session, error) {
	token, err := randomToken()
	if err != nil {
		return "", Session{}, err
	}

	session := Session{
		Username:  username,
		TokenHash: hashToken(token),
		ExpiresAt: time.Now().UTC().Add(ttl),
	}

	s.mu.Lock()
	s.sessions[session.TokenHash] = session
	s.mu.Unlock()

	return token, session, nil
}

func (s *SessionStore) Get(token string) (Session, bool) {
	tokenHash := hashToken(token)

	s.mu.RLock()
	session, ok := s.sessions[tokenHash]
	s.mu.RUnlock()
	if !ok {
		return Session{}, false
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		s.Delete(token)
		return Session{}, false
	}
	return session, true
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	delete(s.sessions, hashToken(token))
	s.mu.Unlock()
}

func (s *SessionStore) CleanupExpired() {
	now := time.Now().UTC()

	s.mu.Lock()
	for key, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, key)
		}
	}
	s.mu.Unlock()
}

func (s *SessionStore) StartCleanup(interval time.Duration) {
	if interval <= 0 {
		interval = time.Hour
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.CleanupExpired()
		}
	}()
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
