# Logger Documentation

## Overview

The Agent Forge logger provides leveled logging functionality that respects the `AF_LOG_LEVEL` configuration. It filters log messages based on severity levels and provides a clean, consistent logging interface throughout the application.

## Features

- **Leveled Logging**: DEBUG, INFO, WARN, ERROR
- **Configuration-based**: Reads log level from `AF_LOG_LEVEL` environment variable
- **Thread-safe**: Safe for concurrent use across goroutines
- **Global and Instance APIs**: Use package-level functions or create custom logger instances
- **Runtime Configuration**: Change log levels at runtime

## Log Levels

The logger supports four log levels in increasing order of severity:

1. **DEBUG**: Detailed debugging information (most verbose)
2. **INFO**: General informational messages
3. **WARN**: Warning messages
4. **ERROR**: Error messages (least verbose)

When a log level is set, only messages at that level or higher are output. For example, if the log level is set to `INFO`, then `DEBUG` messages are filtered out, but `INFO`, `WARN`, and `ERROR` messages are shown.

## Quick Start

### 1. Initialize the Logger

Initialize the global logger during application startup:

```go
import agentforge "github.com/thinktwice/agentForge/src"

func main() {
    // Load configuration
    config, err := agentforge.NewConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Initialize the global logger
    agentforge.InitLogger(config)

    // Now you can use the logger throughout your application
    agentforge.Info("Application started")
}
```

### 2. Use Package-Level Functions

The simplest way to use the logger is through package-level functions:

```go
agentforge.Debug("Debug message: %s", "details")
agentforge.Info("Info message: %d", 42)
agentforge.Warn("Warning: %s", "something might be wrong")
agentforge.Error("Error: %v", err)
```

### 3. Get Logger Instance

You can also get the logger instance for more control:

```go
logger := agentforge.GetLogger()
logger.Info("Using logger instance")
logger.SetLevel(agentforge.DebugLevel)
```

## API Reference

### Initialization Functions

#### `InitLogger(config *Config)`

Initializes the global logger with the provided configuration. This should be called once during application initialization.

```go
config, _ := agentforge.NewConfig()
agentforge.InitLogger(config)
```

#### `GetLogger() *Logger`

Returns the global logger instance. If the logger hasn't been initialized with `InitLogger`, it returns a default logger with INFO level.

```go
logger := agentforge.GetLogger()
```

#### `NewLogger(level LogLevel, output io.Writer) *Logger`

Creates a new logger instance with a specific log level and output writer.

```go
logger := agentforge.NewLogger(agentforge.DebugLevel, os.Stdout)
```

#### `NewLoggerFromConfig(config *Config) *Logger`

Creates a new logger instance using the Config.

```go
config, _ := agentforge.NewConfig()
logger := agentforge.NewLoggerFromConfig(config)
```

### Logging Functions

#### Package-Level Functions

```go
agentforge.Debug(format string, args ...interface{})
agentforge.Info(format string, args ...interface{})
agentforge.Warn(format string, args ...interface{})
agentforge.Error(format string, args ...interface{})
```

#### Logger Instance Methods

```go
logger.Debug(format string, args ...interface{})
logger.Info(format string, args ...interface{})
logger.Warn(format string, args ...interface{})
logger.Error(format string, args ...interface{})
```

### Logger Configuration

#### `SetLevel(level LogLevel)`

Changes the log level at runtime.

```go
logger := agentforge.GetLogger()
logger.SetLevel(agentforge.DebugLevel)
```

#### `GetLevel() LogLevel`

Returns the current log level.

```go
level := logger.GetLevel()
fmt.Printf("Current level: %s\n", level)
```

## Log Levels Constants

```go
agentforge.DebugLevel  // Most verbose
agentforge.InfoLevel   // General information
agentforge.WarnLevel   // Warnings
agentforge.ErrorLevel  // Errors only
```

## Usage Examples

### Example 1: Basic Usage

```go
package main

import (
    agentforge "github.com/thinktwice/agentForge/src"
)

func main() {
    // Initialize from config
    config, _ := agentforge.NewConfig()
    agentforge.InitLogger(config)

    // Use throughout your application
    agentforge.Info("Application started")
    agentforge.Debug("Debug details: %v", someData)
}
```

### Example 2: Custom Logger for Specific Component

