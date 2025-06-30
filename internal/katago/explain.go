package katago

import (
	"context"
	"fmt"
	"math"
)

// MoveExplanation provides detailed explanation for a move.
type MoveExplanation struct {
	Move         string        `json:"move"`
	Explanation  string        `json:"explanation"`
	Winrate      float64       `json:"winrate"`
	ScoreLead    float64       `json:"scoreLead"`
	Visits       int           `json:"visits"`
	Pros         []string      `json:"pros"`
	Cons         []string      `json:"cons"`
	Alternatives []Alternative `json:"alternatives"`
	Strategic    StrategicInfo `json:"strategic"`
}

// Alternative represents an alternative move option.
type Alternative struct {
	Move      string  `json:"move"`
	Winrate   float64 `json:"winrate"`
	Visits    int     `json:"visits"`
	Reasoning string  `json:"reasoning"`
}

// StrategicInfo contains strategic analysis of a move.
type StrategicInfo struct {
	Purpose       []string `json:"purpose"`     // Attack, defense, territory, influence
	Urgency       string   `json:"urgency"`     // Critical, important, optional
	BoardRegion   string   `json:"boardRegion"` // Corner, side, center
	FightingMove  bool     `json:"fightingMove"`
	TerritoryMove bool     `json:"territoryMove"`
	InfluenceMove bool     `json:"influenceMove"`
}

// ExplainMove provides explanation for why a move is good or bad.
func (e *Engine) ExplainMove(ctx context.Context, position *Position, move string) (*MoveExplanation, error) {
	// Analyze the position
	req := &AnalysisRequest{
		Position:         position,
		IncludePolicy:    true,
		IncludeOwnership: true,
	}

	result, err := e.Analyze(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze position: %w", err)
	}

	// Find the move in the analysis
	var moveInfo *MoveInfo
	var moveRank int
	for i, mi := range result.MoveInfos {
		if mi.Move == move {
			moveInfo = &mi
			moveRank = i + 1
			break
		}
	}

	if moveInfo == nil {
		return nil, fmt.Errorf("move %s not found in analysis", move)
	}

	// Get top moves for comparison
	topMoves := result.MoveInfos
	if len(topMoves) > 5 {
		topMoves = topMoves[:5]
	}

	explanation := &MoveExplanation{
		Move:      move,
		Winrate:   moveInfo.Winrate,
		ScoreLead: moveInfo.ScoreLead,
		Visits:    moveInfo.Visits,
	}

	// Analyze move quality
	bestMove := &result.MoveInfos[0]
	winrateDiff := bestMove.Winrate - moveInfo.Winrate

	// Generate main explanation
	switch {
	case moveRank == 1:
		explanation.Explanation = fmt.Sprintf("%s is KataGo's top choice (%.1f%% win rate, %.1f point lead)",
			move, moveInfo.Winrate*100, moveInfo.ScoreLead)
	case winrateDiff < 0.02:
		explanation.Explanation = fmt.Sprintf("%s is nearly as good as the best move (%.1f%% win rate, rank #%d)",
			move, moveInfo.Winrate*100, moveRank)
	case winrateDiff < 0.05:
		explanation.Explanation = fmt.Sprintf("%s is a reasonable move but slightly inferior (%.1f%% win rate, -%1.f%% from best)",
			move, moveInfo.Winrate*100, winrateDiff*100)
	default:
		explanation.Explanation = fmt.Sprintf("%s is questionable, losing %.1f%% win rate compared to %s",
			move, winrateDiff*100, bestMove.Move)
	}

	// Analyze strategic aspects
	explanation.Strategic = analyzeStrategicAspects(move, position, result)

	// Generate pros and cons
	explanation.Pros, explanation.Cons = generateProsAndCons(moveInfo, bestMove, result, position)

	// Add alternatives
	for i, altMove := range topMoves {
		if i >= 3 || altMove.Move == move {
			continue
		}

		alt := Alternative{
			Move:    altMove.Move,
			Winrate: altMove.Winrate,
			Visits:  altMove.Visits,
		}

		// Generate reasoning for alternative
		if i == 0 {
			alt.Reasoning = "KataGo's top choice"
		} else {
			alt.Reasoning = compareMove(&altMove, moveInfo, position)
		}

		explanation.Alternatives = append(explanation.Alternatives, alt)
	}

	return explanation, nil
}

