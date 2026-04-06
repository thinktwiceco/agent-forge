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

func TestTelegramAllowlistPermits(t *testing.T) {
	tok := "x"
	payload := func(username string) map[string]interface{} {
		return map[string]interface{}{
			"message": map[string]interface{}{
				"from": map[string]interface{}{
					"id":       float64(999),
					"username": username,
				},
				"chat": map[string]interface{}{"id": float64(999), "type": "private"},
				"text": "hi",
			},
		}
	}

	if NewTelegramProvider(tok, nil).AllowlistPermits(payload("alice")) {
		t.Fatal("empty allowlist (TELEGRAM_ALLOWED_USER_IDS not set) should deny all")
	}
	if !NewTelegramProvider(tok, []string{"alice"}).AllowlistPermits(payload("alice")) {
		t.Fatal("exact username match should permit")
	}
	if !NewTelegramProvider(tok, []string{"@Alice"}).AllowlistPermits(payload("alice")) {
		t.Fatal("@ prefix and different case should permit")
	}
	if NewTelegramProvider(tok, []string{"bob"}).AllowlistPermits(payload("alice")) {
		t.Fatal("wrong username should deny")
	}
	if NewTelegramProvider(tok, []string{"alice"}).AllowlistPermits(payload("")) {
		t.Fatal("no username in payload should deny")
	}

	cbqPayload := map[string]interface{}{
		"callback_query": map[string]interface{}{
			"id":   "cb1",
			"from": map[string]interface{}{"id": float64(99), "username": "Alice"},
			"message": map[string]interface{}{
				"chat": map[string]interface{}{"id": float64(99)},
			},
		},
	}
	if !NewTelegramProvider(tok, []string{"alice"}).AllowlistPermits(cbqPayload) {
		t.Fatal("callback_query from username should permit")
	}
}
