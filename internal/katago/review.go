package katago

import (
	"context"
	"fmt"
	"math"
	"sort"
)

// MistakeThresholds defines thresholds for categorizing mistakes.
type MistakeThresholds struct {
	Blunder       float64 // Win rate drop >= this is a blunder (default: 0.15)
	Mistake       float64 // Win rate drop >= this is a mistake (default: 0.05)
	Inaccuracy    float64 // Win rate drop >= this is an inaccuracy (default: 0.02)
	MinimumVisits int     // Minimum visits for reliable analysis
}

// DefaultMistakeThresholds returns default thresholds.
func DefaultMistakeThresholds() MistakeThresholds {
	return MistakeThresholds{
		Blunder:       0.15,
		Mistake:       0.05,
		Inaccuracy:    0.02,
		MinimumVisits: 100,
	}
}

// GameReview contains the analysis of a complete game.
type GameReview struct {
	Mistakes     []MistakeInfo  `json:"mistakes"`
	Summary      ReviewSummary  `json:"summary"`
	MoveAnalyses []MoveAnalysis `json:"moveAnalyses,omitempty"`
}

// MistakeInfo describes a mistake in the game.
type MistakeInfo struct {
	MoveNumber   int      `json:"moveNumber"`
	Color        string   `json:"color"`
	PlayedMove   string   `json:"playedMove"`
	BestMove     string   `json:"bestMove"`
	WinrateDrop  float64  `json:"winrateDrop"`
	ScoreDrop    float64  `json:"scoreDrop"`
	Category     string   `json:"category"` // "blunder", "mistake", "inaccuracy"
	Explanation  string   `json:"explanation"`
	BestSequence []string `json:"bestSequence,omitempty"`
}

// ReviewSummary provides overall game statistics.
type ReviewSummary struct {
	TotalMoves        int     `json:"totalMoves"`
	BlackMistakes     int     `json:"blackMistakes"`
	WhiteMistakes     int     `json:"whiteMistakes"`
	BlackBlunders     int     `json:"blackBlunders"`
	WhiteBlunders     int     `json:"whiteBlunders"`
	BlackAccuracy     float64 `json:"blackAccuracy"` // Percentage of accurate moves
	WhiteAccuracy     float64 `json:"whiteAccuracy"`
	GameChangingMoves []int   `json:"gameChangingMoves"`
	EstimatedRank     string  `json:"estimatedRank,omitempty"`
}

// MoveAnalysis contains detailed analysis for a single move.
type MoveAnalysis struct {
	MoveNumber    int        `json:"moveNumber"`
	Color         string     `json:"color"`
	PlayedMove    string     `json:"playedMove"`
	WinrateBefore float64    `json:"winrateBefore"`
	WinrateAfter  float64    `json:"winrateAfter"`
	ScoreBefore   float64    `json:"scoreBefore"`
	ScoreAfter    float64    `json:"scoreAfter"`
	BestMoves     []MoveInfo `json:"bestMoves"`
	Visits        int        `json:"visits"`
}

