package agentforge

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// LogLevel represents the severity level of a log message.
type LogLevel int

const (
	// DebugLevel is for detailed debugging information
	DebugLevel LogLevel = iota
	// InfoLevel is for general informational messages
	InfoLevel
	// WarnLevel is for warning messages
	WarnLevel
	// ErrorLevel is for error messages
	ErrorLevel
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides leveled logging functionality.
//
// The logger respects the AF_LOG_LEVEL configuration and only outputs
// messages at or above the configured level.
type Logger struct {
	level  LogLevel
	logger *log.Logger
	mu     sync.RWMutex
}

var (
	// defaultLogger is the global logger instance used by package-level logging functions
	defaultLogger *Logger
	loggerOnce    sync.Once
)

// NewLogger creates a new Logger instance with the specified log level.
//
// Parameters:
//   - level: The minimum log level to output (DEBUG, INFO, WARN, ERROR)
//   - output: The output writer (e.g., os.Stdout, os.Stderr, or a file)
//
// Returns:
//   - *Logger: A new Logger instance
func NewLogger(level LogLevel, output io.Writer) *Logger {
	return &Logger{
		level:  level,
		logger: log.New(output, "", log.LstdFlags),
	}
}

// NewLoggerFromConfig creates a new Logger instance using the Config.
//
// Parameters:
//   - config: The Config containing the AF_LOG_LEVEL setting
//
// Returns:
//   - *Logger: A new Logger instance configured based on AF_LOG_LEVEL
func NewLoggerFromConfig(config *Config) *Logger {
	level := parseLogLevel(config.AFLogLevel)
	return NewLogger(level, os.Stdout)
}

// InitLogger initializes the global logger with the provided configuration.
//
// This function should be called once during application initialization.
// Subsequent calls will be ignored.
//
// Parameters:
//   - config: The Config containing the AF_LOG_LEVEL setting
func InitLogger(config *Config) {
	loggerOnce.Do(func() {
		defaultLogger = NewLoggerFromConfig(config)
	})
}

// GetLogger returns the global logger instance.
//
// If the logger has not been initialized with InitLogger, it returns
// a default logger with INFO level.
//
// Returns:
//   - *Logger: The global logger instance
func GetLogger() *Logger {
	if defaultLogger == nil {
		// Create a default logger if not initialized
		defaultLogger = NewLogger(InfoLevel, os.Stdout)
	}
	return defaultLogger
}

// SetLevel changes the log level of the logger.
//
// Parameters:
//   - level: The new minimum log level to output
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current log level.
//
// Returns:
//   - LogLevel: The current minimum log level
func (l *Logger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// Debug logs a debug-level message.
//
// Parameters:
//   - format: Printf-style format string
//   - args: Arguments for the format string
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DebugLevel, format, args...)
}

// Info logs an info-level message.
//
// Parameters:
//   - format: Printf-style format string
//   - args: Arguments for the format string
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(InfoLevel, format, args...)
}

// Warn logs a warning-level message.
//
// Parameters:
//   - format: Printf-style format string
//   - args: Arguments for the format string
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WarnLevel, format, args...)
}

// Error logs an error-level message.
//
// Parameters:
//   - format: Printf-style format string
//   - args: Arguments for the format string
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ErrorLevel, format, args...)
}

// log is the internal logging function that checks the level before outputting.
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	l.mu.RLock()
	currentLevel := l.level
	l.mu.RUnlock()

	if level >= currentLevel {
		prefix := fmt.Sprintf("[%s] ", level.String())
		message := fmt.Sprintf(format, args...)
		l.logger.Printf("%s%s", prefix, message)
	}
}

// parseLogLevel converts a string log level to a LogLevel constant.
//
// Parameters:
//   - levelStr: The log level string (DEBUG, INFO, WARN, ERROR)
//
// Returns:
//   - LogLevel: The corresponding LogLevel constant (defaults to INFO if invalid)
func parseLogLevel(levelStr string) LogLevel {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return DebugLevel
	case "INFO":
		return InfoLevel
	case "WARN":
		return WarnLevel
	case "ERROR":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// Package-level convenience functions that use the global logger

// Debug logs a debug-level message using the global logger.
func Debug(format string, args ...interface{}) {
	GetLogger().Debug(format, args...)
}

// Info logs an info-level message using the global logger.
func Info(format string, args ...interface{}) {
	GetLogger().Info(format, args...)
}

// Warn logs a warning-level message using the global logger.
func Warn(format string, args ...interface{}) {
	GetLogger().Warn(format, args...)
}

// Error logs an error-level message using the global logger.
func Error(format string, args ...interface{}) {
	GetLogger().Error(format, args...)
}
