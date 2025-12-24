package agentforge

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"DEBUG", DebugLevel},
		{"debug", DebugLevel},
		{"INFO", InfoLevel},
		{"info", InfoLevel},
		{"WARN", WarnLevel},
		{"warn", WarnLevel},
		{"ERROR", ErrorLevel},
		{"error", ErrorLevel},
		{"invalid", InfoLevel}, // defaults to INFO
		{"", InfoLevel},        // defaults to INFO
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}

	if logger.GetLevel() != InfoLevel {
		t.Errorf("expected log level INFO, got %v", logger.GetLevel())
	}
}

func TestNewLoggerFromConfig(t *testing.T) {
	tests := []struct {
		name          string
		configLevel   string
		expectedLevel LogLevel
	}{
		{"debug level", "DEBUG", DebugLevel},
		{"info level", "INFO", InfoLevel},
		{"warn level", "WARN", WarnLevel},
		{"error level", "ERROR", ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{AFLogLevel: tt.configLevel}
			logger := NewLoggerFromConfig(config)

			if logger.GetLevel() != tt.expectedLevel {
				t.Errorf("expected log level %v, got %v", tt.expectedLevel, logger.GetLevel())
			}
		})
	}
}

func TestLogger_SetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	logger.SetLevel(DebugLevel)
	if logger.GetLevel() != DebugLevel {
		t.Errorf("expected log level DEBUG, got %v", logger.GetLevel())
	}

	logger.SetLevel(ErrorLevel)
	if logger.GetLevel() != ErrorLevel {
		t.Errorf("expected log level ERROR, got %v", logger.GetLevel())
	}
}

func TestLogger_Debug(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(DebugLevel, &buf)

	logger.Debug("debug message: %s", "test")

	output := buf.String()
	if !strings.Contains(output, "[DEBUG]") {
		t.Errorf("expected output to contain [DEBUG], got: %s", output)
	}
	if !strings.Contains(output, "debug message: test") {
		t.Errorf("expected output to contain message, got: %s", output)
	}
}

func TestLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	logger.Info("info message: %s", "test")

	output := buf.String()
	if !strings.Contains(output, "[INFO]") {
		t.Errorf("expected output to contain [INFO], got: %s", output)
	}
	if !strings.Contains(output, "info message: test") {
		t.Errorf("expected output to contain message, got: %s", output)
	}
}

func TestLogger_Warn(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(WarnLevel, &buf)

	logger.Warn("warning message: %s", "test")

	output := buf.String()
	if !strings.Contains(output, "[WARN]") {
		t.Errorf("expected output to contain [WARN], got: %s", output)
	}
	if !strings.Contains(output, "warning message: test") {
		t.Errorf("expected output to contain message, got: %s", output)
	}
}

func TestLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(ErrorLevel, &buf)

	logger.Error("error message: %s", "test")

	output := buf.String()
	if !strings.Contains(output, "[ERROR]") {
		t.Errorf("expected output to contain [ERROR], got: %s", output)
	}
	if !strings.Contains(output, "error message: test") {
		t.Errorf("expected output to contain message, got: %s", output)
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	tests := []struct {
		name           string
		loggerLevel    LogLevel
		messageLevel   LogLevel
		shouldBeLogged bool
	}{
		{"debug logs at debug level", DebugLevel, DebugLevel, true},
		{"info logs at debug level", DebugLevel, InfoLevel, true},
		{"warn logs at debug level", DebugLevel, WarnLevel, true},
		{"error logs at debug level", DebugLevel, ErrorLevel, true},
		{"debug not logged at info level", InfoLevel, DebugLevel, false},
		{"info logs at info level", InfoLevel, InfoLevel, true},
		{"warn logs at info level", InfoLevel, WarnLevel, true},
		{"error logs at info level", InfoLevel, ErrorLevel, true},
		{"debug not logged at warn level", WarnLevel, DebugLevel, false},
		{"info not logged at warn level", WarnLevel, InfoLevel, false},
		{"warn logs at warn level", WarnLevel, WarnLevel, true},
		{"error logs at warn level", WarnLevel, ErrorLevel, true},
		{"debug not logged at error level", ErrorLevel, DebugLevel, false},
		{"info not logged at error level", ErrorLevel, InfoLevel, false},
		{"warn not logged at error level", ErrorLevel, WarnLevel, false},
		{"error logs at error level", ErrorLevel, ErrorLevel, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(tt.loggerLevel, &buf)

			// Log a message at the specified level
			switch tt.messageLevel {
			case DebugLevel:
				logger.Debug("test message")
			case InfoLevel:
				logger.Info("test message")
			case WarnLevel:
				logger.Warn("test message")
			case ErrorLevel:
				logger.Error("test message")
			}

			output := buf.String()
			hasOutput := len(output) > 0

			if hasOutput != tt.shouldBeLogged {
				t.Errorf("expected logged=%v, but got output: %q", tt.shouldBeLogged, output)
			}
		})
	}
}

func TestGetLogger(t *testing.T) {
	// Reset the default logger for this test
	defaultLogger = nil
	loggerOnce = sync.Once{}

	logger := GetLogger()
	if logger == nil {
		t.Fatal("GetLogger returned nil")
	}

	// Should return the same instance
	logger2 := GetLogger()
	if logger != logger2 {
		t.Error("GetLogger should return the same instance")
	}
}

func TestInitLogger(t *testing.T) {
	// Reset the default logger for this test
	defaultLogger = nil
	loggerOnce = sync.Once{}

	config := &Config{AFLogLevel: "DEBUG"}
	InitLogger(config)

	logger := GetLogger()
	if logger.GetLevel() != DebugLevel {
		t.Errorf("expected DEBUG level after InitLogger, got %v", logger.GetLevel())
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	// Reset the default logger for this test
	defaultLogger = nil
	loggerOnce = sync.Once{}

	var buf bytes.Buffer
	defaultLogger = NewLogger(DebugLevel, &buf)

	Debug("debug %s", "test")
	Info("info %s", "test")
	Warn("warn %s", "test")
	Error("error %s", "test")

	output := buf.String()

	if !strings.Contains(output, "[DEBUG]") {
		t.Error("expected output to contain [DEBUG]")
	}
	if !strings.Contains(output, "[INFO]") {
		t.Error("expected output to contain [INFO]")
	}
	if !strings.Contains(output, "[WARN]") {
		t.Error("expected output to contain [WARN]")
	}
	if !strings.Contains(output, "[ERROR]") {
		t.Error("expected output to contain [ERROR]")
	}
}
