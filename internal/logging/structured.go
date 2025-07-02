package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// StructuredLogger provides JSON structured logging with correlation IDs.
type StructuredLogger struct {
	level      Level
	service    string
	version    string
	mu         sync.RWMutex
	encoder    *json.Encoder
	fields     map[string]interface{}
	timeFormat string
}

// LogEntry represents a structured log entry.
type LogEntry struct {
	Timestamp     string                 `json:"timestamp"`
	Level         string                 `json:"level"`
	Service       string                 `json:"service"`
	Version       string                 `json:"version,omitempty"`
	Message       string                 `json:"message"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
	Caller        string                 `json:"caller,omitempty"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
}

// NewStructuredLogger creates a new structured logger.
func NewStructuredLogger(service, version, level string) *StructuredLogger {
	return &StructuredLogger{
		level:      parseLevel(level),
		service:    service,
		version:    version,
		encoder:    json.NewEncoder(os.Stderr),
		fields:     make(map[string]interface{}),
		timeFormat: time.RFC3339Nano,
	}
}

// WithContext returns a logger with correlation and request IDs from context.
func (l *StructuredLogger) WithContext(ctx context.Context) ContextLogger {
	newLogger := &StructuredLogger{
		level:      l.level,
		service:    l.service,
		version:    l.version,
		encoder:    l.encoder,
		fields:     make(map[string]interface{}),
		timeFormat: l.timeFormat,
	}

	// Copy existing fields
	l.mu.RLock()
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	l.mu.RUnlock()

	// Add context values
	if correlationID, ok := CorrelationIDFromContext(ctx); ok {
		newLogger.fields["correlation_id"] = correlationID
	}
	if requestID, ok := RequestIDFromContext(ctx); ok {
		newLogger.fields["request_id"] = requestID
	}

	return newLogger
}

// WithFields returns a logger with additional fields.
func (l *StructuredLogger) WithFields(fields map[string]interface{}) ContextLogger {
	newLogger := &StructuredLogger{
		level:      l.level,
		service:    l.service,
		version:    l.version,
		encoder:    l.encoder,
		fields:     make(map[string]interface{}),
		timeFormat: l.timeFormat,
	}

	// Copy existing fields
	l.mu.RLock()
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	l.mu.RUnlock()

	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// WithField returns a logger with an additional field.
func (l *StructuredLogger) WithField(key string, value interface{}) ContextLogger {
	return l.WithFields(map[string]interface{}{key: value})
}

// addArgsAsFields adds args as key-value pairs to the log entry.
func (l *StructuredLogger) addArgsAsFields(entry *LogEntry, args []interface{}) {
	if len(args) == 0 {
		return
	}

	// Initialize fields map if needed
	if entry.Fields == nil {
		entry.Fields = make(map[string]interface{})
	}

	// Process args as key-value pairs
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			entry.Fields[key] = args[i+1]
		}
	}

	// If we have an odd number of args, add the last one as "extra"
	if len(args)%2 == 1 {
		entry.Fields["extra"] = args[len(args)-1]
	}
}

// log writes a structured log entry.
func (l *StructuredLogger) log(level Level, message string, args ...interface{}) {
	if !l.shouldLog(level) {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(l.timeFormat),
		Level:     levelToString(level),
		Service:   l.service,
		Version:   l.version,
		Message:   message,
	}

	// Handle args as key-value pairs or printf style
	if len(args) > 0 {
		// Check if message contains format verbs
		if strings.Contains(message, "%") {
			// Count format verbs
			verbCount := 0
			for i := 0; i < len(message)-1; i++ {
				if message[i] == '%' && message[i+1] != '%' {
					verbCount++
				}
			}

			// If we have format verbs and enough args to satisfy them
			if verbCount > 0 && len(args) >= verbCount {
				// Extract printf args
				printfArgs := args[:verbCount]
				entry.Message = fmt.Sprintf(message, printfArgs...)

				// Remaining args are key-value pairs
				if len(args) > verbCount {
					l.addArgsAsFields(&entry, args[verbCount:])
				}
			} else {
				// Not enough args for printf, treat all as key-value
				l.addArgsAsFields(&entry, args)
			}
		} else {
			// No format verbs, treat all args as key-value pairs
			l.addArgsAsFields(&entry, args)
		}
	}

	// Add caller information
	if _, file, line, ok := runtime.Caller(2); ok {
		entry.Caller = fmt.Sprintf("%s:%d", file, line)
	}

	// Add fields from logger
	l.mu.RLock()
	if len(l.fields) > 0 {
		// Initialize fields map if not already done
		if entry.Fields == nil {
			entry.Fields = make(map[string]interface{})
		}
		for k, v := range l.fields {
			// Handle special fields
			switch k {
			case "correlation_id":
				if id, ok := v.(string); ok {
					entry.CorrelationID = id
				}
			case "request_id":
				if id, ok := v.(string); ok {
					entry.RequestID = id
				}
			default:
				entry.Fields[k] = v
			}
		}
	}
	l.mu.RUnlock()

	// Write JSON to stderr
	if err := l.encoder.Encode(entry); err != nil {
		// Fallback to basic logging if JSON encoding fails
		fmt.Fprintf(os.Stderr, "[%s] %s: %s (json encoding failed: %v)\n",
			entry.Timestamp, entry.Level, entry.Message, err)
	}
}

// Debug logs a debug message.
func (l *StructuredLogger) Debug(message string, args ...interface{}) {
	l.log(DebugLevel, message, args...)
}

// Info logs an info message.
func (l *StructuredLogger) Info(message string, args ...interface{}) {
	l.log(InfoLevel, message, args...)
}

// Warn logs a warning message.
func (l *StructuredLogger) Warn(message string, args ...interface{}) {
	l.log(WarnLevel, message, args...)
}

// Error logs an error message.
func (l *StructuredLogger) Error(message string, args ...interface{}) {
	l.log(ErrorLevel, message, args...)
}

// Fatal logs a fatal message and exits.
func (l *StructuredLogger) Fatal(message string, args ...interface{}) {
	l.log(ErrorLevel, message, args...)
	os.Exit(1)
}

// SetLevel sets the logging level.
func (l *StructuredLogger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current logging level.
func (l *StructuredLogger) GetLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// shouldLog checks if a message should be logged at the given level.
func (l *StructuredLogger) shouldLog(level Level) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level >= l.level
}

// levelToString converts a Level to its string representation.
func levelToString(level Level) string {
	switch level {
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
