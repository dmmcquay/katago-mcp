package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	// KataGo configuration
	KataGo KataGoConfig `json:"katago"`

	// Server configuration
	Server ServerConfig `json:"server"`

	// Logging configuration
	Logging LoggingConfig `json:"logging"`

	// Rate limiting configuration
	RateLimit RateLimitConfig `json:"rateLimit"`
}

type KataGoConfig struct {
	BinaryPath  string `json:"binaryPath"`
	ModelPath   string `json:"modelPath"`
	ConfigPath  string `json:"configPath"`
	NumThreads  int    `json:"numThreads"`
	MaxVisits   int    `json:"maxVisits"`
	MaxTime     float64 `json:"maxTime"`
}

type ServerConfig struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type LoggingConfig struct {
	Level  string `json:"level"`
	Prefix string `json:"prefix"`
}

type RateLimitConfig struct {
	Enabled         bool              `json:"enabled"`
	RequestsPerMin  int               `json:"requestsPerMin"`
	BurstSize       int               `json:"burstSize"`
	PerToolLimits   map[string]int    `json:"perToolLimits"`
}

func Load(configPath string) (*Config, error) {
	cfg := &Config{
		// Default values
		KataGo: KataGoConfig{
			BinaryPath:  "katago",
			NumThreads:  4,
			MaxVisits:   1000,
			MaxTime:     10.0,
		},
		Server: ServerConfig{
			Name:        "katago-mcp",
			Version:     "0.1.0",
			Description: "KataGo analysis server for MCP",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Prefix: "[katago-mcp] ",
		},
		RateLimit: RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 60,
			BurstSize:      10,
			PerToolLimits:  make(map[string]int),
		},
	}

	// Load from JSON file if provided
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Override with environment variables
	cfg.applyEnvOverrides()

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) applyEnvOverrides() {
	// KataGo settings
	if v := os.Getenv("KATAGO_BINARY_PATH"); v != "" {
		c.KataGo.BinaryPath = v
	}
	if v := os.Getenv("KATAGO_MODEL_PATH"); v != "" {
		c.KataGo.ModelPath = v
	}
	if v := os.Getenv("KATAGO_CONFIG_PATH"); v != "" {
		c.KataGo.ConfigPath = v
	}

	// Logging settings
	if v := os.Getenv("KATAGO_MCP_LOG_LEVEL"); v != "" {
		c.Logging.Level = v
	}

	// Rate limit settings
	if v := os.Getenv("KATAGO_MCP_RATE_LIMIT_ENABLED"); v != "" {
		c.RateLimit.Enabled = strings.ToLower(v) == "true"
	}
}

func (c *Config) validate() error {
	// Validate paths exist if they're absolute paths
	if filepath.IsAbs(c.KataGo.BinaryPath) {
		if _, err := os.Stat(c.KataGo.BinaryPath); err != nil {
			return fmt.Errorf("katago binary not found at %s", c.KataGo.BinaryPath)
		}
	}

	if c.KataGo.ModelPath != "" && filepath.IsAbs(c.KataGo.ModelPath) {
		if _, err := os.Stat(c.KataGo.ModelPath); err != nil {
			return fmt.Errorf("katago model not found at %s", c.KataGo.ModelPath)
		}
	}

	// Validate numeric ranges
	if c.KataGo.NumThreads < 1 {
		c.KataGo.NumThreads = 1
	}
	if c.KataGo.MaxVisits < 1 {
		c.KataGo.MaxVisits = 1
	}
	if c.KataGo.MaxTime < 0.1 {
		c.KataGo.MaxTime = 0.1
	}

	// Validate rate limits
	if c.RateLimit.Enabled {
		if c.RateLimit.RequestsPerMin < 1 {
			c.RateLimit.RequestsPerMin = 1
		}
		if c.RateLimit.BurstSize < 1 {
			c.RateLimit.BurstSize = 1
		}
	}

	return nil
}

func (c *Config) GetKataGoHomeDir() string {
	if home := os.Getenv("KATAGO_HOME"); home != "" {
		return home
	}
	
	userHome, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	
	return filepath.Join(userHome, ".katago")
}

func GetConfigPath() string {
	// Check environment variable first
	if path := os.Getenv("KATAGO_MCP_CONFIG"); path != "" {
		return path
	}

	// Check current directory
	if _, err := os.Stat("config.json"); err == nil {
		return "config.json"
	}

	// Check home directory
	if home, err := os.UserHomeDir(); err == nil {
		configPath := filepath.Join(home, ".katago-mcp", "config.json")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	return ""
}