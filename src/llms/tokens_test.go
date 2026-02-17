package llms

import (
	"testing"
)

func TestApproximateTokenCounter_CountTokens(t *testing.T) {
	counter := &ApproximateTokenCounter{}

	tests := []struct {
		name        string
		text        string
		expectedMin int
		expectedMax int
		description string
	}{
		{
			name:        "empty string",
			text:        "",
			expectedMin: 0,
			expectedMax: 0,
			description: "Empty text should return 0 tokens",
		},
		{
			name:        "simple sentence",
			text:        "Hello world",
			expectedMin: 2,
			expectedMax: 4,
			description: "Simple 2-word sentence",
		},
		{
			name:        "longer natural language",
			text:        "The quick brown fox jumps over the lazy dog",
			expectedMin: 8,
			expectedMax: 14,
			description: "9-word sentence should be ~9-12 tokens",
		},
		{
			name:        "code snippet",
			text:        "function hello() { return 'world'; }",
			expectedMin: 8,
			expectedMax: 15,
			description: "Code has more symbols, different estimation",
		},
		{
			name:        "json data",
			text:        `{"key": "value", "number": 42, "array": [1, 2, 3]}`,
			expectedMin: 10,
			expectedMax: 20,
			description: "JSON with symbols and structure",
		},
		{
			name:        "mixed content",
			text:        "Here is some code: const x = 10; which does something",
			expectedMin: 12,
			expectedMax: 20,
			description: "Natural language mixed with code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := counter.CountTokens(tt.text)

			if result < tt.expectedMin || result > tt.expectedMax {
				t.Errorf("%s: CountTokens(%q) = %d, expected range [%d, %d]",
					tt.description, tt.text, result, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestApproximateTokenCounter_CountMessageTokens(t *testing.T) {
	counter := &ApproximateTokenCounter{}

	t.Run("nil message", func(t *testing.T) {
		result := counter.CountMessageTokens(nil)
		if result != 0 {
			t.Errorf("CountMessageTokens(nil) = %d, expected 0", result)
		}
	})

	t.Run("user message", func(t *testing.T) {
		msg := UserMessage("Hello, how are you?")
		result := counter.CountMessageTokens(msg)

		// Expect: 4 (overhead) + ~5-7 (content) = 9-11
		if result < 7 || result > 15 {
			t.Errorf("User message tokens = %d, expected range [7, 15]", result)
		}
	})

	t.Run("system message", func(t *testing.T) {
		msg := SystemMessage("You are a helpful assistant")
		result := counter.CountMessageTokens(msg)

		// Expect: 4 (overhead) + ~6-8 (content) = 10-12
		if result < 8 || result > 16 {
			t.Errorf("System message tokens = %d, expected range [8, 16]", result)
		}
	})

	t.Run("assistant message with tool calls", func(t *testing.T) {
		toolCalls := []ToolCall{
			{
				ID:   "call-123",
				Name: "search_files",
				Arguments: map[string]any{
					"query": "test",
					"path":  "/home/user",
				},
			},
		}
		msg := AssistantMessageWithToolCalls("Let me search for that", toolCalls, 0, 0, 0)
		result := counter.CountMessageTokens(msg)

		// Expect: message overhead + content + tool call overhead + arguments
		// 4 + 5-7 + 8 + tool name + args = 25-40
		if result < 20 || result > 50 {
			t.Errorf("Assistant message with tool calls tokens = %d, expected range [20, 50]", result)
		}
	})

	t.Run("tool message", func(t *testing.T) {
		msg := ToolMessage("call-123", "Found 5 files matching the query", false)
		result := counter.CountMessageTokens(msg)

		// Expect: 4 (base) + 4 (tool overhead) + ~8-10 (content) = 16-18
		if result < 12 || result > 22 {
			t.Errorf("Tool message tokens = %d, expected range [12, 22]", result)
		}
	})
}

func TestApproximateTokenCounter_CountMessagesTokens(t *testing.T) {
	counter := &ApproximateTokenCounter{}

	t.Run("empty messages", func(t *testing.T) {
		result := counter.CountMessagesTokens([]*UnifiedMessage{})
		if result != 0 {
			t.Errorf("Empty messages = %d tokens, expected 0", result)
		}
	})

	t.Run("conversation with multiple messages", func(t *testing.T) {
		messages := []*UnifiedMessage{
			SystemMessage("You are a helpful assistant"),
			UserMessage("What is the weather?"),
			AssistantMessage("Let me check that for you", 0, 0, 0),
		}

		result := counter.CountMessagesTokens(messages)

		// Each message has overhead + content
		// Expect total: 30-60 tokens
		if result < 25 || result > 70 {
			t.Errorf("Conversation tokens = %d, expected range [25, 70]", result)
		}
	})
}

func TestNewTokenCounter(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		expectedType string
	}{
		{
			name:         "approximate method",
			method:       "approximate",
			expectedType: "*llms.ApproximateTokenCounter",
		},
		{
			name:         "empty method defaults to approximate",
			method:       "",
			expectedType: "*llms.ApproximateTokenCounter",
		},
		{
			name:         "unknown method defaults to approximate",
			method:       "unknown",
			expectedType: "*llms.ApproximateTokenCounter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := NewTokenCounter(tt.method)

			if counter == nil {
				t.Errorf("NewTokenCounter(%q) returned nil", tt.method)
				return
			}

			// Verify it's functional by counting something
			tokens := counter.CountTokens("test")
			if tokens <= 0 {
				t.Errorf("Counter returned %d tokens for 'test', expected > 0", tokens)
			}
		})
	}
}

// Benchmark tests
func BenchmarkApproximateTokenCounter_CountTokens(b *testing.B) {
	counter := &ApproximateTokenCounter{}
	text := "The quick brown fox jumps over the lazy dog. This is a test sentence with multiple words."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter.CountTokens(text)
	}
}

func BenchmarkApproximateTokenCounter_CountMessageTokens(b *testing.B) {
	counter := &ApproximateTokenCounter{}
	msg := UserMessage("Hello, how are you today? I hope you're doing well!")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter.CountMessageTokens(msg)
	}
}