// analyzeStrategicAspects determines the strategic nature of a move.
func analyzeStrategicAspects(move string, position *Position, _ *AnalysisResult) StrategicInfo {
	info := StrategicInfo{
		Purpose: []string{},
	}

	// Determine board region
	x, y := parseCoord(move, position.BoardXSize)
	if x >= 0 && y >= 0 {
		info.BoardRegion = getBoardRegion(x, y, position.BoardXSize)
	}

	// Analyze based on board position and context
	moveNum := len(position.Moves)

	// Opening moves
	if moveNum < 20 {
		if info.BoardRegion == "corner" {
			info.Purpose = append(info.Purpose, "corner enclosure")
			info.TerritoryMove = true
		} else if info.BoardRegion == "side" {
			info.Purpose = append(info.Purpose, "side development")
			info.InfluenceMove = true
		}
		info.Urgency = "important"
	}

	// Check if it's a fighting move (near existing stones)
	if isNearStones(x, y, position) {
		info.Purpose = append(info.Purpose, "local response")
		info.FightingMove = true
		info.Urgency = "critical"
	}

	// Territory vs influence based on location
	if info.BoardRegion == "corner" || info.BoardRegion == "side" {
		info.TerritoryMove = true
		if !contains(info.Purpose, "territory") {
			info.Purpose = append(info.Purpose, "territory")
		}
	} else {
		info.InfluenceMove = true
		if !contains(info.Purpose, "influence") {
			info.Purpose = append(info.Purpose, "influence")
		}
	}

	// Default urgency if not set
	if info.Urgency == "" {
		info.Urgency = "optional"
	}

	return info
}

// getBoardRegion determines which region of the board a move is in.
func getBoardRegion(x, y, boardSize int) string {
	edge := 3 // Consider 3 lines from edge as corner/side

	// Corners
	if (x < edge || x >= boardSize-edge) && (y < edge || y >= boardSize-edge) {
		return "corner"
	}

	// Sides
	if x < edge || x >= boardSize-edge || y < edge || y >= boardSize-edge {
		return "side"
	}

	// Center
	return "center"
}

// isNearStones checks if a move is near existing stones.
func isNearStones(_, _ int, position *Position) bool {
	// Simplified check - in real implementation would check actual board state
	return len(position.Moves) > 4 && len(position.Moves) < 50
}

// generateProsAndCons creates lists of advantages and disadvantages.
func generateProsAndCons(moveInfo, bestMove *MoveInfo, result *AnalysisResult, position *Position) ([]string, []string) {
	pros := []string{}
	cons := []string{}

	// Compare to best move
	winrateDiff := bestMove.Winrate - moveInfo.Winrate

	// Pros
	if moveInfo.Visits > 100 {
		pros = append(pros, "Well-explored by the engine")
	}

	if moveInfo.Prior > 0.1 {
		pros = append(pros, "Natural-looking move")
	}

	if winrateDiff < 0.02 {
		pros = append(pros, "Nearly optimal")
	}

	if moveInfo.ScoreLead > 0 {
		pros = append(pros, fmt.Sprintf("Maintains %.1f point lead", moveInfo.ScoreLead))
	}

	// Move-specific pros based on board position
	x, y := parseCoord(moveInfo.Move, position.BoardXSize)
	region := getBoardRegion(x, y, position.BoardXSize)
	if region == "corner" {
		pros = append(pros, "Secures corner territory")
	} else if region == "side" {
		pros = append(pros, "Develops along the side")
	}

	// Cons
	if winrateDiff > 0.05 {
		cons = append(cons, fmt.Sprintf("Loses %.1f%% win rate", winrateDiff*100))
	}

	if moveInfo.Prior < 0.01 {
		cons = append(cons, "Unconventional choice")
	}

	if moveInfo.Visits < 50 {
		cons = append(cons, "Limited engine exploration")
	}

	if winrateDiff > 0.02 && bestMove.Move != "" {
		cons = append(cons, fmt.Sprintf("%s is better", bestMove.Move))
	}

	// Ensure we have at least one item in each list
	if len(pros) == 0 {
		pros = append(pros, "Playable move")
	}
	if len(cons) == 0 && winrateDiff > 0 {
		cons = append(cons, "Slightly suboptimal")
	}

	return pros, cons
}

// compareMove generates a comparison between two moves.
func compareMove(move1, move2 *MoveInfo, position *Position) string {
	winrateDiff := move1.Winrate - move2.Winrate

	if math.Abs(winrateDiff) < 0.01 {
		return "Similar strength"
	}

	x1, y1 := parseCoord(move1.Move, position.BoardXSize)
	x2, y2 := parseCoord(move2.Move, position.BoardXSize)

	region1 := getBoardRegion(x1, y1, position.BoardXSize)
	region2 := getBoardRegion(x2, y2, position.BoardXSize)

	if region1 != region2 {
		if winrateDiff > 0 {
			return fmt.Sprintf("Prefers %s over %s", region1, region2)
		}
		return fmt.Sprintf("Alternative in %s", region1)
	}

	if winrateDiff > 0.02 {
		return fmt.Sprintf("%.1f%% better", winrateDiff*100)
	}

	return "Slightly different approach"
}

// contains checks if a slice contains a string.
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
