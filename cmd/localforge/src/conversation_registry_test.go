package main

import (
	"testing"
)

func TestConversationRegistryRegisterReplacesAndCancelsOld(t *testing.T) {
	reg := NewConversationRegistry()

	firstCanceled := make(chan struct{}, 1)
	first := reg.Register("chat-1", func() { firstCanceled <- struct{}{} })

	second := reg.Register("chat-1", func() {})
	if first == second {
		t.Fatal("expected a new registration after re-register")
	}

	select {
	case <-firstCanceled:
	default:
		t.Fatal("expected first registration to be canceled on re-register")
	}

	reg.Unregister(second)
}

func TestConversationRegistryUnregisterOnlyOwnsRegistration(t *testing.T) {
	reg := NewConversationRegistry()

	first := reg.Register("chat-1", func() {})
	secondCanceled := make(chan struct{}, 1)
	second := reg.Register("chat-1", func() { secondCanceled <- struct{}{} })

	reg.Unregister(first)
	if reg.Cancel("chat-1") {
		select {
		case <-secondCanceled:
		default:
			t.Fatal("stale unregister removed the active registration")
		}
	}

	reg.Unregister(second)
	if reg.Cancel("chat-1") {
		t.Fatal("expected no registration after Unregister")
	}
}

func TestConversationRegistryCancel(t *testing.T) {
	reg := NewConversationRegistry()
	canceled := make(chan struct{}, 1)
	reg.Register("chat-1", func() { canceled <- struct{}{} })

	if !reg.Cancel("chat-1") {
		t.Fatal("expected Cancel to succeed")
	}

	select {
	case <-canceled:
	default:
		t.Fatal("expected cancel func to be invoked")
	}

	if reg.Cancel("chat-1") {
		t.Fatal("expected Cancel to fail after removal")
	}
}
