package katago

import (
	"testing"
)

func TestSGFParser(t *testing.T) {
	tests := []struct {
		name      string
		sgf       string
		wantMoves int
		wantKomi  float64
		wantRules string
		wantSize  int
	}{
		{
			name: "Basic game",
			sgf: `(;GM[1]FF[4]CA[UTF-8]SZ[19]KM[7.5]
				PB[Black Player]PW[White Player]
				;B[pd];W[dd];B[pp];W[dp])`,
			wantMoves: 4,
			wantKomi:  7.5,
			wantRules: "chinese",
			wantSize:  19,
		},
		{
			name: "Small board",
			sgf: `(;GM[1]FF[4]SZ[13]KM[5.5]RU[Japanese]
				;B[dd];W[jj])`,
			wantMoves: 2,
			wantKomi:  5.5,
			wantRules: "japanese",
			wantSize:  13,
		},
		{
			name: "With handicap stones",
			sgf: `(;GM[1]FF[4]SZ[19]HA[2]AB[pd][dp]KM[0.5]
				;W[dd];B[pp])`,
			wantMoves: 2,
			wantKomi:  0.5,
			wantRules: "chinese",
			wantSize:  19,
		},
		{
			name: "Korean rules",
			sgf: `(;GM[1]FF[4]SZ[19]KM[6.5]RU[Korean]
				;B[dd])`,
			wantMoves: 1,
			wantKomi:  6.5,
			wantRules: "korean",
			wantSize:  19,
		},
		{
			name: "With passes",
			sgf: `(;GM[1]FF[4]SZ[19]KM[7.5]
				;B[dd];W[];B[pp])`,
			wantMoves: 3,
			wantKomi:  7.5,
			wantRules: "chinese",
			wantSize:  19,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewSGFParser(tt.sgf)
			position, err := parser.Parse()
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if len(position.Moves) != tt.wantMoves {
				t.Errorf("Got %d moves, want %d", len(position.Moves), tt.wantMoves)
			}

			if position.Komi != tt.wantKomi {
				t.Errorf("Got komi %.1f, want %.1f", position.Komi, tt.wantKomi)
			}

			if position.Rules != tt.wantRules {
				t.Errorf("Got rules %s, want %s", position.Rules, tt.wantRules)
			}

			if position.BoardXSize != tt.wantSize {
				t.Errorf("Got board size %d, want %d", position.BoardXSize, tt.wantSize)
			}
		})
	}
}

func TestSGFCoordinateConversion(t *testing.T) {
	parser := NewSGFParser("")

	tests := []struct {
		sgfCoord string
		want     string
	}{
		{"aa", "A19"}, // Bottom-left corner
		{"sa", "T19"}, // Bottom-right corner (skipping I)
		{"as", "A1"},  // Top-left corner
		{"ss", "T1"},  // Top-right corner
		{"dd", "D16"}, // Standard corner stone
		{"pd", "Q16"}, // Standard corner stone
		{"jj", "K10"}, // Center (skipping I)
	}

	for _, tt := range tests {
		t.Run(tt.sgfCoord, func(t *testing.T) {
			got := parser.sgfToKataGo(tt.sgfCoord)
			if got != tt.want {
				t.Errorf("sgfToKataGo(%s) = %s, want %s", tt.sgfCoord, got, tt.want)
			}
		})
	}
}

func TestSGFValidation(t *testing.T) {
	validPosition := &Position{
		Rules:      "chinese",
		BoardXSize: 19,
		BoardYSize: 19,
		Moves: []Move{
			{Color: "b", Location: "D4"},
			{Color: "w", Location: "Q16"},
		},
		Komi: 7.5,
	}

	if err := ValidatePosition(validPosition); err != nil {
		t.Errorf("ValidatePosition() should pass for valid position: %v", err)
	}

	// Test invalid board size
	invalidSize := *validPosition
	invalidSize.BoardXSize = 26
	if err := ValidatePosition(&invalidSize); err == nil {
		t.Error("ValidatePosition() should fail for board size > 25")
	}

	// Test invalid rules
	invalidRules := *validPosition
	invalidRules.Rules = "invalid"
	if err := ValidatePosition(&invalidRules); err == nil {
		t.Error("ValidatePosition() should fail for invalid rules")
	}

	// Test invalid move color
	invalidColor := *validPosition
	invalidColor.Moves = []Move{{Color: "x", Location: "D4"}}
	if err := ValidatePosition(&invalidColor); err == nil {
		t.Error("ValidatePosition() should fail for invalid move color")
	}

	// Test invalid coordinate
	invalidCoord := *validPosition
	invalidCoord.Moves = []Move{{Color: "b", Location: "Z99"}}
	if err := ValidatePosition(&invalidCoord); err == nil {
		t.Error("ValidatePosition() should fail for invalid coordinate")
	}
}

func TestSGFComplexExamples(t *testing.T) {
	// Test with variations (should be ignored)
	sgfWithVariations := `(;GM[1]FF[4]SZ[19]KM[7.5]
		;B[dd];W[pp]
		(;B[pd];W[dp])
		(;B[qq];W[od]))`

	parser := NewSGFParser(sgfWithVariations)
	position, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse SGF with variations: %v", err)
	}

	// Should only parse main line
	if len(position.Moves) != 2 {
		t.Errorf("Expected 2 moves in main line, got %d", len(position.Moves))
	}

	// Test with comments
	sgfWithComments := `(;GM[1]FF[4]SZ[19]KM[7.5]C[Test game]
		;B[dd]C[Good opening move];W[pp]C[Standard response])`

	parser = NewSGFParser(sgfWithComments)
	position, err = parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse SGF with comments: %v", err)
	}

	if len(position.Moves) != 2 {
		t.Errorf("Expected 2 moves, got %d", len(position.Moves))
	}

	// Test empty SGF
	emptySGF := `(;GM[1]FF[4]SZ[19]KM[7.5])`
	parser = NewSGFParser(emptySGF)
	position, err = parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse empty SGF: %v", err)
	}

	if len(position.Moves) != 0 {
		t.Errorf("Expected 0 moves in empty game, got %d", len(position.Moves))
	}
}

func TestSGFErrorCases(t *testing.T) {
	testCases := []struct {
		name string
		sgf  string
	}{
		{"No opening parenthesis", "GM[1]FF[4]SZ[19];B[dd]"},
		{"Unclosed property", "(;GM[1]FF[4]SZ[19]B[dd"},
		{"Malformed property", "(;GM[1]FF[4]SZ[19];B)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewSGFParser(tc.sgf)
			_, err := parser.Parse()
			if err == nil {
				t.Errorf("Expected error for malformed SGF: %s", tc.name)
			}
		})
	}
}

func TestSGFPlayerToMove(t *testing.T) {
	// Test with explicit player to move
	sgfWithPlayer := `(;GM[1]FF[4]SZ[19]KM[7.5]PL[W]
		;B[dd];W[pp])`

	parser := NewSGFParser(sgfWithPlayer)
	position, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse SGF: %v", err)
	}

	// Should determine next player based on moves, not initial player
	if len(position.Moves) == 2 && position.InitialPlayer != "w" {
		// Initial player should be set to White as specified
		if position.InitialPlayer != "w" {
			t.Errorf("Expected initial player 'w', got '%s'", position.InitialPlayer)
		}
	}
}
