//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/katago"
	mcpInternal "github.com/dmmcquay/katago-mcp/internal/mcp"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestFindMistakesFullCoverage tests that findMistakes analyzes ALL moves in a game,
// not just the first move. This test would have caught the bug where only move 1 was analyzed.
func TestFindMistakesFullCoverage(t *testing.T) {
	env := SetupTestEnvironment(t)
	engine := env.CreateTestEngine(t)

	// Use a timeout to prevent hanging in CI
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Test with a shorter game that has mistakes at different points
	// Reduced from 31 moves to 15 for faster e2e tests
	sgf := `(;GM[1]FF[4]CA[UTF-8]SZ[19]KM[6.5]
;B[pd];W[dp];B[pp];W[dd];B[fc];W[cf];B[jd];W[qj]
;B[aa]C[Move 9: Black plays useless move in corner - clear mistake]
;W[qm];B[bb]C[Move 11: Another bad move in corner]
;W[nq];B[pq];W[np];B[tt]C[Move 15: Black passes - mistake])`

	// Count moves in the SGF
	parser := katago.NewSGFParser(sgf)
	position, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse test SGF: %v", err)
	}
	expectedMoves := len(position.Moves)
	t.Logf("Test SGF has %d moves", expectedMoves)

	// Review the game with reduced visits for faster e2e tests
	review, err := engine.ReviewGame(ctx, sgf, &katago.MistakeThresholds{
		Blunder:       0.15,
		Mistake:       0.05,
		Inaccuracy:    0.02,
		MinimumVisits: 10, // Reduced from 50 for faster CPU-only tests
	})
	if err != nil {
		t.Fatalf("Failed to review game: %v", err)
	}

	// CRITICAL CHECK: Verify TotalMoves matches actual move count
	if review.Summary.TotalMoves != expectedMoves {
		t.Errorf("CRITICAL: TotalMoves = %d, but SGF has %d moves. Only first move analyzed?",
			review.Summary.TotalMoves, expectedMoves)
	}

	// Verify we found mistakes at different points in the game
	mistakePositions := make(map[int]bool)
	for _, mistake := range review.Mistakes {
		mistakePositions[mistake.MoveNumber] = true
		t.Logf("Found mistake at move %d: %s played %s (best: %s, category: %s)",
			mistake.MoveNumber, mistake.Color, mistake.PlayedMove,
			mistake.BestMove, mistake.Category)
	}

	// Check that we found mistakes from different parts of the game
	foundEarlyMistake := false // Moves 1-10
	foundMidMistake := false   // Moves 11-20
	foundLateMistake := false  // Moves 21+

	for moveNum := range mistakePositions {
		if moveNum <= 10 {
			foundEarlyMistake = true
		} else if moveNum <= 20 {
			foundMidMistake = true
		} else {
			foundLateMistake = true
		}
	}

	// We expect mistakes in at least the early section given our test SGF
	if !foundEarlyMistake {
		t.Error("No mistakes found in moves 1-10, but we placed A19/B18 corner mistakes there")
	}
	// Log which sections had mistakes
	t.Logf("Mistakes found - Early (1-10): %v, Mid (11-20): %v, Late (21+): %v",
		foundEarlyMistake, foundMidMistake, foundLateMistake)

	// Additional checks
	if len(review.Mistakes) == 0 {
		t.Error("No mistakes found at all - analysis may not be working")
	}

	// Verify move numbers are not all 1 (the original bug)
	allMovesAreOne := true
	for _, mistake := range review.Mistakes {
		if mistake.MoveNumber != 1 {
			allMovesAreOne = false
			break
		}
	}
	if allMovesAreOne && len(review.Mistakes) > 0 {
		t.Error("CRITICAL: All mistakes are at move 1 - the exact bug we're testing for!")
	}

	t.Logf("Game review summary: %d total moves, %d mistakes found",
		review.Summary.TotalMoves, len(review.Mistakes))
	t.Logf("Black: %d mistakes, %.1f%% accuracy",
		review.Summary.BlackMistakes, review.Summary.BlackAccuracy)
	t.Logf("White: %d mistakes, %.1f%% accuracy",
		review.Summary.WhiteMistakes, review.Summary.WhiteAccuracy)
}

