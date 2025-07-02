package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLRU_BasicOperations(t *testing.T) {
	cache := NewLRU(3, 0) // Max 3 items, no size limit

	// Test Put and Get
	cache.Put("key1", "value1", 10)
	val, ok := cache.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	// Test Get non-existent key
	val, ok = cache.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, val)

	// Test multiple items
	cache.Put("key2", "value2", 20)
	cache.Put("key3", "value3", 30)

	assert.Equal(t, 3, cache.Len())
	assert.Equal(t, int64(60), cache.Size())
}

func TestLRU_Eviction_ItemLimit(t *testing.T) {
	cache := NewLRU(3, 0) // Max 3 items

	// Add 3 items
	cache.Put("key1", "value1", 10)
	cache.Put("key2", "value2", 20)
	cache.Put("key3", "value3", 30)

	// Access key1 to make it recently used
	_, _ = cache.Get("key1")

	// Add a 4th item, should evict key2 (least recently used)
	cache.Put("key4", "value4", 40)

	// Check eviction
	_, ok := cache.Get("key2")
	assert.False(t, ok, "key2 should have been evicted")

	// Check others still exist
	_, ok = cache.Get("key1")
	assert.True(t, ok, "key1 should still exist")
	_, ok = cache.Get("key3")
	assert.True(t, ok, "key3 should still exist")
	_, ok = cache.Get("key4")
	assert.True(t, ok, "key4 should exist")
}

func TestLRU_Eviction_SizeLimit(t *testing.T) {
	cache := NewLRU(0, 100) // No item limit, 100 bytes size limit

	// Add items totaling 90 bytes
	cache.Put("key1", "value1", 30)
	cache.Put("key2", "value2", 30)
	cache.Put("key3", "value3", 30)

	assert.Equal(t, int64(90), cache.Size())

	// Add item that would exceed size limit
	cache.Put("key4", "value4", 40)

	// Should have evicted key1 (least recently used)
	assert.Equal(t, int64(100), cache.Size())
	_, ok := cache.Get("key1")
	assert.False(t, ok, "key1 should have been evicted")
}

func TestLRU_Eviction_CombinedLimits(t *testing.T) {
	cache := NewLRU(3, 100) // Max 3 items AND 100 bytes

	// Test item limit is hit first
	cache.Put("key1", "value1", 10)
	cache.Put("key2", "value2", 10)
	cache.Put("key3", "value3", 10)
	cache.Put("key4", "value4", 10) // Should trigger eviction by item count

	assert.Equal(t, 3, cache.Len())
	assert.Equal(t, int64(30), cache.Size())

	// Test size limit is hit first
	cache = NewLRU(10, 50) // Max 10 items AND 50 bytes
	cache.Put("key1", "value1", 20)
	cache.Put("key2", "value2", 20)
	cache.Put("key3", "value3", 20) // Should trigger eviction by size

	assert.True(t, cache.Len() <= 10)
	assert.True(t, cache.Size() <= 50)
}

func TestLRU_Update(t *testing.T) {
	cache := NewLRU(3, 0)

	// Add item
	cache.Put("key1", "value1", 10)
	cache.Put("key2", "value2", 20)

	// Update existing item
	cache.Put("key1", "updated", 15)

	// Check updated value
	val, ok := cache.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "updated", val)

	// Check size was adjusted
	assert.Equal(t, int64(35), cache.Size()) // 15 + 20
}

func TestLRU_Delete(t *testing.T) {
	cache := NewLRU(0, 0)

	// Add items
	cache.Put("key1", "value1", 10)
	cache.Put("key2", "value2", 20)

	// Delete existing key
	deleted := cache.Delete("key1")
	assert.True(t, deleted)

	// Check it's gone
	_, ok := cache.Get("key1")
	assert.False(t, ok)

	// Check size adjusted
	assert.Equal(t, int64(20), cache.Size())

	// Delete non-existent key
	deleted = cache.Delete("nonexistent")
	assert.False(t, deleted)
}

func TestLRU_Clear(t *testing.T) {
	cache := NewLRU(0, 0)

	// Add items
	cache.Put("key1", "value1", 10)
	cache.Put("key2", "value2", 20)
	cache.Put("key3", "value3", 30)

	// Clear cache
	cache.Clear()

	// Check everything is gone
	assert.Equal(t, 0, cache.Len())
	assert.Equal(t, int64(0), cache.Size())

	_, ok := cache.Get("key1")
	assert.False(t, ok)
}

func TestLRU_Stats(t *testing.T) {
	cache := NewLRU(3, 0)

	// Initial stats
	stats := cache.Stats()
	assert.Equal(t, 0, stats.Items)
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)

	// Add items and access them
	cache.Put("key1", "value1", 10)
	cache.Put("key2", "value2", 20)

	// Hit
	_, _ = cache.Get("key1")
	// Miss
	_, _ = cache.Get("nonexistent")

	stats = cache.Stats()
	assert.Equal(t, 2, stats.Items)
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, 0.5, stats.HitRate)

	// Test eviction counter
	cache.Put("key3", "value3", 30)
	cache.Put("key4", "value4", 40) // Should evict key2

	stats = cache.Stats()
	assert.Equal(t, int64(1), stats.Evictions)

	// Test ResetStats
	cache.ResetStats()
	stats = cache.Stats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, int64(0), stats.Evictions)
	assert.Equal(t, 3, stats.Items) // Items should not be reset
}

