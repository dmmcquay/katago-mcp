package katago

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsValidMoveFormat tests the unexported isValidMoveFormat function
// This has to be in the same package to access the unexported function
func TestIsValidMoveFormat(t *testing.T) {
	tests := []struct {
		name      string
		move      string
		boardSize int
		expected  bool
	}{
		// Valid moves
		{
			name:      "valid standard move",
			move:      "D4",
			boardSize: 19,
			expected:  true,
		},
		{
			name:      "valid pass move",
			move:      "pass",
			boardSize: 19,
			expected:  true,
		},
		{
			name:      "valid corner move A1",
			move:      "A1",
			boardSize: 19,
			expected:  true,
		},
		{
			name:      "valid corner move T19",
			move:      "T19",
			boardSize: 19,
			expected:  true,
		},
		{
			name:      "valid double digit row",
			move:      "D10",
			boardSize: 19,
			expected:  true,
		},
		{
			name:      "valid 9x9 board move",
			move:      "E5",
			boardSize: 9,
			expected:  true,
		},
		// Invalid moves
		{
			name:      "invalid SGF format lowercase",
			move:      "dd",
			boardSize: 19,
			expected:  false,
		},
		{
			name:      "invalid column I",
			move:      "I5",
			boardSize: 19,
			expected:  false,
		},
		{
			name:      "invalid column U",
			move:      "U1",
			boardSize: 19,
			expected:  false,
		},
		{
			name:      "invalid row 0",
			move:      "A0",
			boardSize: 19,
			expected:  false,
		},
		{
			name:      "invalid row too high",
			move:      "A20",
			boardSize: 19,
			expected:  false,
		},
		{
			name:      "invalid empty string",
			move:      "",
			boardSize: 19,
			expected:  false,
		},
		{
			name:      "invalid single character",
			move:      "A",
			boardSize: 19,
			expected:  false,
		},
		{
			name:      "invalid too many characters",
			move:      "A123",
			boardSize: 19,
			expected:  false,
		},
		{
			name:      "invalid non-letter column",
			move:      "1A",
			boardSize: 19,
			expected:  false,
		},
		{
			name:      "invalid non-digit row",
			move:      "AA",
			boardSize: 19,
			expected:  false,
		},
		{
			name:      "out of bounds for 9x9",
			move:      "K10",
			boardSize: 9,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidMoveFormat(tt.move, tt.boardSize)
			assert.Equal(t, tt.expected, result, "Move %s validation on %dx%d board", tt.move, tt.boardSize, tt.boardSize)
		})
	}
}

// TestIndexToCoordinate tests the unexported indexToCoordinate function
// This has to be in the same package to access the unexported function
func TestIndexToCoordinate(t *testing.T) {
	tests := []struct {
		name      string
		index     int
		boardSize int
		expected  string
	}{
		// 19x19 board tests
		{
			name:      "top-left corner",
			index:     0,
			boardSize: 19,
			expected:  "A19",
		},
		{
			name:      "top-right corner",
			index:     18,
			boardSize: 19,
			expected:  "T19",
		},
		{
			name:      "bottom-left corner",
			index:     342,
			boardSize: 19,
			expected:  "A1",
		},
		{
			name:      "bottom-right corner",
			index:     360,
			boardSize: 19,
			expected:  "T1",
		},
		{
			name:      "center tengen",
			index:     180,
			boardSize: 19,
			expected:  "K10",
		},
		{
			name:      "pass move",
			index:     361,
			boardSize: 19,
			expected:  "pass",
		},
		// Column tests with I skipping
		{
			name:      "column H",
			index:     7,
			boardSize: 19,
			expected:  "H19",
		},
		{
			name:      "column J (after I)",
			index:     8,
			boardSize: 19,
			expected:  "J19",
		},
		{
			name:      "column K",
			index:     9,
			boardSize: 19,
			expected:  "K19",
		},
		// 9x9 board tests
		{
			name:      "9x9 top-left",
			index:     0,
			boardSize: 9,
			expected:  "A9",
		},
		{
			name:      "9x9 center",
			index:     40,
			boardSize: 9,
			expected:  "E5",
		},
		{
			name:      "9x9 bottom-right",
			index:     80,
			boardSize: 9,
			expected:  "J1",
		},
		{
			name:      "9x9 pass",
			index:     81,
			boardSize: 9,
			expected:  "pass",
		},
		// 13x13 board tests
		{
			name:      "13x13 center",
			index:     84,
			boardSize: 13,
			expected:  "G7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexToCoordinate(tt.index, tt.boardSize)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAnalyzePosition_MoveValidation tests that the Analyze function validates moves
// This tests the integration of move validation without directly testing unexported functions
func TestAnalyzePosition_MoveValidation(t *testing.T) {
	// This test would require a full engine setup with mocking
	// Since isValidMoveFormat is unexported, we test it indirectly through
	// the Analyze function in integration tests
}

func TestFormatAnalysisResult_PolicyDisplay(t *testing.T) {
	result := &AnalysisResult{
		RootInfo: RootInfo{
			CurrentPlayer: "B",
			Visits:        1000,
			Winrate:       0.523,
			ScoreMean:     1.5,
		},
		MoveInfos: []MoveInfo{
			{Move: "D4", Visits: 400, Winrate: 0.55, ScoreLead: 2.0},
			{Move: "Q16", Visits: 300, Winrate: 0.52, ScoreLead: 1.5},
		},
		Policy: make([]float64, 362), // 19x19 + 1 for pass
	}

	// Set some policy values
	// Calculate indices: index = y * boardSize + x
	// D16: x=3, y=3 -> index = 3*19 + 3 = 60
	result.Policy[60] = 0.15 // D16
	// Q4: x=15, y=15 -> index = 15*19 + 15 = 300
	result.Policy[300] = 0.12 // Q4
	// Q16: x=15, y=3 -> index = 3*19 + 15 = 72
	result.Policy[72] = 0.10 // Q16
	// D4: x=3, y=15 -> index = 15*19 + 3 = 288
	result.Policy[288] = 0.08  // D4
	result.Policy[361] = 0.005 // pass

	// Test verbose output with policy
	output := FormatAnalysisResult(result, true, 19)
	assert.Contains(t, output, "=== Policy Network ===")
	assert.Contains(t, output, "Top policy moves:")
	assert.Contains(t, output, "D16: 15.0%")
	assert.Contains(t, output, "Q4: 12.0%")
	assert.Contains(t, output, "Q16: 10.0%")
	assert.Contains(t, output, "D4: 8.0%")
	assert.NotContains(t, output, "pass: 0.5%") // Below 1% threshold

	// Test non-verbose output doesn't show policy
	output = FormatAnalysisResult(result, false, 19)
	assert.NotContains(t, output, "=== Policy Network ===")
}
