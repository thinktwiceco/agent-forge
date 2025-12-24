# Configuration Guide

## Overview

The `config.go` file provides a centralized configuration system for the Agent Forge application. Configuration values are loaded from environment variables and `.env` files.

## Configuration Structure

The `Config` struct contains the following fields:

- `AF_LOG_LEVEL`: Logging level for the application
  - Valid values: `DEBUG`, `INFO`, `WARN`, `ERROR`
  - Default: `INFO`
  - See [Logger Documentation](LOGGER.md) for details on how logging works

## Usage

### Basic Usage

```go
import "github.com/thinktwice/agentForge/src"

// Load configuration from .env file (if present) and environment variables
config, err := agentforge.NewConfig()
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

fmt.Printf("Log Level: %s\n", config.AFLogLevel)

// Initialize the logger with the config
agentforge.InitLogger(config)

// Now you can use the logger
agentforge.Info("Application started with log level: %s", config.AFLogLevel)
```

## Environment Variables

### Setting Environment Variables

You can set environment variables in several ways:

#### 1. Using a .env file

Create a `.env` file in the project root:

```bash
# .env file
AF_LOG_LEVEL=DEBUG
```

#### 2. Using shell export

```bash
export AF_LOG_LEVEL=DEBUG
```

#### 3. Inline with command

```bash
AF_LOG_LEVEL=DEBUG go run main.go
```

## Configuration Priority

Configuration values are loaded in the following order (later sources override earlier ones):

1. Default values (hardcoded in the application)
2. `.env` file values
3. System environment variables (highest priority)

## Validation

The configuration system automatically validates all values:

- `AF_LOG_LEVEL` must be one of: `DEBUG`, `INFO`, `WARN`, `ERROR`
- Invalid values will cause `NewConfig()` to return an error

## Example .env File

```bash
# Agent Forge Configuration
# Copy this to .env and update the values as needed

# Logging level for the application
# Valid values: DEBUG, INFO, WARN, ERROR
# Default: INFO
AF_LOG_LEVEL=INFO
```

## Adding New Configuration Fields

To add a new configuration field:

1. Add the field to the `Config` struct in `config.go`
2. Load the field in `NewConfig()` using `getEnv()`
3. Add validation logic in the `validate()` method if needed
4. Update this documentation

Example:

```go
type Config struct {
    AFLogLevel string
    // Add new field here
    NewField string
}

func NewConfig() (*Config, error) {
    _ = godotenv.Load()
    
    config := &Config{
        AFLogLevel: getEnv("AF_LOG_LEVEL", "INFO"),
        NewField:   getEnv("NEW_FIELD", "default_value"),
    }
    
    if err := config.validate(); err != nil {
        return nil, fmt.Errorf("config validation failed: %w", err)
    }
    
    return config, nil
}
```

## See Also

- [Logger Documentation](LOGGER.md) - Comprehensive guide to using the logger system
- [Logger Example](examples/logger_example.go) - Working examples of logger usage
- [Config Example](examples/config_example.go) - Working examples of config usage

