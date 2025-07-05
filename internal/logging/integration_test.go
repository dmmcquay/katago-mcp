package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestStructuredLoggerIntegration(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	// Create logger
	logger := NewStructuredLogger("test-service", "1.0.0", "debug")

	// Create context with IDs
	ctx := context.Background()
	ctx = ContextWithCorrelationID(ctx, "corr-123")
	ctx = ContextWithRequestID(ctx, "req-456")

	// Get logger with context
	ctxLogger := logger.WithContext(ctx).WithField("tool", "test")

	// Log a message
	ctxLogger.Info("Test message")

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}
	output := buf.String()

	// Parse JSON output
	var entry LogEntry
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	// Verify fields
	if entry.Level != "INFO" {
		t.Errorf("Expected level INFO, got %s", entry.Level)
	}
	if entry.Service != "test-service" {
		t.Errorf("Expected service test-service, got %s", entry.Service)
	}
	if entry.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", entry.Version)
	}
	if entry.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got %s", entry.Message)
	}
	if entry.CorrelationID != "corr-123" {
		t.Errorf("Expected correlation ID corr-123, got %s", entry.CorrelationID)
	}
	if entry.RequestID != "req-456" {
		t.Errorf("Expected request ID req-456, got %s", entry.RequestID)
	}
	if entry.Fields["tool"] != "test" {
		t.Errorf("Expected tool field 'test', got %v", entry.Fields["tool"])
	}
}

func TestLoggerAdapter(t *testing.T) {
	// Create a text logger and wrap it
	textLogger := NewLogger("[TEST] ", "info")
	adapter := NewLoggerAdapter(textLogger)

	// Test WithContext
	ctx := context.Background()
	ctx = ContextWithCorrelationID(ctx, "corr-789")
	ctxLogger := adapter.WithContext(ctx)

	// Verify it returns a valid ContextLogger
	if ctxLogger == nil {
		t.Fatal("Expected non-nil context logger")
	}

	// Test WithField
	fieldLogger := adapter.WithField("key", "value")
	if fieldLogger == nil {
		t.Fatal("Expected non-nil field logger")
	}

	// Test WithFields
	fieldsLogger := adapter.WithFields(map[string]interface{}{"k1": "v1", "k2": "v2"})
	if fieldsLogger == nil {
		t.Fatal("Expected non-nil fields logger")
	}
}

func TestFactoryCreation(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		expectType string
	}{
		{
			name: "JSON format",
			config: &Config{
				Level:   "info",
				Format:  FormatJSON,
				Service: "test",
				Version: "1.0",
			},
			expectType: "structured",
		},
		{
			name: "Text format",
			config: &Config{
				Level:  "info",
				Format: FormatText,
				Prefix: "[TEST] ",
			},
			expectType: "text",
		},
		{
			name: "Default format",
			config: &Config{
				Level:   "info",
				Service: "test",
			},
			expectType: "structured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLoggerFromConfig(tt.config)
			if logger == nil {
				t.Fatal("Expected non-nil logger")
			}

			// Check type
			switch logger.(type) {
			case *StructuredLogger:
				if tt.expectType != "structured" {
					t.Errorf("Expected text logger, got structured")
				}
			case *LoggerAdapter:
				if tt.expectType != "text" {
					t.Errorf("Expected structured logger, got text")
				}
			}
		})
	}
}

func TestJSONOutputFormat(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	// Create logger and log with various levels
	logger := NewStructuredLogger("test", "1.0", "debug")

	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warn message")
	logger.Error("Error message")

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}
	output := buf.String()

	// Should have 4 JSON lines
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 4 {
		t.Errorf("Expected 4 log lines, got %d", len(lines))
	}

	// Parse each line as JSON
	expectedLevels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	for i, line := range lines {
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("Failed to parse JSON line %d: %v", i, err)
			continue
		}
		if entry.Level != expectedLevels[i] {
			t.Errorf("Line %d: expected level %s, got %s", i, expectedLevels[i], entry.Level)
		}
	}
}
