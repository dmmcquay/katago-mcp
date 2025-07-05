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
	t.Log("Starting TestFindMistakesFullCoverage")
	env := SetupTestEnvironment(t)
	engine := env.CreateTestEngine(t)

	// Use a timeout to prevent hanging in CI
	// Increased to 10 minutes for CPU-only Docker environment
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Test with a minimal 9x9 game to ensure fast execution in CI
	// Just 3 moves is enough to verify all moves are analyzed
	sgf := `(;GM[1]FF[4]CA[UTF-8]SZ[9]KM[6.5]
;B[ee];W[eg];B[ge])`

	// Count moves in the SGF
	parser := katago.NewSGFParser(sgf)
	position, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse test SGF: %v", err)
	}
	expectedMoves := len(position.Moves)
	t.Logf("Test SGF has %d moves", expectedMoves)

	// Review the game with minimal visits for fast e2e tests in Docker
	review, err := engine.ReviewGame(ctx, sgf, &katago.MistakeThresholds{
		Blunder:       0.15,
		Mistake:       0.05,
		Inaccuracy:    0.02,
		MinimumVisits: 1, // Minimal visits - just enough to verify functionality
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

	// The key test is that all moves were analyzed (checked above with TotalMoves)
	// Finding mistakes is secondary - with minimal analysis (1 visit), we may not find any
	
	if len(review.Mistakes) > 0 {
		// If we found mistakes, verify they're not all at move 1 (the original bug)
		allMovesAreOne := true
		for _, mistake := range review.Mistakes {
			if mistake.MoveNumber != 1 {
				allMovesAreOne = false
				break
			}
		}
		if allMovesAreOne {
			t.Error("CRITICAL: All mistakes are at move 1 - the exact bug we're testing for!")
		}
		
		t.Logf("Found %d mistakes across different moves", len(review.Mistakes))
	} else {
		t.Log("No mistakes found - this is OK with minimal analysis (1 visit)")
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
	t.Log("Starting TestFindMistakesMCPFullCoverage")
	env := SetupTestEnvironment(t)

	// Create MCP server and tools handler
	toolsHandler := setupMCPServer(t, env)

	// Same minimal 9x9 test SGF
	sgf := `(;GM[1]FF[4]CA[UTF-8]SZ[9]KM[6.5]
;B[ee];W[eg];B[ge])`

	// Count expected moves
	parser := katago.NewSGFParser(sgf)
	position, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse test SGF: %v", err)
	}
	expectedMoves := len(position.Moves)

	// Call findMistakes through MCP with minimal visits
	args := map[string]interface{}{
		"sgf":       sgf,
		"maxVisits": 1, // Minimal visits - just enough to verify functionality
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

	// The key test is that all moves were analyzed (checked above)
	// Whether mistakes were found is secondary with minimal analysis
	
	// If mistakes were found, verify they're not all from move 1
	if strings.Contains(resultText, "Move 1:") {
		hasNonMove1Mistake := false
		for i := 2; i <= expectedMoves; i++ {
			moveStr := fmt.Sprintf("Move %d:", i)
			if strings.Contains(resultText, moveStr) {
				hasNonMove1Mistake = true
				break
			}
		}
		
		if !hasNonMove1Mistake {
			t.Error("All mistakes appear to be from move 1 only - the bug we're testing for!")
		}
	} else if strings.Contains(resultText, "No significant mistakes found") {
		t.Log("No mistakes found - this is OK with minimal analysis")
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
			MaxVisits:  10,  // Reduced for faster tests
			MaxTime:    5.0, // Reduced timeout
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