// ReviewGame analyzes a complete game to find mistakes.
func (e *Engine) ReviewGame(ctx context.Context, sgfContent string, thresholds *MistakeThresholds) (*GameReview, error) {
	// Parse SGF
	parser := NewSGFParser(sgfContent)
	position, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse SGF: %w", err)
	}

	if thresholds == nil {
		defaultThresholds := DefaultMistakeThresholds()
		thresholds = &defaultThresholds
	}

	// Analyze each position in the game
	moveAnalyses := make([]MoveAnalysis, 0, len(position.Moves))
	mistakes := []MistakeInfo{}

	// Track statistics
	blackMistakes, whiteMistakes := 0, 0
	blackBlunders, whiteBlunders := 0, 0
	blackAccurateMoves, whiteAccurateMoves := 0, 0
	gameChangingMoves := []int{}

	// Create a copy of position for analysis
	currentPos := &Position{
		Rules:         position.Rules,
		BoardXSize:    position.BoardXSize,
		BoardYSize:    position.BoardYSize,
		InitialStones: position.InitialStones,
		InitialPlayer: position.InitialPlayer,
		Komi:          position.Komi,
		Moves:         []Move{},
	}

	for i := 0; i < len(position.Moves); i++ {
		// Analyze position before the move
		req := &AnalysisRequest{
			Position:  currentPos,
			MaxVisits: &thresholds.MinimumVisits,
		}

		result, err := e.Analyze(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze move %d: %w", i+1, err)
		}

		// Get win rate and score before move
		winrateBefore := result.RootInfo.Winrate
		scoreBefore := result.RootInfo.ScoreMean

		// Apply the actual move
		playedMove := position.Moves[i]
		currentPos.Moves = append(currentPos.Moves, playedMove)

		// Find the played move in analysis results
		var playedMoveInfo *MoveInfo
		var bestMoveInfo *MoveInfo

		if len(result.MoveInfos) > 0 {
			bestMoveInfo = &result.MoveInfos[0]

			for _, moveInfo := range result.MoveInfos {
				if moveInfo.Move == playedMove.Location ||
					(playedMove.Location == "" && moveInfo.Move == "pass") {
					playedMoveInfo = &moveInfo
					break
				}
			}
		}

		// Calculate win rate drop
		winrateAfter := winrateBefore // Default if move not found
		scoreAfter := scoreBefore

		if playedMoveInfo != nil {
			// Adjust for player perspective
			if playedMove.Color == "w" {
				winrateAfter = 1.0 - playedMoveInfo.Winrate
			} else {
				winrateAfter = playedMoveInfo.Winrate
			}
			scoreAfter = playedMoveInfo.ScoreMean
		}

		winrateDrop := winrateBefore - winrateAfter
		scoreDrop := math.Abs(scoreBefore - scoreAfter)

		// Create move analysis
		analysis := MoveAnalysis{
			MoveNumber:    i + 1,
			Color:         playedMove.Color,
			PlayedMove:    playedMove.Location,
			WinrateBefore: winrateBefore,
			WinrateAfter:  winrateAfter,
			ScoreBefore:   scoreBefore,
			ScoreAfter:    scoreAfter,
			BestMoves:     result.MoveInfos,
			Visits:        result.RootInfo.Visits,
		}
		moveAnalyses = append(moveAnalyses, analysis)

		// Check if this is a mistake
		if bestMoveInfo != nil && playedMoveInfo != nil && bestMoveInfo.Move != playedMove.Location {
			// Categorize the mistake
			category := ""
			switch {
			case winrateDrop >= thresholds.Blunder:
				category = "blunder"
				if playedMove.Color == "b" {
					blackBlunders++
				} else {
					whiteBlunders++
				}
			case winrateDrop >= thresholds.Mistake:
				category = "mistake"
				if playedMove.Color == "b" {
					blackMistakes++
				} else {
					whiteMistakes++
				}
			case winrateDrop >= thresholds.Inaccuracy:
				category = "inaccuracy"
			}

			if category != "" {
				mistake := MistakeInfo{
					MoveNumber:   i + 1,
					Color:        playedMove.Color,
					PlayedMove:   playedMove.Location,
					BestMove:     bestMoveInfo.Move,
					WinrateDrop:  winrateDrop,
					ScoreDrop:    scoreDrop,
					Category:     category,
					Explanation:  generateMistakeExplanation(category, winrateDrop, scoreDrop),
					BestSequence: bestMoveInfo.PV,
				}
				mistakes = append(mistakes, mistake)
			}
		} else if winrateDrop < thresholds.Inaccuracy {
			// Accurate move
			if playedMove.Color == "b" {
				blackAccurateMoves++
			} else {
				whiteAccurateMoves++
			}
		}

		// Check if game-changing move (large swing in evaluation)
		if winrateDrop > 0.10 || winrateDrop < -0.10 {
			gameChangingMoves = append(gameChangingMoves, i+1)
		}
	}

	// Calculate accuracy percentages
	blackMoves := 0
	whiteMoves := 0
	for _, move := range position.Moves {
		if move.Color == "b" {
			blackMoves++
		} else {
			whiteMoves++
		}
	}

	blackAccuracy := 0.0
	if blackMoves > 0 {
		blackAccuracy = float64(blackAccurateMoves) / float64(blackMoves) * 100
	}

	whiteAccuracy := 0.0
	if whiteMoves > 0 {
		whiteAccuracy = float64(whiteAccurateMoves) / float64(whiteMoves) * 100
	}

	// Create summary
	summary := ReviewSummary{
		TotalMoves:        len(position.Moves),
		BlackMistakes:     blackMistakes,
		WhiteMistakes:     whiteMistakes,
		BlackBlunders:     blackBlunders,
		WhiteBlunders:     whiteBlunders,
		BlackAccuracy:     blackAccuracy,
		WhiteAccuracy:     whiteAccuracy,
		GameChangingMoves: gameChangingMoves,
		EstimatedRank:     estimateRank(blackAccuracy, whiteAccuracy, blackBlunders+whiteBlunders),
	}

	return &GameReview{
		Mistakes:     mistakes,
		Summary:      summary,
		MoveAnalyses: moveAnalyses,
	}, nil
}

// generateMistakeExplanation creates a human-readable explanation of the mistake.
func generateMistakeExplanation(category string, winrateDrop, scoreDrop float64) string {
	switch category {
	case "blunder":
		return fmt.Sprintf("This move loses %.1f%% win rate (%.1f points). A severe mistake that significantly damages the position.",
			winrateDrop*100, scoreDrop)
	case "mistake":
		return fmt.Sprintf("This move loses %.1f%% win rate (%.1f points). A clear error that weakens the position.",
			winrateDrop*100, scoreDrop)
	case "inaccuracy":
		return fmt.Sprintf("This move loses %.1f%% win rate (%.1f points). A minor inaccuracy that could be improved.",
			winrateDrop*100, scoreDrop)
	default:
		return ""
	}
}

// estimateRank provides a rough estimate of playing strength based on accuracy.
func estimateRank(blackAccuracy, whiteAccuracy float64, totalBlunders int) string {
	avgAccuracy := (blackAccuracy + whiteAccuracy) / 2

	// Very rough estimates based on accuracy and blunders
	switch {
	case avgAccuracy > 95 && totalBlunders == 0:
		return "Professional"
	case avgAccuracy > 90 && totalBlunders <= 1:
		return "1-3 dan"
	case avgAccuracy > 80 && totalBlunders <= 2:
		return "1-5 kyu"
	case avgAccuracy > 75 && totalBlunders <= 3:
		return "5-10 kyu"
	case avgAccuracy > 70:
		return "10-20 kyu"
	default:
		return "20-30 kyu"
	}
}

// FindTopMistakes returns the most significant mistakes in a game.
func FindTopMistakes(review *GameReview, limit int) []MistakeInfo {
	// Sort mistakes by win rate drop
	mistakes := make([]MistakeInfo, len(review.Mistakes))
	copy(mistakes, review.Mistakes)

	sort.Slice(mistakes, func(i, j int) bool {
		return mistakes[i].WinrateDrop > mistakes[j].WinrateDrop
	})

	if limit > 0 && limit < len(mistakes) {
		return mistakes[:limit]
	}
	return mistakes
}
