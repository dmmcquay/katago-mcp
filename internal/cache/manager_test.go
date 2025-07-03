package cache

import (
	"testing"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_CacheKey(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test: ", "debug"))
	cfg := &config.CacheConfig{
		Enabled:      true,
		MaxItems:     10,
		MaxSizeBytes: 1024,
		TTLSeconds:   60,
	}
	manager := NewManager(cfg, logger)

	// Test cache key generation
	query1 := map[string]interface{}{
		"rules":      "tromp-taylor",
		"boardXSize": 19,
		"boardYSize": 19,
		"moves": [][]interface{}{
			{"B", "C4"},
			{"W", "Q16"},
		},
		"maxVisits": 100,
	}

	key1, err := manager.CacheKey(query1)
	require.NoError(t, err)
	assert.NotEmpty(t, key1)

	// Same query should produce same key
	key2, err := manager.CacheKey(query1)
	require.NoError(t, err)
	assert.Equal(t, key1, key2)

	// Different query should produce different key
	query2 := map[string]interface{}{
		"rules":      "tromp-taylor",
		"boardXSize": 19,
		"boardYSize": 19,
		"moves": [][]interface{}{
			{"B", "D4"}, // Different move
			{"W", "Q16"},
		},
		"maxVisits": 100,
	}

	key3, err := manager.CacheKey(query2)
	require.NoError(t, err)
	assert.NotEqual(t, key1, key3)
}

func TestManager_GetPut(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test: ", "debug"))
	cfg := &config.CacheConfig{
		Enabled:      true,
		MaxItems:     10,
		MaxSizeBytes: 1024,
		TTLSeconds:   60,
	}
	manager := NewManager(cfg, logger)

	// Put and get
	key := "test-key"
	value := map[string]interface{}{
		"result": "test-value",
		"moves":  []string{"C4", "D4"},
	}

	manager.Put(key, value, 100)

	retrieved, ok := manager.Get(key)
	assert.True(t, ok)
	assert.Equal(t, value, retrieved)

	// Test non-existent key
	_, ok = manager.Get("non-existent")
	assert.False(t, ok)
}

func TestManager_TTL(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test: ", "debug"))
	cfg := &config.CacheConfig{
		Enabled:      true,
		MaxItems:     10,
		MaxSizeBytes: 1024,
		TTLSeconds:   1, // 1 second TTL
	}
	manager := NewManager(cfg, logger)

	key := "ttl-test"
	value := "test-value"

	manager.Put(key, value, 10)

	// Should exist immediately
	retrieved, ok := manager.Get(key)
	assert.True(t, ok)
	assert.Equal(t, value, retrieved)

	// Wait for TTL to expire
	time.Sleep(2 * time.Second)

	// Should be expired
	_, ok = manager.Get(key)
	assert.False(t, ok)
}

func TestManager_Disabled(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test: ", "debug"))

	// Test with nil config
	manager := NewManager(nil, logger)
	assert.False(t, manager.IsEnabled())

	// Test with disabled config
	cfg := &config.CacheConfig{
		Enabled: false,
	}
	manager = NewManager(cfg, logger)
	assert.False(t, manager.IsEnabled())

	// Operations should be no-ops
	manager.Put("key", "value", 100)
	_, ok := manager.Get("key")
	assert.False(t, ok)

	stats := manager.Stats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
}

func TestManager_Stats(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test: ", "debug"))
	cfg := &config.CacheConfig{
		Enabled:      true,
		MaxItems:     10,
		MaxSizeBytes: 1024,
		TTLSeconds:   60,
	}
	manager := NewManager(cfg, logger)

	// Initial stats
	stats := manager.Stats()
	assert.Equal(t, 0, stats.Items)

	// Add items
	manager.Put("key1", "value1", 50)
	manager.Put("key2", "value2", 50)

	stats = manager.Stats()
	assert.Equal(t, 2, stats.Items)
	// Size will be more than 100 due to timedEntry wrapper overhead
	assert.Greater(t, stats.Size, int64(100))
}

func TestManager_Clear(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test: ", "debug"))
	cfg := &config.CacheConfig{
		Enabled:      true,
		MaxItems:     10,
		MaxSizeBytes: 1024,
		TTLSeconds:   60,
	}
	manager := NewManager(cfg, logger)

	// Add items
	manager.Put("key1", "value1", 50)
	manager.Put("key2", "value2", 50)

	stats := manager.Stats()
	assert.Equal(t, 2, stats.Items)

	// Clear
	manager.Clear()

	stats = manager.Stats()
	assert.Equal(t, 0, stats.Items)
	assert.Equal(t, int64(0), stats.Size)
}

func TestEstimateSize(t *testing.T) {
	testCases := []struct {
		name     string
		response interface{}
		minSize  int64
	}{
		{
			name:     "simple string",
			response: "hello world",
			minSize:  10,
		},
		{
			name: "struct",
			response: struct {
				Name  string
				Value int
			}{Name: "test", Value: 42},
			minSize: 20,
		},
		{
			name:     "slice",
			response: []string{"a", "b", "c"},
			minSize:  10,
		},
		{
			name: "map",
			response: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
			minSize: 20,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			size := EstimateSize(tc.response)
			assert.Greater(t, size, tc.minSize)
		})
	}
}

// TestManager_Integration tests the manager with realistic KataGo responses
func TestManager_Integration(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test: ", "debug"))
	cfg := &config.CacheConfig{
		Enabled:      true,
		MaxItems:     100,
		MaxSizeBytes: 10 * 1024 * 1024, // 10MB
		TTLSeconds:   300,              // 5 minutes
	}
	manager := NewManager(cfg, logger)

	// Simulate KataGo query
	query := map[string]interface{}{
		"rules":      "tromp-taylor",
		"boardXSize": 19,
		"boardYSize": 19,
		"moves": [][]interface{}{
			{"B", "D4"},
			{"W", "Q16"},
			{"B", "D16"},
			{"W", "Q4"},
		},
		"maxVisits":    1000,
		"analyzeTurns": []int{3, 4},
	}

	// Generate cache key
	key, err := manager.CacheKey(query)
	require.NoError(t, err)

	// Simulate KataGo response
	response := map[string]interface{}{
		"id":         "q1",
		"turnNumber": 4,
		"moveInfos": []map[string]interface{}{
			{
				"move":      "C3",
				"visits":    523,
				"winrate":   0.523,
				"scoreLead": 0.8,
			},
			{
				"move":      "R16",
				"visits":    312,
				"winrate":   0.518,
				"scoreLead": 0.6,
			},
		},
		"rootInfo": map[string]interface{}{
			"visits":        1000,
			"winrate":       0.521,
			"scoreLead":     0.7,
			"currentPlayer": "B",
		},
	}

	// Cache the response
	size := EstimateSize(response)
	manager.Put(key, response, size)

	// Retrieve and verify
	cached, ok := manager.Get(key)
	assert.True(t, ok)
	assert.NotNil(t, cached)

	// Check stats
	stats := manager.Stats()
	assert.Equal(t, 1, stats.Items)
	assert.Greater(t, stats.Size, int64(100))

	// Simulate cache hit
	_, ok = manager.Get(key)
	assert.True(t, ok)

	// Non-existent key should miss
	differentQuery := map[string]interface{}{
		"rules":      "chinese",
		"boardXSize": 19,
		"boardYSize": 19,
		"moves":      [][]interface{}{},
		"maxVisits":  1000,
	}

	differentKey, err := manager.CacheKey(differentQuery)
	require.NoError(t, err)
	assert.NotEqual(t, key, differentKey)

	_, ok = manager.Get(differentKey)
	assert.False(t, ok)
}

