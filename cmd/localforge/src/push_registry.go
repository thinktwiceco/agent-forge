package main

import (
	"sync"

	"github.com/thinktwiceco/agent-forge/src/core"
)

// PushRegistration identifies a single SSE push subscription. Unregister only
// removes the registration if it still owns the chatId slot.
type PushRegistration struct {
	chatId string
	ch     chan core.ExtendedChunkResponse
}

func (r *PushRegistration) Channel() <-chan core.ExtendedChunkResponse {
	return r.ch
}

// PushRegistry maps conversationId → active push registration.
// The handlePush SSE handler registers here; the agent's chunk router pushes into it.
type PushRegistry struct {
	mu   sync.RWMutex
	regs map[string]*PushRegistration
}

func NewPushRegistry() *PushRegistry {
	return &PushRegistry{
		regs: make(map[string]*PushRegistration),
	}
}

// Register creates a new buffered channel for chatId, closing any prior registration.
func (r *PushRegistry) Register(chatId string) *PushRegistration {
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.regs[chatId]; ok {
		close(old.ch)
		delete(r.regs, chatId)
	}
	ch := make(chan core.ExtendedChunkResponse, 64)
	reg := &PushRegistration{chatId: chatId, ch: ch}
	r.regs[chatId] = reg
	return reg
}

// Unregister removes and closes the registration only if it still owns chatId.
func (r *PushRegistry) Unregister(reg *PushRegistration) {
	if reg == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.regs[reg.chatId]
	if !ok || current != reg {
		return
	}
	close(reg.ch)
	delete(r.regs, reg.chatId)
}

// Push sends a chunk to the channel registered for chatId.
// Non-blocking: silently drops if there is no listener or the buffer is full.
func (r *PushRegistry) Push(chatId string, chunk core.ExtendedChunkResponse) {
	r.mu.RLock()
	reg, ok := r.regs[chatId]
	r.mu.RUnlock()
	if !ok {
		return
	}
	select {
	case reg.ch <- chunk:
	default:
	}
}

// Broadcast sends a chunk to all registered push channels.
// Non-blocking per channel: drops if a channel's buffer is full.
func (r *PushRegistry) Broadcast(chunk core.ExtendedChunkResponse) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, reg := range r.regs {
		select {
		case reg.ch <- chunk:
		default:
		}
	}
}
