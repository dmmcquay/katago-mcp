package katago

import (
	"strings"
	"testing"
)

func TestParseCoord(t *testing.T) {
	tests := []struct {
		name      string
		coord     string
		boardSize int
		wantX     int
		wantY     int
	}{
		{
			name:      "top-left corner",
			coord:     "A19",
			boardSize: 19,
			wantX:     0,
			wantY:     0,
		},
		{
			name:      "bottom-right corner",
			coord:     "T1",
			boardSize: 19,
			wantX:     18,
			wantY:     18,
		},
		{
			name:      "center point (tengen)",
			coord:     "K10",
			boardSize: 19,
			wantX:     9,
			wantY:     9,
		},
		{
			name:      "skip I coordinate",
			coord:     "J10",
			boardSize: 19,
			wantX:     8,
			wantY:     9,
		},
		{
			name:      "9x9 center",
			coord:     "E5",
			boardSize: 9,
			wantX:     4,
			wantY:     4,
		},
		{
			name:      "pass move",
			coord:     "pass",
			boardSize: 19,
			wantX:     -1,
			wantY:     -1,
		},
		{
			name:      "empty move",
			coord:     "",
			boardSize: 19,
			wantX:     -1,
			wantY:     -1,
		},
		{
			name:      "invalid coordinate",
			coord:     "Z99",
			boardSize: 19,
			wantX:     -1,
			wantY:     -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotX, gotY := parseCoord(tt.coord, tt.boardSize)
			if gotX != tt.wantX || gotY != tt.wantY {
				t.Errorf("parseCoord(%s, %d) = (%d, %d), want (%d, %d)",
					tt.coord, tt.boardSize, gotX, gotY, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestCoordToString(t *testing.T) {
	tests := []struct {
		name      string
		x         int
		y         int
		boardSize int
		want      string
	}{
		{
			name:      "top-left corner",
			x:         0,
			y:         0,
			boardSize: 19,
			want:      "A19",
		},
		{
			name:      "bottom-right corner",
			x:         18,
			y:         18,
			boardSize: 19,
			want:      "T1",
		},
		{
			name:      "center point (tengen)",
			x:         9,
			y:         9,
			boardSize: 19,
			want:      "K10",
		},
		{
			name:      "skip I coordinate",
			x:         8,
			y:         9,
			boardSize: 19,
			want:      "J10",
		},
		{
			name:      "9x9 center",
			x:         4,
			y:         4,
			boardSize: 9,
			want:      "E5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := coordToString(tt.x, tt.y, tt.boardSize)
			if got != tt.want {
				t.Errorf("coordToString(%d, %d, %d) = %s, want %s",
					tt.x, tt.y, tt.boardSize, got, tt.want)
			}
		})
	}
}

func TestFindGroup(t *testing.T) {
	// Create a simple 9x9 board with a few stones
	board := make([][]string, 9)
	for i := range board {
		board[i] = make([]string, 9)
		for j := range board[i] {
			board[i][j] = "."
		}
	}

	// Place a black group
	board[2][2] = "B"
	board[2][3] = "B"
	board[3][2] = "B"

	// Place a white stone
	board[5][5] = "W"

	visited := make([][]bool, 9)
	for i := range visited {
		visited[i] = make([]bool, 9)
	}

	// Test finding the black group
	group := findGroup(2, 2, board, visited)
	if len(group) != 3 {
		t.Errorf("Expected group size 3, got %d", len(group))
	}

	// Reset visited
	for i := range visited {
		for j := range visited[i] {
			visited[i][j] = false
		}
	}

	// Test finding single white stone
	group = findGroup(5, 5, board, visited)
	if len(group) != 1 {
		t.Errorf("Expected group size 1, got %d", len(group))
	}

	// Test empty point
	for i := range visited {
		for j := range visited[i] {
			visited[i][j] = false
		}
	}
	group = findGroup(0, 0, board, visited)
	if len(group) != 0 {
		t.Errorf("Expected empty group for empty point, got %d stones", len(group))
	}
}

func TestIsGroupDead(t *testing.T) {
	territoryMap := &TerritoryMap{
		Ownership: make([][]float64, 9),
	}

	// Initialize ownership map - positive is black, negative is white
	// C9-C6 (y=0-3) will be black territory
	// C5-C1 (y=4-8) will be white territory
	for i := range territoryMap.Ownership {
		territoryMap.Ownership[i] = make([]float64, 9)
		for j := range territoryMap.Ownership[i] {
			if i < 5 {
				territoryMap.Ownership[i][j] = 0.9 // Strong black territory (top)
			} else {
				territoryMap.Ownership[i][j] = -0.9 // Strong white territory (bottom)
			}
		}
	}

	tests := []struct {
		name      string
		group     []string
		color     string
		threshold float64
		wantDead  bool
	}{
		{
			name:      "white stone in black territory",
			group:     []string{"C6"}, // C6 maps to y=3, which is in black territory
			color:     "W",
			threshold: 0.85,
			wantDead:  true,
		},
		{
			name:      "black stone in white territory",
			group:     []string{"C3"}, // C3 maps to y=6, which is in white territory
			color:     "B",
			threshold: 0.85,
			wantDead:  true,
		},
		{
			name:      "black stone in black territory",
			group:     []string{"C6"}, // C6 maps to y=3, which is in black territory
			color:     "B",
			threshold: 0.85,
			wantDead:  false,
		},
		{
			name:      "empty group",
			group:     []string{},
			color:     "B",
			threshold: 0.85,
			wantDead:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGroupDead(tt.group, tt.color, territoryMap, tt.threshold)
			if got != tt.wantDead {
				t.Errorf("isGroupDead() = %v, want %v", got, tt.wantDead)
			}
		})
	}
}

func TestGetTerritoryVisualization(t *testing.T) {
	estimate := &TerritoryEstimate{
		Map: &TerritoryMap{
			Territory: [][]string{
				{"B", "B", "?", "W", "W"},
				{"B", "B", "?", "W", "W"},
				{"?", "?", "?", "?", "?"},
				{"B", "B", "?", "W", "W"},
				{"B", "B", "?", "W", "W"},
			},
		},
		BlackTerritory: 8,
		WhiteTerritory: 8,
		DamePoints:     9,
		ScoreString:    "B+0.5",
	}

	viz := GetTerritoryVisualization(estimate)

	// Check that visualization contains expected elements
	if !strings.Contains(viz, "●") {
		t.Error("Visualization should contain black territory markers (●)")
	}
	if !strings.Contains(viz, "○") {
		t.Error("Visualization should contain white territory markers (○)")
	}
	if !strings.Contains(viz, "·") {
		t.Error("Visualization should contain dame point markers (·)")
	}
	if !strings.Contains(viz, "Black territory: 8") {
		t.Error("Visualization should show black territory count")
	}
	if !strings.Contains(viz, "White territory: 8") {
		t.Error("Visualization should show white territory count")
	}
	if !strings.Contains(viz, "Score: B+0.5") {
		t.Error("Visualization should show score")
	}

	// Test nil map
	emptyEstimate := &TerritoryEstimate{}
	viz = GetTerritoryVisualization(emptyEstimate)
	if viz != "No territory data available" {
		t.Errorf("Expected 'No territory data available', got %s", viz)
	}
}

func TestTerritoryEstimateStruct(t *testing.T) {
	// Test that TerritoryEstimate struct can be properly created
	estimate := TerritoryEstimate{
		Map: &TerritoryMap{
			Territory:  [][]string{{"B", "W", "?"}},
			Ownership:  [][]float64{{0.9, -0.9, 0.1}},
			DeadStones: []string{"D4", "Q16"},
		},
		BlackTerritory: 180,
		WhiteTerritory: 180,
		DamePoints:     1,
		ScoreEstimate:  -6.5,
		ScoreString:    "W+6.5",
	}

	if estimate.BlackTerritory != 180 {
		t.Errorf("Expected black territory 180, got %d", estimate.BlackTerritory)
	}
	if estimate.ScoreString != "W+6.5" {
		t.Errorf("Expected score string 'W+6.5', got %s", estimate.ScoreString)
	}
	if len(estimate.Map.DeadStones) != 2 {
		t.Errorf("Expected 2 dead stones, got %d", len(estimate.Map.DeadStones))
	}
}
