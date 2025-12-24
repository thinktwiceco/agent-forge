package agentforge

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestLoggerIntegrationWithConfig tests that the logger properly integrates with Config
func TestLoggerIntegrationWithConfig(t *testing.T) {
	// Save original environment
	originalLogLevel := os.Getenv("AF_LOG_LEVEL")
	defer func() {
		if originalLogLevel != "" {
			os.Setenv("AF_LOG_LEVEL", originalLogLevel)
		} else {
			os.Unsetenv("AF_LOG_LEVEL")
		}
	}()

	tests := []struct {
		name           string
		envValue       string
		shouldLogInfo  bool
		shouldLogDebug bool
	}{
		{
			name:           "DEBUG level shows all",
			envValue:       "DEBUG",
			shouldLogInfo:  true,
			shouldLogDebug: true,
		},
		{
			name:           "INFO level shows info but not debug",
			envValue:       "INFO",
			shouldLogInfo:  true,
			shouldLogDebug: false,
		},
		{
			name:           "ERROR level shows neither",
			envValue:       "ERROR",
			shouldLogInfo:  false,
			shouldLogDebug: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			os.Setenv("AF_LOG_LEVEL", tt.envValue)

			// Create config
			config, err := NewConfig()
			if err != nil {
				t.Fatalf("Failed to create config: %v", err)
			}

			// Create logger from config with custom output
			var buf bytes.Buffer
			logger := NewLoggerFromConfig(config)
			logger.logger.SetOutput(&buf)

			// Log messages
			logger.Debug("debug message")
			logger.Info("info message")

			output := buf.String()

			// Check debug message
			hasDebug := strings.Contains(output, "[DEBUG]")
			if hasDebug != tt.shouldLogDebug {
				t.Errorf("expected debug logged=%v, got output: %q", tt.shouldLogDebug, output)
			}

			// Check info message
			hasInfo := strings.Contains(output, "[INFO]")
			if hasInfo != tt.shouldLogInfo {
				t.Errorf("expected info logged=%v, got output: %q", tt.shouldLogInfo, output)
			}
		})
	}
}

// TestLoggerWithConfigValidation tests that invalid log levels are caught
func TestLoggerWithConfigValidation(t *testing.T) {
	// Save original environment
	originalLogLevel := os.Getenv("AF_LOG_LEVEL")
	defer func() {
		if originalLogLevel != "" {
			os.Setenv("AF_LOG_LEVEL", originalLogLevel)
		} else {
			os.Unsetenv("AF_LOG_LEVEL")
		}
	}()

	// Set invalid log level
	os.Setenv("AF_LOG_LEVEL", "INVALID")

	// Create config - should fail validation
	_, err := NewConfig()
	if err == nil {
		t.Error("expected error for invalid log level, got nil")
	}

	if !strings.Contains(err.Error(), "invalid AF_LOG_LEVEL") {
		t.Errorf("expected error message about invalid log level, got: %v", err)
	}
}
