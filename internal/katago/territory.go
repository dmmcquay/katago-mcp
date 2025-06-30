package katago

import (
	"context"
	"fmt"
	"math"
)

// TerritoryMap represents ownership probabilities for each board point.
type TerritoryMap struct {
	BoardXSize int         `json:"boardXSize"`
	BoardYSize int         `json:"boardYSize"`
	Ownership  [][]float64 `json:"ownership"` // -1 (white) to 1 (black)
	Territory  [][]string  `json:"territory"` // "B", "W", or "?" for uncertain
}

// TerritoryEstimate contains the territory analysis results.
type TerritoryEstimate struct {
	Map            *TerritoryMap `json:"map"`
	BlackTerritory int           `json:"blackTerritory"`
	WhiteTerritory int           `json:"whiteTerritory"`
	DamePoints     int           `json:"damePoints"`  // Neutral points
	BlackDead      []string      `json:"blackDead"`   // Dead black stones
	WhiteDead      []string      `json:"whiteDead"`   // Dead white stones
	FinalScore     float64       `json:"finalScore"`  // Positive = black wins
	ScoreString    string        `json:"scoreString"` // Human readable score
}

// EstimateTerritory analyzes territory ownership for a position.
func (e *Engine) EstimateTerritory(ctx context.Context, position *Position, threshold float64) (*TerritoryEstimate, error) {
	// Validate position
	if err := ValidatePosition(position); err != nil {
		return nil, fmt.Errorf("invalid position: %w", err)
	}

	// Default threshold for territory determination
	if threshold <= 0 {
		threshold = 0.60 // 60% confidence
	}

	// Request ownership map in analysis
	req := &AnalysisRequest{
		Position:         position,
		IncludeOwnership: true,
	}

	result, err := e.Analyze(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze position: %w", err)
	}

	if len(result.Ownership) == 0 {
		return nil, fmt.Errorf("ownership data not available")
	}

	// Create territory map
	territoryMap := &TerritoryMap{
		BoardXSize: position.BoardXSize,
		BoardYSize: position.BoardYSize,
		Ownership:  result.Ownership,
		Territory:  make([][]string, position.BoardYSize),
	}

	// Convert ownership values to territory
	blackTerritory := 0
	whiteTerritory := 0
	damePoints := 0

	for y := 0; y < position.BoardYSize; y++ {
		territoryMap.Territory[y] = make([]string, position.BoardXSize)
		for x := 0; x < position.BoardXSize; x++ {
			ownership := result.Ownership[y][x]

			switch {
			case ownership > threshold:
				territoryMap.Territory[y][x] = "B"
				blackTerritory++
			case ownership < -threshold:
				territoryMap.Territory[y][x] = "W"
				whiteTerritory++
			default:
				territoryMap.Territory[y][x] = "?"
				damePoints++
			}
		}
	}

	// Identify dead stones (simplified - would need more sophisticated analysis)
	blackDead, whiteDead := identifyDeadStones(position, territoryMap)

	// Calculate final score
	// Territory scoring: territory + captures + komi
	blackScore := float64(blackTerritory) + float64(len(whiteDead))
	whiteScore := float64(whiteTerritory) + float64(len(blackDead)) + position.Komi
	finalScore := blackScore - whiteScore

	// Format score string
	scoreString := formatScore(finalScore)

	return &TerritoryEstimate{
		Map:            territoryMap,
		BlackTerritory: blackTerritory,
		WhiteTerritory: whiteTerritory,
		DamePoints:     damePoints,
		BlackDead:      blackDead,
		WhiteDead:      whiteDead,
		FinalScore:     finalScore,
		ScoreString:    scoreString,
	}, nil
}

