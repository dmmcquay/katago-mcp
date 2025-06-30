package katago

import (
	"strings"
	"testing"
)

func TestParseCoordinate(t *testing.T) {
	tests := []struct {
		name      string
		coord     string
		boardSize int
		wantX     int
		wantY     int
	}{
		{
			name:      "A1 corner",
			coord:     "A1",
			boardSize: 19,
			wantX:     0,
			wantY:     18,
		},
		{
			name:      "T19 corner",
			coord:     "T19",
			boardSize: 19,
			wantX:     18,
			wantY:     0,
		},
		{
			name:      "K10 center",
			coord:     "K10",
			boardSize: 19,
			wantX:     9, // K is 10th letter, skipping I
			wantY:     9,
		},
		{
			name:      "D4 corner approach",
			coord:     "D4",
			boardSize: 19,
			wantX:     3,
			wantY:     15,
		},
		{
			name:      "invalid coordinate",
			coord:     "X99",
			boardSize: 19,
			wantX:     -1,
			wantY:     -1,
		},
		{
			name:      "9x9 board center",
			coord:     "E5",
			boardSize: 9,
			wantX:     4,
			wantY:     4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := parseCoordinate(tt.coord, tt.boardSize)
			if x != tt.wantX || y != tt.wantY {
				t.Errorf("parseCoordinate(%s, %d) = (%d, %d), want (%d, %d)",
					tt.coord, tt.boardSize, x, y, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestFormatCoordinate(t *testing.T) {
	tests := []struct {
		name      string
		x         int
		y         int
		boardSize int
		want      string
	}{
		{
			name:      "A1 corner",
			x:         0,
			y:         18,
			boardSize: 19,
			want:      "A1",
		},
		{
			name:      "T19 corner",
			x:         18,
			y:         0,
			boardSize: 19,
			want:      "T19",
		},
		{
			name:      "K10 center",
			x:         9,
			y:         9,
			boardSize: 19,
			want:      "K10",
		},
		{
			name:      "after I column",
			x:         8,
			y:         9,
			boardSize: 19,
			want:      "J10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCoordinate(tt.x, tt.y, tt.boardSize)
			if got != tt.want {
				t.Errorf("formatCoordinate(%d, %d, %d) = %s, want %s",
					tt.x, tt.y, tt.boardSize, got, tt.want)
			}
		})
	}
}

func TestFormatScore(t *testing.T) {
	tests := []struct {
		name  string
		score float64
		want  string
	}{
		{
			name:  "black wins by 7.5",
			score: 7.5,
			want:  "B+7.5",
		},
		{
			name:  "white wins by 2.5",
			score: -2.5,
			want:  "W+2.5",
		},
		{
			name:  "jigo",
			score: 0.0,
			want:  "Jigo (Draw)",
		},
		{
			name:  "near jigo rounds to jigo",
			score: 0.3,
			want:  "Jigo (Draw)",
		},
		{
			name:  "black wins by 0.5",
			score: 0.6,
			want:  "B+0.5",
		},
		{
			name:  "rounding test",
			score: 7.3,
			want:  "B+7.5",
		},
		{
			name:  "rounding test 2",
			score: 7.2,
			want:  "B+7.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatScore(tt.score)
			if got != tt.want {
				t.Errorf("formatScore(%f) = %s, want %s", tt.score, got, tt.want)
			}
		})
	}
}

func TestIdentifyDeadStones(t *testing.T) {
	// Create a simple position with stones
	position := &Position{
		BoardXSize: 9,
		BoardYSize: 9,
		InitialStones: []Stone{
			{Color: "b", Location: "A1"},
			{Color: "w", Location: "B1"},
			{Color: "w", Location: "A2"},
			{Color: "b", Location: "H8"},
			{Color: "w", Location: "J9"},
		},
		Moves: []Move{},
	}

	// Create territory map where A1 area is white territory
	territoryMap := &TerritoryMap{
		BoardXSize: 9,
		BoardYSize: 9,
		Territory:  make([][]string, 9),
	}

	// Initialize territory
	for y := 0; y < 9; y++ {
		territoryMap.Territory[y] = make([]string, 9)
		for x := 0; x < 9; x++ {
			// Bottom left is white territory
			if x < 3 && y > 5 {
				territoryMap.Territory[y][x] = "W"
			} else if x > 5 && y < 3 {
				// Top right is black territory
				territoryMap.Territory[y][x] = "B"
			} else {
				territoryMap.Territory[y][x] = "?"
			}
		}
	}

	blackDead, whiteDead := identifyDeadStones(position, territoryMap)

	// A1 black stone should be dead (in white territory)
	if len(blackDead) != 1 || blackDead[0] != "A1" {
		t.Errorf("Expected black dead stones [A1], got %v", blackDead)
	}

	// J9 white stone should be dead (in black territory)
	if len(whiteDead) != 1 || whiteDead[0] != "J9" {
		t.Errorf("Expected white dead stones [J9], got %v", whiteDead)
	}
}

func TestGetTerritoryVisualization(t *testing.T) {
	estimate := &TerritoryEstimate{
		Map: &TerritoryMap{
			BoardXSize: 9,
			BoardYSize: 9,
			Territory: [][]string{
				{"B", "B", "B", "?", "?", "?", "W", "W", "W"},
				{"B", "B", "B", "?", "?", "?", "W", "W", "W"},
				{"B", "B", "B", "?", "?", "?", "W", "W", "W"},
				{"?", "?", "?", "?", "?", "?", "?", "?", "?"},
				{"?", "?", "?", "?", "?", "?", "?", "?", "?"},
				{"?", "?", "?", "?", "?", "?", "?", "?", "?"},
				{"B", "B", "B", "?", "?", "?", "W", "W", "W"},
				{"B", "B", "B", "?", "?", "?", "W", "W", "W"},
				{"B", "B", "B", "?", "?", "?", "W", "W", "W"},
			},
		},
		BlackTerritory: 27,
		WhiteTerritory: 27,
		DamePoints:     27,
		ScoreString:    "Jigo (Draw)",
	}

	viz := GetTerritoryVisualization(estimate)

	// Check that visualization contains expected elements
	if !strings.Contains(viz, "A B C D E F G H J") {
		t.Error("Visualization should contain column labels")
	}
	if !strings.Contains(viz, "●") {
		t.Error("Visualization should contain black territory markers")
	}
	if !strings.Contains(viz, "○") {
		t.Error("Visualization should contain white territory markers")
	}
	if !strings.Contains(viz, "·") {
		t.Error("Visualization should contain neutral point markers")
	}
	if !strings.Contains(viz, "Black territory: 27") {
		t.Error("Visualization should show black territory count")
	}
	if !strings.Contains(viz, "White territory: 27") {
		t.Error("Visualization should show white territory count")
	}
	if !strings.Contains(viz, "Score: Jigo (Draw)") {
		t.Error("Visualization should show score")
	}
}
