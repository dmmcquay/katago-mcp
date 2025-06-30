package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/dmmcquay/katago-mcp/internal/katago"
	"github.com/dmmcquay/katago-mcp/internal/logging"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolsHandler manages MCP tools for KataGo.
type ToolsHandler struct {
	engine *katago.Engine
	logger *logging.Logger
}

// NewToolsHandler creates a new tools handler.
func NewToolsHandler(engine *katago.Engine, logger *logging.Logger) *ToolsHandler {
	return &ToolsHandler{
		engine: engine,
		logger: logger,
	}
}

// RegisterTools registers all tools with the MCP server.
func (h *ToolsHandler) RegisterTools(s *server.MCPServer) {
	// Register analyzePosition tool
	analyzePositionTool := mcp.NewTool("analyzePosition",
		mcp.WithDescription("Analyze a Go position using KataGo. Provide either SGF content or a position object."),
		mcp.WithString("sgf",
			mcp.Description("SGF content to analyze"),
		),
		mcp.WithObject("position",
			mcp.Description("Position object with rules, board size, moves, etc."),
		),
		mcp.WithNumber("moveNumber",
			mcp.Description("Move number to analyze (for SGF input). If not specified, analyzes the final position."),
		),
		mcp.WithNumber("maxVisits",
			mcp.Description("Maximum visits for analysis (overrides default)"),
		),
		mcp.WithNumber("maxTime",
			mcp.Description("Maximum time in seconds for analysis (overrides default)"),
		),
		mcp.WithBoolean("includePolicy",
			mcp.Description("Include policy network output"),
		),
		mcp.WithBoolean("includeOwnership",
			mcp.Description("Include ownership map"),
		),
		mcp.WithBoolean("verbose",
			mcp.Description("Include more detailed output"),
		),
	)
	s.AddTool(analyzePositionTool, h.handleAnalyzePosition)

	// Register getEngineStatus tool
	getEngineStatusTool := mcp.NewTool("getEngineStatus",
		mcp.WithDescription("Get the status of the KataGo engine"),
	)
	s.AddTool(getEngineStatusTool, h.handleGetEngineStatus)

	// Register startEngine tool
	startEngineTool := mcp.NewTool("startEngine",
		mcp.WithDescription("Start the KataGo engine if not already running"),
	)
	s.AddTool(startEngineTool, h.handleStartEngine)

	// Register stopEngine tool
	stopEngineTool := mcp.NewTool("stopEngine",
		mcp.WithDescription("Stop the KataGo engine"),
	)
	s.AddTool(stopEngineTool, h.handleStopEngine)

	// Register findMistakes tool
	findMistakesTool := mcp.NewTool("findMistakes",
		mcp.WithDescription("Analyze a game to find mistakes and blunders"),
		mcp.WithString("sgf",
			mcp.Required(),
			mcp.Description("SGF content of the game to analyze"),
		),
		mcp.WithNumber("blunderThreshold",
			mcp.Description("Win rate drop threshold for blunders (default: 0.15)"),
		),
		mcp.WithNumber("mistakeThreshold",
			mcp.Description("Win rate drop threshold for mistakes (default: 0.05)"),
		),
		mcp.WithNumber("inaccuracyThreshold",
			mcp.Description("Win rate drop threshold for inaccuracies (default: 0.02)"),
		),
		mcp.WithBoolean("includeAnalysis",
			mcp.Description("Include detailed move-by-move analysis"),
		),
		mcp.WithNumber("topMistakes",
			mcp.Description("Limit number of mistakes to return (0 for all)"),
		),
	)
	s.AddTool(findMistakesTool, h.handleFindMistakes)

	// Register evaluateTerritory tool
	evaluateTerritoryTool := mcp.NewTool("evaluateTerritory",
		mcp.WithDescription("Estimate territory control and final score"),
		mcp.WithString("sgf",
			mcp.Description("SGF content to analyze"),
		),
		mcp.WithObject("position",
			mcp.Description("Position object with rules, board size, moves, etc."),
		),
		mcp.WithNumber("threshold",
			mcp.Description("Ownership threshold for territory (default: 0.60)"),
		),
		mcp.WithBoolean("includeVisualization",
			mcp.Description("Include text visualization of territory"),
		),
	)
	s.AddTool(evaluateTerritoryTool, h.handleEvaluateTerritory)

	// Register explainMove tool
	explainMoveTool := mcp.NewTool("explainMove",
		mcp.WithDescription("Get detailed explanation for why a move is good or bad"),
		mcp.WithString("sgf",
			mcp.Description("SGF content of the position"),
		),
		mcp.WithObject("position",
			mcp.Description("Position object with rules, board size, moves, etc."),
		),
		mcp.WithString("move",
			mcp.Required(),
			mcp.Description("The move to explain (e.g., 'D4', 'Q16', 'pass')"),
		),
	)
	s.AddTool(explainMoveTool, h.handleExplainMove)
}

