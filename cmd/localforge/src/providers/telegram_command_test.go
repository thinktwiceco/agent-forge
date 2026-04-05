package providers

import (
	"testing"
)

func TestIsTelegramNewConversationCommand(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"/new_conversation", true},
		{"/NEW_CONVERSATION", true},
		{"/new_conversation@MyBot", true},
		{"  /new_conversation  ", true},
		{"/new_conversation extra", false},
		{"/start", false},
		{"", false},
		{"hello", false},
	}
	for _, tt := range tests {
		if got := IsTelegramNewConversationCommand(tt.text); got != tt.want {
			t.Errorf("IsTelegramNewConversationCommand(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

func TestTelegramMessageText(t *testing.T) {
	payload := map[string]interface{}{
		"message": map[string]interface{}{
			"text": "hi",
		},
	}
	text, ok := TelegramMessageText(payload)
	if !ok || text != "hi" {
		t.Fatalf("expected hi, ok=true; got %q, %v", text, ok)
	}
	if _, ok := TelegramMessageText(map[string]interface{}{}); ok {
		t.Fatal("expected false for empty payload")
	}
}
