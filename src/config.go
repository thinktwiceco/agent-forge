package agentforge

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds the application configuration loaded from environment variables.
//
// Configuration values are loaded from:
//  1. .env file (if present)
//  2. System environment variables (takes precedence over .env)
type Config struct {
	// AF_LOG_LEVEL defines the logging level for the application.
	// Valid values: DEBUG, INFO, WARN, ERROR
	// Default: INFO
	AFLogLevel string

	// AF_DEEPSEEK_API_KEY is the API key for DeepSeek LLM provider.
	// Optional - only required if using DeepSeek models
	AFDeepSeekAPIKey string

	// AF_TOGETHERAI_API_KEY is the API key for TogetherAI LLM provider.
	// Optional - only required if using TogetherAI models
	AFTogetherAIAPIKey string

	// AF_OPENAI_API_KEY is the API key for OpenAI LLM provider.
	// Optional - only required if using OpenAI models
	AFOpenAIAPIKey string
}

// NewConfig creates a new Config instance by loading environment variables.
//
// It attempts to load a .env file from the current directory first, then
// reads configuration values from environment variables. Environment variables
// take precedence over .env file values.
//
// Returns:
//   - *Config: The loaded configuration
//   - error: An error if configuration loading fails
func NewConfig() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	config := &Config{
		AFLogLevel:         getEnv("AF_LOG_LEVEL", "INFO"),
		AFDeepSeekAPIKey:   getEnv("AF_DEEPSEEK_API_KEY", ""),
		AFTogetherAIAPIKey: getEnv("AF_TOGETHERAI_API_KEY", ""),
		AFOpenAIAPIKey:     getEnv("AF_OPENAI_API_KEY", ""),
	}

	// Validate the configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// validate ensures that the configuration values are valid.
//
// Returns:
//   - error: An error describing which configuration value is invalid, or nil if validation passes
func (c *Config) validate() error {
	// Validate log level
	validLogLevels := map[string]bool{
		"DEBUG": true,
		"INFO":  true,
		"WARN":  true,
		"ERROR": true,
	}

	logLevel := strings.ToUpper(c.AFLogLevel)
	if !validLogLevels[logLevel] {
		return fmt.Errorf("invalid AF_LOG_LEVEL: %s (must be DEBUG, INFO, WARN, or ERROR)", c.AFLogLevel)
	}

	// Normalize the log level to uppercase
	c.AFLogLevel = logLevel

	return nil
}

// getEnv retrieves an environment variable value or returns a default value if not set.
//
// Parameters:
//   - key: The environment variable name
//   - defaultValue: The default value to return if the environment variable is not set
//
// Returns:
//   - string: The environment variable value or the default value
func getEnv(key, defaultValue string) string {
	value, err := GetEnvVar(key)

	if err != nil {
		return defaultValue
	}

	return value
}
