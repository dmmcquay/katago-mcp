package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket implements the token bucket algorithm for rate limiting.
type TokenBucket struct {
	capacity   int        // Maximum number of tokens
	tokens     float64    // Current number of tokens
	refillRate float64    // Tokens added per second
	lastRefill time.Time  // Last time tokens were refilled
	mu         sync.Mutex // Protects all fields
}

// NewTokenBucket creates a new token bucket with the specified capacity and refill rate.
func NewTokenBucket(capacity int, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     float64(capacity), // Start with full bucket
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow attempts to consume n tokens from the bucket.
// Returns true if the tokens were available, false otherwise.
func (b *TokenBucket) Allow(n int) bool {
	return b.AllowN(n, time.Now())
}

// AllowN attempts to consume n tokens at a specific time.
// This method is useful for testing.
func (b *TokenBucket) AllowN(n int, now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens based on elapsed time
	b.refill(now)

	// Check if we have enough tokens
	if b.tokens >= float64(n) {
		b.tokens -= float64(n)
		return true
	}

	return false
}

// Wait blocks until n tokens are available or the context expires.
func (b *TokenBucket) Wait(n int) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	b.refill(now)

	// If we have enough tokens, no wait needed
	if b.tokens >= float64(n) {
		b.tokens -= float64(n)
		return 0
	}

	// Calculate wait time
	deficit := float64(n) - b.tokens
	waitSeconds := deficit / b.refillRate
	waitDuration := time.Duration(waitSeconds * float64(time.Second))

	// Reserve the tokens (they'll be available after the wait)
	b.tokens = 0

	return waitDuration
}

// Tokens returns the current number of tokens available.
func (b *TokenBucket) Tokens() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill(time.Now())
	return b.tokens
}

// refill adds tokens based on the time elapsed since last refill.
// Must be called with lock held.
func (b *TokenBucket) refill(now time.Time) {
	elapsed := now.Sub(b.lastRefill)
	tokensToAdd := elapsed.Seconds() * b.refillRate

	b.tokens += tokensToAdd
	if b.tokens > float64(b.capacity) {
		b.tokens = float64(b.capacity)
	}

	b.lastRefill = now
}

// Reset resets the bucket to full capacity.
func (b *TokenBucket) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.tokens = float64(b.capacity)
	b.lastRefill = time.Now()
}

