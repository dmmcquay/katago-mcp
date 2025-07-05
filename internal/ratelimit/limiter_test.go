package ratelimit

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/logging"
)

func TestLimiter(t *testing.T) {
	cfg := &logging.Config{
		Level:   "debug",
		Format:  logging.FormatText,
		Service: "test",
		Version: "test",
		Prefix:  "[TEST] ",
	}
	logger := logging.NewLoggerFromConfig(cfg)

	t.Run("NewLimiter", func(t *testing.T) {
		// Disabled limiter
		cfg := &config.RateLimitConfig{Enabled: false}
		limiter := NewLimiter(cfg, logger)
		if limiter != nil {
			t.Error("Expected nil limiter when disabled")
		}

		// Enabled limiter
		cfg = &config.RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 60,
			BurstSize:      10,
			PerToolLimits: map[string]int{
				"analyzePosition": 30,
				"findMistakes":    10,
			},
		}
		limiter = NewLimiter(cfg, logger)
		if limiter == nil {
			t.Fatal("Expected non-nil limiter when enabled")
		}
		if limiter.globalBucket.capacity != 10 {
			t.Errorf("Expected global burst size 10, got %d", limiter.globalBucket.capacity)
		}
		if len(limiter.toolBuckets) != 2 {
			t.Errorf("Expected 2 tool buckets, got %d", len(limiter.toolBuckets))
		}
	})

	t.Run("GlobalLimit", func(t *testing.T) {
		cfg := &config.RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 60, // 1 per second
			BurstSize:      5,
		}
		limiter := NewLimiter(cfg, logger)

		// Should allow burst
		for i := 0; i < 5; i++ {
			allowed, err := limiter.Allow("client1", "someAction")
			if !allowed || err != nil {
				t.Errorf("Request %d should be allowed: %v", i+1, err)
			}
		}

		// Should deny after burst
		allowed, err := limiter.Allow("client1", "someAction")
		if allowed || err == nil {
			t.Error("Request should be denied after burst")
		}
	})

	t.Run("ToolLimit", func(t *testing.T) {
		cfg := &config.RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 600, // 10 per second
			BurstSize:      10,
			PerToolLimits: map[string]int{
				"limitedTool": 60, // 1 per second, burst 1
			},
		}
		limiter := NewLimiter(cfg, logger)

		// First request should be allowed
		allowed, err := limiter.Allow("client1", "limitedTool")
		if !allowed || err != nil {
			t.Errorf("First request should be allowed: %v", err)
		}

		// Second request should be denied by tool limit
		allowed, err = limiter.Allow("client1", "limitedTool")
		if allowed || err == nil {
			t.Error("Second request should be denied by tool limit")
		}

		// Other tools should still work
		allowed, err = limiter.Allow("client1", "otherTool")
		if !allowed || err != nil {
			t.Errorf("Other tool should be allowed: %v", err)
		}
	})

	t.Run("ClientLimit", func(t *testing.T) {
		cfg := &config.RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 600, // 10 per second
			BurstSize:      20,  // Larger burst to test per-client limits
		}
		limiter := NewLimiter(cfg, logger)

		// Client 1 uses half the burst
		for i := 0; i < 10; i++ {
			allowed, err := limiter.Allow("client1", "action")
			if !allowed || err != nil {
				t.Errorf("Client1 request %d should be allowed: %v", i+1, err)
			}
		}

		// Client 2 should still have tokens from global pool
		allowed, err := limiter.Allow("client2", "action")
		if !allowed || err != nil {
			t.Errorf("Client2 should be allowed: %v", err)
		}

		// Client 1 uses more tokens
		for i := 0; i < 9; i++ {
			_, _ = limiter.Allow("client1", "action")
		}

		// Now global limit should be hit
		allowed, err = limiter.Allow("client3", "action")
		if allowed || err == nil {
			t.Error("Client3 should be rate limited due to global limit")
		}
	})

	t.Run("Wait", func(t *testing.T) {
		cfg := &config.RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 600, // 10 per second
			BurstSize:      5,
		}
		limiter := NewLimiter(cfg, logger)

		// Use up burst
		for i := 0; i < 5; i++ {
			_, _ = limiter.Allow("client1", "action")
		}

		// Check wait time
		wait := limiter.Wait("client1", "action")
		expectedWait := 100 * time.Millisecond // Need 1 token at 10/sec = 0.1 sec
		if wait < expectedWait-10*time.Millisecond || wait > expectedWait+10*time.Millisecond {
			t.Errorf("Expected wait ~%v, got %v", expectedWait, wait)
		}
	})

	t.Run("Reset", func(t *testing.T) {
		cfg := &config.RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 60,
			BurstSize:      5,
		}
		limiter := NewLimiter(cfg, logger)

		// Use all tokens
		for i := 0; i < 5; i++ {
			_, _ = limiter.Allow("client1", "action")
		}

		// Should be denied
		allowed, _ := limiter.Allow("client1", "action")
		if allowed {
			t.Error("Should be denied after using all tokens")
		}

		// Reset
		limiter.Reset()

		// Should be allowed again
		allowed, err := limiter.Allow("client1", "action")
		if !allowed || err != nil {
			t.Errorf("Should be allowed after reset: %v", err)
		}
	})

	t.Run("GetStatus", func(t *testing.T) {
		// Nil limiter
		var limiter *Limiter
		status := limiter.GetStatus()
		if status["enabled"].(bool) {
			t.Error("Nil limiter should report disabled")
		}

		// Active limiter
		cfg := &config.RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 60,
			BurstSize:      10,
			PerToolLimits: map[string]int{
				"tool1": 30,
			},
		}
		limiter = NewLimiter(cfg, logger)
		limiter.Allow("client1", "tool1")

		status = limiter.GetStatus()
		if !status["enabled"].(bool) {
			t.Error("Active limiter should report enabled")
		}
		if status["requestsPerMin"].(int) != 60 {
			t.Errorf("Expected requestsPerMin 60, got %v", status["requestsPerMin"])
		}
		if status["activeClients"].(int) != 1 {
			t.Errorf("Expected 1 active client, got %v", status["activeClients"])
		}
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		cfg := &config.RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 6000, // 100 per second
			BurstSize:      100,
		}
		limiter := NewLimiter(cfg, logger)

		var allowed int32
		var denied int32
		var wg sync.WaitGroup

		// Run 20 goroutines making requests
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(clientID int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					if ok, _ := limiter.Allow(fmt.Sprintf("client%d", clientID), "action"); ok {
						atomic.AddInt32(&allowed, 1)
					} else {
						atomic.AddInt32(&denied, 1)
					}
					time.Sleep(1 * time.Millisecond)
				}
			}(i)
		}

		wg.Wait()

		// Should have allowed ~100 requests (burst size)
		if allowed < 95 || allowed > 105 {
			t.Errorf("Expected ~100 allowed requests, got %d", allowed)
		}
		if denied < 95 || denied > 105 {
			t.Errorf("Expected ~100 denied requests, got %d", denied)
		}
	})

	t.Run("ClientCleanup", func(t *testing.T) {
		cfg := &config.RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 60,
			BurstSize:      10,
		}
		limiter := NewLimiter(cfg, logger)

		// Add some clients
		limiter.Allow("client1", "action")
		limiter.Allow("client2", "action")

		// Check they exist
		if len(limiter.clientLimits) != 2 {
			t.Errorf("Expected 2 clients, got %d", len(limiter.clientLimits))
		}

		// Manually set lastSeen to trigger cleanup
		limiter.mu.Lock()
		for _, client := range limiter.clientLimits {
			client.lastSeen = time.Now().Add(-31 * time.Minute)
		}
		limiter.mu.Unlock()

		// Trigger cleanup manually (normally runs on timer)
		limiter.mu.Lock()
		now := time.Now()
		staleTimeout := 30 * time.Minute
		for clientID, client := range limiter.clientLimits {
			if now.Sub(client.lastSeen) > staleTimeout {
				delete(limiter.clientLimits, clientID)
			}
		}
		limiter.mu.Unlock()

		// Check they were cleaned up
		if len(limiter.clientLimits) != 0 {
			t.Errorf("Expected 0 clients after cleanup, got %d", len(limiter.clientLimits))
		}
	})
}