func TestLRU_LRUOrder(t *testing.T) {
	cache := NewLRU(3, 0)

	// Add items in order
	cache.Put("key1", "value1", 10)
	cache.Put("key2", "value2", 20)
	cache.Put("key3", "value3", 30)

	// Access in different order: key2, key1, key3
	cache.Get("key2")
	cache.Get("key1")
	cache.Get("key3")

	// Add new item, should evict the least recently used (key2)
	cache.Put("key4", "value4", 40)

	// key2 should be evicted (accessed first, so least recent)
	_, ok := cache.Get("key2")
	assert.False(t, ok)

	// Others should still exist
	_, ok = cache.Get("key1")
	assert.True(t, ok)
	_, ok = cache.Get("key3")
	assert.True(t, ok)
	_, ok = cache.Get("key4")
	assert.True(t, ok)
}

func TestLRU_ConcurrentAccess(t *testing.T) {
	cache := NewLRU(100, 0)
	numGoroutines := 10
	numOperations := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent puts and gets
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)

				// Put
				cache.Put(key, value, int64(len(value)))

				// Get
				val, ok := cache.Get(key)
				if ok {
					assert.Equal(t, value, val)
				}

				// Sometimes delete
				if j%10 == 0 {
					cache.Delete(key)
				}
			}
		}(i)
	}

	// Concurrent stats reading
	go func() {
		for i := 0; i < 100; i++ {
			_ = cache.Stats()
			time.Sleep(time.Millisecond)
		}
	}()

	wg.Wait()
}

func TestLRU_ConcurrentEviction(t *testing.T) {
	cache := NewLRU(10, 0) // Small cache to force evictions
	numGoroutines := 50
	numOperations := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)
				cache.Put(key, value, 10)

				// Try to get various keys
				testKey := fmt.Sprintf("key-%d-%d", (id+1)%numGoroutines, j)
				cache.Get(testKey)
			}
		}(i)
	}

	wg.Wait()

	// Cache should still be within limits
	assert.True(t, cache.Len() <= 10)
}

func TestLRU_ZeroLimits(t *testing.T) {
	// Both limits zero means unlimited
	cache := NewLRU(0, 0)

	// Add many items
	for i := 0; i < 1000; i++ {
		cache.Put(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i), 10)
	}

	// All should still be there
	assert.Equal(t, 1000, cache.Len())
	assert.Equal(t, int64(10000), cache.Size())
}

func TestLRU_EdgeCases(t *testing.T) {
	t.Run("EmptyKeyValue", func(t *testing.T) {
		cache := NewLRU(10, 0)
		cache.Put("", "empty key", 10)
		val, ok := cache.Get("")
		assert.True(t, ok)
		assert.Equal(t, "empty key", val)

		cache.Put("key", "", 0)
		val, ok = cache.Get("key")
		assert.True(t, ok)
		assert.Equal(t, "", val)
	})

	t.Run("NilValue", func(t *testing.T) {
		cache := NewLRU(10, 0)
		cache.Put("nil", nil, 0)
		val, ok := cache.Get("nil")
		assert.True(t, ok)
		assert.Nil(t, val)
	})

	t.Run("LargeSize", func(t *testing.T) {
		cache := NewLRU(0, 1000)
		// Add item larger than max size
		cache.Put("huge", "data", 2000)
		// Should still be added (single item can exceed limit)
		val, ok := cache.Get("huge")
		assert.True(t, ok)
		assert.Equal(t, "data", val)

		// But adding another should evict it
		cache.Put("small", "data", 10)
		_, ok = cache.Get("huge")
		assert.False(t, ok)
	})
}

func BenchmarkLRU_Put(b *testing.B) {
	cache := NewLRU(1000, 0)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		cache.Put(key, i, 8)
	}
}

func BenchmarkLRU_Get_Hit(b *testing.B) {
	cache := NewLRU(1000, 0)
	// Pre-populate
	for i := 0; i < 1000; i++ {
		cache.Put(fmt.Sprintf("key%d", i), i, 8)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%1000)
		cache.Get(key)
	}
}

func BenchmarkLRU_Get_Miss(b *testing.B) {
	cache := NewLRU(1000, 0)
	// Pre-populate
	for i := 0; i < 1000; i++ {
		cache.Put(fmt.Sprintf("key%d", i), i, 8)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("miss%d", i)
		cache.Get(key)
	}
}

func BenchmarkLRU_Concurrent(b *testing.B) {
	cache := NewLRU(10000, 0)
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key%d", i)
			if i%2 == 0 {
				cache.Put(key, i, 8)
			} else {
				cache.Get(key)
			}
			i++
		}
	})
}

// TestLRU_AnalysisCache tests caching of KataGo analysis results
func TestLRU_AnalysisCache(t *testing.T) {
	// This simulates caching analysis results
	type AnalysisResult struct {
		Position string
		WinRate  float64
		Moves    []string
	}

	// 100 cached positions, up to 10MB total
	cache := NewLRU(100, 10*1024*1024)

	// Cache some analysis results
	result1 := &AnalysisResult{
		Position: "B[dd];W[pp];B[dq]",
		WinRate:  0.52,
		Moves:    []string{"Q16", "Q4", "D16"},
	}

	// Estimate size (rough)
	size := int64(len(result1.Position) + 8 + len(result1.Moves)*5)
	cache.Put(result1.Position, result1, size)

	// Retrieve
	val, ok := cache.Get(result1.Position)
	require.True(t, ok)

	retrieved := val.(*AnalysisResult)
	assert.Equal(t, result1.WinRate, retrieved.WinRate)
	assert.Equal(t, result1.Moves, retrieved.Moves)
}