```go
package mypackage

import (
    "os"
    agentforge "github.com/thinktwice/agentForge/src"
)

type MyService struct {
    logger *agentforge.Logger
}

func NewMyService() *MyService {
    // Create a custom logger for this service
    logger := agentforge.NewLogger(agentforge.InfoLevel, os.Stdout)
    
    return &MyService{
        logger: logger,
    }
}

func (s *MyService) DoSomething() {
    s.logger.Info("Doing something...")
    s.logger.Debug("Debug info: %v", details)
}
```

### Example 3: Conditional Logging

```go
func processData(data []string) error {
    logger := agentforge.GetLogger()
    
    logger.Debug("Processing %d items", len(data))
    
    for i, item := range data {
        logger.Debug("Processing item %d: %s", i, item)
        
        if err := process(item); err != nil {
            logger.Error("Failed to process item %d: %v", i, err)
            return err
        }
    }
    
    logger.Info("Successfully processed %d items", len(data))
    return nil
}
```

### Example 4: Changing Log Level at Runtime

```go
func main() {
    config, _ := agentforge.NewConfig()
    agentforge.InitLogger(config)
    
    logger := agentforge.GetLogger()
    
    // Normal operation with INFO level
    logger.Info("Running normally")
    
    // Enable debug mode for troubleshooting
    if debugMode {
        logger.SetLevel(agentforge.DebugLevel)
        logger.Debug("Debug mode enabled")
    }
}
```

## Environment Configuration

Set the log level using the `AF_LOG_LEVEL` environment variable:

```bash
# In .env file
AF_LOG_LEVEL=DEBUG

# Or as environment variable
export AF_LOG_LEVEL=INFO

# Or inline
AF_LOG_LEVEL=ERROR go run main.go
```

Valid values: `DEBUG`, `INFO`, `WARN`, `ERROR` (case-insensitive)

## Log Output Format

Log messages are output in the following format:

```
2025/12/22 19:06:15 [LEVEL] message
```

Example output:

```
2025/12/22 19:06:15 [DEBUG] Debug message with value: 42
2025/12/22 19:06:15 [INFO] This is an info message
2025/12/22 19:06:15 [WARN] This is a warning
2025/12/22 19:06:15 [ERROR] This is an error
```

## Best Practices

### 1. Initialize Early

Initialize the logger as early as possible in your application:

```go
func main() {
    config, err := agentforge.NewConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    agentforge.InitLogger(config)
    
    // Rest of your application
}
```

### 2. Use Appropriate Log Levels

- **DEBUG**: Detailed information for debugging (e.g., variable values, function calls)
- **INFO**: General informational messages (e.g., "Server started", "Request completed")
- **WARN**: Warnings that don't prevent operation (e.g., "Deprecated API used")
- **ERROR**: Errors that need attention (e.g., "Failed to connect to database")

### 3. Use Structured Logging

Include context in your log messages:

```go
// Good
agentforge.Info("User %s logged in from IP %s", username, ipAddress)

// Bad
agentforge.Info("User logged in")
```

### 4. Avoid Logging Sensitive Information

Never log passwords, tokens, or other sensitive data:

```go
// Bad
agentforge.Debug("API key: %s", apiKey)

// Good
agentforge.Debug("API key configured: %v", apiKey != "")
```

### 5. Use Debug Level Appropriately

Debug logs should be detailed but not overwhelming:

```go
func processItems(items []Item) {
    agentforge.Debug("Processing %d items", len(items))
    
    for _, item := range items {
        // Don't log every iteration in production
        agentforge.Debug("Processing item: %+v", item)
    }
    
    agentforge.Info("Processed %d items successfully", len(items))
}
```

## Integration with Existing Code

The logger has been integrated into the Agent Forge codebase:

- **Agent**: Uses `Debug` level for message logging
- **DelegateTool**: Uses `Info` level for delegation events
- **Tests**: Keep using `fmt.Print` for test output (not filtered by log level)

## Thread Safety

The logger is thread-safe and can be used concurrently from multiple goroutines:

```go
func worker(id int) {
    logger := agentforge.GetLogger()
    logger.Info("Worker %d started", id)
    // Safe to use from multiple goroutines
}

func main() {
    agentforge.InitLogger(config)
    
    for i := 0; i < 10; i++ {
        go worker(i)
    }
}
```

## See Also

- [Configuration Guide](CONFIG.md) - For setting up `AF_LOG_LEVEL`
- [Examples](examples/logger_example.go) - Complete working examples

