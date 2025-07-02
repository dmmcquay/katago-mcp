package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestStructuredLoggerKeyValueArgs(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		args     []interface{}
		wantMsg  string
		wantKeys []string
	}{
		{
			name:     "key-value pairs",
			message:  "Tool request received",
			args:     []interface{}{"tool", "getEngineStatus", "client", "anonymous", "duration", "100ms"},
			wantMsg:  "Tool request received",
			wantKeys: []string{"tool", "client", "duration"},
		},
		{
			name:     "printf style",
			message:  "Processing %d items",
			args:     []interface{}{42},
			wantMsg:  "Processing 42 items",
			wantKeys: []string{},
		},
		{
			name:     "mixed format with extra args",
			message:  "Rate limit exceeded",
			args:     []interface{}{"client", "test-client", "tool", "analyzePosition"},
			wantMsg:  "Rate limit exceeded",
			wantKeys: []string{"client", "tool"},
		},
		{
			name:     "odd number of args",
			message:  "Global rate limit exceeded",
			args:     []interface{}{"client", "anonymous", "extra-value"},
			wantMsg:  "Global rate limit exceeded",
			wantKeys: []string{"client", "extra"},
		},
		{
			name:     "empty args",
			message:  "Simple message",
			args:     []interface{}{},
			wantMsg:  "Simple message",
			wantKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			var buf bytes.Buffer
			encoder := json.NewEncoder(&buf)

			logger := &StructuredLogger{
				level:      InfoLevel,
				service:    "test",
				version:    "1.0",
				encoder:    encoder,
				fields:     make(map[string]interface{}),
				timeFormat: "2006-01-02T15:04:05.000000Z",
			}

			// Log the message
			logger.Info(tt.message, tt.args...)

			// Parse the output
			var entry LogEntry
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("Failed to parse log output: %v\nOutput: %s", err, buf.String())
			}

			// Check message
			if entry.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", entry.Message, tt.wantMsg)
			}

			// Check fields
			for _, key := range tt.wantKeys {
				if entry.Fields == nil {
					t.Errorf("Expected field %q but Fields is nil", key)
					continue
				}
				if _, ok := entry.Fields[key]; !ok {
					t.Errorf("Expected field %q not found in Fields: %v", key, entry.Fields)
				}
			}

			// Ensure no %!(EXTRA in output
			if strings.Contains(buf.String(), "%!(EXTRA") {
				t.Errorf("Output contains %%!(EXTRA formatting error: %s", buf.String())
			}
		})
	}
}

func TestStructuredLoggerPrintfCompatibility(t *testing.T) {
	tests := []struct {
		name    string
		message string
		args    []interface{}
		want    string
	}{
		{
			name:    "basic printf",
			message: "Found %d items in %s",
			args:    []interface{}{5, "bucket"},
			want:    "Found 5 items in bucket",
		},
		{
			name:    "printf with extra args treated as kv",
			message: "Error: %v",
			args:    []interface{}{"connection failed", "retry", 3, "delay", "5s"},
			want:    "Error: connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			encoder := json.NewEncoder(&buf)

			logger := &StructuredLogger{
				level:      InfoLevel,
				service:    "test",
				version:    "1.0",
				encoder:    encoder,
				fields:     make(map[string]interface{}),
				timeFormat: "2006-01-02T15:04:05.000000Z",
			}

			logger.Info(tt.message, tt.args...)

			var entry LogEntry
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("Failed to parse log output: %v", err)
			}

			if entry.Message != tt.want {
				t.Errorf("Message = %q, want %q", entry.Message, tt.want)
			}
		})
	}
}
