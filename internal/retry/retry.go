package retry

import (
	"context"
	"crypto/rand"
	"math"
	"math/big"
	"time"
)

// Config defines retry behavior configuration.
type Config struct {
	// MaxAttempts is the maximum number of retry attempts (0 = infinite).
	MaxAttempts int
	// InitialDelay is the initial delay between retries.
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration
	// Multiplier is the exponential backoff multiplier.
	Multiplier float64
	// Jitter adds randomness to retry delays (0-1).
	Jitter float64
}

// DefaultConfig returns a default retry configuration.
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  0, // Infinite retries
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}
}

// Manager handles retry logic with exponential backoff.
type Manager struct {
	config Config
}

// NewManager creates a new retry manager.
func NewManager(config Config) *Manager {
	return &Manager{
		config: config,
	}
}

// Run executes the given function with retry logic.
// It returns nil if the function succeeds, or the last error if all retries fail.
func (m *Manager) Run(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error
	attempt := 0

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try the function
		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err
		attempt++

		// Check if we've exceeded max attempts
		if m.config.MaxAttempts > 0 && attempt >= m.config.MaxAttempts {
			return lastErr
		}

		// Calculate next delay
		delay := m.calculateDelay(attempt)

		// Wait for the delay or context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}
}

// calculateDelay calculates the delay for the given attempt number.
func (m *Manager) calculateDelay(attempt int) time.Duration {
	// Calculate exponential delay
	delay := float64(m.config.InitialDelay) * math.Pow(m.config.Multiplier, float64(attempt-1))

	// Cap at max delay
	if delay > float64(m.config.MaxDelay) {
		delay = float64(m.config.MaxDelay)
	}

	// Add jitter
	if m.config.Jitter > 0 {
		jitter := delay * m.config.Jitter
		// Random value between -jitter and +jitter using crypto/rand
		maxJitter := int64(jitter * 2)
		if maxJitter > 0 {
			n, err := rand.Int(rand.Reader, big.NewInt(maxJitter))
			if err == nil {
				randomJitter := float64(n.Int64()) - jitter
				delay += randomJitter
			}
		}
	}

	// Ensure delay is not negative
	if delay < 0 {
		delay = 0
	}

	return time.Duration(delay)
}

// NextDelay returns the delay that would be used for the given attempt.
// This is useful for testing and logging.
func (m *Manager) NextDelay(attempt int) time.Duration {
	return m.calculateDelay(attempt)
}
