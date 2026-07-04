package main

import (
	"testing"

	"github.com/thinktwiceco/agent-forge/src/core"
)

func TestPushRegistryRegisterReplacesAndClosesOldChannel(t *testing.T) {
	reg := NewPushRegistry()

	first := reg.Register("chat-1")
	firstCh := first.Channel()

	second := reg.Register("chat-1")
	if first == second {
		t.Fatal("expected a new registration after re-register")
	}

	_, open := <-firstCh
	if open {
		t.Fatal("expected first channel to be closed on re-register")
	}

	reg.Unregister(second)
}

func TestPushRegistryUnregisterOnlyOwnsRegistration(t *testing.T) {
	reg := NewPushRegistry()

	first := reg.Register("chat-1")
	second := reg.Register("chat-1")

	reg.Unregister(first)
	select {
	case _, ok := <-second.Channel():
		if !ok {
			t.Fatal("stale unregister closed the active registration")
		}
	default:
	}

	reg.Unregister(second)
	if _, ok := <-second.Channel(); ok {
		t.Fatal("expected active registration channel to close")
	}
}

func TestPushRegistryPushDelivered(t *testing.T) {
	reg := NewPushRegistry()
	sub := reg.Register("chat-1")

	chunk := core.ExtendedChunkResponse{Content: "hello"}
	reg.Push("chat-1", chunk)

	got := <-sub.Channel()
	if got.Content != "hello" {
		t.Fatalf("expected hello, got %q", got.Content)
	}

	reg.Unregister(sub)
}
