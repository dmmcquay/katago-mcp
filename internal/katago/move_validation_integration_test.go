//go:build integration
// +build integration

package katago

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMoveValidation_Integration tests move validation functionality
// This doesn't require KataGo to be running
func TestMoveValidation_Integration(t *testing.T) {
	tests := []struct {
		name      string
		move      string
		boardSize int
		expected  bool
	}{
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
			name:      "invalid SGF format",
			move:      "dd",
			boardSize: 19,
			expected:  false,
		},
		{
			name:      "invalid I column",
			move:      "I5",
			boardSize: 19,
			expected:  false,
		},
		{
			name:      "valid edge move",
			move:      "A1",
			boardSize: 19,
			expected:  true,
		},
		{
			name:      "valid T19 corner",
			move:      "T19",
			boardSize: 19,
			expected:  true,
		},
		{
			name:      "out of bounds row",
			move:      "A20",
			boardSize: 19,
			expected:  false,
		},
		{
			name:      "out of bounds column",
			move:      "U1",
			boardSize: 19,
			expected:  false,
		},
		{
			name:      "valid 9x9 center",
			move:      "E5",
			boardSize: 9,
			expected:  true,
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

// TestPolicyDecoding_Integration tests policy array decoding
func TestPolicyDecoding_Integration(t *testing.T) {
	boardSize := 19

	// Test specific indices
	tests := []struct {
		index    int
		expected string
	}{
		{0, "A19"},    // Top-left
		{18, "T19"},   // Top-right
		{342, "A1"},   // Bottom-left
		{360, "T1"},   // Bottom-right
		{180, "K10"},  // Row 9 (10 from bottom), Col 9 (K)
		{190, "A9"},   // Row 10 (9 from bottom), Col 0 (A)
		{361, "pass"}, // Pass move
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := indexToCoordinate(tt.index, boardSize)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCoordinateRoundTrip_Integration tests conversion both ways
func TestCoordinateRoundTrip_Integration(t *testing.T) {
	// Test that SGF to KataGo conversion works correctly
	parser := &SGFParser{boardSize: 19}

	tests := []struct {
		sgf      string
		expected string
	}{
		{"aa", "A19"},
		{"ss", "T1"},
		{"dd", "D16"},
		{"jj", "K10"},
		{"pd", "Q16"},
		{"dp", "D4"},
	}

	for _, tt := range tests {
		t.Run(tt.sgf, func(t *testing.T) {
			result := parser.sgfToKataGo(tt.sgf)
			assert.Equal(t, tt.expected, result)
		})
	}
}
