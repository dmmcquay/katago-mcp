package shutdown

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/logging"
)

func TestShutdownManager(t *testing.T) {
	logConfig := &logging.Config{
		Level:   "debug",
		Format:  logging.FormatJSON,
		Service: "test",
		Version: "1.0.0",
	}
	logger, closer := logging.NewLoggerFromConfig(logConfig)
	if closer != nil {
		defer closer.Close()
	}

	t.Run("shutdown functions all called", func(t *testing.T) {
		manager := NewManager(logger)
		var counters [3]atomic.Int32

		for i := 0; i < 3; i++ {
			idx := i
			manager.Register(fmt.Sprintf("component-%d", idx), func(ctx context.Context) error {
				counters[idx].Add(1)
				return nil
			})
		}

		manager.Shutdown(5 * time.Second)
		manager.WaitForShutdown()

		// Verify all functions were called exactly once
		for i := 0; i < 3; i++ {
			if counters[i].Load() != 1 {
				t.Errorf("Expected component-%d to be called once, got %d", i, counters[i].Load())
			}
		}
	})

	t.Run("shutdown with errors", func(t *testing.T) {
		manager := NewManager(logger)
		errExpected := errors.New("shutdown error")

		manager.Register("failing-component", func(ctx context.Context) error {
			return errExpected
		})

		manager.Register("successful-component", func(ctx context.Context) error {
			return nil
		})

		manager.Shutdown(5 * time.Second)
		manager.WaitForShutdown()
	})

	t.Run("shutdown timeout", func(t *testing.T) {
		manager := NewManager(logger)

		manager.Register("slow-component", func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
				return nil
			}
		})

		start := time.Now()
		manager.Shutdown(100 * time.Millisecond)
		manager.WaitForShutdown()
		elapsed := time.Since(start)

		// Should timeout around 100ms, not wait full 2 seconds
		if elapsed > 500*time.Millisecond {
			t.Errorf("Shutdown took too long: %v", elapsed)
		}
	})

	t.Run("concurrent shutdown calls", func(t *testing.T) {
		manager := NewManager(logger)
		var counter atomic.Int32

		manager.Register("component", func(ctx context.Context) error {
			counter.Add(1)
			time.Sleep(100 * time.Millisecond)
			return nil
		})

		// Call shutdown multiple times concurrently
		for i := 0; i < 5; i++ {
			go manager.Shutdown(5 * time.Second)
		}

		manager.WaitForShutdown()

		// Should only execute once
		if counter.Load() != 1 {
			t.Errorf("Expected shutdown function to be called once, got %d", counter.Load())
		}
	})

	t.Run("done channel", func(t *testing.T) {
		manager := NewManager(logger)

		manager.Register("quick-component", func(ctx context.Context) error {
			return nil
		})

		done := manager.Done()

		// Channel should not be closed yet
		select {
		case <-done:
			t.Error("Done channel closed before shutdown")
		default:
			// Expected
		}

		manager.Shutdown(5 * time.Second)

		// Channel should be closed after shutdown
		select {
		case <-done:
			// Expected
		case <-time.After(1 * time.Second):
			t.Error("Done channel not closed after shutdown")
		}
	})
}
