package katago

import (
	"strings"
	"testing"
)

func TestGetBoardRegion(t *testing.T) {
	tests := []struct {
		name      string
		x         int
		y         int
		boardSize int
		want      string
	}{
		{
			name:      "3-3 corner",
			x:         2,
			y:         2,
			boardSize: 19,
			want:      "corner",
		},
		{
			name:      "4-4 corner",
			x:         3,
			y:         3,
			boardSize: 19,
			want:      "corner",
		},
		{
			name:      "top edge",
			x:         9,
			y:         1,
			boardSize: 19,
			want:      "side",
		},
		{
			name:      "left edge",
			x:         1,
			y:         9,
			boardSize: 19,
			want:      "side",
		},
		{
			name:      "center",
			x:         9,
			y:         9,
			boardSize: 19,
			want:      "center",
		},
		{
			name:      "9x9 corner",
			x:         2,
			y:         2,
			boardSize: 9,
			want:      "corner",
		},
		{
			name:      "9x9 center",
			x:         4,
			y:         4,
			boardSize: 9,
			want:      "center",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBoardRegion(tt.x, tt.y, tt.boardSize)
			if got != tt.want {
				t.Errorf("getBoardRegion(%d, %d, %d) = %s, want %s",
					tt.x, tt.y, tt.boardSize, got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		str   string
		want  bool
	}{
		{
			name:  "string exists",
			slice: []string{"apple", "banana", "cherry"},
			str:   "banana",
			want:  true,
		},
		{
			name:  "string not exists",
			slice: []string{"apple", "banana", "cherry"},
			str:   "orange",
			want:  false,
		},
		{
			name:  "empty slice",
			slice: []string{},
			str:   "apple",
			want:  false,
		},
		{
			name:  "empty string",
			slice: []string{"apple", "", "cherry"},
			str:   "",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.slice, tt.str)
			if got != tt.want {
				t.Errorf("contains(%v, %s) = %v, want %v",
					tt.slice, tt.str, got, tt.want)
			}
		})
	}
}

func TestAnalyzeStrategicAspects(t *testing.T) {
	// Test opening corner move
	position := &Position{
		BoardXSize: 19,
		BoardYSize: 19,
		Moves:      []Move{},
	}

	result := &AnalysisResult{}

	info := analyzeStrategicAspects("D4", position, result)

	if info.BoardRegion != "corner" {
		t.Errorf("Expected corner region, got %s", info.BoardRegion)
	}

	if !info.TerritoryMove {
		t.Error("Opening corner move should be territory-oriented")
	}

	if info.Urgency != "important" {
		t.Errorf("Expected 'important' urgency for opening, got %s", info.Urgency)
	}

	// Test mid-game move with more stones
	position2 := &Position{
		BoardXSize: 19,
		BoardYSize: 19,
		Moves: []Move{
			{Color: "B", Location: "D4"},
			{Color: "W", Location: "Q16"},
			{Color: "B", Location: "D16"},
			{Color: "W", Location: "Q4"},
			{Color: "B", Location: "D10"},
		},
	}

	info2 := analyzeStrategicAspects("E5", position2, result)
	if !info2.FightingMove {
		t.Error("Move near existing stones in mid-game should be fighting move")
	}
	if info2.Urgency != "critical" {
		t.Errorf("Expected 'critical' urgency for fighting move, got %s", info2.Urgency)
	}
}

func TestGenerateProsAndCons(t *testing.T) {
	moveInfo := &MoveInfo{
		Move:      "D4",
		Winrate:   0.58,
		ScoreLead: 5.5,
		Prior:     0.15,
		Visits:    200,
	}

	bestMove := &MoveInfo{
		Move:      "Q16",
		Winrate:   0.60,
		ScoreLead: 6.0,
		Prior:     0.20,
		Visits:    300,
	}

	position := &Position{
		BoardXSize: 19,
		BoardYSize: 19,
	}

	pros, cons := generateProsAndCons(moveInfo, bestMove, position)

	// Should have at least one pro and con
	if len(pros) == 0 {
		t.Error("Expected at least one pro")
	}
	if len(cons) == 0 {
		t.Error("Expected at least one con for suboptimal move")
	}

	// Check for specific pros
	hasVisitsPro := false
	hasNaturalPro := false
	for _, pro := range pros {
		if strings.Contains(pro, "Well-explored") {
			hasVisitsPro = true
		}
		if strings.Contains(pro, "Natural-looking") {
			hasNaturalPro = true
		}
	}

	if !hasVisitsPro {
		t.Error("Expected pro about being well-explored (200 visits)")
	}
	if !hasNaturalPro {
		t.Error("Expected pro about being natural-looking (0.15 prior)")
	}

	// Check for winrate loss con
	hasWinrateCon := false
	for _, con := range cons {
		if strings.Contains(con, "win rate") {
			hasWinrateCon = true
		}
	}
	if !hasWinrateCon {
		t.Error("Expected con about win rate loss")
	}
}

func TestCompareMove(t *testing.T) {
	move1 := &MoveInfo{
		Move:    "D4",
		Winrate: 0.52,
	}

	move2 := &MoveInfo{
		Move:    "Q16",
		Winrate: 0.50,
	}

	position := &Position{
		BoardXSize: 19,
		BoardYSize: 19,
	}

	result := compareMove(move1, move2, position)

	// Should indicate move1 is better
	if !strings.Contains(result, "better") && !strings.Contains(result, "Prefers") {
		t.Errorf("Expected comparison to show move1 is better, got: %s", result)
	}

	// Test similar moves
	move2.Winrate = 0.515
	result = compareMove(move1, move2, position)
	if result != "Similar strength" {
		t.Errorf("Expected 'Similar strength' for close winrates, got: %s", result)
	}
}

func TestMoveExplanationStruct(t *testing.T) {
	explanation := MoveExplanation{
		Move:        "D4",
		Explanation: "This is the top choice",
		Winrate:     0.52,
		ScoreLead:   2.5,
		Visits:      1000,
		Pros:        []string{"Secures corner", "Natural move"},
		Cons:        []string{},
		Alternatives: []Alternative{
			{
				Move:      "Q16",
				Winrate:   0.51,
				Visits:    900,
				Reasoning: "Similar strength",
			},
		},
		Strategic: StrategicInfo{
			Purpose:       []string{"corner enclosure"},
			Urgency:       "important",
			BoardRegion:   "corner",
			TerritoryMove: true,
		},
	}

	if explanation.Move != "D4" {
		t.Errorf("Expected move D4, got %s", explanation.Move)
	}
	if len(explanation.Pros) != 2 {
		t.Errorf("Expected 2 pros, got %d", len(explanation.Pros))
	}
	if !explanation.Strategic.TerritoryMove {
		t.Error("Expected territory move for corner")
	}
	if len(explanation.Alternatives) != 1 {
		t.Errorf("Expected 1 alternative, got %d", len(explanation.Alternatives))
	}
}