// TestFindMistakesMCPFullCoverage tests the same thing through the MCP interface
func TestFindMistakesMCPFullCoverage(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create MCP server and tools handler
	toolsHandler := setupMCPServer(t, env)

	// Same test SGF as above - shortened for faster tests
	sgf := `(;GM[1]FF[4]CA[UTF-8]SZ[19]KM[6.5]
;B[pd];W[dp];B[pp];W[dd];B[fc];W[cf];B[jd];W[qj]
;B[aa]C[Move 9: Black plays useless move in corner - clear mistake]
;W[qm];B[bb]C[Move 11: Another bad move in corner]
;W[nq];B[pq];W[np];B[tt]C[Move 15: Black passes - mistake])`

	// Count expected moves
	parser := katago.NewSGFParser(sgf)
	position, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse test SGF: %v", err)
	}
	expectedMoves := len(position.Moves)

	// Call findMistakes through MCP with reduced visits for faster tests
	args := map[string]interface{}{
		"sgf":       sgf,
		"maxVisits": 10, // Reduced from 50 for faster CPU-only tests
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "findMistakes",
			Arguments: args,
		},
	}

	ctx := context.Background()
	result, err := toolsHandler.HandleFindMistakes(ctx, request)
	if err != nil {
		t.Fatalf("Tool call failed: %v", err)
	}

	// Extract text from result
	var resultText string
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			resultText = textContent.Text
		}
	}

	if resultText == "" {
		t.Fatal("No result text returned")
	}

	// Parse the result to check total moves reported
	// Look for "Total moves: X" in the output
	totalMovesFound := false
	expectedMovesStr := fmt.Sprintf("%d", expectedMoves)
	if strings.Contains(resultText, "Total moves: "+expectedMovesStr) {
		totalMovesFound = true
	}

	if !totalMovesFound {
		// Check if it incorrectly reports "Total moves: 1"
		if strings.Contains(resultText, "Total moves: 1") {
			t.Error("CRITICAL: findMistakes reports 'Total moves: 1' - the exact bug we're testing for!")
		} else {
			t.Errorf("Expected 'Total moves: %d' in output, got: %s", expectedMoves, resultText)
		}
	}

	// Verify mistakes are found at different move numbers
	hasNonMove1Mistake := false
	// Simple check: look for "Move X:" where X > 1
	for i := 2; i <= expectedMoves; i++ {
		moveStr := fmt.Sprintf("Move %d:", i)
		if strings.Contains(resultText, moveStr) {
			hasNonMove1Mistake = true
			break
		}
	}

	if !hasNonMove1Mistake && strings.Contains(resultText, "Move 1:") {
		t.Error("All mistakes appear to be from move 1 only")
	}

	t.Logf("MCP findMistakes result length: %d characters", len(resultText))
}

// setupMCPServer is a helper to create the MCP server for testing
func setupMCPServer(t *testing.T, env *TestEnvironment) *mcpInternal.ToolsHandler {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Name:    "test-katago-mcp",
			Version: "test",
		},
		KataGo: config.KataGoConfig{
			BinaryPath: env.BinaryPath,
			ModelPath:  env.ModelPath,
			ConfigPath: env.ConfigPath,
			NumThreads: 1,
			MaxVisits:  100,
			MaxTime:    10.0,
		},
	}

	engine := katago.NewEngine(&cfg.KataGo, env.Logger, nil)

	// Start engine
	ctx := context.Background()
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	t.Cleanup(func() {
		engine.Stop()
	})

	// Create tools handler
	return mcpInternal.NewToolsHandler(engine, env.Logger)
}
