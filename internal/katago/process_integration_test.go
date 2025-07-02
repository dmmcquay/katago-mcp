//go:build integration
// +build integration

package katago

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/logging"
)

func TestEngineLifecycle(t *testing.T) {
	// Skip if KataGo not available
	if _, err := DetectKataGo(); err != nil {
		t.Skip("KataGo not installed, skipping engine tests")
	}

	cfg := &config.KataGoConfig{
		BinaryPath: "katago",
		NumThreads: 2,
		MaxVisits:  100,
		MaxTime:    1.0,
	}

	logger := logging.NewLoggerAdapter(logging.NewLogger("test: ", "debug"))
	engine := NewEngine(cfg, logger, nil)

	ctx := context.Background()

	// Test starting engine
	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	// Check if running
	if !engine.IsRunning() {
		t.Error("Engine should be running after Start()")
	}

	// Give engine time to initialize
	time.Sleep(100 * time.Millisecond)

	// Test stopping engine
	err = engine.Stop()
	if err != nil {
		t.Fatalf("Failed to stop engine: %v", err)
	}

	// Check if stopped
	if engine.IsRunning() {
		t.Error("Engine should not be running after Stop()")
	}
}

func TestEngineAnalysis(t *testing.T) {
	// Skip if KataGo not available
	detection, err := DetectKataGo()
	if err != nil {
		t.Skip("KataGo not installed, skipping engine tests")
	}

	// Skip if no model or config found
	if detection.ModelPath == "" || detection.ConfigPath == "" {
		t.Skip("KataGo model or config not found, skipping analysis tests")
	}

	cfg := &config.KataGoConfig{
		BinaryPath: detection.BinaryPath,
		ModelPath:  detection.ModelPath,
		ConfigPath: detection.ConfigPath,
		NumThreads: 2,
		MaxVisits:  100,
		MaxTime:    5.0, // Longer timeout for CI environment
	}

	logger := logging.NewLoggerAdapter(logging.NewLogger("test: ", "debug"))
	engine := NewEngine(cfg, logger, nil)

	ctx := context.Background()

	// Start engine
	err = engine.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer engine.Stop()

	// Give engine time to initialize
	time.Sleep(200 * time.Millisecond)

	// Create a simple position
	position := &Position{
		Rules:      "chinese",
		BoardXSize: 19,
		BoardYSize: 19,
		Komi:       7.5,
		Moves: []Move{
			{Color: "b", Location: "D4"},
			{Color: "w", Location: "Q16"},
		},
	}

	req := &AnalysisRequest{
		Position: position,
	}

	// Analyze position
	result, err := engine.Analyze(ctx, req)
	if err != nil {
		t.Fatalf("Failed to analyze position: %v", err)
	}

	// Check result
	if len(result.MoveInfos) == 0 {
		t.Error("Expected at least one move in analysis result")
	}

	if result.RootInfo.CurrentPlayer == "" {
		t.Error("Expected current player in root info")
	}

	// Test with SGF
	sgf := `(;GM[1]FF[4]CA[UTF-8]SZ[19]KM[7.5]
		PB[Black]PW[White]
		;B[pd];W[dd];B[pp];W[dp])`

	sgfResult, err := engine.AnalyzeSGF(ctx, sgf, 0)
	if err != nil {
		t.Fatalf("Failed to analyze SGF: %v", err)
	}

	if len(sgfResult.MoveInfos) == 0 {
		t.Error("Expected at least one move in SGF analysis result")
	}
}

func TestEngineContextCancellation(t *testing.T) {
	// Skip if KataGo not available
	if _, err := DetectKataGo(); err != nil {
		t.Skip("KataGo not installed, skipping engine tests")
	}

	cfg := &config.KataGoConfig{
		BinaryPath: "katago",
		NumThreads: 2,
		MaxVisits:  10000, // High visits to ensure analysis takes time
		MaxTime:    10.0,
	}

	logger := logging.NewLoggerAdapter(logging.NewLogger("test: ", "debug"))
	engine := NewEngine(cfg, logger, nil)

	ctx, cancel := context.WithCancel(context.Background())

	// Start engine
	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer engine.Stop()

	// Cancel context quickly
	cancel()

	// Engine should stop soon
	time.Sleep(100 * time.Millisecond)

	// Try to use engine after context cancellation
	position := &Position{
		Rules:      "chinese",
		BoardXSize: 19,
		BoardYSize: 19,
		Komi:       7.5,
	}

	req := &AnalysisRequest{
		Position: position,
	}

	// This should fail or handle gracefully
	_, err = engine.Analyze(context.Background(), req)
	if err == nil {
		// It might still work if the engine hasn't shut down yet
		// but subsequent calls should fail
		time.Sleep(500 * time.Millisecond)
	}
}

func TestEngineWithEnvironment(t *testing.T) {
	// Test that engine respects KATAGO_HOME environment variable
	tempDir := t.TempDir()
	os.Setenv("KATAGO_HOME", tempDir)
	defer os.Unsetenv("KATAGO_HOME")

	cfg := &config.KataGoConfig{
		BinaryPath: "katago",
		NumThreads: 1,
		MaxVisits:  10,
		MaxTime:    0.1,
	}

	logger := logging.NewLoggerAdapter(logging.NewLogger("test: ", "debug"))
	engine := NewEngine(cfg, logger, nil)

	// This should work even if KataGo isn't installed in the temp directory
	// The engine should handle the error gracefully
	ctx := context.Background()
	err := engine.Start(ctx)

	// We expect this to fail since KataGo won't be in the temp directory
	if err == nil {
		engine.Stop()
		t.Log("Engine started successfully even with custom KATAGO_HOME")
	}
}

func TestEnginePing(t *testing.T) {
	// Skip if KataGo not available
	if _, err := DetectKataGo(); err != nil {
		t.Skip("KataGo not installed, skipping engine tests")
	}

	cfg := &config.KataGoConfig{
		BinaryPath: "katago",
		NumThreads: 2,
		MaxVisits:  100,
		MaxTime:    1.0,
	}

	logger := logging.NewLoggerAdapter(logging.NewLogger("test: ", "debug"))
	engine := NewEngine(cfg, logger, nil)

	ctx := context.Background()

	// Test ping when engine is not running
	err := engine.Ping(ctx)
	if err == nil {
		t.Error("Expected error when pinging stopped engine")
	}

	// Start engine
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer engine.Stop()

	// Wait for engine to be ready
	time.Sleep(1 * time.Second)

	// Test ping when engine is running
	err = engine.Ping(ctx)
	if err != nil {
		t.Errorf("Failed to ping running engine: %v", err)
	}

	// Test ping with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err = engine.Ping(timeoutCtx)
	if err != nil {
		t.Errorf("Failed to ping with timeout: %v", err)
	}
}
