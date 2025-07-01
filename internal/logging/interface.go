package logging

import "context"

// LoggerInterface defines the common interface for all loggers.
type LoggerInterface interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})

	SetLevel(level Level)
	GetLevel() Level
}

// ContextLogger extends LoggerInterface with context support.
type ContextLogger interface {
	LoggerInterface
	WithContext(ctx context.Context) ContextLogger
	WithField(key string, value interface{}) ContextLogger
	WithFields(fields map[string]interface{}) ContextLogger
}

// Ensure our loggers implement the interfaces.
var (
	_ LoggerInterface = (*Logger)(nil)
	_ LoggerInterface = (*StructuredLogger)(nil)
	_ ContextLogger   = (*StructuredLogger)(nil)
)
