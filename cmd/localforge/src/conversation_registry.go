package main

import (
	"context"
	"sync"
)

// ConversationRegistry tracks active chat requests and their cancel functions
type ConversationRegistry struct {
	mu       sync.RWMutex
	registry map[string]context.CancelFunc
}

// NewConversationRegistry creates a new conversation registry
func NewConversationRegistry() *ConversationRegistry {
	return &ConversationRegistry{
		registry: make(map[string]context.CancelFunc),
	}
}

// Register stores a cancel function for a conversation
func (r *ConversationRegistry) Register(conversationID string, cancel context.CancelFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registry[conversationID] = cancel
}

// Cancel calls the cancel function for a conversation and removes it from registry
func (r *ConversationRegistry) Cancel(conversationID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	cancel, exists := r.registry[conversationID]
	if !exists {
		return false
	}

	cancel()
	delete(r.registry, conversationID)
	return true
}

// Unregister removes a conversation from the registry without calling cancel
func (r *ConversationRegistry) Unregister(conversationID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.registry, conversationID)
}
