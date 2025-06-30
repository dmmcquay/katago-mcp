package katago

import (
	"context"
	"fmt"
	"strings"
)

// MoveExplanation provides detailed reasoning for a move.
type MoveExplanation struct {
	Move         string            `json:"move"`
	MoveNumber   int               `json:"moveNumber,omitempty"`
	Explanation  string            `json:"explanation"`
	Pros         []string          `json:"pros"`
	Cons         []string          `json:"cons"`
	Alternatives []AlternativeMove `json:"alternatives"`
	Strategic    StrategicInfo     `json:"strategic"`
	Continuation []string          `json:"continuation,omitempty"`
}

// AlternativeMove describes an alternative to the chosen move.
type AlternativeMove struct {
	Move        string  `json:"move"`
	Reason      string  `json:"reason"`
	WinrateDiff float64 `json:"winrateDiff"`
	ScoreDiff   float64 `json:"scoreDiff"`
}

// StrategicInfo contains strategic evaluation of a move.
type StrategicInfo struct {
	Purpose       []string `json:"purpose"`     // Attack, defense, territory, influence
	Urgency       string   `json:"urgency"`     // Critical, important, optional
	BoardRegion   string   `json:"boardRegion"` // Corner, side, center
	FightingMove  bool     `json:"fightingMove"`
	TerritoryMove bool     `json:"territoryMove"`
	InfluenceMove bool     `json:"influenceMove"`
}

// ExplainMove provides detailed explanation for a specific move.
func (e *Engine) ExplainMove(ctx context.Context, position *Position, move string) (*MoveExplanation, error) {
	// Validate position
	if err := ValidatePosition(position); err != nil {
		return nil, fmt.Errorf("invalid position: %w", err)
	}

	// Analyze the position with policy information
	req := &AnalysisRequest{
		Position:      position,
		IncludePolicy: true,
		MaxVisits:     intPtr(1000), // More visits for detailed analysis
	}

	result, err := e.Analyze(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze position: %w", err)
	}

	// Find the specific move in the analysis
	var moveInfo *MoveInfo
	var moveRank int
	for i, info := range result.MoveInfos {
		if info.Move == move {
			moveInfo = &info
			moveRank = i + 1
			break
		}
	}

	if moveInfo == nil {
		return nil, fmt.Errorf("move %s not found in analysis", move)
	}

	// Generate explanation based on analysis
	explanation := &MoveExplanation{
		Move:         move,
		MoveNumber:   len(position.Moves) + 1,
		Continuation: moveInfo.PV,
	}

	// Analyze move characteristics
	explanation.Strategic = analyzeStrategicPurpose(move, position, moveInfo, result)

	// Generate main explanation
	explanation.Explanation = generateMoveExplanation(moveInfo, moveRank, result, explanation.Strategic)

	// List pros and cons
	explanation.Pros = generatePros(moveInfo, moveRank, explanation.Strategic)
	explanation.Cons = generateCons(moveInfo, moveRank, result.MoveInfos, explanation.Strategic)

	// Find alternatives
	explanation.Alternatives = findAlternatives(moveInfo, result.MoveInfos, position)

	return explanation, nil
}

// analyzeStrategicPurpose determines the strategic purpose of a move.
func analyzeStrategicPurpose(move string, position *Position, _ *MoveInfo, result *AnalysisResult) StrategicInfo {
	info := StrategicInfo{
		Purpose:       []string{},
		Urgency:       "optional",
		BoardRegion:   determineBoardRegion(move, position.BoardXSize),
		FightingMove:  false,
		TerritoryMove: false,
		InfluenceMove: false,
	}

	// Determine urgency based on win rate
	if len(result.MoveInfos) > 1 {
		topWinrate := result.MoveInfos[0].Winrate
		secondWinrate := result.MoveInfos[1].Winrate

		if topWinrate-secondWinrate > 0.10 {
			info.Urgency = "critical"
		} else if topWinrate-secondWinrate > 0.05 {
			info.Urgency = "important"
		}
	}

	// Analyze based on board position and game phase
	moveCount := len(position.Moves)
	gamePhase := "opening"
	if moveCount > 100 {
		gamePhase = "endgame"
	} else if moveCount > 30 {
		gamePhase = "middlegame"
	}

	// Determine move purposes
	x, y := parseCoordinate(move, position.BoardXSize)

	// Corner moves in opening are usually territory-oriented
	if gamePhase == "opening" && info.BoardRegion == "corner" {
		info.TerritoryMove = true
		info.Purpose = append(info.Purpose, "Secure corner territory")
	}

	// Check if it's near existing stones (fighting)
	if isNearStones(x, y, position) {
		info.FightingMove = true
		info.Purpose = append(info.Purpose, "Local fight")
	}

	// Center moves are often influence-oriented
	if info.BoardRegion == "center" && gamePhase != "endgame" {
		info.InfluenceMove = true
		info.Purpose = append(info.Purpose, "Build influence")
	}

	// Endgame moves are usually territory-focused
	if gamePhase == "endgame" {
		info.TerritoryMove = true
		info.Purpose = append(info.Purpose, "Endgame point")
	}

	return info
}

