package katago

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/cache"
	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/logging"
	"github.com/dmmcquay/katago-mcp/internal/retry"
)

// Supervisor manages the KataGo engine lifecycle with auto-restart capability.
type Supervisor struct {
	engine       EngineInterface
	config       *config.KataGoConfig
	logger       logging.ContextLogger
	retryManager *retry.Manager

	mu                  sync.RWMutex
	running             bool
	stopCh              chan struct{}
	restartCh           chan struct{}
	healthCheckInterval time.Duration
}

// NewSupervisor creates a new KataGo supervisor.
func NewSupervisor(cfg *config.KataGoConfig, logger logging.ContextLogger, cacheManager *cache.Manager) *Supervisor {
	retryConfig := retry.Config{
		MaxAttempts:  0, // Infinite retries
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}

	return &Supervisor{
		engine:              NewEngine(cfg, logger, cacheManager),
		config:              cfg,
		logger:              logger,
		retryManager:        retry.NewManager(retryConfig),
		stopCh:              make(chan struct{}),
		restartCh:           make(chan struct{}, 1),
		healthCheckInterval: 30 * time.Second,
	}
}

// Start starts the supervisor and the KataGo engine.
func (s *Supervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("supervisor already running")
	}

	s.running = true
	go s.supervise(ctx)

	return nil
}

// Stop stops the supervisor and the KataGo engine.
func (s *Supervisor) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)

	// Stop the engine
	return s.engine.Stop()
}

// GetEngine returns the underlying KataGo engine.
func (s *Supervisor) GetEngine() EngineInterface {
	return s.engine
}

// Restart triggers a manual restart of the KataGo engine.
func (s *Supervisor) Restart() {
	select {
	case s.restartCh <- struct{}{}:
		s.logger.Info("Manual restart requested")
	default:
		// Channel is full, restart already pending
	}
}

// supervise monitors the KataGo engine and restarts it if needed.
func (s *Supervisor) supervise(ctx context.Context) {
	s.logger.Info("Starting KataGo supervisor")

	// Start the engine initially
	s.startEngineWithRetry(ctx)

	// Health check ticker
	healthTicker := time.NewTicker(s.healthCheckInterval)
	defer healthTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Supervisor context cancelled")
			return

		case <-s.stopCh:
			s.logger.Info("Supervisor stopped")
			return

		case <-s.restartCh:
			s.logger.Info("Processing restart request")
			if err := s.engine.Stop(); err != nil {
				s.logger.Error("Failed to stop engine for restart", "error", err)
			}
			s.startEngineWithRetry(ctx)

		case <-healthTicker.C:
			// Check if engine is healthy
			if !s.engine.IsRunning() {
				s.logger.Warn("KataGo engine not running, restarting")
				s.startEngineWithRetry(ctx)
			} else {
				// Ping to check responsiveness
				pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				err := s.engine.Ping(pingCtx)
				cancel()

				if err != nil {
					s.logger.Error("KataGo engine health check failed", "error", err)
					if err := s.engine.Stop(); err != nil {
						s.logger.Error("Failed to stop unhealthy engine", "error", err)
					}
					s.startEngineWithRetry(ctx)
				}
			}
		}
	}
}

// startEngineWithRetry starts the engine with exponential backoff retry.
func (s *Supervisor) startEngineWithRetry(ctx context.Context) {
	err := s.retryManager.Run(ctx, func(retryCtx context.Context) error {
		// Check if we should stop
		select {
		case <-s.stopCh:
			return fmt.Errorf("supervisor stopped")
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		s.logger.Info("Starting KataGo engine")

		// Start the engine
		if err := s.engine.Start(retryCtx); err != nil {
			s.logger.Error("Failed to start KataGo engine", "error", err)
			return err
		}

		// Verify it's responsive
		pingCtx, cancel := context.WithTimeout(retryCtx, 10*time.Second)
		defer cancel()

		if err := s.engine.Ping(pingCtx); err != nil {
			s.logger.Error("KataGo engine not responsive after start", "error", err)
			// Stop the engine before retrying
			_ = s.engine.Stop()
			return err
		}

		s.logger.Info("KataGo engine started successfully")
		return nil
	})

	if err != nil {
		s.logger.Error("Failed to start KataGo engine after retries", "error", err)
	}
}
