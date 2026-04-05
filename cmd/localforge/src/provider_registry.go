package main

import (
	"context"
	"sync"
)

// Provider defines the interface for messaging platform providers
type Provider interface {
	Name() string
	SendMessage(ctx context.Context, recipient string, message string) error
	ExtractRecipient(payload map[string]interface{}) (string, error)
}

// AllowlistProvider is an optional extension of Provider for platforms that
// support sender-level access control. If a Provider implements this interface
// the webhook handler will call IsAllowed before forwarding a message.
type AllowlistProvider interface {
	Provider
	// IsAllowed returns true when the given recipient (chat / user ID) is
	// permitted to interact with the agent.
	IsAllowed(recipient string) bool
}

// EditableProvider is an optional extension of Provider for platforms that
// support sending a placeholder message and later editing it in-place (e.g.
// Telegram). Handlers should use a type assertion to check for this capability
// rather than provider-name comparisons.
type EditableProvider interface {
	Provider
	// SendInitialMessage sends a placeholder and returns an opaque message
	// reference that can be passed to UpdateMessage.
	SendInitialMessage(ctx context.Context, recipient string, text string) (msgRef string, err error)
	// UpdateMessage replaces the content of a previously sent message.
	// If the update fails, implementations should fall back to SendMessage.
	UpdateMessage(ctx context.Context, recipient string, msgRef string, text string) error
	// SendTypingAction signals to the user that the bot is working.
	SendTypingAction(ctx context.Context, recipient string) error
}

// ProviderRegistry manages registered messaging providers
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry
func (r *ProviderRegistry) Register(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name
func (r *ProviderRegistry) Get(name string) Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providers[name]
}

// Has checks if a provider is registered
func (r *ProviderRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.providers[name]
	return exists
}

// GetAll returns all registered providers.
func (r *ProviderRegistry) GetAll() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p)
	}
	return out
}
