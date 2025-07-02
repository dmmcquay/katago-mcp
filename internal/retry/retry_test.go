package retry

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryManager(t *testing.T) {
	t.Run("successful on first attempt", func(t *testing.T) {
		config := Config{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
			Jitter:       0,
		}
		manager := NewManager(config)

		var attempts atomic.Int32
		err := manager.Run(context.Background(), func(ctx context.Context) error {
			attempts.Add(1)
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts.Load() != 1 {
			t.Errorf("Expected 1 attempt, got %d", attempts.Load())
		}
	})

	t.Run("retries on failure", func(t *testing.T) {
		config := Config{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
			Jitter:       0,
		}
		manager := NewManager(config)

		var attempts atomic.Int32
		expectedErr := errors.New("test error")

		start := time.Now()
		err := manager.Run(context.Background(), func(ctx context.Context) error {
			attempts.Add(1)
			return expectedErr
		})
		elapsed := time.Since(start)

		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
		if attempts.Load() != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts.Load())
		}
		// Should have delays: 10ms + 20ms = 30ms minimum
		if elapsed < 30*time.Millisecond {
			t.Errorf("Expected at least 30ms elapsed, got %v", elapsed)
		}
	})

	t.Run("succeeds after retries", func(t *testing.T) {
		config := Config{
			MaxAttempts:  5,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
			Jitter:       0,
		}
		manager := NewManager(config)

		var attempts atomic.Int32
		err := manager.Run(context.Background(), func(ctx context.Context) error {
			count := attempts.Add(1)
			if count < 3 {
				return errors.New("temporary error")
			}
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts.Load() != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts.Load())
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		config := Config{
			MaxAttempts:  0, // Infinite
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
			Jitter:       0,
		}
		manager := NewManager(config)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		var attempts atomic.Int32
		start := time.Now()
		err := manager.Run(ctx, func(ctx context.Context) error {
			attempts.Add(1)
			return errors.New("always fails")
		})
		elapsed := time.Since(start)

		if err != context.DeadlineExceeded {
			t.Errorf("Expected context deadline exceeded, got %v", err)
		}
		// Should have made 1 attempt before timeout
		if attempts.Load() != 1 {
			t.Errorf("Expected 1 attempt, got %d", attempts.Load())
		}
		// Should timeout around 50ms
		if elapsed > 100*time.Millisecond {
			t.Errorf("Took too long to cancel: %v", elapsed)
		}
	})

	t.Run("exponential backoff", func(t *testing.T) {
		config := Config{
			MaxAttempts:  4,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
			Jitter:       0,
		}
		manager := NewManager(config)

		// Test delay calculation
		delays := []time.Duration{
			manager.NextDelay(1), // 10ms
			manager.NextDelay(2), // 20ms
			manager.NextDelay(3), // 40ms
			manager.NextDelay(4), // 80ms
			manager.NextDelay(5), // 100ms (capped)
		}

		expected := []time.Duration{
			10 * time.Millisecond,
			20 * time.Millisecond,
			40 * time.Millisecond,
			80 * time.Millisecond,
			100 * time.Millisecond, // Max delay
		}

		for i, delay := range delays {
			if delay != expected[i] {
				t.Errorf("Attempt %d: expected delay %v, got %v", i+1, expected[i], delay)
			}
		}
	})

	t.Run("infinite retries", func(t *testing.T) {
		config := Config{
			MaxAttempts:  0, // Infinite
			InitialDelay: 5 * time.Millisecond,
			MaxDelay:     10 * time.Millisecond,
			Multiplier:   2.0,
			Jitter:       0,
		}
		manager := NewManager(config)

		var attempts atomic.Int32
		ctx, cancel := context.WithCancel(context.Background())

		// Cancel after some attempts
		go func() {
			for attempts.Load() < 5 {
				time.Sleep(5 * time.Millisecond)
			}
			cancel()
		}()

		err := manager.Run(ctx, func(ctx context.Context) error {
			attempts.Add(1)
			return errors.New("always fails")
		})

		if err != context.Canceled {
			t.Errorf("Expected context canceled, got %v", err)
		}
		// Should have made at least 5 attempts
		if attempts.Load() < 5 {
			t.Errorf("Expected at least 5 attempts, got %d", attempts.Load())
		}
	})

	t.Run("jitter", func(t *testing.T) {
		config := Config{
			MaxAttempts:  1,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
			Jitter:       0.5,
		}
		manager := NewManager(config)

		// Get multiple delay calculations
		delays := make([]time.Duration, 10)
		for i := range delays {
			delays[i] = manager.NextDelay(1)
		}

		// Check that delays vary due to jitter
		allSame := true
		for i := 1; i < len(delays); i++ {
			if delays[i] != delays[0] {
				allSame = false
				break
			}
		}

		if allSame {
			t.Error("Expected delays to vary with jitter, but all were the same")
		}

		// Check delays are within expected range (50ms to 150ms with 0.5 jitter)
		for i, delay := range delays {
			if delay < 50*time.Millisecond || delay > 150*time.Millisecond {
				t.Errorf("Delay %d out of expected range: %v", i, delay)
			}
		}
	})
}
