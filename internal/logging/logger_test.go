package logging

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		testFunc  func(*Logger)
		shouldLog bool
	}{
		{
			name:      "debug level logs everything",
			logLevel:  "debug",
			testFunc:  func(l *Logger) { l.Debug("test") },
			shouldLog: true,
		},
		{
			name:      "info level skips debug",
			logLevel:  "info",
			testFunc:  func(l *Logger) { l.Debug("test") },
			shouldLog: false,
		},
		{
			name:      "info level logs info",
			logLevel:  "info",
			testFunc:  func(l *Logger) { l.Info("test") },
			shouldLog: true,
		},
		{
			name:      "error level only logs errors",
			logLevel:  "error",
			testFunc:  func(l *Logger) { l.Warn("test") },
			shouldLog: false,
		},
		{
			name:      "error level logs errors",
			logLevel:  "error",
			testFunc:  func(l *Logger) { l.Error("test") },
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger("[TEST] ", tt.logLevel)
			logger.logger = log.New(&buf, "[TEST] ", log.LstdFlags)

			tt.testFunc(logger)

			output := buf.String()
			hasOutput := len(output) > 0

			if hasOutput != tt.shouldLog {
				t.Errorf("Expected shouldLog=%v but got output=%v", tt.shouldLog, hasOutput)
			}
		})
	}
}

func TestLoggerWithRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("[TEST] ", "info")
	logger.logger = log.New(&buf, "[TEST] ", 0)

	reqLogger := logger.WithRequestID("req-123")
	reqLogger.logger.SetOutput(&buf)

	reqLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "req-123") {
		t.Errorf("Expected request ID in output, got: %s", output)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", DebugLevel},
		{"DEBUG", DebugLevel},
		{"info", InfoLevel},
		{"INFO", InfoLevel},
		{"warn", WarnLevel},
		{"warning", WarnLevel},
		{"error", ErrorLevel},
		{"ERROR", ErrorLevel},
		{"unknown", InfoLevel}, // default
		{"", InfoLevel},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := parseLevel(tt.input)
			if level != tt.expected {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, level, tt.expected)
			}
		})
	}
}

func TestLoggerSetLevel(t *testing.T) {
	logger := NewLogger("[TEST] ", "info")

	if logger.GetLevel() != InfoLevel {
		t.Errorf("Expected initial level to be InfoLevel")
	}

	logger.SetLevel(DebugLevel)

	if logger.GetLevel() != DebugLevel {
		t.Errorf("Expected level to be DebugLevel after SetLevel")
	}
}

func TestLoggerOutput(t *testing.T) {
	// Save original stderr
	oldStderr := os.Stderr
	defer func() { os.Stderr = oldStderr }()

	// Create a pipe to capture output
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger := NewLogger("[TEST] ", "info")
	logger.Info("test message %d", 42)

	// Close writer and restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "[INFO]") {
		t.Errorf("Expected [INFO] in output")
	}
	if !strings.Contains(output, "test message 42") {
		t.Errorf("Expected formatted message in output")
	}
	if !strings.Contains(output, "[TEST]") {
		t.Errorf("Expected prefix in output")
	}
}
