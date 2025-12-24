package agentforge

import (
	"os"
	"testing"
)

func TestNewConfig(t *testing.T) {
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
		name        string
		envValue    string
		expected    string
		shouldError bool
	}{
		{
			name:        "default log level",
			envValue:    "",
			expected:    "INFO",
			shouldError: false,
		},
		{
			name:        "debug log level",
			envValue:    "DEBUG",
			expected:    "DEBUG",
			shouldError: false,
		},
		{
			name:        "info log level",
			envValue:    "INFO",
			expected:    "INFO",
			shouldError: false,
		},
		{
			name:        "warn log level",
			envValue:    "WARN",
			expected:    "WARN",
			shouldError: false,
		},
		{
			name:        "error log level",
			envValue:    "ERROR",
			expected:    "ERROR",
			shouldError: false,
		},
		{
			name:        "lowercase log level should normalize",
			envValue:    "debug",
			expected:    "DEBUG",
			shouldError: false,
		},
		{
			name:        "invalid log level",
			envValue:    "INVALID",
			expected:    "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envValue != "" {
				os.Setenv("AF_LOG_LEVEL", tt.envValue)
			} else {
				os.Unsetenv("AF_LOG_LEVEL")
			}

			// Create config
			config, err := NewConfig()

			// Check error expectation
			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check log level
			if config.AFLogLevel != tt.expected {
				t.Errorf("expected log level %s, got %s", tt.expected, config.AFLogLevel)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		shouldError bool
	}{
		{
			name: "valid DEBUG level",
			config: &Config{
				AFLogLevel: "DEBUG",
			},
			shouldError: false,
		},
		{
			name: "valid INFO level",
			config: &Config{
				AFLogLevel: "INFO",
			},
			shouldError: false,
		},
		{
			name: "valid WARN level",
			config: &Config{
				AFLogLevel: "WARN",
			},
			shouldError: false,
		},
		{
			name: "valid ERROR level",
			config: &Config{
				AFLogLevel: "ERROR",
			},
			shouldError: false,
		},
		{
			name: "invalid log level",
			config: &Config{
				AFLogLevel: "INVALID",
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.shouldError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "environment variable exists",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "environment variable does not exist",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalValue := os.Getenv(tt.key)
			defer func() {
				if originalValue != "" {
					os.Setenv(tt.key, originalValue)
				} else {
					os.Unsetenv(tt.key)
				}
			}()

			// Set test environment variable
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}

			// Test getEnv
			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
