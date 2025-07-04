package katago

import (
	"context"
	"fmt"
	"strings"
)

// MistakeThresholds defines thresholds for categorizing mistakes.
type MistakeThresholds struct {
	Blunder       float64 // Win rate drop >= this is a blunder (default: 0.15)
	Mistake       float64 // Win rate drop >= this is a mistake (default: 0.05)
	Inaccuracy    float64 // Win rate drop >= this is an inaccuracy (default: 0.02)
	MinimumVisits int     // Minimum visits for reliable analysis
}

// DefaultMistakeThresholds returns default thresholds.
func DefaultMistakeThresholds() *MistakeThresholds {
	return &MistakeThresholds{
		Blunder:       0.15,
		Mistake:       0.05,
		Inaccuracy:    0.02,
		MinimumVisits: 50,
	}
}

// Mistake represents a suboptimal move in a game.
type Mistake struct {
	MoveNumber   int     `json:"moveNumber"`
	Color        string  `json:"color"`
	PlayedMove   string  `json:"playedMove"`
	BestMove     string  `json:"bestMove"`
	WinrateDrop  float64 `json:"winrateDrop"`
	Category     string  `json:"category"` // "blunder", "mistake", "inaccuracy"
	Explanation  string  `json:"explanation"`
	PlayedWR     float64 `json:"playedWinrate"`
	BestWR       float64 `json:"bestWinrate"`
	PolicyPlayed float64 `json:"policyPlayed,omitempty"`
	PolicyBest   float64 `json:"policyBest,omitempty"`
}

// GameReview contains the analysis of an entire game.
type GameReview struct {
	Mistakes []Mistake     `json:"mistakes"`
	Summary  ReviewSummary `json:"summary"`
}

// ReviewSummary provides overall game statistics.
type ReviewSummary struct {
	TotalMoves     int     `json:"totalMoves"`
	BlackMistakes  int     `json:"blackMistakes"`
	WhiteMistakes  int     `json:"whiteMistakes"`
	BlackBlunders  int     `json:"blackBlunders"`
	WhiteBlunders  int     `json:"whiteBlunders"`
	BlackAccuracy  float64 `json:"blackAccuracy"` // Percentage of good moves
	WhiteAccuracy  float64 `json:"whiteAccuracy"`
	EstimatedLevel string  `json:"estimatedLevel,omitempty"`
}

