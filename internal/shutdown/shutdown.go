package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/logging"
)

// Manager coordinates graceful shutdown of multiple components.
type Manager struct {
	logger        logging.ContextLogger
	shutdownFuncs []func(context.Context) error
	mu            sync.Mutex
	done          chan struct{}
	shutdownOnce  sync.Once
}

// NewManager creates a new shutdown manager.
func NewManager(logger logging.ContextLogger) *Manager {
	return &Manager{
		logger: logger,
		done:   make(chan struct{}),
	}
}

// Register adds a shutdown function to be called during graceful shutdown.
// Functions are called in reverse order of registration (LIFO).
func (m *Manager) Register(name string, fn func(context.Context) error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	wrappedFn := func(ctx context.Context) error {
		m.logger.Info("Shutting down component", "component", name)
		start := time.Now()
		err := fn(ctx)
		elapsed := time.Since(start)
		if err != nil {
			m.logger.Error("Failed to shutdown component",
				"component", name,
				"error", err,
				"elapsed", elapsed)
		} else {
			m.logger.Info("Component shutdown complete",
				"component", name,
				"elapsed", elapsed)
		}
		return err
	}

	m.shutdownFuncs = append([]func(context.Context) error{wrappedFn}, m.shutdownFuncs...)
}

// HandleSignals sets up signal handling for graceful shutdown.
// It listens for SIGINT and SIGTERM.
func (m *Manager) HandleSignals() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		m.logger.Info("Received shutdown signal", "signal", sig)
		m.Shutdown(30 * time.Second)
	}()
}

// Shutdown performs graceful shutdown with the given timeout.
func (m *Manager) Shutdown(timeout time.Duration) {
	m.shutdownOnce.Do(func() {
		m.logger.Info("Starting graceful shutdown", "timeout", timeout)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Run all shutdown functions
		var wg sync.WaitGroup
		errors := make([]error, 0)
		var errorsMu sync.Mutex

		for _, fn := range m.shutdownFuncs {
			wg.Add(1)
			shutdownFn := fn
			go func() {
				defer wg.Done()
				if err := shutdownFn(ctx); err != nil {
					errorsMu.Lock()
					errors = append(errors, err)
					errorsMu.Unlock()
				}
			}()
		}

		// Wait for all shutdown functions to complete or timeout
		shutdownDone := make(chan struct{})
		go func() {
			wg.Wait()
			close(shutdownDone)
		}()

		select {
		case <-shutdownDone:
			if len(errors) > 0 {
				m.logger.Error("Graceful shutdown completed with errors",
					"errors", len(errors))
			} else {
				m.logger.Info("Graceful shutdown completed successfully")
			}
		case <-ctx.Done():
			m.logger.Error("Graceful shutdown timed out",
				"timeout", timeout)
		}

		close(m.done)
	})
}

// Done returns a channel that's closed when shutdown is complete.
func (m *Manager) Done() <-chan struct{} {
	return m.done
}

// WaitForShutdown blocks until shutdown is complete.
func (m *Manager) WaitForShutdown() {
	<-m.done
}
