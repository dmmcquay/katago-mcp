package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/logging"
)

// Manager handles caching of KataGo analysis results.
type Manager struct {
	cache   *LRU
	logger  logging.ContextLogger
	enabled bool
	ttl     time.Duration
}

// NewManager creates a new cache manager.
func NewManager(cfg *config.CacheConfig, logger logging.ContextLogger) *Manager {
	if cfg == nil || !cfg.Enabled {
		return &Manager{
			enabled: false,
			logger:  logger,
		}
	}

	cache := NewLRU(cfg.MaxItems, cfg.MaxSizeBytes)

	return &Manager{
		cache:   cache,
		logger:  logger,
		enabled: cfg.Enabled,
		ttl:     time.Duration(cfg.TTLSeconds) * time.Second,
	}
}

// CacheKey generates a cache key for an analysis query.
func (m *Manager) CacheKey(query map[string]interface{}) (string, error) {
	// Extract relevant fields for cache key
	// We only cache based on position and analysis parameters
	keyData := map[string]interface{}{
		"rules":         query["rules"],
		"boardXSize":    query["boardXSize"],
		"boardYSize":    query["boardYSize"],
		"moves":         query["moves"],
		"initialStones": query["initialStones"],
		"maxVisits":     query["maxVisits"],
		"analyzeTurns":  query["analyzeTurns"],
	}

	// Convert to JSON for consistent ordering
	data, err := json.Marshal(keyData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cache key: %w", err)
	}

	// Generate SHA256 hash
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// Get retrieves a cached analysis result.
func (m *Manager) Get(key string) (interface{}, bool) {
	if !m.enabled || m.cache == nil {
		return nil, false
	}

	// Get from cache
	val, ok := m.cache.Get(key)
	if !ok {
		return nil, false
	}

	// Check if it's a timed entry
	if entry, ok := val.(*timedEntry); ok {
		// Check TTL
		if m.ttl > 0 && time.Since(entry.timestamp) > m.ttl {
			// Expired, remove it
			m.cache.Delete(key)
			m.logger.Debug("Cache entry expired", "key", key, "age", time.Since(entry.timestamp))
			return nil, false
		}
		return entry.value, true
	}

	// Return raw value (backward compatibility)
	return val, true
}

// Put stores an analysis result in the cache.
func (m *Manager) Put(key string, value interface{}, size int64) {
	if !m.enabled || m.cache == nil {
		return
	}

	// Wrap with timestamp if TTL is enabled
	var storedValue interface{}
	if m.ttl > 0 {
		storedValue = &timedEntry{
			value:     value,
			timestamp: time.Now(),
		}
		// Add overhead for timestamp
		size += 64
	} else {
		storedValue = value
	}

	m.cache.Put(key, storedValue, size)
	m.logger.Debug("Cached analysis result", "key", key, "size", size)
}

// Stats returns cache statistics.
func (m *Manager) Stats() Stats {
	if !m.enabled || m.cache == nil {
		return Stats{}
	}
	return m.cache.Stats()
}

// Clear clears the cache.
func (m *Manager) Clear() {
	if m.cache != nil {
		m.cache.Clear()
	}
}

// IsEnabled returns whether caching is enabled.
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// timedEntry wraps a value with a timestamp for TTL support.
type timedEntry struct {
	value     interface{}
	timestamp time.Time
}

// EstimateSize estimates the size of an analysis response in bytes.
func EstimateSize(response interface{}) int64 {
	// Simple estimation based on JSON encoding
	data, err := json.Marshal(response)
	if err != nil {
		// Fallback to a reasonable estimate
		return 1024
	}
	return int64(len(data))
}
