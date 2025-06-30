package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultConfig(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	// Check default values
	if cfg.KataGo.BinaryPath != "katago" {
		t.Errorf("Expected default binary path 'katago', got %s", cfg.KataGo.BinaryPath)
	}
	if cfg.KataGo.NumThreads != 4 {
		t.Errorf("Expected default threads 4, got %d", cfg.KataGo.NumThreads)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("Expected default log level 'info', got %s", cfg.Logging.Level)
	}
	if !cfg.RateLimit.Enabled {
		t.Error("Expected rate limiting to be enabled by default")
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	testConfig := Config{
		KataGo: KataGoConfig{
			BinaryPath: "/usr/local/bin/katago",
			ModelPath:  "/path/to/model.bin.gz",
			NumThreads: 8,
			MaxVisits:  2000,
		},
		Logging: LoggingConfig{
			Level: "debug",
		},
		RateLimit: RateLimitConfig{
			Enabled:        false,
			RequestsPerMin: 120,
		},
	}

	data, err := json.MarshalIndent(testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config from file: %v", err)
	}

	// Verify loaded values
	if cfg.KataGo.BinaryPath != testConfig.KataGo.BinaryPath {
		t.Errorf("Expected binary path %s, got %s", testConfig.KataGo.BinaryPath, cfg.KataGo.BinaryPath)
	}
	if cfg.KataGo.NumThreads != testConfig.KataGo.NumThreads {
		t.Errorf("Expected threads %d, got %d", testConfig.KataGo.NumThreads, cfg.KataGo.NumThreads)
	}
	if cfg.Logging.Level != testConfig.Logging.Level {
		t.Errorf("Expected log level %s, got %s", testConfig.Logging.Level, cfg.Logging.Level)
	}
	if cfg.RateLimit.Enabled != testConfig.RateLimit.Enabled {
		t.Errorf("Expected rate limit enabled %v, got %v", testConfig.RateLimit.Enabled, cfg.RateLimit.Enabled)
	}
}

func TestEnvOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("KATAGO_BINARY_PATH", "/custom/katago")
	os.Setenv("KATAGO_MODEL_PATH", "/custom/model.bin.gz")
	os.Setenv("KATAGO_MCP_LOG_LEVEL", "debug")
	os.Setenv("KATAGO_MCP_RATE_LIMIT_ENABLED", "false")

	defer func() {
		os.Unsetenv("KATAGO_BINARY_PATH")
		os.Unsetenv("KATAGO_MODEL_PATH")
		os.Unsetenv("KATAGO_MCP_LOG_LEVEL")
		os.Unsetenv("KATAGO_MCP_RATE_LIMIT_ENABLED")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Failed to load config with env overrides: %v", err)
	}

	// Verify environment overrides
	if cfg.KataGo.BinaryPath != "/custom/katago" {
		t.Errorf("Expected env override for binary path, got %s", cfg.KataGo.BinaryPath)
	}
	if cfg.KataGo.ModelPath != "/custom/model.bin.gz" {
		t.Errorf("Expected env override for model path, got %s", cfg.KataGo.ModelPath)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Expected env override for log level, got %s", cfg.Logging.Level)
	}
	if cfg.RateLimit.Enabled {
		t.Error("Expected rate limiting to be disabled by env override")
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name      string
		modify    func(*Config)
		wantError bool
	}{
		{
			name: "valid config",
			modify: func(c *Config) {
				// No modifications, should be valid
			},
			wantError: false,
		},
		{
			name: "negative threads",
			modify: func(c *Config) {
				c.KataGo.NumThreads = -1
			},
			wantError: false, // Should be corrected to 1
		},
		{
			name: "zero visits",
			modify: func(c *Config) {
				c.KataGo.MaxVisits = 0
			},
			wantError: false, // Should be corrected to 1
		},
		{
			name: "negative time",
			modify: func(c *Config) {
				c.KataGo.MaxTime = -1
			},
			wantError: false, // Should be corrected to 0.1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _ := Load("")
			tt.modify(cfg)
			err := cfg.validate()

			if (err != nil) != tt.wantError {
				t.Errorf("validate() error = %v, wantError %v", err, tt.wantError)
			}

			// Check corrections
			if cfg.KataGo.NumThreads < 1 {
				t.Error("NumThreads should be at least 1")
			}
			if cfg.KataGo.MaxVisits < 1 {
				t.Error("MaxVisits should be at least 1")
			}
			if cfg.KataGo.MaxTime < 0.1 {
				t.Error("MaxTime should be at least 0.1")
			}
		})
	}
}

func TestGetKataGoHomeDir(t *testing.T) {
	cfg := &Config{}

	// Test with KATAGO_HOME env var
	os.Setenv("KATAGO_HOME", "/custom/katago/home")
	defer os.Unsetenv("KATAGO_HOME")

	homeDir := cfg.GetKataGoHomeDir()
	if homeDir != "/custom/katago/home" {
		t.Errorf("Expected KATAGO_HOME env var, got %s", homeDir)
	}

	// Test without env var (should use ~/.katago)
	os.Unsetenv("KATAGO_HOME")
	homeDir = cfg.GetKataGoHomeDir()
	if !strings.HasSuffix(homeDir, ".katago") {
		t.Errorf("Expected path ending with .katago, got %s", homeDir)
	}
}

func TestGetConfigPath(t *testing.T) {
	// Test with environment variable
	os.Setenv("KATAGO_MCP_CONFIG", "/custom/config.json")
	defer os.Unsetenv("KATAGO_MCP_CONFIG")

	path := GetConfigPath()
	if path != "/custom/config.json" {
		t.Errorf("Expected env var path, got %s", path)
	}

	// Test without env var (might find config.json in current dir or return empty)
	os.Unsetenv("KATAGO_MCP_CONFIG")
	path = GetConfigPath()
	// This could be empty or a found config file, both are valid
	t.Logf("Config path without env var: %s", path)
}