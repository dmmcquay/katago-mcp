package katago

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzePosition_MoveFormatUnmarshaling(t *testing.T) {
	tests := []struct {
		name          string
		responseJSON  string
		expectedMoves []string
		expectedError bool
	}{
		{
			name: "standard move format",
			responseJSON: `{
				"id": "test1",
				"turnNumber": 1,
				"moveInfos": [
					{"move": "D4", "visits": 100, "winrate": 0.55, "scoreLead": 0.2, "scoreMean": 1.5, "prior": 0.25, "pv": ["D4", "Q16"]},
					{"move": "Q16", "visits": 80, "winrate": 0.54, "scoreLead": 0.1, "scoreMean": 1.2, "prior": 0.20, "pv": ["Q16", "D4"]},
					{"move": "pass", "visits": 5, "winrate": 0.30, "scoreLead": -5.0, "scoreMean": -10.0, "prior": 0.01, "pv": ["pass"]}
				],
				"rootInfo": {
					"visits": 185,
					"winrate": 0.55,
					"scoreLead": 0.2,
					"scoreMean": 1.5,
					"scoreStdev": 0.5,
					"currentPlayer": "B"
				}
			}`,
			expectedMoves: []string{"D4", "Q16", "pass"},
			expectedError: false,
		},
		{
			name: "lowercase sgf format (should fail or be converted)",
			responseJSON: `{
				"id": "test2",
				"turnNumber": 1,
				"moveInfos": [
					{"move": "dd", "visits": 100, "winrate": 0.55, "scoreLead": 0.2, "scoreMean": 1.5, "prior": 0.25, "pv": ["dd", "pd"]}
				],
				"rootInfo": {
					"visits": 100,
					"winrate": 0.55,
					"scoreLead": 0.2,
					"scoreMean": 1.5,
					"scoreStdev": 0.5,
					"currentPlayer": "B"
				}
			}`,
			expectedMoves: []string{"dd"}, // This might need conversion to "D4"
			expectedError: false,
		},
		{
			name: "edge coordinates",
			responseJSON: `{
				"id": "test3",
				"turnNumber": 1,
				"moveInfos": [
					{"move": "A1", "visits": 50, "winrate": 0.45, "scoreLead": -0.5, "scoreMean": -1.0, "prior": 0.10, "pv": ["A1"]},
					{"move": "T19", "visits": 40, "winrate": 0.44, "scoreLead": -0.6, "scoreMean": -1.1, "prior": 0.08, "pv": ["T19"]}
				],
				"rootInfo": {
					"visits": 90,
					"winrate": 0.45,
					"scoreLead": -0.5,
					"scoreMean": -1.0,
					"scoreStdev": 0.3,
					"currentPlayer": "W"
				}
			}`,
			expectedMoves: []string{"A1", "T19"},
			expectedError: false,
		},
		{
			name: "invalid move format",
			responseJSON: `{
				"id": "test4",
				"turnNumber": 1,
				"moveInfos": [
					{"move": "invalid", "visits": 100, "winrate": 0.55, "scoreLead": 0.2, "scoreMean": 1.5, "prior": 0.25, "pv": ["invalid"]}
				],
				"rootInfo": {
					"visits": 100,
					"winrate": 0.55,
					"scoreLead": 0.2,
					"scoreMean": 1.5,
					"scoreStdev": 0.5,
					"currentPlayer": "B"
				}
			}`,
			expectedMoves: []string{"invalid"},
			expectedError: false, // Currently doesn't validate move format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response Response
			err := json.Unmarshal([]byte(tt.responseJSON), &response)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, response.MoveInfos, len(tt.expectedMoves))

			for i, moveInfo := range response.MoveInfos {
				assert.Equal(t, tt.expectedMoves[i], moveInfo.Move)
			}
		})
	}
}

func TestAnalyzePosition_PolicyDecoding(t *testing.T) {
	// Test decoding policy array to move coordinates
	// Policy is a flat array of size boardYSize * boardXSize + 1
	// Last element is pass probability

	boardSize := 19
	policySize := boardSize*boardSize + 1

	// Create a sample policy array with some non-zero values
	policy := make([]float64, policySize)

	// Set some move probabilities
	// D4 (3,3) in 0-indexed = index 3*19+3 = 60
	policy[60] = 0.25
	// Q16 (15,2) in 0-indexed = index 2*19+15 = 53
	policy[53] = 0.20
	// Pass move is last
	policy[policySize-1] = 0.01

	// Test coordinate conversion
	tests := []struct {
		index    int
		expected string
	}{
		{60, "D16"},   // (3,3) from top = D16
		{53, "Q17"},   // (15,2) from top = Q17
		{0, "A19"},    // (0,0) - top-left
		{18, "T19"},   // (18,0) - top-right
		{342, "A1"},   // (0,18) - bottom-left
		{360, "T1"},   // (18,18) - bottom-right
		{361, "pass"}, // Last index is pass
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			coord := indexToCoordinate(tt.index, boardSize)
			assert.Equal(t, tt.expected, coord)
		})
	}
}

func TestMoveFormatValidation(t *testing.T) {
	tests := []struct {
		move    string
		isValid bool
	}{
		{"D4", true},
		{"Q16", true},
		{"A1", true},
		{"T19", true},
		{"pass", true},
		{"dd", false},  // SGF format, should be uppercase
		{"I1", false},  // 'I' is skipped in Go coordinates
		{"U1", false},  // Out of bounds for 19x19
		{"A0", false},  // Row 0 doesn't exist
		{"A20", false}, // Row 20 doesn't exist for 19x19
		{"", false},    // Empty move
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.move, func(t *testing.T) {
			isValid := isValidMoveFormat(tt.move, 19)
			assert.Equal(t, tt.isValid, isValid, "Move %s validation", tt.move)
		})
	}
}
