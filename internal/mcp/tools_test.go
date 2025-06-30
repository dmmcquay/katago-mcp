package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/katago"
	"github.com/dmmcquay/katago-mcp/internal/logging"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestAnalyzePositionTool(t *testing.T) {
	// Create mock engine
	cfg := &config.KataGoConfig{
		BinaryPath: "mock-katago",
		NumThreads: 1,
		MaxVisits:  10,
		MaxTime:    0.1,
	}
	logger := logging.NewLogger("test: ", "debug")
	engine := katago.NewEngine(cfg, logger)

	handler := NewToolsHandler(engine, logger)

	ctx := context.Background()

	// Test with simple SGF
	sgf := `(;GM[1]FF[4]SZ[19]KM[7.5];B[dd];W[pp])`

	args := map[string]interface{}{
		"sgf": sgf,
	}

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "analyzePosition",
			Arguments: args,
		},
	}

	// This should fail because mock engine won't start
	_, err := handler.handleAnalyzePosition(ctx, req)
	if err == nil {
		t.Error("Expected error for mock engine")
	}
}

func TestEngineStatusTool(t *testing.T) {
	cfg := &config.KataGoConfig{
		BinaryPath: "mock-katago",
		NumThreads: 1,
		MaxVisits:  10,
		MaxTime:    0.1,
	}
	logger := logging.NewLogger("test: ", "debug")
	engine := katago.NewEngine(cfg, logger)

	handler := NewToolsHandler(engine, logger)

	ctx := context.Background()
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "getEngineStatus",
		},
	}

	result, err := handler.handleGetEngineStatus(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Error("Expected result")
	}

	// Check that result contains status information
	if len(result.Content) == 0 {
		t.Error("Expected content in result")
	}
}

func TestStartStopEngineTool(t *testing.T) {
	cfg := &config.KataGoConfig{
		BinaryPath: "mock-katago",
		NumThreads: 1,
		MaxVisits:  10,
		MaxTime:    0.1,
	}
	logger := logging.NewLogger("test: ", "debug")
	engine := katago.NewEngine(cfg, logger)

	handler := NewToolsHandler(engine, logger)

	ctx := context.Background()

	// Test start engine
	startReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "startEngine",
		},
	}

	// This should fail because mock engine won't start
	_, err := handler.handleStartEngine(ctx, startReq)
	if err == nil {
		t.Error("Expected error for mock engine start")
	}

	// Test stop engine
	stopReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "stopEngine",
		},
	}

	result, err := handler.handleStopEngine(ctx, stopReq)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Error("Expected result")
	}
}

func TestAnalyzePositionArguments(t *testing.T) {
	cfg := &config.KataGoConfig{
		BinaryPath: "mock-katago",
		NumThreads: 1,
		MaxVisits:  10,
		MaxTime:    0.1,
	}
	logger := logging.NewLogger("test: ", "debug")
	engine := katago.NewEngine(cfg, logger)

	handler := NewToolsHandler(engine, logger)

	ctx := context.Background()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name:    "No arguments",
			args:    nil,
			wantErr: true,
		},
		{
			name:    "Empty arguments",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "Valid SGF",
			args: map[string]interface{}{
				"sgf": "(;GM[1]FF[4]SZ[19]KM[7.5];B[dd];W[pp])",
			},
			wantErr: true, // Will fail because engine won't start
		},
		{
			name: "Valid position object",
			args: map[string]interface{}{
				"position": map[string]interface{}{
					"rules":      "chinese",
					"boardXSize": 19,
					"boardYSize": 19,
					"komi":       7.5,
					"moves": []interface{}{
						map[string]interface{}{"color": "b", "location": "D4"},
						map[string]interface{}{"color": "w", "location": "Q16"},
					},
				},
			},
			wantErr: true, // Will fail because engine won't start
		},
		{
			name: "Invalid SGF",
			args: map[string]interface{}{
				"sgf": "invalid sgf",
			},
			wantErr: true,
		},
		{
			name: "SGF with move number",
			args: map[string]interface{}{
				"sgf":        "(;GM[1]FF[4]SZ[19]KM[7.5];B[dd];W[pp];B[pd])",
				"moveNumber": 2,
			},
			wantErr: true, // Will fail because engine won't start
		},
		{
			name: "With analysis options",
			args: map[string]interface{}{
				"sgf":              "(;GM[1]FF[4]SZ[19]KM[7.5];B[dd];W[pp])",
				"maxVisits":        100,
				"maxTime":          5.0,
				"includePolicy":    true,
				"includeOwnership": true,
				"verbose":          true,
			},
			wantErr: true, // Will fail because engine won't start
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "analyzePosition",
					Arguments: tt.args,
				},
			}

			_, err := handler.handleAnalyzePosition(ctx, req)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleAnalyzePosition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPositionObjectParsing(t *testing.T) {
	// Test that position objects are correctly parsed
	positionData := map[string]interface{}{
		"rules":      "chinese",
		"boardXSize": float64(19), // JSON numbers are float64
		"boardYSize": float64(19),
		"komi":       7.5,
		"moves": []interface{}{
			map[string]interface{}{
				"color":    "b",
				"location": "D4",
			},
			map[string]interface{}{
				"color":    "w",
				"location": "Q16",
			},
		},
	}

	// Marshal and unmarshal to simulate JSON processing
	data, err := json.Marshal(positionData)
	if err != nil {
		t.Fatalf("Failed to marshal position: %v", err)
	}

	var position katago.Position
	if err := json.Unmarshal(data, &position); err != nil {
		t.Fatalf("Failed to parse position: %v", err)
	}

	// Verify parsing
	if position.Rules != "chinese" {
		t.Errorf("Expected rules 'chinese', got '%s'", position.Rules)
	}
	if position.BoardXSize != 19 {
		t.Errorf("Expected board size 19, got %d", position.BoardXSize)
	}
	if len(position.Moves) != 2 {
		t.Errorf("Expected 2 moves, got %d", len(position.Moves))
	}
	if position.Moves[0].Color != "b" || position.Moves[0].Location != "D4" {
		t.Errorf("Unexpected first move: %+v", position.Moves[0])
	}
}
