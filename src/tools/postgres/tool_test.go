package postgres

import (
	"testing"
)

// Test that the tool can be created successfully
func TestNewPostgresTool(t *testing.T) {
	tool := NewPostgresTool(
		"postgresql://testuser:testpass@localhost:5432/testdb?sslmode=disable",
		"read",
		[]string{"users", "products", "orders"},
		[]string{"public"},
	)

	if tool == nil {
		t.Fatal("expected tool to be created, got nil")
	}

	// Verify tool has the correct name
	if tool.GetName() != "postgres" {
		t.Errorf("expected tool name 'postgres', got '%s'", tool.GetName())
	}
}

// Test that write mode is configured correctly
func TestNewPostgresTool_WriteMode(t *testing.T) {
	tool := NewPostgresTool(
		"postgresql://testuser:testpass@localhost:5432/testdb?sslmode=disable",
		"write",
		[]string{"users", "products", "orders"},
		[]string{"public"},
	)

	if tool == nil {
		t.Fatal("expected tool to be created, got nil")
	}
}
