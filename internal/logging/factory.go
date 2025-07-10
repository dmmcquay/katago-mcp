package logging

import (
	"io"
	"os"
	"strings"

	"github.com/dmmcquay/katago-mcp/internal/config"
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
	File    *config.LoggingConfig // File logging config from main config
}

// NewLoggerFromConfig creates a logger based on configuration.
func NewLoggerFromConfig(cfg *Config) (ContextLogger, io.Closer) {
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

	// Set up writers
	writers := []io.Writer{os.Stderr} // Always log to stderr
	var fileWriter *FileWriter

	// Add file writer if enabled
	if cfg.File != nil && cfg.File.File.Enabled && cfg.File.File.Path != "" {
		fw, err := NewFileWriter(
			cfg.File.File.Path,
			cfg.File.File.MaxSize,
			cfg.File.File.MaxBackups,
			cfg.File.File.MaxAge,
			cfg.File.File.Compress,
		)
		if err != nil {
			// Log error to stderr and continue without file logging
			logger := NewLogger("[katago-mcp] ", "error")
			logger.Error("Failed to create file writer: %v", err)
		} else {
			fileWriter = fw
			writers = append(writers, fw)
		}
	}

	// Create multi-writer if we have multiple outputs
	var writer io.Writer
	if len(writers) > 1 {
		writer = NewMultiWriter(writers...)
	} else {
		writer = writers[0]
	}

	// Create appropriate logger based on format
	var logger ContextLogger
	switch format {
	case FormatJSON:
		logger = NewStructuredLoggerWithWriter(writer, cfg.Service, cfg.Version, cfg.Level)
	case FormatText:
		// Use adapter to make old logger compatible
		basicLogger := NewLoggerWithWriter(writer, cfg.Prefix, cfg.Level)
		logger = NewLoggerAdapter(basicLogger)
	default:
		// Default to structured logging
		logger = NewStructuredLoggerWithWriter(writer, cfg.Service, cfg.Version, cfg.Level)
	}

	// Return logger and closer (fileWriter implements io.Closer)
	if fileWriter != nil {
		return logger, fileWriter
	}
	return logger, nil
}

// MustGetLogger creates a logger or panics.
func MustGetLogger(cfg *Config) (ContextLogger, io.Closer) {
	logger, closer := NewLoggerFromConfig(cfg)
	if logger == nil {
		panic("failed to create logger")
	}
	return logger, closer
}
