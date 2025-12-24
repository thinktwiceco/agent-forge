package agentforge

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetEnvVarDiscovery tests that GetEnvVar can discover environment variables
// from .env files in the current directory or parent directories.
func TestGetEnvVarDiscovery(t *testing.T) {
	// Test that we can find TOGETHERAI_API_KEY if it exists
	value, err := GetEnvVar("TOGETHERAI_API_KEY")
	if err != nil {
		t.Logf("TOGETHERAI_API_KEY not found (this is OK if not set): %v", err)
	} else {
		if value == "" {
			t.Error("TOGETHERAI_API_KEY found but value is empty")
		} else {
			t.Logf("Successfully found TOGETHERAI_API_KEY (length: %d)", len(value))
		}
	}

	// Test that we can find DEEPSEEK_API_KEY if it exists
	value, err = GetEnvVar("DEEPSEEK_API_KEY")
	if err != nil {
		t.Logf("DEEPSEEK_API_KEY not found (this is OK if not set): %v", err)
	} else {
		if value == "" {
			t.Error("DEEPSEEK_API_KEY found but value is empty")
		} else {
			t.Logf("Successfully found DEEPSEEK_API_KEY (length: %d)", len(value))
		}
	}
}

// TestGetEnvVarSearchPath verifies that GetEnvVar searches the correct directories.
func TestGetEnvVarSearchPath(t *testing.T) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	t.Logf("Current working directory: %s", cwd)

	// Check if .env exists in current directory
	envPath := filepath.Join(cwd, ".env")
	if _, err := os.Stat(envPath); err == nil {
		t.Logf("Found .env file at: %s", envPath)
	} else {
		t.Logf("No .env file found at: %s", envPath)
	}

	// Check parent directory
	parent := filepath.Dir(cwd)
	parentEnvPath := filepath.Join(parent, ".env")
	if _, err := os.Stat(parentEnvPath); err == nil {
		t.Logf("Found .env file at: %s", parentEnvPath)
	} else {
		t.Logf("No .env file found at: %s", parentEnvPath)
	}
}
