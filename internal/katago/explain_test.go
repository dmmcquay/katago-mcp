package katago

import (
	"strings"
	"testing"
)

func TestDetermineBoardRegion(t *testing.T) {
	tests := []struct {
		name      string
		move      string
		boardSize int
		want      string
	}{
		{
			name:      "3-3 corner",
			move:      "C3",
			boardSize: 19,
			want:      "corner",
		},
		{
			name:      "4-4 corner",
			move:      "D4",
			boardSize: 19,
			want:      "corner",
		},
		{
			name:      "star point corner",
			move:      "Q16",
			boardSize: 19,
			want:      "corner",
		},
		{
			name:      "side move",
			move:      "K3",
			boardSize: 19,
			want:      "side",
		},
		{
			name:      "center move",
			move:      "K10",
			boardSize: 19,
			want:      "center",
		},
		{
			name:      "tengen",
			move:      "J10",
			boardSize: 19,
			want:      "center",
		},
		{
			name:      "pass",
			move:      "pass",
			boardSize: 19,
			want:      "pass",
		},
		{
			name:      "9x9 corner",
			move:      "C3",
			boardSize: 9,
			want:      "corner",
		},
		{
			name:      "9x9 center",
			move:      "E5",
			boardSize: 9,
			want:      "center",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineBoardRegion(tt.move, tt.boardSize)
			if got != tt.want {
				t.Errorf("determineBoardRegion(%s, %d) = %s, want %s",
					tt.move, tt.boardSize, got, tt.want)
			}
		})
	}
}