// determineBoardRegion determines which region of the board a move is in.
func determineBoardRegion(move string, boardSize int) string {
	if move == "" || move == "pass" {
		return "pass"
	}

	x, y := parseCoordinate(move, boardSize)
	if x < 0 || y < 0 {
		return "unknown"
	}

	// Define regions based on distance from edge
	edgeDist := minInt(x, y)
	edgeDist = minInt(edgeDist, boardSize-1-x)
	edgeDist = minInt(edgeDist, boardSize-1-y)

	if edgeDist <= 3 {
		// Check if corner
		if (x <= 3 || x >= boardSize-4) && (y <= 3 || y >= boardSize-4) {
			return "corner"
		}
		return "side"
	}
	return "center"
}

// isNearStones checks if a move is near existing stones.
func isNearStones(x, y int, position *Position) bool {
	// Check if any moves are within 3 points
	for _, move := range position.Moves {
		if move.Location == "" {
			continue
		}
		mx, my := parseCoordinate(move.Location, position.BoardXSize)
		if mx < 0 || my < 0 {
			continue
		}

		dist := abs(mx-x) + abs(my-y)
		if dist <= 3 {
			return true
		}
	}
	return false
}

// generateMoveExplanation creates the main explanation text.
func generateMoveExplanation(moveInfo *MoveInfo, rank int, _ *AnalysisResult, strategic StrategicInfo) string {
	var parts []string

	// Rank and basic evaluation
	if rank == 1 {
		parts = append(parts, fmt.Sprintf("This is KataGo's top choice (%.1f%% win rate).", moveInfo.Winrate*100))
	} else {
		parts = append(parts, fmt.Sprintf("This is KataGo's #%d choice (%.1f%% win rate).", rank, moveInfo.Winrate*100))
	}

	// Strategic purpose
	if len(strategic.Purpose) > 0 {
		parts = append(parts, fmt.Sprintf("The move aims to: %s.", strings.Join(strategic.Purpose, ", ")))
	}

	// Urgency
	switch strategic.Urgency {
	case "critical":
		parts = append(parts, "This is a critical move that significantly affects the game outcome.")
	case "important":
		parts = append(parts, "This is an important move that should be played soon.")
	}

	// Expected continuation
	if len(moveInfo.PV) > 1 {
		continuation := strings.Join(moveInfo.PV[1:minInt(6, len(moveInfo.PV))], " ")
		parts = append(parts, fmt.Sprintf("Expected continuation: %s", continuation))
	}

	return strings.Join(parts, " ")
}

// generatePros lists the advantages of a move.
func generatePros(moveInfo *MoveInfo, rank int, strategic StrategicInfo) []string {
	pros := []string{}

	if rank == 1 {
		pros = append(pros, "Highest win rate according to KataGo")
	}

	if moveInfo.Winrate > 0.55 {
		pros = append(pros, "Maintains a strong advantage")
	}

	if strategic.TerritoryMove {
		pros = append(pros, fmt.Sprintf("Secures approximately %.1f points", moveInfo.ScoreLead))
	}

	if strategic.FightingMove {
		pros = append(pros, "Addresses the local tactical situation")
	}

	if strategic.InfluenceMove {
		pros = append(pros, "Builds valuable influence for future fighting")
	}

	if strategic.Urgency == "critical" {
		pros = append(pros, "Critical move that prevents opponent's strong continuation")
	}

	return pros
}

// generateCons lists potential disadvantages of a move.
func generateCons(moveInfo *MoveInfo, rank int, allMoves []MoveInfo, strategic StrategicInfo) []string {
	cons := []string{}

	if rank > 1 && len(allMoves) > 0 {
		winrateLoss := allMoves[0].Winrate - moveInfo.Winrate
		if winrateLoss > 0.02 {
			cons = append(cons, fmt.Sprintf("Loses %.1f%% win rate compared to best move", winrateLoss*100))
		}
	}

	if strategic.BoardRegion == "center" && len(allMoves) < 20 {
		cons = append(cons, "May be premature to play in center")
	}

	if !strategic.FightingMove && strategic.Urgency != "critical" {
		cons = append(cons, "Might miss urgent moves elsewhere")
	}

	if moveInfo.ScoreLead < -5 {
		cons = append(cons, "Position remains difficult for this player")
	}

	return cons
}

// findAlternatives identifies alternative moves.
func findAlternatives(chosen *MoveInfo, allMoves []MoveInfo, position *Position) []AlternativeMove {
	alternatives := []AlternativeMove{}

	for i, move := range allMoves {
		if i >= 5 || move.Move == chosen.Move {
			continue
		}

		alt := AlternativeMove{
			Move:        move.Move,
			WinrateDiff: move.Winrate - chosen.Winrate,
			ScoreDiff:   move.ScoreLead - chosen.ScoreLead,
		}

		// Generate reason based on characteristics
		switch {
		case alt.WinrateDiff > 0.02:
			alt.Reason = fmt.Sprintf("Higher win rate (+%.1f%%)", alt.WinrateDiff*100)
		case move.Move == "pass":
			alt.Reason = "Pass if game is essentially over"
		default:
			region := determineBoardRegion(move.Move, position.BoardXSize)
			switch region {
			case "corner":
				alt.Reason = "Alternative corner approach"
			case "side":
				alt.Reason = "Side development"
			case "center":
				alt.Reason = "Central influence"
			default:
				alt.Reason = "Alternative continuation"
			}
		}

		alternatives = append(alternatives, alt)
	}

	return alternatives
}

// Helper functions.
func intPtr(i int) *int {
	return &i
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
