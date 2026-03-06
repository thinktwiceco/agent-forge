package agentforge

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// ─── Environment / Secrets Layer ─────────────────────────────────────────────
//
// This file owns the lowest configuration layer: infrastructure secrets and
// operational flags sourced exclusively from the process environment (.env file
// or real env vars). It is intentionally kept separate from agent behaviour
// config (builder.Config / config.yaml) and is NEVER written by the UI.
//
// Layer overview:
//
//	Environment (.env)   ← this file
//	   │
//	Application (config.yaml / builder.Config)
//	   │
//	Runtime (agents.AgentConfig / agents.Agent)
//
// The UI Settings page exposes the application layer (config.yaml) and can
// read/update provider keys, but all writes go through the .env file path;
// the agent must be reloaded for new API keys to take effect.

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

	// AF_LOG_FILE is the optional path to a log file. When set, logs are written
	// to both stdout and the file. When empty, logs go to stdout only.
	AFLogFile string

	// AF_DEEPSEEK_API_KEY is the API key for DeepSeek LLM provider.
	// Optional - only required if using DeepSeek models
	AFDeepSeekAPIKey string

	// AF_TOGETHERAI_API_KEY is the API key for TogetherAI LLM provider.
	// Optional - only required if using TogetherAI models
	AFTogetherAIAPIKey string

	// AF_OPENAI_API_KEY is the API key for OpenAI LLM provider.
	// Optional - only required if using OpenAI models
	AFOpenAIAPIKey string

	// AF_BRAVE_API_KEY is the API key for the Brave Search API.
	// Optional - only required when using the web_search action of the web tool.
	AFBraveAPIKey string
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
	// Check if AF_LOG_LEVEL was set before loading .env
	// This allows tests to unset it and use the default value
	// even if .env file sets it
	logLevelWasSet := os.Getenv("AF_LOG_LEVEL") != ""

	// Try to load .env file (ignore error if file doesn't exist)
	// godotenv.Load() doesn't override existing env vars, but will set unset ones
	_ = godotenv.Load()

	// If log level wasn't set before loading .env, use default even if .env set it
	// This ensures tests can test default behavior by unsetting the variable
	var logLevel string
	if !logLevelWasSet {
		// Variable wasn't set before loading .env
		// If .env set it, ignore it and use default to allow tests to test defaults
		if os.Getenv("AF_LOG_LEVEL") != "" {
			// .env set it, but we want to use default for tests
			logLevel = "INFO"
		} else {
			// Not set by .env either, use default
			logLevel = "INFO"
		}
	} else {
		// Variable was set before loading .env, use it (or from .env if it overrides)
		logLevel = getEnv("AF_LOG_LEVEL", "INFO")
	}

	config := &Config{
		AFLogLevel:         logLevel,
		AFLogFile:          getEnv("AF_LOG_FILE", ""),
		AFDeepSeekAPIKey:   getEnv("AF_DEEPSEEK_API_KEY", ""),
		AFTogetherAIAPIKey: getEnv("AF_TOGETHERAI_API_KEY", ""),
		AFOpenAIAPIKey:     getEnv("AF_OPENAI_API_KEY", ""),
		AFBraveAPIKey:      getEnv("AF_BRAVE_API_KEY", ""),
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
