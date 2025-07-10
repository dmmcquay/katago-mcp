package katago

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/logging"
)

// mockEngine is a mock implementation of EngineInterface for testing.
type mockEngine struct {
	running    atomic.Bool
	startCount atomic.Int32
	stopCount  atomic.Int32
	pingCount  atomic.Int32
	failStart  atomic.Bool
	failPing   atomic.Bool
	startDelay time.Duration
}

func (m *mockEngine) Start(ctx context.Context) error {
	m.startCount.Add(1)
	if m.failStart.Load() {
		return errors.New("start failed")
	}
	if m.startDelay > 0 {
		time.Sleep(m.startDelay)
	}
	m.running.Store(true)
	return nil
}

func (m *mockEngine) Stop() error {
	m.stopCount.Add(1)
	m.running.Store(false)
	return nil
}

func (m *mockEngine) IsRunning() bool {
	return m.running.Load()
}

func (m *mockEngine) Ping(ctx context.Context) error {
	m.pingCount.Add(1)
	if m.failPing.Load() {
		return errors.New("ping failed")
	}
	if !m.running.Load() {
		return errors.New("engine not running")
	}
	return nil
}

func (m *mockEngine) Analyze(ctx context.Context, req *AnalysisRequest) (*AnalysisResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockEngine) AnalyzeSGF(ctx context.Context, sgf string, moveNum int) (*AnalysisResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockEngine) ReviewGame(ctx context.Context, sgf string, thresholds *MistakeThresholds) (*GameReview, error) {
	return nil, errors.New("not implemented")
}

func (m *mockEngine) EstimateTerritory(ctx context.Context, position *Position, threshold float64) (*TerritoryEstimate, error) {
	return nil, errors.New("not implemented")
}

func (m *mockEngine) ExplainMove(ctx context.Context, position *Position, move string) (*MoveExplanation, error) {
	return nil, errors.New("not implemented")
}

func TestSupervisor(t *testing.T) {
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

	t.Run("start and stop", func(t *testing.T) {
		cfg := &config.KataGoConfig{}
		supervisor := NewSupervisor(cfg, logger, nil)

		// Replace engine with mock
		mock := &mockEngine{}
		supervisor.engine = mock

		ctx := context.Background()

		// Start supervisor
		err := supervisor.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start supervisor: %v", err)
		}

		// Wait for engine to start
		time.Sleep(100 * time.Millisecond)

		if mock.startCount.Load() != 1 {
			t.Errorf("Expected 1 start call, got %d", mock.startCount.Load())
		}

		// Stop supervisor
		err = supervisor.Stop()
		if err != nil {
			t.Fatalf("Failed to stop supervisor: %v", err)
		}

		if mock.stopCount.Load() != 1 {
			t.Errorf("Expected 1 stop call, got %d", mock.stopCount.Load())
		}
	})

	t.Run("auto restart on failure", func(t *testing.T) {
		cfg := &config.KataGoConfig{}
		supervisor := NewSupervisor(cfg, logger, nil)
		supervisor.healthCheckInterval = 100 * time.Millisecond

		// Replace engine with mock
		mock := &mockEngine{}
		supervisor.engine = mock

		ctx := context.Background()

		// Start supervisor
		err := supervisor.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start supervisor: %v", err)
		}

		// Wait for initial start
		time.Sleep(50 * time.Millisecond)

		// Simulate engine crash
		mock.running.Store(false)

		// Wait for health check and restart
		time.Sleep(200 * time.Millisecond)

		// Should have restarted
		if mock.startCount.Load() < 2 {
			t.Errorf("Expected at least 2 start calls, got %d", mock.startCount.Load())
		}

		// Stop supervisor
		_ = supervisor.Stop()
	})

	t.Run("manual restart", func(t *testing.T) {
		cfg := &config.KataGoConfig{}
		supervisor := NewSupervisor(cfg, logger, nil)

		// Replace engine with mock
		mock := &mockEngine{}
		supervisor.engine = mock

		ctx := context.Background()

		// Start supervisor
		err := supervisor.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start supervisor: %v", err)
		}

		// Wait for initial start
		time.Sleep(50 * time.Millisecond)

		startsBefore := mock.startCount.Load()
		stopsBefore := mock.stopCount.Load()

		// Trigger manual restart
		supervisor.Restart()

		// Wait for restart
		time.Sleep(100 * time.Millisecond)

		if mock.startCount.Load() <= startsBefore {
			t.Errorf("Expected more starts after restart, before: %d, after: %d",
				startsBefore, mock.startCount.Load())
		}
		if mock.stopCount.Load() <= stopsBefore {
			t.Errorf("Expected more stops after restart, before: %d, after: %d",
				stopsBefore, mock.stopCount.Load())
		}

		// Stop supervisor
		_ = supervisor.Stop()
	})

	t.Run("retry on start failure", func(t *testing.T) {
		cfg := &config.KataGoConfig{}
		supervisor := NewSupervisor(cfg, logger, nil)

		// Replace engine with mock that fails initially
		mock := &mockEngine{}
		mock.failStart.Store(true)
		supervisor.engine = mock

		ctx := context.Background()

		// Start supervisor
		err := supervisor.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start supervisor: %v", err)
		}

		// Wait for initial attempt and first retry
		time.Sleep(1500 * time.Millisecond)

		// Should have tried multiple times
		if mock.startCount.Load() < 2 {
			t.Errorf("Expected at least 2 start attempts, got %d", mock.startCount.Load())
		}

		// Allow start to succeed
		mock.failStart.Store(false)

		// Wait for successful start
		time.Sleep(2 * time.Second)

		if !mock.IsRunning() {
			t.Error("Expected engine to be running after retry")
		}

		// Stop supervisor
		_ = supervisor.Stop()
	})

	t.Run("health check with ping failure", func(t *testing.T) {
		cfg := &config.KataGoConfig{}
		supervisor := NewSupervisor(cfg, logger, nil)
		supervisor.healthCheckInterval = 100 * time.Millisecond

		// Replace engine with mock
		mock := &mockEngine{}
		supervisor.engine = mock

		ctx := context.Background()

		// Start supervisor
		err := supervisor.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start supervisor: %v", err)
		}

		// Wait for initial start
		time.Sleep(50 * time.Millisecond)

		// Make ping fail
		mock.failPing.Store(true)

		// Wait for health check to detect failure and restart
		time.Sleep(200 * time.Millisecond)

		// Should have restarted due to ping failure
		if mock.startCount.Load() < 2 {
			t.Errorf("Expected at least 2 start calls after ping failure, got %d", mock.startCount.Load())
		}

		// Stop supervisor
		_ = supervisor.Stop()
	})
}