func TestIsNearStones(t *testing.T) {
	position := &Position{
		BoardXSize: 19,
		BoardYSize: 19,
		Moves: []Move{
			{Color: "b", Location: "D4"},
			{Color: "w", Location: "Q16"},
			{Color: "b", Location: "D16"},
			{Color: "w", Location: "Q4"},
		},
	}

	tests := []struct {
		name string
		x    int
		y    int
		want bool
	}{
		{
			name: "next to D4",
			x:    3,
			y:    14, // D5
			want: true,
		},
		{
			name: "3 points from D4",
			x:    3,
			y:    12, // D7
			want: true,
		},
		{
			name: "4 points from D4",
			x:    3,
			y:    11, // D8
			want: false,
		},
		{
			name: "center with no nearby stones",
			x:    9,
			y:    9, // K10
			want: false,
		},
		{
			name: "diagonal from Q16",
			x:    14,
			y:    4, // O14
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNearStones(tt.x, tt.y, position)
			if got != tt.want {
				t.Errorf("isNearStones(%d, %d) = %v, want %v",
					tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestAnalyzeStrategicPurpose(t *testing.T) {
	// Test opening corner move
	position := &Position{
		BoardXSize: 19,
		BoardYSize: 19,
		Moves:      []Move{},
	}

	moveInfo := &MoveInfo{
		Move:    "D4",
		Winrate: 0.52,
	}

	result := &AnalysisResult{
		MoveInfos: []MoveInfo{
			*moveInfo,
			{Move: "Q16", Winrate: 0.51},
		},
	}

	info := analyzeStrategicPurpose("D4", position, moveInfo, result)

	if info.BoardRegion != "corner" {
		t.Errorf("Expected corner region, got %s", info.BoardRegion)
	}

	if !info.TerritoryMove {
		t.Error("Opening corner move should be territory-oriented")
	}

	if len(info.Purpose) == 0 {
		t.Error("Should have at least one purpose")
	}

	// Test fighting move
	position2 := &Position{
		BoardXSize: 19,
		BoardYSize: 19,
		Moves: []Move{
			{Color: "b", Location: "D4"},
			{Color: "w", Location: "D5"},
		},
	}

	info2 := analyzeStrategicPurpose("E4", position2, moveInfo, result)
	if !info2.FightingMove {
		t.Error("Move near existing stones should be fighting move")
	}
}

func TestGeneratePros(t *testing.T) {
	moveInfo := &MoveInfo{
		Move:      "D4",
		Winrate:   0.58,
		ScoreLead: 5.5,
	}

	strategic := StrategicInfo{
		TerritoryMove: true,
		Urgency:       "critical",
	}

	pros := generatePros(moveInfo, 1, strategic)

	// Should have multiple pros
	if len(pros) < 2 {
		t.Errorf("Expected multiple pros for a good move, got %d: %v", len(pros), pros)
	}

	// Check for expected pros
	hasWinratePro := false
	hasUrgencyPro := false
	hasTerritoryPro := false
	for _, pro := range pros {
		if strings.Contains(pro, "win rate") {
			hasWinratePro = true
		}
		if strings.Contains(pro, "Critical") || strings.Contains(pro, "prevents") {
			hasUrgencyPro = true
		}
		if strings.Contains(pro, "Secures") || strings.Contains(pro, "points") {
			hasTerritoryPro = true
		}
	}

	if !hasWinratePro {
		t.Error("Expected pro about win rate")
	}
	if !hasUrgencyPro {
		t.Error("Expected pro about urgency")
	}
	if !hasTerritoryPro {
		t.Error("Expected pro about territory since TerritoryMove is true")
	}
}

func TestGenerateCons(t *testing.T) {
	moveInfo := &MoveInfo{
		Move:      "K10",
		Winrate:   0.48,
		ScoreLead: -2.5,
	}

	allMoves := []MoveInfo{
		{Move: "D4", Winrate: 0.52},
		*moveInfo,
	}

	strategic := StrategicInfo{
		BoardRegion: "center",
		Urgency:     "optional",
	}

	cons := generateCons(moveInfo, 2, allMoves, strategic)

	// Should have cons for suboptimal move
	if len(cons) == 0 {
		t.Error("Expected cons for suboptimal move")
	}

	// Check for win rate loss con
	hasWinrateLoss := false
	for _, con := range cons {
		if strings.Contains(con, "win rate") {
			hasWinrateLoss = true
			break
		}
	}

	if !hasWinrateLoss {
		t.Error("Expected con about win rate loss")
	}
}

func TestFindAlternatives(t *testing.T) {
	chosen := &MoveInfo{
		Move:      "K10",
		Winrate:   0.48,
		ScoreLead: 0.0,
	}

	allMoves := []MoveInfo{
		{Move: "D4", Winrate: 0.52, ScoreLead: 2.5},
		{Move: "Q16", Winrate: 0.51, ScoreLead: 2.0},
		*chosen,
		{Move: "D16", Winrate: 0.47, ScoreLead: -0.5},
		{Move: "pass", Winrate: 0.20, ScoreLead: -10.0},
	}

	position := &Position{
		BoardXSize: 19,
		BoardYSize: 19,
	}

	alternatives := findAlternatives(chosen, allMoves, position)

	// Should have alternatives (but limited to top moves)
	if len(alternatives) == 0 {
		t.Error("Expected some alternatives")
	}

	// First alternative should be best move
	if len(alternatives) > 0 && alternatives[0].Move != "D4" {
		t.Errorf("Expected D4 as first alternative, got %s", alternatives[0].Move)
	}

	// Check win rate differences
	if len(alternatives) > 0 && alternatives[0].WinrateDiff <= 0 {
		t.Error("First alternative should have positive win rate difference")
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test intPtr
	ptr := intPtr(42)
	if ptr == nil || *ptr != 42 {
		t.Error("intPtr should return pointer to int")
	}

	// Test minInt
	if minInt(5, 3) != 3 {
		t.Error("minInt(5, 3) should be 3")
	}
	if minInt(2, 7) != 2 {
		t.Error("minInt(2, 7) should be 2")
	}

	// Test abs
	if abs(5) != 5 {
		t.Error("abs(5) should be 5")
	}
	if abs(-5) != 5 {
		t.Error("abs(-5) should be 5")
	}
	if abs(0) != 0 {
		t.Error("abs(0) should be 0")
	}
}
