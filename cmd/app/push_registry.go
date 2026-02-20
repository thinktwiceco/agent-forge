package main

import (
	"sync"

	"github.com/thinktwiceco/agent-forge/src/core"
)

// PushRegistry maps conversationId → buffered channel of chunks.
// The handlePush SSE handler registers a channel here; the agent's chunk router pushes into it.
type PushRegistry struct {
	mu    sync.RWMutex
	chans map[string]chan core.ExtendedChunkResponse
}

func NewPushRegistry() *PushRegistry {
	return &PushRegistry{
		chans: make(map[string]chan core.ExtendedChunkResponse),
	}
}

// Register creates and returns a buffered channel for chatId.
// Call Unregister when the SSE connection closes to clean up.
func (r *PushRegistry) Register(chatId string) <-chan core.ExtendedChunkResponse {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch := make(chan core.ExtendedChunkResponse, 64)
	r.chans[chatId] = ch
	return ch
}

// Unregister removes and closes the channel for chatId.
func (r *PushRegistry) Unregister(chatId string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ch, ok := r.chans[chatId]; ok {
		close(ch)
		delete(r.chans, chatId)
	}
}

// Push sends a chunk to the channel registered for chatId.
// Non-blocking: silently drops if there is no listener or the buffer is full.
func (r *PushRegistry) Push(chatId string, chunk core.ExtendedChunkResponse) {
	r.mu.RLock()
	ch, ok := r.chans[chatId]
	r.mu.RUnlock()
	if !ok {
		return
	}
	select {
	case ch <- chunk:
	default:
	}
}
