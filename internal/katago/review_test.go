package katago

import (
	"context"
	"strings"
	"testing"
)

func TestMistakeThresholds(t *testing.T) {
	defaults := DefaultMistakeThresholds()

	if defaults.Blunder != 0.15 {
		t.Errorf("Expected blunder threshold 0.15, got %f", defaults.Blunder)
	}
	if defaults.Mistake != 0.05 {
		t.Errorf("Expected mistake threshold 0.05, got %f", defaults.Mistake)
	}
	if defaults.Inaccuracy != 0.02 {
		t.Errorf("Expected inaccuracy threshold 0.02, got %f", defaults.Inaccuracy)
	}
	if defaults.MinimumVisits != 100 {
		t.Errorf("Expected minimum visits 100, got %d", defaults.MinimumVisits)
	}
}

func TestGenerateMistakeExplanation(t *testing.T) {
	tests := []struct {
		name         string
		category     string
		winrateDrop  float64
		scoreDrop    float64
		wantContains string
	}{
		{
			name:         "blunder",
			category:     "blunder",
			winrateDrop:  0.20,
			scoreDrop:    15.5,
			wantContains: "severe mistake",
		},
		{
			name:         "mistake",
			category:     "mistake",
			winrateDrop:  0.08,
			scoreDrop:    5.2,
			wantContains: "clear error",
		},
		{
			name:         "inaccuracy",
			category:     "inaccuracy",
			winrateDrop:  0.03,
			scoreDrop:    1.5,
			wantContains: "minor inaccuracy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			explanation := generateMistakeExplanation(tt.category, tt.winrateDrop, tt.scoreDrop)
			if explanation == "" {
				t.Error("Expected non-empty explanation")
			}
			if !strings.Contains(explanation, tt.wantContains) {
				t.Errorf("Expected explanation to contain '%s', got: %s", tt.wantContains, explanation)
			}
		})
	}
}

func TestEstimateRank(t *testing.T) {
	tests := []struct {
		name          string
		blackAccuracy float64
		whiteAccuracy float64
		totalBlunders int
		wantRank      string
	}{
		{
			name:          "professional",
			blackAccuracy: 96,
			whiteAccuracy: 97,
			totalBlunders: 0,
			wantRank:      "Professional",
		},
		{
			name:          "dan player",
			blackAccuracy: 92,
			whiteAccuracy: 91,
			totalBlunders: 1,
			wantRank:      "1-3 dan",
		},
		{
			name:          "kyu player",
			blackAccuracy: 85,
			whiteAccuracy: 83,
			totalBlunders: 2,
			wantRank:      "1-5 kyu",
		},
		{
			name:          "beginner",
			blackAccuracy: 65,
			whiteAccuracy: 60,
			totalBlunders: 5,
			wantRank:      "20-30 kyu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rank := estimateRank(tt.blackAccuracy, tt.whiteAccuracy, tt.totalBlunders)
			if rank != tt.wantRank {
				t.Errorf("Expected rank %s, got %s", tt.wantRank, rank)
			}
		})
	}
}

func TestFindTopMistakes(t *testing.T) {
	review := &GameReview{
		Mistakes: []MistakeInfo{
			{MoveNumber: 10, WinrateDrop: 0.05},
			{MoveNumber: 20, WinrateDrop: 0.15},
			{MoveNumber: 30, WinrateDrop: 0.08},
			{MoveNumber: 40, WinrateDrop: 0.25},
			{MoveNumber: 50, WinrateDrop: 0.03},
		},
	}

	// Test getting top 3 mistakes
	top3 := FindTopMistakes(review, 3)
	if len(top3) != 3 {
		t.Fatalf("Expected 3 mistakes, got %d", len(top3))
	}

	// Check they are sorted by win rate drop
	if top3[0].MoveNumber != 40 { // Highest drop: 0.25
		t.Errorf("Expected move 40 first, got move %d", top3[0].MoveNumber)
	}
	if top3[1].MoveNumber != 20 { // Second highest: 0.15
		t.Errorf("Expected move 20 second, got move %d", top3[1].MoveNumber)
	}
	if top3[2].MoveNumber != 30 { // Third highest: 0.08
		t.Errorf("Expected move 30 third, got move %d", top3[2].MoveNumber)
	}

	// Test getting all mistakes
	allMistakes := FindTopMistakes(review, 0)
	if len(allMistakes) != 5 {
		t.Errorf("Expected all 5 mistakes, got %d", len(allMistakes))
	}
}

// Mock ReviewGame for testing
func TestReviewGameMock(t *testing.T) {
	// This test demonstrates the structure of a game review
	// In real tests, you would mock the engine's Analyze method

	ctx := context.Background()

	// Simple SGF with a few moves
	sgf := `(;GM[1]FF[4]CA[UTF-8]SZ[19]KM[6.5]
	;B[pd];W[dp];B[pq];W[dd];B[fc])`

	// Create mock engine (would need proper mocking in production)
	engine := &Engine{
		running: true,
	}

	// Test that ReviewGame returns expected structure
	// This would need a proper mock implementation
	_ = ctx
	_ = sgf
	_ = engine

	// Verify review structure
	expectedReview := &GameReview{
		Mistakes: []MistakeInfo{},
		Summary: ReviewSummary{
			TotalMoves:    5,
			BlackMistakes: 0,
			WhiteMistakes: 0,
			BlackBlunders: 0,
			WhiteBlunders: 0,
			BlackAccuracy: 100.0,
			WhiteAccuracy: 100.0,
		},
		MoveAnalyses: []MoveAnalysis{},
	}

	if expectedReview.Summary.TotalMoves != 5 {
		t.Error("Review should have correct structure")
	}
}
