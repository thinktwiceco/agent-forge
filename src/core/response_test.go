package core

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/thinktwiceco/agent-forge/src/llms"
)

func TestResponseCh_Lifecycle(t *testing.T) {
	rc := NewResponseCh("test-agent", "trace", "chat-1", nil)

	// Test Start returns a channel
	chunkCh := rc.Start()

	// Send data
	go func() {
		chunk := llms.ChunkResponse{
			Content: "test",
			Status:  llms.StatusStreaming,
		}
		bytes, _ := json.Marshal(chunk)
		rc.Response <- bytes
		rc.Close()
	}()

	// Receive data
	received := false
	for chunk := range chunkCh {
		if chunk.Content == "test" {
			received = true
		}
	}

	if !received {
		t.Error("Expected to receive chunk with content 'test'")
	}
}

func TestResponseCh_ErrorHandling(t *testing.T) {
	rc := NewResponseCh("test-agent", "trace", "chat-1", nil)
	chunkCh := rc.Start()

	go func() {
		rc.Error <- errors.New("test error")
		// Do not Close() here, as it races with Error processing in Start().
		// Start() will return and close chunkCh upon receiving the error.
	}()

	var errReceived bool
	for chunk := range chunkCh {
		if chunk.Status == llms.StatusError {
			errReceived = true
			if chunk.Content != "test error" {
				t.Errorf("Expected error content 'test error', got '%s'", chunk.Content)
			}
		}
	}

	if !errReceived {
		t.Error("Expected to receive error chunk")
	}
}

func TestResponseCh_TrySend(t *testing.T) {
	rc := NewResponseCh("test-agent", "trace", "chat-1", nil)

	// Send to open channel
	success := rc.TrySend([]byte("{}"))
	if !success {
		t.Error("Expected TrySend to succeed on open channel")
	}

	// Consume the channel to prevent blocking/leak in real usage,
	// here we just close it.
	rc.Close()

	// Send to closed channel
	// Give a small delay for close to propagate if needed (though mutex protects state)
	time.Sleep(10 * time.Millisecond)

	success = rc.TrySend([]byte("{}"))
	if success {
		// Note: TrySend checks the 'closed' flag.
		// If we just close the channel without setting flag via Close(), it might panic and return false.
		// Since we called rc.Close(), it sets rc.closed = true.
		t.Error("Expected TrySend to fail on closed channel")
	}
}

func TestResponseCh_ChatIdGetterSetter(t *testing.T) {
	rc := NewResponseCh("test-agent", "trace", "initial-chat-id", nil)

	// Test initial chatId
	if rc.GetChatId() != "initial-chat-id" {
		t.Errorf("Expected initial chatId 'initial-chat-id', got '%s'", rc.GetChatId())
	}

	// Test SetChatId
	rc.SetChatId("updated-chat-id")
	if rc.GetChatId() != "updated-chat-id" {
		t.Errorf("Expected updated chatId 'updated-chat-id', got '%s'", rc.GetChatId())
	}

	// Test thread safety by setting multiple times
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			rc.SetChatId("concurrent-id")
			_ = rc.GetChatId()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Final value should be set
	if rc.GetChatId() != "concurrent-id" {
		t.Errorf("Expected final chatId 'concurrent-id', got '%s'", rc.GetChatId())
	}
}
