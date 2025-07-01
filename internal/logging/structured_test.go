package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestStructuredLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger("test-service", "1.0.0", "info")
	logger.encoder = json.NewEncoder(&buf)

	// Test info message
	logger.Info("test message")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Level != "INFO" {
		t.Errorf("Expected level INFO, got %s", entry.Level)
	}
	if entry.Service != "test-service" {
		t.Errorf("Expected service test-service, got %s", entry.Service)
	}
	if entry.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", entry.Version)
	}
	if entry.Message != "test message" {
		t.Errorf("Expected message 'test message', got %s", entry.Message)
	}
	if entry.Timestamp == "" {
		t.Error("Expected timestamp to be set")
	}
}

func TestStructuredLoggerLevels(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		logFunc   func(*StructuredLogger)
		shouldLog bool
	}{
		{
			name:      "debug level logs debug",
			logLevel:  "debug",
			logFunc:   func(l *StructuredLogger) { l.Debug("test") },
			shouldLog: true,
		},
		{
			name:      "info level skips debug",
			logLevel:  "info",
			logFunc:   func(l *StructuredLogger) { l.Debug("test") },
			shouldLog: false,
		},
		{
			name:      "warn level logs warn",
			logLevel:  "warn",
			logFunc:   func(l *StructuredLogger) { l.Warn("test") },
			shouldLog: true,
		},
		{
			name:      "error level logs error",
			logLevel:  "error",
			logFunc:   func(l *StructuredLogger) { l.Error("test") },
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewStructuredLogger("test", "1.0", tt.logLevel)
			logger.encoder = json.NewEncoder(&buf)

			tt.logFunc(logger)

			hasOutput := buf.Len() > 0
			if hasOutput != tt.shouldLog {
				t.Errorf("Expected shouldLog=%v but got output=%v", tt.shouldLog, hasOutput)
			}
		})
	}
}

func TestStructuredLoggerWithContext(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger("test-service", "1.0.0", "info")

	ctx := context.Background()
	ctx = ContextWithCorrelationID(ctx, "corr-123")
	ctx = ContextWithRequestID(ctx, "req-456")

	contextLogger := logger.WithContext(ctx).(*StructuredLogger)
	contextLogger.encoder = json.NewEncoder(&buf)

	contextLogger.Info("test with context")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.CorrelationID != "corr-123" {
		t.Errorf("Expected correlation ID corr-123, got %s", entry.CorrelationID)
	}
	if entry.RequestID != "req-456" {
		t.Errorf("Expected request ID req-456, got %s", entry.RequestID)
	}
}

func TestStructuredLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger("test-service", "1.0.0", "info")

	fieldLogger := logger.WithFields(map[string]interface{}{
		"user_id": "user-123",
		"action":  "analyze",
	}).(*StructuredLogger)
	fieldLogger.encoder = json.NewEncoder(&buf)

	fieldLogger.Info("test with fields")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Fields == nil {
		t.Fatal("Expected fields to be set")
	}
	if entry.Fields["user_id"] != "user-123" {
		t.Errorf("Expected user_id field to be user-123, got %v", entry.Fields["user_id"])
	}
	if entry.Fields["action"] != "analyze" {
		t.Errorf("Expected action field to be analyze, got %v", entry.Fields["action"])
	}
}

func TestStructuredLoggerFormatting(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger("test-service", "1.0.0", "info")
	logger.encoder = json.NewEncoder(&buf)

	logger.Info("test %s %d", "message", 42)

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Message != "test message 42" {
		t.Errorf("Expected formatted message 'test message 42', got %s", entry.Message)
	}
}

func TestGenerateIDs(t *testing.T) {
	// Test correlation ID generation
	corrID := GenerateCorrelationID()
	if !strings.HasPrefix(corrID, "corr_") {
		t.Errorf("Expected correlation ID to start with 'corr_', got %s", corrID)
	}
	if len(corrID) < 10 { // corr_ + at least 5 chars
		t.Errorf("Expected correlation ID to be at least 10 chars, got %d", len(corrID))
	}

	// Test request ID generation
	reqID := GenerateRequestID()
	if !strings.HasPrefix(reqID, "req_") {
		t.Errorf("Expected request ID to start with 'req_', got %s", reqID)
	}
	if len(reqID) < 9 { // req_ + at least 5 chars
		t.Errorf("Expected request ID to be at least 9 chars, got %d", len(reqID))
	}

	// Ensure IDs are unique
	corrID2 := GenerateCorrelationID()
	reqID2 := GenerateRequestID()
	if corrID == corrID2 {
		t.Error("Expected unique correlation IDs")
	}
	if reqID == reqID2 {
		t.Error("Expected unique request IDs")
	}
}

func TestContextFunctions(t *testing.T) {
	ctx := context.Background()

	// Test correlation ID
	ctx = ContextWithCorrelationID(ctx, "test-corr-id")
	corrID, ok := CorrelationIDFromContext(ctx)
	if !ok {
		t.Error("Expected to find correlation ID in context")
	}
	if corrID != "test-corr-id" {
		t.Errorf("Expected correlation ID test-corr-id, got %s", corrID)
	}

	// Test request ID
	ctx = ContextWithRequestID(ctx, "test-req-id")
	reqID, ok := RequestIDFromContext(ctx)
	if !ok {
		t.Error("Expected to find request ID in context")
	}
	if reqID != "test-req-id" {
		t.Errorf("Expected request ID test-req-id, got %s", reqID)
	}

	// Test missing IDs
	emptyCtx := context.Background()
	_, ok = CorrelationIDFromContext(emptyCtx)
	if ok {
		t.Error("Expected not to find correlation ID in empty context")
	}
	_, ok = RequestIDFromContext(emptyCtx)
	if ok {
		t.Error("Expected not to find request ID in empty context")
	}
}

func TestStructuredLoggerCaller(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger("test-service", "1.0.0", "info")
	logger.encoder = json.NewEncoder(&buf)

	logger.Info("test caller")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Caller == "" {
		t.Error("Expected caller information to be set")
	}
	if !strings.Contains(entry.Caller, "structured_test.go") {
		t.Errorf("Expected caller to contain structured_test.go, got %s", entry.Caller)
	}
}

func TestLevelToString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{Level(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := levelToString(tt.level)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
