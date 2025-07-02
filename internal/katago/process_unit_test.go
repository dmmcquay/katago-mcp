package katago

import (
	"context"
	"testing"

	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/logging"
)

// TestEnginePingWithMock tests the Ping functionality without starting a real process.
func TestEnginePingWithMock(t *testing.T) {
	cfg := &config.KataGoConfig{
		BinaryPath: "katago",
		NumThreads: 2,
		MaxVisits:  100,
		MaxTime:    1.0,
	}

	logger := logging.NewLoggerAdapter(logging.NewLogger("test: ", "debug"))
	engine := NewEngine(cfg, logger)

	ctx := context.Background()

	// Test ping when engine is not running
	err := engine.Ping(ctx)
	if err == nil {
		t.Error("Expected error when pinging stopped engine")
	}
	if err.Error() != "engine not running" {
		t.Errorf("Expected 'engine not running' error, got: %v", err)
	}

	// Manually set engine as running (simulating a started engine)
	engine.mu.Lock()
	engine.running = true
	// Note: In a real scenario, engine.cmd would be set, but for this test
	// we're just testing the logic when cmd is nil
	engine.mu.Unlock()

	// Test ping when engine is "running" but process is nil
	err = engine.Ping(ctx)
	if err == nil {
		t.Error("Expected error when process is nil")
	}
	if err.Error() != "engine process not found" {
		t.Errorf("Expected 'engine process not found' error, got: %v", err)
	}
}

// TestEngineIsRunning tests the IsRunning method.
func TestEngineIsRunning(t *testing.T) {
	cfg := &config.KataGoConfig{
		BinaryPath: "katago",
	}

	logger := logging.NewLoggerAdapter(logging.NewLogger("test: ", "debug"))
	engine := NewEngine(cfg, logger)

	// Initially should not be running
	if engine.IsRunning() {
		t.Error("New engine should not be running")
	}

	// Manually set as running
	engine.mu.Lock()
	engine.running = true
	engine.mu.Unlock()

	// Now should be running
	if !engine.IsRunning() {
		t.Error("Engine should be running after setting running=true")
	}

	// Set as not running
	engine.mu.Lock()
	engine.running = false
	engine.mu.Unlock()

	// Should not be running
	if engine.IsRunning() {
		t.Error("Engine should not be running after setting running=false")
	}
}
