package logging

import (
	"context"
	"fmt"
	"strings"
)

// LoggerAdapter adapts the old Logger to work with the new ContextLogger interface.
type LoggerAdapter struct {
	*Logger
	fields map[string]interface{}
}

// NewLoggerAdapter creates a new adapter for the legacy logger.
func NewLoggerAdapter(logger *Logger) *LoggerAdapter {
	return &LoggerAdapter{
		Logger: logger,
		fields: make(map[string]interface{}),
	}
}

// WithContext returns a new logger with context values.
func (l *LoggerAdapter) WithContext(ctx context.Context) ContextLogger {
	newLogger := &LoggerAdapter{
		Logger: l.Logger,
		fields: make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add context values
	if correlationID, ok := CorrelationIDFromContext(ctx); ok {
		return newLogger.WithField("correlation_id", correlationID)
	}
	if requestID, ok := RequestIDFromContext(ctx); ok {
		return newLogger.WithField("request_id", requestID)
	}

	return newLogger
}

// WithField returns a new logger with an additional field.
func (l *LoggerAdapter) WithField(key string, value interface{}) ContextLogger {
	newLogger := &LoggerAdapter{
		Logger: l.Logger,
		fields: make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	newLogger.fields[key] = value

	// Handle special fields
	if key == "request_id" {
		if reqID, ok := value.(string); ok {
			newLogger.Logger = l.WithRequestID(reqID)
		}
	}

	return newLogger
}

// WithFields returns a new logger with additional fields.
func (l *LoggerAdapter) WithFields(fields map[string]interface{}) ContextLogger {
	newLogger := l
	for k, v := range fields {
		if adapted, ok := newLogger.WithField(k, v).(*LoggerAdapter); ok {
			newLogger = adapted
		}
	}
	return newLogger
}

// Override logging methods to include field information.
func (l *LoggerAdapter) Debug(format string, args ...interface{}) {
	msg, fields := l.parseArgs(format, args...)
	l.Logger.Debug(l.formatWithFields(msg, fields))
}

func (l *LoggerAdapter) Info(format string, args ...interface{}) {
	msg, fields := l.parseArgs(format, args...)
	l.Logger.Info(l.formatWithFields(msg, fields))
}

func (l *LoggerAdapter) Warn(format string, args ...interface{}) {
	msg, fields := l.parseArgs(format, args...)
	l.Logger.Warn(l.formatWithFields(msg, fields))
}

func (l *LoggerAdapter) Error(format string, args ...interface{}) {
	msg, fields := l.parseArgs(format, args...)
	l.Logger.Error(l.formatWithFields(msg, fields))
}

func (l *LoggerAdapter) Fatal(format string, args ...interface{}) {
	msg, fields := l.parseArgs(format, args...)
	l.Logger.Fatal(l.formatWithFields(msg, fields))
}

// parseArgs separates the message from key-value pairs.
func (l *LoggerAdapter) parseArgs(format string, args ...interface{}) (message string, fields map[string]interface{}) {
	fields = make(map[string]interface{})

	// If args length is odd, first arg might be for format string
	if len(args)%2 == 1 {
		// Check if format string contains %
		if strings.Contains(format, "%") {
			return fmt.Sprintf(format, args...), fields
		}
		// Otherwise treat all as key-value pairs starting from second arg
		return format, fields
	}

	// Parse key-value pairs
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			fields[key] = args[i+1]
		}
	}

	return format, fields
}

// formatWithFields adds field information to the message.
func (l *LoggerAdapter) formatWithFields(format string, extraFields map[string]interface{}) string {
	// Combine adapter fields with extra fields
	allFields := make(map[string]interface{})
	for k, v := range l.fields {
		allFields[k] = v
	}
	for k, v := range extraFields {
		allFields[k] = v
	}

	if len(allFields) == 0 {
		return format
	}

	// Add fields to the message
	fieldStr := ""
	for k, v := range allFields {
		if fieldStr != "" {
			fieldStr += " "
		}
		fieldStr += fmt.Sprintf("%s=%v", k, v)
	}

	return fmt.Sprintf("%s [%s]", format, fieldStr)
}
