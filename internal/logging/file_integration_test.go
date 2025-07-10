package logging

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dmmcquay/katago-mcp/internal/config"
)

func TestLoggerWithFileOutput(t *testing.T) {
	// Create temp directory
	tmpDir, err := ioutil.TempDir("", "katago-mcp-logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")

	// Create logger config with file output
	cfg := &Config{
		Level:   "info",
		Format:  FormatText,
		Service: "test-service",
		Version: "1.0.0",
		Prefix:  "[TEST] ",
		File: &config.LoggingConfig{
			Level:  "info",
			Prefix: "[TEST] ",
			File: struct {
				Enabled    bool   `json:"enabled"`
				Path       string `json:"path"`
				MaxSize    int    `json:"maxSize"`
				MaxBackups int    `json:"maxBackups"`
				MaxAge     int    `json:"maxAge"`
				Compress   bool   `json:"compress"`
			}{
				Enabled:    true,
				Path:       logPath,
				MaxSize:    100,
				MaxBackups: 3,
				MaxAge:     30,
				Compress:   false,
			},
		},
	}

	// Create logger
	logger, closer := NewLoggerFromConfig(cfg)
	if closer != nil {
		defer closer.Close()
	}

	// Log some messages
	logger.Info("Test info message")
	logger.Error("Test error message")
	logger.Debug("Test debug message") // Should not appear with info level

	// Close to flush
	if closer != nil {
		closer.Close()
	}

	// Read and verify file contents
	content, err := ioutil.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Test info message") {
		t.Error("Log file should contain info message")
	}
	if !strings.Contains(contentStr, "Test error message") {
		t.Error("Log file should contain error message")
	}
	if strings.Contains(contentStr, "Test debug message") {
		t.Error("Log file should not contain debug message (info level)")
	}
}

func TestStructuredLoggerWithFileOutput(t *testing.T) {
	// Create temp directory
	tmpDir, err := ioutil.TempDir("", "katago-mcp-structured-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "structured.log")

	// Create logger config with file output
	cfg := &Config{
		Level:   "info",
		Format:  FormatJSON,
		Service: "test-service",
		Version: "1.0.0",
		File: &config.LoggingConfig{
			Level: "info",
			File: struct {
				Enabled    bool   `json:"enabled"`
				Path       string `json:"path"`
				MaxSize    int    `json:"maxSize"`
				MaxBackups int    `json:"maxBackups"`
				MaxAge     int    `json:"maxAge"`
				Compress   bool   `json:"compress"`
			}{
				Enabled:    true,
				Path:       logPath,
				MaxSize:    100,
				MaxBackups: 3,
				MaxAge:     30,
				Compress:   false,
			},
		},
	}

	// Create logger
	logger, closer := NewLoggerFromConfig(cfg)
	if closer != nil {
		defer closer.Close()
	}

	// Log with fields
	logger.WithField("user_id", "123").Info("User logged in")
	logger.WithFields(map[string]interface{}{
		"error_code": 500,
		"method":     "POST",
	}).Error("Request failed")

	// Close to flush
	if closer != nil {
		closer.Close()
	}

	// Read and verify file contents
	content, err := ioutil.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	// Should contain JSON structured logs
	if !strings.Contains(contentStr, `"message":"User logged in"`) {
		t.Error("Log file should contain user login message")
	}
	if !strings.Contains(contentStr, `"user_id":"123"`) {
		t.Error("Log file should contain user_id field")
	}
	if !strings.Contains(contentStr, `"error_code":500`) {
		t.Error("Log file should contain error_code field")
	}
	if !strings.Contains(contentStr, `"level":"INFO"`) {
		t.Error("Log file should contain INFO level")
	}
	if !strings.Contains(contentStr, `"level":"ERROR"`) {
		t.Error("Log file should contain ERROR level")
	}
}