// handleAnalyzePosition handles the analyzePosition tool.
func (h *ToolsHandler) handleAnalyzePosition(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Ensure engine is running
	if !h.engine.IsRunning() {
		if err := h.engine.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start engine: %w", err)
		}
		// Give engine a moment to initialize
		// In a real implementation, we might want to wait for a ready signal
	}

	args := request.Params.Arguments
	if args == nil {
		return nil, fmt.Errorf("missing arguments")
	}

	// Parse arguments
	argsMap, ok := args.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments format")
	}

	// Create analysis request
	req := &katago.AnalysisRequest{}

	// Handle SGF input
	if sgfVal, ok := argsMap["sgf"]; ok {
		sgf, ok := sgfVal.(string)
		if !ok {
			return nil, fmt.Errorf("sgf must be a string")
		}

		// Parse SGF to get position
		parser := katago.NewSGFParser(sgf)
		position, err := parser.Parse()
		if err != nil {
			return nil, fmt.Errorf("failed to parse SGF: %w", err)
		}

		// Handle move number
		if moveNumVal, ok := argsMap["moveNumber"]; ok {
			moveNum := 0
			switch v := moveNumVal.(type) {
			case float64:
				moveNum = int(v)
			case int:
				moveNum = v
			case string:
				moveNum, _ = strconv.Atoi(v)
			}

			if moveNum > 0 && moveNum < len(position.Moves) {
				position.Moves = position.Moves[:moveNum]
			}
		}

		req.Position = position
	} else if posVal, ok := argsMap["position"]; ok {
		// Handle position object input
		posData, err := json.Marshal(posVal)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal position: %w", err)
		}

		var position katago.Position
		if err := json.Unmarshal(posData, &position); err != nil {
			return nil, fmt.Errorf("failed to parse position: %w", err)
		}

		req.Position = &position
	} else {
		return nil, fmt.Errorf("must provide either 'sgf' or 'position' parameter")
	}

	// Handle optional parameters
	if maxVisitsVal, ok := argsMap["maxVisits"]; ok {
		maxVisits := 0
		switch v := maxVisitsVal.(type) {
		case float64:
			maxVisits = int(v)
		case int:
			maxVisits = v
		}
		if maxVisits > 0 {
			req.MaxVisits = &maxVisits
		}
	}

	if maxTimeVal, ok := argsMap["maxTime"]; ok {
		maxTime := 0.0
		switch v := maxTimeVal.(type) {
		case float64:
			maxTime = v
		case int:
			maxTime = float64(v)
		}
		if maxTime > 0 {
			req.MaxTime = &maxTime
		}
	}

	if includePolicyVal, ok := argsMap["includePolicy"]; ok {
		if includePolicy, ok := includePolicyVal.(bool); ok {
			req.IncludePolicy = includePolicy
		}
	}

	if includeOwnershipVal, ok := argsMap["includeOwnership"]; ok {
		if includeOwnership, ok := includeOwnershipVal.(bool); ok {
			req.IncludeOwnership = includeOwnership
		}
	}

	verbose := false
	if verboseVal, ok := argsMap["verbose"]; ok {
		if v, ok := verboseVal.(bool); ok {
			verbose = v
		}
	}

	// Perform analysis
	result, err := h.engine.Analyze(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	// Format result
	if verbose || (!req.IncludePolicy && !req.IncludeOwnership) {
		// Return formatted text for simple cases
		formatted := katago.FormatAnalysisResult(result, verbose)
		return mcp.NewToolResultText(formatted), nil
	}

	// Return JSON for complex cases
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format result: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleGetEngineStatus handles the getEngineStatus tool.
func (h *ToolsHandler) handleGetEngineStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status := "stopped"
	if h.engine.IsRunning() {
		status = "running"
	}

	info := fmt.Sprintf("KataGo engine status: %s", status)
	return mcp.NewToolResultText(info), nil
}

