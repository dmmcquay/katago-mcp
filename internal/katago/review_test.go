package katago

import (
	"testing"
)

func TestDefaultMistakeThresholds(t *testing.T) {
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
	if defaults.MinimumVisits != 50 {
		t.Errorf("Expected minimum visits 50, got %d", defaults.MinimumVisits)
	}
}

func TestEstimateLevel(t *testing.T) {
	tests := []struct {
		name    string
		summary ReviewSummary
		want    string
	}{
		{
			name: "professional level",
			summary: ReviewSummary{
				TotalMoves:    100,
				BlackAccuracy: 96,
				WhiteAccuracy: 97,
				BlackBlunders: 0,
				WhiteBlunders: 0,
			},
			want: "Professional",
		},
		{
			name: "strong amateur",
			summary: ReviewSummary{
				TotalMoves:    100,
				BlackAccuracy: 91,
				WhiteAccuracy: 92,
				BlackBlunders: 1,
				WhiteBlunders: 1,
			},
			want: "Strong Amateur (5d+)",
		},
		{
			name: "amateur dan",
			summary: ReviewSummary{
				TotalMoves:    100,
				BlackAccuracy: 86,
				WhiteAccuracy: 87,
				BlackBlunders: 2,
				WhiteBlunders: 2,
			},
			want: "Amateur Dan (1d-4d)",
		},
		{
			name: "strong kyu",
			summary: ReviewSummary{
				TotalMoves:    100,
				BlackAccuracy: 81,
				WhiteAccuracy: 82,
				BlackBlunders: 4,
				WhiteBlunders: 3,
			},
			want: "Strong Kyu (5k-1k)",
		},
		{
			name: "mid kyu",
			summary: ReviewSummary{
				TotalMoves:    100,
				BlackAccuracy: 71,
				WhiteAccuracy: 73,
				BlackBlunders: 6,
				WhiteBlunders: 5,
			},
			want: "Mid Kyu (10k-6k)",
		},
		{
			name: "weak kyu",
			summary: ReviewSummary{
				TotalMoves:    100,
				BlackAccuracy: 61,
				WhiteAccuracy: 62,
				BlackBlunders: 8,
				WhiteBlunders: 7,
			},
			want: "Weak Kyu (15k-11k)",
		},
		{
			name: "beginner",
			summary: ReviewSummary{
				TotalMoves:    100,
				BlackAccuracy: 45,
				WhiteAccuracy: 48,
				BlackBlunders: 15,
				WhiteBlunders: 12,
			},
			want: "Beginner (20k-16k)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateLevel(tt.summary)
			if got != tt.want {
				t.Errorf("estimateLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMistakeStruct(t *testing.T) {
	// Test that Mistake struct can be properly created and contains expected fields
	mistake := Mistake{
		MoveNumber:   42,
		Color:        "B",
		PlayedMove:   "D4",
		BestMove:     "Q16",
		WinrateDrop:  0.15,
		Category:     "blunder",
		Explanation:  "This move loses 15.0% win rate",
		PlayedWR:     0.35,
		BestWR:       0.50,
		PolicyPlayed: 0.02,
		PolicyBest:   0.45,
	}

	if mistake.MoveNumber != 42 {
		t.Errorf("Expected move number 42, got %d", mistake.MoveNumber)
	}
	if mistake.Category != "blunder" {
		t.Errorf("Expected category 'blunder', got %s", mistake.Category)
	}
	if mistake.WinrateDrop != 0.15 {
		t.Errorf("Expected winrate drop 0.15, got %f", mistake.WinrateDrop)
	}
}

func TestGameReviewStruct(t *testing.T) {
	// Test that GameReview struct can be properly created
	review := GameReview{
		Mistakes: []Mistake{
			{
				MoveNumber:  10,
				Color:       "B",
				PlayedMove:  "D4",
				BestMove:    "Q16",
				WinrateDrop: 0.05,
				Category:    "mistake",
			},
		},
		Summary: ReviewSummary{
			TotalMoves:     50,
			BlackMistakes:  1,
			WhiteMistakes:  0,
			BlackBlunders:  0,
			WhiteBlunders:  0,
			BlackAccuracy:  90.0,
			WhiteAccuracy:  95.0,
			EstimatedLevel: "Amateur Dan (1d-4d)",
		},
	}

	if len(review.Mistakes) != 1 {
		t.Errorf("Expected 1 mistake, got %d", len(review.Mistakes))
	}
	if review.Summary.TotalMoves != 50 {
		t.Errorf("Expected 50 total moves, got %d", review.Summary.TotalMoves)
	}
}
