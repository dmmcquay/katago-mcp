//go:build integration
// +build integration

package e2e

import (
	"context"
	"testing"

	"github.com/dmmcquay/katago-mcp/internal/katago"
)

// TestFindMistakesAnalyzesAllMoves ensures that findMistakes analyzes every move in the game
func TestFindMistakesAnalyzesAllMoves(t *testing.T) {
	env := SetupTestEnvironment(t)
	engine := env.CreateTestEngine(t)

	ctx := context.Background()

	// Create a simple 5-move game
	sgf := `(;GM[1]FF[4]CA[UTF-8]SZ[9]KM[5.5]
;B[ee];W[eg];B[ge];W[gg];B[ce])`

	// Review the game with very low thresholds to catch analysis
	thresholds := &katago.MistakeThresholds{
		Blunder:       0.15,
		Mistake:       0.05,
		Inaccuracy:    0.02,
		MinimumVisits: 1, // Very low to ensure we get results quickly
	}

	review, err := engine.ReviewGame(ctx, sgf, thresholds)
	if err != nil {
		t.Fatalf("Failed to review game: %v", err)
	}

	// The game has 5 moves, so we should have analyzed all 5
	if review.Summary.TotalMoves != 5 {
		t.Errorf("Expected to analyze 5 moves, but only analyzed %d", review.Summary.TotalMoves)
	}

	// Black played 3 moves (moves 1, 3, 5)
	// White played 2 moves (moves 2, 4)
	totalAnalyzedMoves := 0
	if review.Summary.BlackAccuracy >= 0 {
		// If we have accuracy, it means we analyzed black's moves
		totalAnalyzedMoves += 3
	}
	if review.Summary.WhiteAccuracy >= 0 {
		// If we have accuracy, it means we analyzed white's moves
		totalAnalyzedMoves += 2
	}

	t.Logf("Review summary: Total moves=%d, Black accuracy=%.1f%%, White accuracy=%.1f%%",
		review.Summary.TotalMoves, review.Summary.BlackAccuracy, review.Summary.WhiteAccuracy)

	// Log any mistakes found
	for _, mistake := range review.Mistakes {
		t.Logf("Found mistake at move %d: %s played %s (best: %s)",
			mistake.MoveNumber, mistake.Color, mistake.PlayedMove, mistake.BestMove)
	}
}
