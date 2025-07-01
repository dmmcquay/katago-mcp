package logging

import (
	"os"
	"strings"
)

// LogFormat represents the log output format.
type LogFormat string

const (
	// FormatText is the traditional text format.
	FormatText LogFormat = "text"
	// FormatJSON is structured JSON format.
	FormatJSON LogFormat = "json"
)

// Config represents logging configuration.
type Config struct {
	Level   string
	Format  LogFormat
	Service string
	Version string
	Prefix  string
}

// NewLoggerFromConfig creates a logger based on configuration.
func NewLoggerFromConfig(cfg *Config) ContextLogger {
	// Default to JSON format in production
	format := cfg.Format
	if format == "" {
		// Check environment variable
		if envFormat := os.Getenv("KATAGO_LOG_FORMAT"); envFormat != "" {
			format = LogFormat(strings.ToLower(envFormat))
		} else {
			format = FormatJSON
		}
	}

	// Create appropriate logger based on format
	switch format {
	case FormatJSON:
		return NewStructuredLogger(cfg.Service, cfg.Version, cfg.Level)
	case FormatText:
		// Use adapter to make old logger compatible
		logger := NewLogger(cfg.Prefix, cfg.Level)
		return NewLoggerAdapter(logger)
	default:
		// Default to structured logging
		return NewStructuredLogger(cfg.Service, cfg.Version, cfg.Level)
	}
}

// MustGetLogger creates a logger or panics.
func MustGetLogger(cfg *Config) ContextLogger {
	logger := NewLoggerFromConfig(cfg)
	if logger == nil {
		panic("failed to create logger")
	}
	return logger
}
