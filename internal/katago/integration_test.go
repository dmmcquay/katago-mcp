//go:build integration
// +build integration

package katago

import (
	"context"
	"testing"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/logging"
)

// TestEngineWithRealKataGo tests the engine with a real KataGo process.
// Run with: go test -tags=integration ./internal/katago
func TestEngineWithRealKataGo(t *testing.T) {
	// Skip if KataGo not available
	detection, err := DetectKataGo()
	if err != nil {
		t.Skip("KataGo not installed, skipping integration tests")
	}

	// Skip if no model or config found
	if detection.ModelPath == "" || detection.ConfigPath == "" {
		t.Skip("KataGo model or config not found, skipping integration tests")
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
	engine := NewEngine(cfg, logger)

	ctx := context.Background()

	// Start engine
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer engine.Stop()

	// Give engine time to initialize
	time.Sleep(1 * time.Second)

	// Test ping
	if err := engine.Ping(ctx); err != nil {
		t.Errorf("Failed to ping engine: %v", err)
	}

	// Test analysis
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

	result, err := engine.Analyze(ctx, req)
	if err != nil {
		t.Fatalf("Failed to analyze position: %v", err)
	}

	if len(result.MoveInfos) == 0 {
		t.Error("Expected at least one move in analysis result")
	}
}
