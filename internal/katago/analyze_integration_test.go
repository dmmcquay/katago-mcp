//go:build integration
// +build integration

package katago

import (
	"context"
	"testing"

	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzePosition_Integration(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create test engine
	cfg := &config.KataGoConfig{
		BinaryPath: "/usr/local/bin/katago", // Adjust as needed
		ConfigPath: "",
		ModelPath:  "",
	}
	logger := logging.NewTestLogger(t)
	engine := NewEngine(cfg, logger, nil)

	ctx := context.Background()
	err := engine.Start(ctx)
	require.NoError(t, err)
	defer engine.Stop(ctx)

	tests := []struct {
		name    string
		request *AnalysisRequest
		check   func(t *testing.T, result *AnalysisResult)
	}{
		{
			name: "analyze empty board with policy",
			request: &AnalysisRequest{
				Position: &Position{
					Rules:      "chinese",
					BoardXSize: 19,
					BoardYSize: 19,
					Komi:       6.5,
					Moves:      []Move{},
				},
				IncludePolicy: true,
				MaxVisits:     intPtr(100),
			},
			check: func(t *testing.T, result *AnalysisResult) {
				// Should have move suggestions
				assert.NotEmpty(t, result.MoveInfos)

				// Should have policy array
				assert.NotNil(t, result.Policy)
				assert.Len(t, result.Policy, 19*19+1) // 361 + pass

				// Check policy decoding works
				formatted := FormatAnalysisResult(result, true, 19)
				assert.Contains(t, formatted, "Policy Network")
				assert.Contains(t, formatted, "Top policy moves:")
			},
		},
		{
			name: "reject invalid move format",
			request: &AnalysisRequest{
				Position: &Position{
					Rules:      "chinese",
					BoardXSize: 19,
					BoardYSize: 19,
					Komi:       6.5,
					Moves: []Move{
						{Color: "b", Location: "dd"}, // SGF format
					},
				},
			},
			check: func(t *testing.T, result *AnalysisResult) {
				// Should fail before getting here
				t.Fatal("Should have failed with invalid move format")
			},
		},
		{
			name: "analyze position with moves",
			request: &AnalysisRequest{
				Position: &Position{
					Rules:      "chinese",
					BoardXSize: 19,
					BoardYSize: 19,
					Komi:       6.5,
					Moves: []Move{
						{Color: "b", Location: "D4"},
						{Color: "w", Location: "Q16"},
						{Color: "b", Location: "D16"},
						{Color: "w", Location: "Q4"},
					},
				},
				MaxVisits: intPtr(50),
			},
			check: func(t *testing.T, result *AnalysisResult) {
				// Should have move suggestions
				assert.NotEmpty(t, result.MoveInfos)

				// Current player should be black (after 4 moves)
				assert.Equal(t, "B", result.RootInfo.CurrentPlayer)

				// All suggested moves should be valid format
				for _, moveInfo := range result.MoveInfos {
					assert.True(t, isValidMoveFormat(moveInfo.Move, 19))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Analyze(ctx, tt.request)

			if tt.name == "reject invalid move format" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid move format")
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			tt.check(t, result)
		})
	}
}

func intPtr(i int) *int {
	return &i
}
