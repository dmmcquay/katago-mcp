package ratelimit

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	t.Run("NewTokenBucket", func(t *testing.T) {
		bucket := NewTokenBucket(10, 1.0)
		if bucket.capacity != 10 {
			t.Errorf("Expected capacity 10, got %d", bucket.capacity)
		}
		if bucket.tokens != 10.0 {
			t.Errorf("Expected initial tokens 10, got %f", bucket.tokens)
		}
		if bucket.refillRate != 1.0 {
			t.Errorf("Expected refill rate 1.0, got %f", bucket.refillRate)
		}
	})

	t.Run("Allow", func(t *testing.T) {
		bucket := NewTokenBucket(5, 1.0)

		// Should allow when tokens available
		if !bucket.Allow(3) {
			t.Error("Expected Allow(3) to return true")
		}
		if bucket.tokens != 2.0 {
			t.Errorf("Expected 2 tokens remaining, got %f", bucket.tokens)
		}

		// Should not allow when insufficient tokens
		if bucket.Allow(3) {
			t.Error("Expected Allow(3) to return false")
		}
		// Allow small timing variance
		if bucket.tokens < 1.99 || bucket.tokens > 2.01 {
			t.Errorf("Expected tokens ~2, got %f", bucket.tokens)
		}

		// Should allow exact amount
		if !bucket.Allow(2) {
			t.Error("Expected Allow(2) to return true")
		}
		// Allow small timing variance
		if bucket.tokens < -0.01 || bucket.tokens > 0.01 {
			t.Errorf("Expected ~0 tokens remaining, got %f", bucket.tokens)
		}
	})

	t.Run("Refill", func(t *testing.T) {
		bucket := NewTokenBucket(10, 2.0) // 2 tokens per second
		now := time.Now()

		// Consume all tokens
		bucket.AllowN(10, now)
		if bucket.tokens != 0.0 {
			t.Errorf("Expected 0 tokens, got %f", bucket.tokens)
		}

		// Check refill after 1 second
		future := now.Add(1 * time.Second)
		bucket.AllowN(0, future) // Just trigger refill
		if bucket.tokens < 1.9 || bucket.tokens > 2.1 {
			t.Errorf("Expected ~2 tokens after 1 second, got %f", bucket.tokens)
		}

		// Check capacity limit
		farFuture := now.Add(10 * time.Second)
		bucket.AllowN(0, farFuture)
		if bucket.tokens != 10.0 {
			t.Errorf("Expected tokens capped at capacity 10, got %f", bucket.tokens)
		}
	})

	t.Run("Wait", func(t *testing.T) {
		bucket := NewTokenBucket(5, 10.0) // 10 tokens per second

		// No wait when tokens available
		wait := bucket.Wait(3)
		if wait != 0 {
			t.Errorf("Expected no wait, got %v", wait)
		}
		if bucket.tokens != 2.0 {
			t.Errorf("Expected 2 tokens remaining, got %f", bucket.tokens)
		}

		// Calculate wait time when insufficient tokens
		wait = bucket.Wait(5)
		expectedWait := 300 * time.Millisecond // Need 3 more tokens at 10/sec = 0.3 sec
		if wait < expectedWait-10*time.Millisecond || wait > expectedWait+10*time.Millisecond {
			t.Errorf("Expected wait ~%v, got %v", expectedWait, wait)
		}
		if bucket.tokens != 0.0 {
			t.Errorf("Expected 0 tokens after reservation, got %f", bucket.tokens)
		}
	})

	t.Run("Reset", func(t *testing.T) {
		bucket := NewTokenBucket(10, 1.0)
		bucket.Allow(10) // Consume all

		bucket.Reset()
		if bucket.tokens != 10.0 {
			t.Errorf("Expected full capacity after reset, got %f", bucket.tokens)
		}
	})

	t.Run("Concurrent", func(t *testing.T) {
		bucket := NewTokenBucket(100, 10.0)
		var allowed int32
		done := make(chan bool)

		// Run 10 goroutines trying to consume tokens
		for i := 0; i < 10; i++ {
			go func() {
				localAllowed := 0
				for j := 0; j < 20; j++ {
					if bucket.Allow(1) {
						localAllowed++
					}
					time.Sleep(1 * time.Millisecond)
				}
				atomic.AddInt32(&allowed, int32(localAllowed))
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Should have allowed ~100 requests (initial capacity + some refill)
		// With race detection, timing can vary more
		if allowed < 85 || allowed > 120 {
			t.Errorf("Expected ~100 allowed requests, got %d", allowed)
		}
	})
}