// identifyDeadStones identifies potentially dead stones based on territory.
func identifyDeadStones(position *Position, territoryMap *TerritoryMap) (blackDead, whiteDead []string) {
	blackDead = []string{}
	whiteDead = []string{}

	// Build board state from moves
	board := make([][]string, position.BoardYSize)
	for y := 0; y < position.BoardYSize; y++ {
		board[y] = make([]string, position.BoardXSize)
		for x := 0; x < position.BoardXSize; x++ {
			board[y][x] = ""
		}
	}

	// Place initial stones
	for _, stone := range position.InitialStones {
		x, y := parseCoordinate(stone.Location, position.BoardXSize)
		if x >= 0 && y >= 0 {
			board[y][x] = stone.Color
		}
	}

	// Apply moves
	for _, move := range position.Moves {
		if move.Location != "" {
			x, y := parseCoordinate(move.Location, position.BoardXSize)
			if x >= 0 && y >= 0 {
				board[y][x] = move.Color
			}
		}
	}

	// Find dead stones (stones in opponent's territory)
	for y := 0; y < position.BoardYSize; y++ {
		for x := 0; x < position.BoardXSize; x++ {
			stone := board[y][x]
			territory := territoryMap.Territory[y][x]

			if stone == "b" && territory == "W" {
				blackDead = append(blackDead, formatCoordinate(x, y, position.BoardXSize))
			} else if stone == "w" && territory == "B" {
				whiteDead = append(whiteDead, formatCoordinate(x, y, position.BoardXSize))
			}
		}
	}

	return blackDead, whiteDead
}

// parseCoordinate converts KataGo coordinate to x,y.
func parseCoordinate(coord string, boardSize int) (x, y int) {
	if len(coord) < 2 {
		return -1, -1
	}

	// Parse column (A-T, skipping I)
	col := coord[0]
	x = -1
	if col >= 'A' && col <= 'H' {
		x = int(col - 'A')
	} else if col >= 'J' && col <= 'T' {
		x = int(col - 'A' - 1)
	}

	// Parse row
	row := coord[1:]
	y = -1
	if row != "" {
		var rowNum int
		if _, err := fmt.Sscanf(row, "%d", &rowNum); err == nil && rowNum > 0 && rowNum <= boardSize {
			y = boardSize - rowNum
		}
	}

	return x, y
}

// formatCoordinate converts x,y to KataGo coordinate.
func formatCoordinate(x, y, boardSize int) string {
	// Convert x to letter (A-T, skipping I)
	var col string
	if x < 8 {
		col = string(rune('A' + x))
	} else {
		col = string(rune('A' + x + 1))
	}

	// Convert y to row number
	row := boardSize - y

	return fmt.Sprintf("%s%d", col, row)
}

// formatScore formats the score in a human-readable way.
func formatScore(score float64) string {
	if math.Abs(score) < 0.5 {
		return "Jigo (Draw)"
	}

	winner := "B"
	points := score
	if score < 0 {
		winner = "W"
		points = -score
	}

	// Round to nearest 0.5
	points = math.Round(points*2) / 2

	return fmt.Sprintf("%s+%.1f", winner, points)
}

// GetTerritoryVisualization creates a text visualization of the territory.
func GetTerritoryVisualization(estimate *TerritoryEstimate) string {
	result := ""

	// Add column labels
	result += "   "
	for x := 0; x < estimate.Map.BoardXSize; x++ {
		if x < 8 {
			result += fmt.Sprintf("%c ", 'A'+x)
		} else {
			result += fmt.Sprintf("%c ", 'A'+x+1) // Skip 'I'
		}
	}
	result += "\n"

	// Add rows
	for y := 0; y < estimate.Map.BoardYSize; y++ {
		row := estimate.Map.BoardYSize - y
		result += fmt.Sprintf("%2d ", row)

		for x := 0; x < estimate.Map.BoardXSize; x++ {
			territory := estimate.Map.Territory[y][x]
			switch territory {
			case "B":
				result += "● " // Black territory
			case "W":
				result += "○ " // White territory
			default:
				result += "· " // Uncertain/dame
			}
		}
		result += fmt.Sprintf("%2d\n", row)
	}

	// Add column labels again
	result += "   "
	for x := 0; x < estimate.Map.BoardXSize; x++ {
		if x < 8 {
			result += fmt.Sprintf("%c ", 'A'+x)
		} else {
			result += fmt.Sprintf("%c ", 'A'+x+1)
		}
	}
	result += "\n\n"

	// Add summary
	result += fmt.Sprintf("Black territory: %d\n", estimate.BlackTerritory)
	result += fmt.Sprintf("White territory: %d\n", estimate.WhiteTerritory)
	result += fmt.Sprintf("Dame points: %d\n", estimate.DamePoints)
	result += fmt.Sprintf("Score: %s\n", estimate.ScoreString)

	return result
}