// handleStartEngine handles the startEngine tool.
func (h *ToolsHandler) handleStartEngine(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.engine.IsRunning() {
		return mcp.NewToolResultText("KataGo engine is already running"), nil
	}

	if err := h.engine.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start engine: %w", err)
	}

	return mcp.NewToolResultText("KataGo engine started successfully"), nil
}

// handleStopEngine handles the stopEngine tool.
func (h *ToolsHandler) handleStopEngine(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !h.engine.IsRunning() {
		return mcp.NewToolResultText("KataGo engine is not running"), nil
	}

	if err := h.engine.Stop(); err != nil {
		return nil, fmt.Errorf("failed to stop engine: %w", err)
	}

	return mcp.NewToolResultText("KataGo engine stopped successfully"), nil
}

// handleFindMistakes handles the findMistakes tool.
func (h *ToolsHandler) handleFindMistakes(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Ensure engine is running
	if !h.engine.IsRunning() {
		if err := h.engine.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start engine: %w", err)
		}
	}

	args := request.Params.Arguments
	if args == nil {
		return nil, fmt.Errorf("missing arguments")
	}

	argsMap, ok := args.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments format")
	}

	// Get SGF content
	sgfVal, ok := argsMap["sgf"]
	if !ok {
		return nil, fmt.Errorf("missing required parameter: sgf")
	}
	sgf, ok := sgfVal.(string)
	if !ok {
		return nil, fmt.Errorf("sgf must be a string")
	}

	// Parse thresholds
	thresholds := katago.DefaultMistakeThresholds()

	if val, ok := argsMap["blunderThreshold"]; ok {
		if threshold, ok := val.(float64); ok {
			thresholds.Blunder = threshold
		}
	}
	if val, ok := argsMap["mistakeThreshold"]; ok {
		if threshold, ok := val.(float64); ok {
			thresholds.Mistake = threshold
		}
	}
	if val, ok := argsMap["inaccuracyThreshold"]; ok {
		if threshold, ok := val.(float64); ok {
			thresholds.Inaccuracy = threshold
		}
	}

	// Review the game
	review, err := h.engine.ReviewGame(ctx, sgf, &thresholds)
	if err != nil {
		return nil, fmt.Errorf("failed to review game: %w", err)
	}

	// Check if we should include full analysis
	includeAnalysis := false
	if val, ok := argsMap["includeAnalysis"]; ok {
		includeAnalysis, _ = val.(bool)
	}

	// Limit mistakes if requested
	mistakes := review.Mistakes
	if val, ok := argsMap["topMistakes"]; ok {
		if limit, ok := val.(float64); ok && limit > 0 {
			mistakes = katago.FindTopMistakes(review, int(limit))
		}
	}

	// Prepare result
	result := map[string]interface{}{
		"mistakes": mistakes,
		"summary":  review.Summary,
	}

	if includeAnalysis {
		result["moveAnalyses"] = review.MoveAnalyses
	}

	// Format as JSON
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format result: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleEvaluateTerritory handles the evaluateTerritory tool.
func (h *ToolsHandler) handleEvaluateTerritory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Ensure engine is running
	if !h.engine.IsRunning() {
		if err := h.engine.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start engine: %w", err)
		}
	}

	args := request.Params.Arguments
	if args == nil {
		return nil, fmt.Errorf("missing arguments")
	}

	argsMap, ok := args.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments format")
	}

	// Get position (from SGF or position object)
	var position *katago.Position

	if sgfVal, ok := argsMap["sgf"]; ok {
		sgf, ok := sgfVal.(string)
		if !ok {
			return nil, fmt.Errorf("sgf must be a string")
		}

		parser := katago.NewSGFParser(sgf)
		pos, err := parser.Parse()
		if err != nil {
			return nil, fmt.Errorf("failed to parse SGF: %w", err)
		}
		position = pos
	} else if posVal, ok := argsMap["position"]; ok {
		posData, err := json.Marshal(posVal)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal position: %w", err)
		}

		var pos katago.Position
		if err := json.Unmarshal(posData, &pos); err != nil {
			return nil, fmt.Errorf("failed to parse position: %w", err)
		}
		position = &pos
	} else {
		return nil, fmt.Errorf("must provide either 'sgf' or 'position' parameter")
	}

	// Get threshold
	threshold := 0.60
	if val, ok := argsMap["threshold"]; ok {
		if t, ok := val.(float64); ok {
			threshold = t
		}
	}

	// Estimate territory
	estimate, err := h.engine.EstimateTerritory(ctx, position, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate territory: %w", err)
	}

	// Check if visualization requested
	includeViz := false
	if val, ok := argsMap["includeVisualization"]; ok {
		includeViz, _ = val.(bool)
	}

	if includeViz {
		// Return text visualization
		viz := katago.GetTerritoryVisualization(estimate)
		return mcp.NewToolResultText(viz), nil
	}

	// Return JSON result
	resultJSON, err := json.MarshalIndent(estimate, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format result: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleExplainMove handles the explainMove tool.
func (h *ToolsHandler) handleExplainMove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Ensure engine is running
	if !h.engine.IsRunning() {
		if err := h.engine.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start engine: %w", err)
		}
	}

	args := request.Params.Arguments
	if args == nil {
		return nil, fmt.Errorf("missing arguments")
	}

	argsMap, ok := args.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments format")
	}

	// Get move to explain
	moveVal, ok := argsMap["move"]
	if !ok {
		return nil, fmt.Errorf("missing required parameter: move")
	}
	move, ok := moveVal.(string)
	if !ok {
		return nil, fmt.Errorf("move must be a string")
	}

	// Get position (from SGF or position object)
	var position *katago.Position

	if sgfVal, ok := argsMap["sgf"]; ok {
		sgf, ok := sgfVal.(string)
		if !ok {
			return nil, fmt.Errorf("sgf must be a string")
		}

		parser := katago.NewSGFParser(sgf)
		pos, err := parser.Parse()
		if err != nil {
			return nil, fmt.Errorf("failed to parse SGF: %w", err)
		}
		position = pos
	} else if posVal, ok := argsMap["position"]; ok {
		posData, err := json.Marshal(posVal)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal position: %w", err)
		}

		var pos katago.Position
		if err := json.Unmarshal(posData, &pos); err != nil {
			return nil, fmt.Errorf("failed to parse position: %w", err)
		}
		position = &pos
	} else {
		return nil, fmt.Errorf("must provide either 'sgf' or 'position' parameter")
	}

	// Get explanation
	explanation, err := h.engine.ExplainMove(ctx, position, move)
	if err != nil {
		return nil, fmt.Errorf("failed to explain move: %w", err)
	}

	// Format result
	resultJSON, err := json.MarshalIndent(explanation, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format result: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}
