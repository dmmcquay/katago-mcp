package cache

import (
	"container/list"
	"sync"
	"time"
)

// entry represents a cache entry with value and metadata.
type entry struct {
	key       string
	value     interface{}
	size      int64
	timestamp time.Time
}

// LRU implements a thread-safe least-recently-used cache with size limits.
type LRU struct {
	mu           sync.RWMutex
	maxItems     int
	maxSizeBytes int64
	currentSize  int64
	items        map[string]*list.Element
	evictionList *list.List

	// Metrics
	hits      int64
	misses    int64
	evictions int64
}

// NewLRU creates a new LRU cache with the given limits.
// maxItems: maximum number of items (0 = unlimited).
// maxSizeBytes: maximum total size in bytes (0 = unlimited).
func NewLRU(maxItems int, maxSizeBytes int64) *LRU {
	return &LRU{
		maxItems:     maxItems,
		maxSizeBytes: maxSizeBytes,
		items:        make(map[string]*list.Element),
		evictionList: list.New(),
	}
}

// Get retrieves a value from the cache.
func (c *LRU) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.evictionList.MoveToFront(elem)
		c.hits++
		e, ok := elem.Value.(*entry)
		if !ok {
			return nil, false
		}
		return e.value, true
	}

	c.misses++
	return nil, false
}

// Put adds or updates a value in the cache.
// size is the approximate size of the value in bytes.
func (c *LRU) Put(key string, value interface{}, size int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if key already exists
	if elem, ok := c.items[key]; ok {
		// Update existing entry
		c.evictionList.MoveToFront(elem)
		e, ok := elem.Value.(*entry)
		if !ok {
			return
		}
		c.currentSize += size - e.size // Adjust size
		e.value = value
		e.size = size
		e.timestamp = time.Now()
		return
	}

	// Add new entry
	e := &entry{
		key:       key,
		value:     value,
		size:      size,
		timestamp: time.Now(),
	}
	elem := c.evictionList.PushFront(e)
	c.items[key] = elem
	c.currentSize += size

	// Evict if necessary
	c.evict()
}

// evict removes entries until cache is within limits.
func (c *LRU) evict() {
	// Special case: if we only have one item and it exceeds size limit, keep it
	if c.evictionList.Len() == 1 && c.maxSizeBytes > 0 && c.currentSize > c.maxSizeBytes {
		return
	}

	for c.evictionList.Len() > 0 {
		// Check if we need to evict
		needEvict := false
		if c.maxItems > 0 && c.evictionList.Len() > c.maxItems {
			needEvict = true
		}
		if c.maxSizeBytes > 0 && c.currentSize > c.maxSizeBytes {
			needEvict = true
		}

		if !needEvict {
			break
		}

		// Remove least recently used
		elem := c.evictionList.Back()
		if elem != nil {
			c.removeElement(elem)
			c.evictions++
		}
	}
}

// removeElement removes an element from the cache.
func (c *LRU) removeElement(elem *list.Element) {
	c.evictionList.Remove(elem)
	e, ok := elem.Value.(*entry)
	if !ok {
		return
	}
	delete(c.items, e.key)
	c.currentSize -= e.size
}

// Delete removes a key from the cache.
func (c *LRU) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
		return true
	}
	return false
}

// Clear removes all entries from the cache.
func (c *LRU) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.evictionList.Init()
	c.currentSize = 0
}

// Len returns the number of items in the cache.
func (c *LRU) Len() int {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.evictionList == nil {
		return 0
	}
	return c.evictionList.Len()
}

// Size returns the total size of items in the cache.
func (c *LRU) Size() int64 {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentSize
}

// Stats returns cache statistics.
type Stats struct {
	Items     int
	Size      int64
	Hits      int64
	Misses    int64
	Evictions int64
	HitRate   float64
}

// Stats returns current cache statistics.
func (c *LRU) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}

	return Stats{
		Items:     c.evictionList.Len(),
		Size:      c.currentSize,
		Hits:      c.hits,
		Misses:    c.misses,
		Evictions: c.evictions,
		HitRate:   hitRate,
	}
}

// ResetStats resets hit/miss/eviction counters.
func (c *LRU) ResetStats() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hits = 0
	c.misses = 0
	c.evictions = 0
}
