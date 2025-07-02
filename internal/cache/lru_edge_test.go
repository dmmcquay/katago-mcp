package cache

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestLRU_MemoryPressure tests behavior under memory pressure
func TestLRU_MemoryPressure(t *testing.T) {
	// Skip in short mode as this test uses significant memory
	if testing.Short() {
		t.Skip("Skipping memory pressure test in short mode")
	}

	// Create cache with 100MB limit
	cache := NewLRU(0, 100*1024*1024)

	// Add large items
	for i := 0; i < 1000; i++ {
		// Each item is ~1MB
		largeData := make([]byte, 1024*1024)
		for j := range largeData {
			largeData[j] = byte(i % 256)
		}
		cache.Put(fmt.Sprintf("large%d", i), largeData, int64(len(largeData)))

		// Cache should maintain size limit
		assert.True(t, cache.Size() <= 100*1024*1024+1024*1024, "Cache exceeded size limit")
	}

	// Force GC to check for memory leaks
	runtime.GC()
	runtime.GC()
}

// TestLRU_RaceConditions tests for race conditions with race detector
func TestLRU_RaceConditions(t *testing.T) {
	cache := NewLRU(100, 0)
	done := make(chan bool)

	// Writer goroutines
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				cache.Put(fmt.Sprintf("key%d-%d", id, j), j, 8)
			}
			done <- true
		}(i)
	}

	// Reader goroutines
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				cache.Get(fmt.Sprintf("key%d-%d", id, j))
			}
			done <- true
		}(i)
	}

	// Deleter goroutines
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 50; j++ {
				cache.Delete(fmt.Sprintf("key%d-%d", id, j))
			}
			done <- true
		}(i)
	}

	// Stats reader
	go func() {
		for i := 0; i < 100; i++ {
			_ = cache.Stats()
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Clear operations
	go func() {
		time.Sleep(50 * time.Millisecond)
		cache.Clear()
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 27; i++ {
		<-done
	}
}

// TestLRU_PanicRecovery tests that cache operations don't panic
func TestLRU_PanicRecovery(t *testing.T) {
	cache := NewLRU(10, 100)

	// Test with various edge cases that shouldn't panic
	testCases := []struct {
		name string
		fn   func()
	}{
		{
			name: "nil cache operations",
			fn: func() {
				var nilCache *LRU
				// These should not panic
				assert.NotPanics(t, func() { nilCache.Len() })
				assert.NotPanics(t, func() { nilCache.Size() })
			},
		},
		{
			name: "negative size",
			fn: func() {
				// Negative size should be handled gracefully
				cache.Put("negative", "value", -100)
				assert.Equal(t, int64(-100), cache.Size())
			},
		},
		{
			name: "concurrent clear during operations",
			fn: func() {
				var wg sync.WaitGroup
				wg.Add(2)

				go func() {
					defer wg.Done()
					for i := 0; i < 100; i++ {
						cache.Put(fmt.Sprintf("key%d", i), i, 8)
					}
				}()

				go func() {
					defer wg.Done()
					time.Sleep(time.Millisecond)
					cache.Clear()
				}()

				wg.Wait()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.fn()
		})
	}
}

// TestLRU_Deadlock tests for potential deadlocks
func TestLRU_Deadlock(t *testing.T) {
	cache := NewLRU(10, 0)
	timeout := time.After(5 * time.Second)
	done := make(chan bool)

	go func() {
		// Nested operations that could cause deadlock
		cache.Put("key1", "value1", 10)

		// Get while holding lock (internal)
		val, _ := cache.Get("key1")

		// Update same key
		cache.Put("key1", val, 10)

		// Delete and re-add
		cache.Delete("key1")
		cache.Put("key1", "new", 10)

		// Clear and add
		cache.Clear()
		cache.Put("key2", "value2", 10)

		done <- true
	}()

	select {
	case <-done:
		// Success - no deadlock
	case <-timeout:
		t.Fatal("Test timed out - possible deadlock")
	}
}

// TestLRU_SecurityEdgeCases tests security-related edge cases
func TestLRU_SecurityEdgeCases(t *testing.T) {
	cache := NewLRU(100, 1024*1024) // 1MB limit

	t.Run("KeyCollisions", func(t *testing.T) {
		// Test that similar keys don't collide
		cache.Put("key", "value1", 10)
		cache.Put("key ", "value2", 10) // trailing space
		cache.Put(" key", "value3", 10) // leading space
		cache.Put("KEY", "value4", 10)  // different case

		// All should be distinct
		val, _ := cache.Get("key")
		assert.Equal(t, "value1", val)
		val, _ = cache.Get("key ")
		assert.Equal(t, "value2", val)
		val, _ = cache.Get(" key")
		assert.Equal(t, "value3", val)
		val, _ = cache.Get("KEY")
		assert.Equal(t, "value4", val)
	})

	t.Run("LargeKeys", func(t *testing.T) {
		// Test with very large keys
		largeKey := make([]byte, 1024*1024) // 1MB key
		for i := range largeKey {
			largeKey[i] = 'a'
		}

		cache.Put(string(largeKey), "value", 10)
		val, ok := cache.Get(string(largeKey))
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})

	t.Run("UnicodeKeys", func(t *testing.T) {
		// Test with various Unicode characters
		unicodeKeys := []string{
			"中文",
			"🔑",
			"\u0000null",
			"مفتاح",
			"κλειδί",
			"🔑🔒🗝️",
		}

		for i, key := range unicodeKeys {
			cache.Put(key, fmt.Sprintf("value%d", i), 10)
		}

		for i, key := range unicodeKeys {
			val, ok := cache.Get(key)
			assert.True(t, ok, "Failed to get key: %s", key)
			assert.Equal(t, fmt.Sprintf("value%d", i), val)
		}
	})
}

// TestLRU_EvictionOrder tests that eviction follows strict LRU order
func TestLRU_EvictionOrder(t *testing.T) {
	cache := NewLRU(3, 0)

	// Add items in order: 0, 1, 2
	cache.Put("key0", "value0", 10)
	cache.Put("key1", "value1", 10)
	cache.Put("key2", "value2", 10)

	// Access key0 to make it most recently used
	cache.Get("key0")

	// Add a new item, should evict key1 (least recently used)
	cache.Put("key3", "value3", 10)

	// Check eviction
	_, ok := cache.Get("key1")
	assert.False(t, ok, "key1 should have been evicted")

	// Check others still exist
	_, ok = cache.Get("key0")
	assert.True(t, ok, "key0 should still exist (recently accessed)")
	_, ok = cache.Get("key2")
	assert.True(t, ok, "key2 should still exist")
	_, ok = cache.Get("key3")
	assert.True(t, ok, "key3 should exist (just added)")
}

// TestLRU_CacheCoherence tests cache coherence under concurrent modifications
func TestLRU_CacheCoherence(t *testing.T) {
	cache := NewLRU(1000, 0)
	numGoroutines := 20
	numOperations := 100

	// Shared counter for verification
	var mu sync.Mutex
	existence := make(map[string]bool)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("shared-key-%d", j)

				switch j % 3 {
				case 0: // Put
					cache.Put(key, id, 8)
					mu.Lock()
					existence[key] = true
					mu.Unlock()

				case 1: // Get and verify
					if val, ok := cache.Get(key); ok {
						// Value should be an int
						assert.IsType(t, 0, val)
					}

				case 2: // Delete
					if id == 0 { // Only one goroutine deletes
						cache.Delete(key)
						mu.Lock()
						delete(existence, key)
						mu.Unlock()
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final state consistency
	mu.Lock()
	for key, shouldExist := range existence {
		_, exists := cache.Get(key)
		if shouldExist {
			assert.True(t, exists, "Key %s should exist but doesn't", key)
		}
	}
	mu.Unlock()
}
