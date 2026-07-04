package main

import (
	"context"
	"sync"
)

// ChatRegistration identifies a single active chat request. Unregister only
// removes the registration if it still owns the conversation slot.
type ChatRegistration struct {
	conversationID string
	cancel         context.CancelFunc
}

// ConversationRegistry tracks active chat requests and their cancel functions
type ConversationRegistry struct {
	mu       sync.RWMutex
	registry map[string]*ChatRegistration
}

// NewConversationRegistry creates a new conversation registry
func NewConversationRegistry() *ConversationRegistry {
	return &ConversationRegistry{
		registry: make(map[string]*ChatRegistration),
	}
}

// Register stores a cancel function for a conversation, canceling any prior registration.
func (r *ConversationRegistry) Register(conversationID string, cancel context.CancelFunc) *ChatRegistration {
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.registry[conversationID]; ok {
		old.cancel()
	}
	reg := &ChatRegistration{conversationID: conversationID, cancel: cancel}
	r.registry[conversationID] = reg
	return reg
}

// Cancel calls the cancel function for a conversation and removes it from registry
func (r *ConversationRegistry) Cancel(conversationID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	reg, exists := r.registry[conversationID]
	if !exists {
		return false
	}

	reg.cancel()
	delete(r.registry, conversationID)
	return true
}

// Unregister removes the registration only if it still owns the conversation slot.
func (r *ConversationRegistry) Unregister(reg *ChatRegistration) {
	if reg == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.registry[reg.conversationID]
	if !ok || current != reg {
		return
	}
	delete(r.registry, reg.conversationID)
}