// ReviewGame analyzes a complete game to find mistakes.
func (e *Engine) ReviewGame(ctx context.Context, sgf string, thresholds *MistakeThresholds) (*GameReview, error) {
	if thresholds == nil {
		thresholds = DefaultMistakeThresholds()
	}

	// Parse the game
	parser := NewSGFParser(sgf)
	fullGame, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse SGF: %w", err)
	}

	e.logger.Info("Parsed SGF game", "totalMoves", len(fullGame.Moves), "boardSize", fullGame.BoardXSize)

	review := &GameReview{
		Mistakes: []Mistake{},
	}

	// Track statistics
	blackMoves, whiteMoves := 0, 0
	blackGoodMoves, whiteGoodMoves := 0, 0

	// Analyze each position after each move
	for i := 1; i <= len(fullGame.Moves); i++ {
		// Log progress every 50 moves
		if i%50 == 0 {
			e.logger.Info("Analyzing game progress", "moveNumber", i, "totalMoves", len(fullGame.Moves))
		}

		// Create position before the move at index i-1
		position := &Position{
			Rules:         fullGame.Rules,
			BoardXSize:    fullGame.BoardXSize,
			BoardYSize:    fullGame.BoardYSize,
			Moves:         fullGame.Moves[:i-1], // Position before move i
			InitialStones: fullGame.InitialStones,
		}

		// The move we're evaluating
		currentMove := fullGame.Moves[i-1]
		color := strings.ToUpper(currentMove.Color)

		// Track move counts
		if color == "B" {
			blackMoves++
		} else {
			whiteMoves++
		}

		// Analyze position
		req := &AnalysisRequest{
			Position:         position,
			IncludePolicy:    true,
			IncludeOwnership: false,
		}
		if thresholds.MinimumVisits > 0 {
			visits := thresholds.MinimumVisits
			req.MaxVisits = &visits
		}

		result, err := e.Analyze(ctx, req)
		if err != nil {
			e.logger.Error("Failed to analyze position at move %d: %v", i+1, err)
			continue
		}

		// Skip if not enough visits
		if result.RootInfo.Visits < thresholds.MinimumVisits {
			continue
		}

		// Get the actual played move
		playedMove := currentMove.Location

		// Find the played move in analysis
		var playedInfo *MoveInfo
		for _, mi := range result.MoveInfos {
			if mi.Move == playedMove {
				playedInfo = &mi
				break
			}
		}

		// If we didn't find the played move, it might be a pass or very bad
		if playedInfo == nil && playedMove != "" {
			// Estimate a low winrate for unanalyzed moves
			playedInfo = &MoveInfo{
				Move:    playedMove,
				Winrate: result.RootInfo.Winrate * 0.8, // Rough estimate
			}
		}

		// Get best move
		if len(result.MoveInfos) == 0 {
			continue
		}
		bestMove := result.MoveInfos[0]

		// Calculate winrate drop
		var winrateDrop float64
		if playedInfo != nil {
			winrateDrop = bestMove.Winrate - playedInfo.Winrate
		} else if playedMove == "" {
			// Pass move when better moves exist
			winrateDrop = bestMove.Winrate - result.RootInfo.Winrate
		}

		// Categorize mistake
		switch {
		case winrateDrop >= thresholds.Blunder:
			mistake := Mistake{
				MoveNumber:  i,
				Color:       color,
				PlayedMove:  playedMove,
				BestMove:    bestMove.Move,
				WinrateDrop: winrateDrop,
				Category:    "blunder",
				Explanation: fmt.Sprintf("This move loses %.1f%% win rate", winrateDrop*100),
			}
			if playedInfo != nil {
				mistake.PlayedWR = playedInfo.Winrate
				mistake.PolicyPlayed = playedInfo.Prior
			}
			mistake.BestWR = bestMove.Winrate
			mistake.PolicyBest = bestMove.Prior

			review.Mistakes = append(review.Mistakes, mistake)
			if color == "B" {
				review.Summary.BlackBlunders++
			} else {
				review.Summary.WhiteBlunders++
			}
		case winrateDrop >= thresholds.Mistake:
			mistake := Mistake{
				MoveNumber:  i,
				Color:       color,
				PlayedMove:  playedMove,
				BestMove:    bestMove.Move,
				WinrateDrop: winrateDrop,
				Category:    "mistake",
				Explanation: fmt.Sprintf("This move loses %.1f%% win rate", winrateDrop*100),
			}
			if playedInfo != nil {
				mistake.PlayedWR = playedInfo.Winrate
				mistake.PolicyPlayed = playedInfo.Prior
			}
			mistake.BestWR = bestMove.Winrate
			mistake.PolicyBest = bestMove.Prior

			review.Mistakes = append(review.Mistakes, mistake)
			if color == "B" {
				review.Summary.BlackMistakes++
			} else {
				review.Summary.WhiteMistakes++
			}
		case winrateDrop >= thresholds.Inaccuracy:
			// Track inaccuracies but don't add to main mistakes
			// Could add to a separate list if needed
		default:
			// Good move
			if color == "B" {
				blackGoodMoves++
			} else {
				whiteGoodMoves++
			}
		}
	}

	// Calculate summary statistics
	review.Summary.TotalMoves = len(fullGame.Moves)
	if blackMoves > 0 {
		review.Summary.BlackAccuracy = float64(blackGoodMoves) / float64(blackMoves) * 100
	}
	if whiteMoves > 0 {
		review.Summary.WhiteAccuracy = float64(whiteGoodMoves) / float64(whiteMoves) * 100
	}

	// Estimate playing level based on accuracy and mistakes
	review.Summary.EstimatedLevel = estimateLevel(review.Summary)

	return review, nil
}

// estimateLevel provides a rough estimate of playing strength.
func estimateLevel(summary ReviewSummary) string {
	avgAccuracy := (summary.BlackAccuracy + summary.WhiteAccuracy) / 2
	blunderRate := float64(summary.BlackBlunders+summary.WhiteBlunders) / float64(summary.TotalMoves)

	switch {
	case avgAccuracy > 95 && blunderRate < 0.01:
		return "Professional"
	case avgAccuracy > 90 && blunderRate < 0.025:
		return "Strong Amateur (5d+)"
	case avgAccuracy > 85 && blunderRate < 0.045:
		return "Amateur Dan (1d-4d)"
	case avgAccuracy > 80 && blunderRate < 0.075:
		return "Strong Kyu (5k-1k)"
	case avgAccuracy > 70 && blunderRate < 0.12:
		return "Mid Kyu (10k-6k)"
	case avgAccuracy > 60:
		return "Weak Kyu (15k-11k)"
	default:
		return "Beginner (20k-16k)"
	}
}
