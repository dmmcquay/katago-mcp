package metrics

import (
	"sync"
	"time"
)

// Collector collects metrics for the application.
type Collector struct {
	mu sync.RWMutex

	// Tool metrics
	toolCalls     map[string]int64
	toolErrors    map[string]int64
	toolDurations map[string][]time.Duration

	// Rate limit metrics
	rateLimitHits  int64
	rateLimitTotal int64
}

// NewCollector creates a new metrics collector.
func NewCollector() *Collector {
	return &Collector{
		toolCalls:     make(map[string]int64),
		toolErrors:    make(map[string]int64),
		toolDurations: make(map[string][]time.Duration),
	}
}

// RecordToolCall records a tool call with its status and duration.
func (c *Collector) RecordToolCall(tool, status string, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.toolCalls[tool]++

	if status == "error" {
		c.toolErrors[tool]++
	}

	if status == "rate_limited" {
		c.rateLimitHits++
	}

	c.rateLimitTotal++

	// Keep last 100 durations for each tool
	durations := c.toolDurations[tool]
	durations = append(durations, duration)
	if len(durations) > 100 {
		durations = durations[1:]
	}
	c.toolDurations[tool] = durations
}

// GetStats returns current metrics statistics.
func (c *Collector) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := make(map[string]interface{})

	// Tool stats
	toolStats := make(map[string]interface{})
	for tool, calls := range c.toolCalls {
		errors := c.toolErrors[tool]
		errorRate := float64(0)
		if calls > 0 {
			errorRate = float64(errors) / float64(calls)
		}

		// Calculate average duration
		var totalDuration time.Duration
		durations := c.toolDurations[tool]
		for _, d := range durations {
			totalDuration += d
		}
		avgDuration := time.Duration(0)
		if len(durations) > 0 {
			avgDuration = totalDuration / time.Duration(len(durations))
		}

		toolStats[tool] = map[string]interface{}{
			"calls":           calls,
			"errors":          errors,
			"error_rate":      errorRate,
			"avg_duration_ms": avgDuration.Milliseconds(),
		}
	}
	stats["tools"] = toolStats

	// Rate limit stats
	rateLimitRate := float64(0)
	if c.rateLimitTotal > 0 {
		rateLimitRate = float64(c.rateLimitHits) / float64(c.rateLimitTotal)
	}
	stats["rate_limits"] = map[string]interface{}{
		"hits":  c.rateLimitHits,
		"total": c.rateLimitTotal,
		"rate":  rateLimitRate,
	}

	return stats
}

// Reset clears all metrics.
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.toolCalls = make(map[string]int64)
	c.toolErrors = make(map[string]int64)
	c.toolDurations = make(map[string][]time.Duration)
	c.rateLimitHits = 0
	c.rateLimitTotal = 0
}

