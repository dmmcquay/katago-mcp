package katago

import (
	"context"
	"fmt"
	"strings"
)

// AnalysisRequest represents a request to analyze a position
type AnalysisRequest struct {
	// Position to analyze
	Position *Position

	// Analysis parameters (override defaults if specified)
	MaxVisits  *int     `json:"maxVisits,omitempty"`
	MaxTime    *float64 `json:"maxTime,omitempty"`

	// Optional parameters
	IncludePolicy      bool     `json:"includePolicy,omitempty"`
	IncludeOwnership   bool     `json:"includeOwnership,omitempty"`
	IncludeMovesOwnership bool  `json:"includeMovesOwnership,omitempty"`
	IncludePVVisits    bool     `json:"includePVVisits,omitempty"`
	AvoidMoves         []string `json:"avoidMoves,omitempty"`
	AllowMoves         []string `json:"allowMoves,omitempty"`
}

// AnalysisResult represents the analysis result
type AnalysisResult struct {
	// Move analysis
	MoveInfos []MoveInfo `json:"moveInfos"`
	
	// Root position info
	RootInfo RootInfo `json:"rootInfo"`
	
	// Policy prior (if requested)
	Policy map[string]float64 `json:"policy,omitempty"`
	
	// Ownership map (if requested)
	Ownership [][]float64 `json:"ownership,omitempty"`
	
	// Move-specific ownership (if requested)
	MovesOwnership map[string][][]float64 `json:"movesOwnership,omitempty"`
}

// Analyze analyzes a position using KataGo
func (e *Engine) Analyze(ctx context.Context, req *AnalysisRequest) (*AnalysisResult, error) {
	// Validate request
	if err := ValidatePosition(req.Position); err != nil {
		return nil, fmt.Errorf("invalid position: %w", err)
	}

	// Build query
	query := map[string]interface{}{
		"action": "analyze",
		"includePolicy": req.IncludePolicy,
		"includeOwnership": req.IncludeOwnership,
		"includeMovesOwnership": req.IncludeMovesOwnership,
		"includePVVisits": req.IncludePVVisits,
	}

	// Add position data
	query["rules"] = req.Position.Rules
	query["boardXSize"] = req.Position.BoardXSize
	query["boardYSize"] = req.Position.BoardYSize
	
	if req.Position.Komi != 0 {
		query["komi"] = req.Position.Komi
	}

	// Add initial stones
	if len(req.Position.InitialStones) > 0 {
		stones := make([][]interface{}, len(req.Position.InitialStones))
		for i, stone := range req.Position.InitialStones {
			stones[i] = []interface{}{stone.Color, stone.Location}
		}
		query["initialStones"] = stones
	}

	// Add moves
	if len(req.Position.Moves) > 0 {
		moves := make([][]interface{}, len(req.Position.Moves))
		for i, move := range req.Position.Moves {
			if move.Location == "" {
				moves[i] = []interface{}{move.Color, "pass"}
			} else {
				moves[i] = []interface{}{move.Color, move.Location}
			}
		}
		query["moves"] = moves
	}

	// Add initial player
	if req.Position.InitialPlayer != "" {
		query["initialPlayer"] = req.Position.InitialPlayer
	}

	// Override analysis parameters if specified
	if req.MaxVisits != nil {
		query["maxVisits"] = *req.MaxVisits
	}
	if req.MaxTime != nil {
		query["maxTime"] = *req.MaxTime
	}

	// Add move restrictions
	if len(req.AvoidMoves) > 0 {
		avoid := make([]map[string]interface{}, len(req.AvoidMoves))
		for i, move := range req.AvoidMoves {
			avoid[i] = map[string]interface{}{
				"moves": []string{move},
				"untilDepth": 1,
			}
		}
		query["avoidMoves"] = avoid
	}

	if len(req.AllowMoves) > 0 {
		query["allowMoves"] = req.AllowMoves
	}

	// Send query
	resp, err := e.sendQuery(query)
	if err != nil {
		return nil, err
	}

	// Convert response to result
	result := &AnalysisResult{
		MoveInfos: resp.MoveInfos,
		RootInfo:  resp.RootInfo,
	}

	// Extract additional data from raw response
	if req.IncludePolicy {
		if policyData, ok := resp.Raw["policy"].([]interface{}); ok {
			result.Policy = make(map[string]float64)
			for _, item := range policyData {
				if arr, ok := item.([]interface{}); ok && len(arr) == 2 {
					if move, ok := arr[0].(string); ok {
						if value, ok := arr[1].(float64); ok {
							result.Policy[move] = value
						}
					}
				}
			}
		}
	}

	if req.IncludeOwnership {
		if ownershipData, ok := resp.Raw["ownership"].([]interface{}); ok {
			result.Ownership = make([][]float64, len(ownershipData))
			for i, row := range ownershipData {
				if rowData, ok := row.([]interface{}); ok {
					result.Ownership[i] = make([]float64, len(rowData))
					for j, val := range rowData {
						if v, ok := val.(float64); ok {
							result.Ownership[i][j] = v
						}
					}
				}
			}
		}
	}

	if req.IncludeMovesOwnership {
		if movesOwnershipData, ok := resp.Raw["movesOwnership"].(map[string]interface{}); ok {
			result.MovesOwnership = make(map[string][][]float64)
			for move, ownership := range movesOwnershipData {
				if ownershipArray, ok := ownership.([]interface{}); ok {
					moveOwnership := make([][]float64, len(ownershipArray))
					for i, row := range ownershipArray {
						if rowData, ok := row.([]interface{}); ok {
							moveOwnership[i] = make([]float64, len(rowData))
							for j, val := range rowData {
								if v, ok := val.(float64); ok {
									moveOwnership[i][j] = v
								}
							}
						}
					}
					result.MovesOwnership[move] = moveOwnership
				}
			}
		}
	}

	return result, nil
}

