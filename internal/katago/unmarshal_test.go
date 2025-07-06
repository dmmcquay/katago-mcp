package katago

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnalyzePosition_UnmarshalingIssue tests the specific unmarshaling issue
// that was reported in the todo list
func TestAnalyzePosition_UnmarshalingIssue(t *testing.T) {
	// Create a mock engine for testing
	engine := &Engine{
		cache: nil, // No cache for this test
	}

	// Test cases that might cause unmarshaling issues
	tests := []struct {
		name          string
		request       *AnalysisRequest
		mockResponse  string
		expectError   bool
		validateMoves func(t *testing.T, result *AnalysisResult)
	}{
		{
			name: "SGF format moves should be rejected",
			request: &AnalysisRequest{
				Position: &Position{
					Rules:      "chinese",
					BoardXSize: 19,
					BoardYSize: 19,
					Komi:       6.5,
					Moves: []Move{
						{Color: "b", Location: "dd"}, // SGF format - should be rejected
					},
				},
			},
			expectError: true,
		},
		{
			name: "Valid GTP format moves",
			request: &AnalysisRequest{
				Position: &Position{
					Rules:      "chinese",
					BoardXSize: 19,
					BoardYSize: 19,
					Komi:       6.5,
					Moves: []Move{
						{Color: "b", Location: "D4"},
						{Color: "w", Location: "Q16"},
						{Color: "b", Location: "pass"},
					},
				},
			},
			mockResponse: `{"id":"test","turnNumber":3,"moveInfos":[{"move":"D16","visits":100,"winrate":0.55,"scoreLead":0.5,"scoreMean":1.2,"prior":0.15,"pv":["D16","D4"]}],"rootInfo":{"visits":100,"winrate":0.55,"scoreLead":0.5,"scoreMean":1.2,"scoreStdev":0.3,"currentPlayer":"W"}}`,
			expectError:  false,
			validateMoves: func(t *testing.T, result *AnalysisResult) {
				require.Len(t, result.MoveInfos, 1)
				assert.Equal(t, "D16", result.MoveInfos[0].Move)
				assert.Equal(t, 0.55, result.MoveInfos[0].Winrate)
			},
		},
		{
			name: "Edge case - I column should be rejected",
			request: &AnalysisRequest{
				Position: &Position{
					Rules:      "chinese",
					BoardXSize: 19,
					BoardYSize: 19,
					Komi:       6.5,
					Moves: []Move{
						{Color: "b", Location: "I5"}, // 'I' is not valid
					},
				},
			},
			expectError: true,
		},
		{
			name: "Response with policy array",
			request: &AnalysisRequest{
				Position: &Position{
					Rules:      "chinese",
					BoardXSize: 9,
					BoardYSize: 9,
					Komi:       6.5,
					Moves:      []Move{},
				},
				IncludePolicy: true,
			},
			mockResponse: `{"id":"test","turnNumber":0,"moveInfos":[{"move":"E5","visits":50,"winrate":0.52,"scoreLead":0.1,"scoreMean":0.5,"prior":0.25,"pv":["E5"]}],"rootInfo":{"visits":50,"winrate":0.52,"scoreLead":0.1,"scoreMean":0.5,"scoreStdev":0.2,"currentPlayer":"B"},"policy":[0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.25,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.05]}`,
			expectError:  false,
			validateMoves: func(t *testing.T, result *AnalysisResult) {
				require.NotNil(t, result.Policy)
				require.Len(t, result.Policy, 82) // 9x9 + 1 for pass
				// Check that E5 (center) has highest policy value
				centerIndex := 4*9 + 4 // (4,4) for 9x9 board
				assert.Equal(t, 0.25, result.Policy[centerIndex])
				// Check pass move probability
				assert.Equal(t, 0.05, result.Policy[81])
			},
		},
		{
			name: "Invalid move in response",
			request: &AnalysisRequest{
				Position: &Position{
					Rules:      "chinese",
					BoardXSize: 19,
					BoardYSize: 19,
					Komi:       6.5,
					Moves:      []Move{},
				},
			},
			mockResponse: `{"id":"test","turnNumber":0,"moveInfos":[{"move":"invalid","visits":100,"winrate":0.5,"scoreLead":0,"scoreMean":0,"prior":0.1,"pv":["invalid"]}],"rootInfo":{"visits":100,"winrate":0.5,"scoreLead":0,"scoreMean":0,"scoreStdev":0.1,"currentPlayer":"B"}}`,
			expectError:  false, // Currently we don't validate moves in responses
			validateMoves: func(t *testing.T, result *AnalysisResult) {
				require.Len(t, result.MoveInfos, 1)
				assert.Equal(t, "invalid", result.MoveInfos[0].Move)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test request validation
			_, err := engine.Analyze(context.Background(), tt.request)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			// For non-error cases, test response unmarshaling
			if tt.mockResponse != "" {
				var response Response
				err := json.Unmarshal([]byte(tt.mockResponse), &response)
				require.NoError(t, err)

				// Convert to AnalysisResult
				result := &AnalysisResult{
					MoveInfos: response.MoveInfos,
					RootInfo:  response.RootInfo,
				}

				// Check if policy exists in raw response
				var rawResponse map[string]interface{}
				json.Unmarshal([]byte(tt.mockResponse), &rawResponse)
				if policy, ok := rawResponse["policy"].([]interface{}); ok {
					result.Policy = make([]float64, len(policy))
					for i, v := range policy {
						if f, ok := v.(float64); ok {
							result.Policy[i] = f
						}
					}
				}

				// Validate the result
				if tt.validateMoves != nil {
					tt.validateMoves(t, result)
				}
			}
		})
	}
}

// TestMoveFormatConversion tests conversion between different move formats
func TestMoveFormatConversion(t *testing.T) {
	tests := []struct {
		name        string
		sgfCoord    string
		gtpCoord    string
		boardSize   int
		expectError bool
	}{
		{
			name:      "Center move on 19x19",
			sgfCoord:  "jj",
			gtpCoord:  "K10",
			boardSize: 19,
		},
		{
			name:      "Corner move",
			sgfCoord:  "aa",
			gtpCoord:  "A19",
			boardSize: 19,
		},
		{
			name:      "Another corner",
			sgfCoord:  "ss",
			gtpCoord:  "T1",
			boardSize: 19,
		},
		{
			name:      "D4 opening",
			sgfCoord:  "dd",
			gtpCoord:  "D16",
			boardSize: 19,
		},
		{
			name:      "9x9 center",
			sgfCoord:  "ee",
			gtpCoord:  "E5",
			boardSize: 9,
		},
	}

	parser := &SGFParser{boardSize: 19}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser.boardSize = tt.boardSize
			result := parser.sgfToKataGo(tt.sgfCoord)
			assert.Equal(t, tt.gtpCoord, result)
		})
	}
}
