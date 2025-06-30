package katago

import (
	"context"
	"fmt"
	"strings"
)

// TerritoryEstimate contains territory ownership information.
type TerritoryEstimate struct {
	Map            *TerritoryMap `json:"map"`
	BlackTerritory int           `json:"blackTerritory"`
	WhiteTerritory int           `json:"whiteTerritory"`
	DamePoints     int           `json:"damePoints"`
	ScoreEstimate  float64       `json:"scoreEstimate"`
	ScoreString    string        `json:"scoreString"`
}

// TerritoryMap represents the ownership of each board point.
type TerritoryMap struct {
	Territory  [][]string  `json:"territory"`  // "B", "W", or "?" for each point
	Ownership  [][]float64 `json:"ownership"`  // -1.0 to 1.0 (-1 = white, 1 = black)
	DeadStones []string    `json:"deadStones"` // List of dead stone groups
}

// EstimateTerritory analyzes territory ownership for a position.
func (e *Engine) EstimateTerritory(ctx context.Context, position *Position, threshold float64) (*TerritoryEstimate, error) {
	// Default threshold
	if threshold <= 0 || threshold > 1 {
		threshold = 0.85
	}

	// Request ownership analysis
	req := &AnalysisRequest{
		Position:         position,
		IncludeOwnership: true,
		IncludePolicy:    false,
	}

	result, err := e.Analyze(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze position: %w", err)
	}

	if len(result.Ownership) == 0 {
		return nil, fmt.Errorf("no ownership data returned")
	}

	// Create territory map
	boardSize := position.BoardXSize
	if position.BoardYSize != boardSize {
		return nil, fmt.Errorf("non-square boards not fully supported")
	}

	territoryMap := &TerritoryMap{
		Territory: make([][]string, boardSize),
		Ownership: make([][]float64, boardSize),
	}

	blackTerritory := 0
	whiteTerritory := 0
	damePoints := 0

	// Convert ownership to territory
	for y := 0; y < boardSize; y++ {
		territoryMap.Territory[y] = make([]string, boardSize)
		territoryMap.Ownership[y] = make([]float64, boardSize)

		for x := 0; x < boardSize; x++ {
			idx := y*boardSize + x
			if idx >= len(result.Ownership) {
				continue
			}

			ownership := result.Ownership[idx]
			territoryMap.Ownership[y][x] = ownership

			// Determine territory based on threshold
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

	// Identify dead stones (simplified - stones in opponent's strong territory)
	deadStones := identifyDeadStones(position, territoryMap, threshold)
	territoryMap.DeadStones = deadStones

	// Calculate score
	komi := 6.5 // Default komi, should get from position.Rules
	scoreEstimate := float64(blackTerritory-whiteTerritory) - komi

	var scoreString string
	if scoreEstimate > 0 {
		scoreString = fmt.Sprintf("B+%.1f", scoreEstimate)
	} else {
		scoreString = fmt.Sprintf("W+%.1f", -scoreEstimate)
	}

	return &TerritoryEstimate{
		Map:            territoryMap,
		BlackTerritory: blackTerritory,
		WhiteTerritory: whiteTerritory,
		DamePoints:     damePoints,
		ScoreEstimate:  scoreEstimate,
		ScoreString:    scoreString,
	}, nil
}

// identifyDeadStones finds stones that are likely dead.
func identifyDeadStones(position *Position, territoryMap *TerritoryMap, threshold float64) []string {
	deadStones := []string{}
	boardSize := position.BoardXSize

	// Build current board state
	board := make([][]string, boardSize)
	for y := 0; y < boardSize; y++ {
		board[y] = make([]string, boardSize)
		for x := 0; x < boardSize; x++ {
			board[y][x] = "."
		}
	}

	// Apply initial stones
	for _, stone := range position.InitialStones {
		x, y := parseCoord(stone.Location, boardSize)
		if x >= 0 && y >= 0 {
			board[y][x] = stone.Color
		}
	}

	// Apply moves
	for _, move := range position.Moves {
		if move.Location != "" && move.Location != "pass" { // Not a pass
			x, y := parseCoord(move.Location, boardSize)
			if x >= 0 && y >= 0 {
				board[y][x] = move.Color
			}
		}
	}

	// Check each stone
	visited := make([][]bool, boardSize)
	for y := 0; y < boardSize; y++ {
		visited[y] = make([]bool, boardSize)
	}

	for y := 0; y < boardSize; y++ {
		for x := 0; x < boardSize; x++ {
			if board[y][x] != "." && !visited[y][x] {
				// Check if this stone group is dead
				group := findGroup(x, y, board, visited)
				if isGroupDead(group, board[y][x], territoryMap, threshold) {
					deadStones = append(deadStones, group...)
				}
			}
		}
	}

	return deadStones
}

// findGroup finds all stones connected to the given position.
func findGroup(x, y int, board [][]string, visited [][]bool) []string {
	boardSize := len(board)
	if x < 0 || x >= boardSize || y < 0 || y >= boardSize || visited[y][x] {
		return []string{}
	}

	color := board[y][x]
	if color == "." {
		return []string{}
	}

	visited[y][x] = true
	group := []string{coordToString(x, y, boardSize)}

	// Check adjacent points
	directions := [][2]int{{0, 1}, {1, 0}, {0, -1}, {-1, 0}}
	for _, dir := range directions {
		nx, ny := x+dir[0], y+dir[1]
		if nx >= 0 && nx < boardSize && ny >= 0 && ny < boardSize &&
			board[ny][nx] == color && !visited[ny][nx] {
			subgroup := findGroup(nx, ny, board, visited)
			group = append(group, subgroup...)
		}
	}

	return group
}

// isGroupDead checks if a group of stones is likely dead.
func isGroupDead(group []string, color string, territoryMap *TerritoryMap, threshold float64) bool {
	if len(group) == 0 {
		return false
	}

	// A group is dead if it's entirely surrounded by strong opponent territory
	// For black stones: dead if in strong white territory (ownership < -threshold)
	// For white stones: dead if in strong black territory (ownership > threshold)

	for _, coord := range group {
		x, y := parseCoord(coord, len(territoryMap.Ownership))
		if x >= 0 && y >= 0 && y < len(territoryMap.Ownership) && x < len(territoryMap.Ownership[y]) {
			ownership := territoryMap.Ownership[y][x]
			if color == "B" {
				// Black stone is alive if ownership is positive (black territory)
				// Dead if ownership < -threshold (strong white territory)
				if ownership > -threshold {
					return false // Not dead - either in black territory or contested
				}
			} else if color == "W" {
				// White stone is alive if ownership is negative (white territory)
				// Dead if ownership > threshold (strong black territory)
				if ownership < threshold {
					return false // Not dead - either in white territory or contested
				}
			}
		}
	}

	return true
}

// parseCoord converts a coordinate string to x,y indices.
func parseCoord(coord string, boardSize int) (x, y int) {
	if len(coord) < 2 {
		return -1, -1
	}

	// Handle pass
	if coord == "pass" || coord == "" {
		return -1, -1
	}

	col := coord[0]
	row := coord[1:]

	// Convert column letter to x coordinate
	x = -1
	if col >= 'A' && col <= 'Z' {
		x = int(col - 'A')
		if col > 'I' {
			x-- // Skip 'I' in Go coordinates
		}
	}

	// Convert row number to y coordinate
	y = -1
	if row != "" {
		var rowNum int
		_, _ = fmt.Sscanf(row, "%d", &rowNum)
		y = boardSize - rowNum
	}

	if x < 0 || x >= boardSize || y < 0 || y >= boardSize {
		return -1, -1
	}

	return x, y
}

// coordToString converts x,y indices to a coordinate string.
func coordToString(x, y, boardSize int) string {
	col := 'A' + x
	if x >= 8 {
		col++ // Skip 'I'
	}
	row := boardSize - y
	return fmt.Sprintf("%c%d", col, row)
}

// GetTerritoryVisualization returns a visual representation of the territory.
func GetTerritoryVisualization(estimate *TerritoryEstimate) string {
	if estimate.Map == nil || len(estimate.Map.Territory) == 0 {
		return "No territory data available"
	}

	var sb strings.Builder
	boardSize := len(estimate.Map.Territory)

	// Column labels
	sb.WriteString("   ")
	for x := 0; x < boardSize; x++ {
		col := 'A' + x
		if x >= 8 {
			col++ // Skip 'I'
		}
		sb.WriteString(fmt.Sprintf(" %c", col))
	}
	sb.WriteString("\n")

	// Board with territory markers
	for y := 0; y < boardSize; y++ {
		row := boardSize - y
		sb.WriteString(fmt.Sprintf("%2d ", row))
		for x := 0; x < boardSize; x++ {
			switch estimate.Map.Territory[y][x] {
			case "B":
				sb.WriteString(" ●") // Black territory
			case "W":
				sb.WriteString(" ○") // White territory
			default:
				sb.WriteString(" ·") // Dame or unclear
			}
		}
		sb.WriteString(fmt.Sprintf(" %d\n", row))
	}

	// Column labels again
	sb.WriteString("   ")
	for x := 0; x < boardSize; x++ {
		col := 'A' + x
		if x >= 8 {
			col++ // Skip 'I'
		}
		sb.WriteString(fmt.Sprintf(" %c", col))
	}
	sb.WriteString("\n\n")

	// Summary
	sb.WriteString(fmt.Sprintf("Black territory: %d\n", estimate.BlackTerritory))
	sb.WriteString(fmt.Sprintf("White territory: %d\n", estimate.WhiteTerritory))
	sb.WriteString(fmt.Sprintf("Dame points: %d\n", estimate.DamePoints))
	sb.WriteString(fmt.Sprintf("Score: %s\n", estimate.ScoreString))

	return sb.String()
}