// AnalyzeSGF analyzes a position from SGF content
func (e *Engine) AnalyzeSGF(ctx context.Context, sgfContent string, moveNum int) (*AnalysisResult, error) {
	// Parse SGF
	parser := NewSGFParser(sgfContent)
	position, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse SGF: %w", err)
	}

	// Truncate moves if specified
	if moveNum > 0 && moveNum < len(position.Moves) {
		position.Moves = position.Moves[:moveNum]
	}

	// Create analysis request
	req := &AnalysisRequest{
		Position: position,
	}

	return e.Analyze(ctx, req)
}

// FormatAnalysisResult formats an analysis result as human-readable text
func FormatAnalysisResult(result *AnalysisResult, verbose bool) string {
	var sb strings.Builder

	// Root info
	sb.WriteString("=== Position Analysis ===\n")
	sb.WriteString(fmt.Sprintf("Current player: %s\n", result.RootInfo.CurrentPlayer))
	sb.WriteString(fmt.Sprintf("Visits: %d\n", result.RootInfo.Visits))
	sb.WriteString(fmt.Sprintf("Win rate: %.1f%%\n", result.RootInfo.Winrate*100))
	sb.WriteString(fmt.Sprintf("Score: %.1f\n", result.RootInfo.ScoreMean))
	sb.WriteString("\n")

	// Top moves
	sb.WriteString("=== Top Moves ===\n")
	for i, move := range result.MoveInfos {
		if i >= 10 && !verbose {
			break
		}

		sb.WriteString(fmt.Sprintf("%2d. %-4s ", i+1, move.Move))
		sb.WriteString(fmt.Sprintf("visits:%6d ", move.Visits))
		sb.WriteString(fmt.Sprintf("win:%.1f%% ", move.Winrate*100))
		sb.WriteString(fmt.Sprintf("score:%+.1f", move.ScoreLead))

		if verbose && len(move.PV) > 0 {
			sb.WriteString(" pv: ")
			for j, pv := range move.PV {
				if j > 0 {
					sb.WriteString(" ")
				}
				sb.WriteString(pv)
				if j >= 10 {
					sb.WriteString("...")
					break
				}
			}
		}

		sb.WriteString("\n")
	}

	// Policy priors
	if len(result.Policy) > 0 && verbose {
		sb.WriteString("\n=== Policy Network ===\n")
		// Sort by policy value
		type policyMove struct {
			move  string
			value float64
		}
		policies := make([]policyMove, 0, len(result.Policy))
		for move, value := range result.Policy {
			policies = append(policies, policyMove{move, value})
		}
		// Simple bubble sort
		for i := 0; i < len(policies); i++ {
			for j := i + 1; j < len(policies); j++ {
				if policies[j].value > policies[i].value {
					policies[i], policies[j] = policies[j], policies[i]
				}
			}
		}
		for i, p := range policies {
			if i >= 10 {
				break
			}
			sb.WriteString(fmt.Sprintf("%-4s %.1f%%\n", p.move, p.value*100))
		}
	}

	return sb.String()
}