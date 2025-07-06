package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dmmcquay/katago-mcp/internal/katago"
	"github.com/dmmcquay/katago-mcp/internal/logging"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolsHandler manages MCP tools for KataGo.
type ToolsHandler struct {
	engine     katago.EngineInterface
	logger     logging.ContextLogger
	middleware *Middleware
}

// NewToolsHandler creates a new tools handler.
func NewToolsHandler(engine katago.EngineInterface, logger logging.ContextLogger) *ToolsHandler {
	return &ToolsHandler{
		engine: engine,
		logger: logger,
	}
}

// SetMiddleware sets the middleware for the tools handler.
func (h *ToolsHandler) SetMiddleware(middleware *Middleware) {
	h.middleware = middleware
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
	handler := h.HandleAnalyzePosition
	if h.middleware != nil {
		handler = h.middleware.WrapTool("analyzePosition", handler)
	}
	s.AddTool(analyzePositionTool, handler)

	// Register getEngineStatus tool
	getEngineStatusTool := mcp.NewTool("getEngineStatus",
		mcp.WithDescription("Get the status of the KataGo engine"),
	)
	statusHandler := h.HandleGetEngineStatus
	if h.middleware != nil {
		statusHandler = h.middleware.WrapTool("getEngineStatus", statusHandler)
	}
	s.AddTool(getEngineStatusTool, statusHandler)

	// Register startEngine tool
	startEngineTool := mcp.NewTool("startEngine",
		mcp.WithDescription("Start the KataGo engine if not already running"),
	)
	startHandler := h.HandleStartEngine
	if h.middleware != nil {
		startHandler = h.middleware.WrapTool("startEngine", startHandler)
	}
	s.AddTool(startEngineTool, startHandler)

	// Register stopEngine tool
	stopEngineTool := mcp.NewTool("stopEngine",
		mcp.WithDescription("Stop the KataGo engine"),
	)
	stopHandler := h.HandleStopEngine
	if h.middleware != nil {
		stopHandler = h.middleware.WrapTool("stopEngine", stopHandler)
	}
	s.AddTool(stopEngineTool, stopHandler)

	// Register findMistakes tool
	findMistakesTool := mcp.NewTool("findMistakes",
		mcp.WithDescription("Analyze a game to find mistakes, blunders, and missed opportunities"),
		mcp.WithString("sgf",
			mcp.Description("SGF content of the game to review"),
			mcp.Required(),
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
		mcp.WithNumber("maxVisits",
			mcp.Description("Maximum visits per position (default: from config)"),
		),
	)
	mistakesHandler := h.HandleFindMistakes
	if h.middleware != nil {
		mistakesHandler = h.middleware.WrapToolWithRetry("findMistakes", mistakesHandler, 2)
	}
	s.AddTool(findMistakesTool, mistakesHandler)

	// Register evaluateTerritory tool
	evaluateTerritoryTool := mcp.NewTool("evaluateTerritory",
		mcp.WithDescription("Evaluate territory ownership and control"),
		mcp.WithString("sgf",
			mcp.Description("SGF content to analyze"),
			mcp.Required(),
		),
		mcp.WithNumber("threshold",
			mcp.Description("Ownership threshold (0.0-1.0, default: 0.85)"),
		),
		mcp.WithBoolean("includeEstimates",
			mcp.Description("Include detailed point estimates"),
		),
	)
	territoryHandler := h.HandleEvaluateTerritory
	if h.middleware != nil {
		territoryHandler = h.middleware.WrapTool("evaluateTerritory", territoryHandler)
	}
	s.AddTool(evaluateTerritoryTool, territoryHandler)

	// Register explainMove tool
	explainMoveTool := mcp.NewTool("explainMove",
		mcp.WithDescription("Get explanations for why a move is good or bad"),
		mcp.WithString("sgf",
			mcp.Description("SGF content of the position"),
			mcp.Required(),
		),
		mcp.WithString("move",
			mcp.Description("Move to explain (e.g., 'D4', 'Q16', 'pass')"),
			mcp.Required(),
		),
		mcp.WithNumber("maxVisits",
			mcp.Description("Maximum visits for analysis"),
		),
	)
	explainHandler := h.HandleExplainMove
	if h.middleware != nil {
		explainHandler = h.middleware.WrapTool("explainMove", explainHandler)
	}
	s.AddTool(explainMoveTool, explainHandler)
}

// HandleAnalyzePosition handles the analyzePosition tool.
func (h *ToolsHandler) HandleAnalyzePosition(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Generate correlation ID for this request
	ctx = logging.ContextWithCorrelationID(ctx, logging.GenerateCorrelationID())
	ctx = logging.ContextWithRequestID(ctx, logging.GenerateRequestID())
	logger := h.logger.WithContext(ctx).WithField("tool", "analyzePosition")

	logger.Info("Handling analyzePosition request")

	// Ensure engine is running
	if !h.engine.IsRunning() {
		logger.Debug("Starting KataGo engine")
		if err := h.engine.Start(ctx); err != nil {
			logger.Error("Failed to start engine: %v", err)
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
		boardSize := 19 // Default
		if position != nil {
			boardSize = position.BoardXSize
		}
		formatted := katago.FormatAnalysisResult(result, verbose, boardSize)
		return mcp.NewToolResultText(formatted), nil
	}

	// Return JSON for complex cases
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format result: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// HandleGetEngineStatus handles the getEngineStatus tool.
func (h *ToolsHandler) HandleGetEngineStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Generate correlation ID for this request
	ctx = logging.ContextWithCorrelationID(ctx, logging.GenerateCorrelationID())
	ctx = logging.ContextWithRequestID(ctx, logging.GenerateRequestID())
	logger := h.logger.WithContext(ctx).WithField("tool", "getEngineStatus")

	logger.Info("Handling getEngineStatus request")

	status := "stopped"
	if h.engine.IsRunning() {
		status = "running"
	}

	logger.Debug("Engine status checked", "status", status)
	info := fmt.Sprintf("KataGo engine status: %s", status)
	return mcp.NewToolResultText(info), nil
}

// HandleStartEngine handles the startEngine tool.
func (h *ToolsHandler) HandleStartEngine(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Generate correlation ID for this request
	ctx = logging.ContextWithCorrelationID(ctx, logging.GenerateCorrelationID())
	ctx = logging.ContextWithRequestID(ctx, logging.GenerateRequestID())
	logger := h.logger.WithContext(ctx).WithField("tool", "startEngine")

	logger.Info("Handling startEngine request")

	if h.engine.IsRunning() {
		logger.Debug("Engine already running")
		return mcp.NewToolResultText("KataGo engine is already running"), nil
	}

	logger.Info("Starting KataGo engine")
	if err := h.engine.Start(ctx); err != nil {
		logger.Error("Failed to start engine: %v", err)
		return nil, fmt.Errorf("failed to start engine: %w", err)
	}

	logger.Info("KataGo engine started successfully")
	return mcp.NewToolResultText("KataGo engine started successfully"), nil
}

// HandleStopEngine handles the stopEngine tool.
func (h *ToolsHandler) HandleStopEngine(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Generate correlation ID for this request
	ctx = logging.ContextWithCorrelationID(ctx, logging.GenerateCorrelationID())
	ctx = logging.ContextWithRequestID(ctx, logging.GenerateRequestID())
	logger := h.logger.WithContext(ctx).WithField("tool", "stopEngine")

	logger.Info("Handling stopEngine request")

	if !h.engine.IsRunning() {
		logger.Debug("Engine not running")
		return mcp.NewToolResultText("KataGo engine is not running"), nil
	}

	logger.Info("Stopping KataGo engine")
	if err := h.engine.Stop(); err != nil {
		logger.Error("Failed to stop engine: %v", err)
		return nil, fmt.Errorf("failed to stop engine: %w", err)
	}

	logger.Info("KataGo engine stopped successfully")
	return mcp.NewToolResultText("KataGo engine stopped successfully"), nil
}

// HandleFindMistakes handles the findMistakes tool.
func (h *ToolsHandler) HandleFindMistakes(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Generate correlation ID for this request
	ctx = logging.ContextWithCorrelationID(ctx, logging.GenerateCorrelationID())
	ctx = logging.ContextWithRequestID(ctx, logging.GenerateRequestID())
	logger := h.logger.WithContext(ctx).WithField("tool", "findMistakes")

	logger.Info("Handling findMistakes request")

	// Ensure engine is running
	if !h.engine.IsRunning() {
		logger.Debug("Starting KataGo engine")
		if err := h.engine.Start(ctx); err != nil {
			logger.Error("Failed to start engine: %v", err)
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
		return nil, fmt.Errorf("missing required parameter 'sgf'")
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

	if val, ok := argsMap["maxVisits"]; ok {
		if visits, ok := val.(float64); ok {
			thresholds.MinimumVisits = int(visits)
		}
	}

	// Review the game
	logger.Info("Reviewing game", "thresholds", thresholds)
	review, err := h.engine.ReviewGame(ctx, sgf, thresholds)
	if err != nil {
		logger.Error("Failed to review game: %v", err)
		return nil, fmt.Errorf("failed to review game: %w", err)
	}
	logger.Info("Game review completed",
		"totalMoves", review.Summary.TotalMoves,
		"mistakes", len(review.Mistakes))

	// Format the result
	var sb strings.Builder
	sb.WriteString("# Game Review\n\n")

	// Summary
	sb.WriteString("## Summary\n")
	sb.WriteString(fmt.Sprintf("- Total moves: %d\n", review.Summary.TotalMoves))
	sb.WriteString(fmt.Sprintf("- Black accuracy: %.1f%%\n", review.Summary.BlackAccuracy))
	sb.WriteString(fmt.Sprintf("- White accuracy: %.1f%%\n", review.Summary.WhiteAccuracy))
	sb.WriteString(fmt.Sprintf("- Black mistakes/blunders: %d/%d\n",
		review.Summary.BlackMistakes, review.Summary.BlackBlunders))
	sb.WriteString(fmt.Sprintf("- White mistakes/blunders: %d/%d\n",
		review.Summary.WhiteMistakes, review.Summary.WhiteBlunders))

	if review.Summary.EstimatedLevel != "" {
		sb.WriteString(fmt.Sprintf("- Estimated level: %s\n", review.Summary.EstimatedLevel))
	}

	// Mistakes
	if len(review.Mistakes) > 0 {
		sb.WriteString("\n## Mistakes Found\n\n")
		for i := range review.Mistakes {
			mistake := &review.Mistakes[i]
			sb.WriteString(fmt.Sprintf("### Move %d (%s)\n", mistake.MoveNumber, mistake.Color))
			sb.WriteString(fmt.Sprintf("- **Category**: %s\n", mistake.Category))
			sb.WriteString(fmt.Sprintf("- **Played**: %s (%.1f%% WR)\n",
				mistake.PlayedMove, mistake.PlayedWR*100))
			sb.WriteString(fmt.Sprintf("- **Better**: %s (%.1f%% WR)\n",
				mistake.BestMove, mistake.BestWR*100))
			sb.WriteString(fmt.Sprintf("- **Win rate drop**: %.1f%%\n", mistake.WinrateDrop*100))
			sb.WriteString(fmt.Sprintf("- %s\n\n", mistake.Explanation))
		}
	} else {
		sb.WriteString("\n## No significant mistakes found!\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

// HandleEvaluateTerritory handles the evaluateTerritory tool.
func (h *ToolsHandler) HandleEvaluateTerritory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Generate correlation ID for this request
	ctx = logging.ContextWithCorrelationID(ctx, logging.GenerateCorrelationID())
	ctx = logging.ContextWithRequestID(ctx, logging.GenerateRequestID())
	logger := h.logger.WithContext(ctx).WithField("tool", "evaluateTerritory")

	logger.Info("Handling evaluateTerritory request")

	// Ensure engine is running
	if !h.engine.IsRunning() {
		logger.Debug("Starting KataGo engine")
		if err := h.engine.Start(ctx); err != nil {
			logger.Error("Failed to start engine: %v", err)
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
		return nil, fmt.Errorf("missing required parameter 'sgf'")
	}
	sgf, ok := sgfVal.(string)
	if !ok {
		return nil, fmt.Errorf("sgf must be a string")
	}

	// Parse SGF
	parser := katago.NewSGFParser(sgf)
	position, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse SGF: %w", err)
	}

	// Get threshold
	threshold := 0.85
	if val, ok := argsMap["threshold"]; ok {
		if t, ok := val.(float64); ok && t > 0 && t <= 1 {
			threshold = t
		}
	}

	// Estimate territory
	logger.Info("Estimating territory", "threshold", threshold)
	estimate, err := h.engine.EstimateTerritory(ctx, position, threshold)
	if err != nil {
		logger.Error("Failed to estimate territory: %v", err)
		return nil, fmt.Errorf("failed to estimate territory: %w", err)
	}
	logger.Debug("Territory estimation completed")

	// Check if detailed estimates requested
	includeEstimates := false
	if val, ok := argsMap["includeEstimates"]; ok {
		if b, ok := val.(bool); ok {
			includeEstimates = b
		}
	}

	// Format result
	if includeEstimates {
		// Return JSON with full details
		resultJSON, err := json.MarshalIndent(estimate, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to format result: %w", err)
		}
		return mcp.NewToolResultText(string(resultJSON)), nil
	}

	// Return visualization
	viz := katago.GetTerritoryVisualization(estimate)
	return mcp.NewToolResultText(viz), nil
}

// HandleExplainMove handles the explainMove tool.
func (h *ToolsHandler) HandleExplainMove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Generate correlation ID for this request
	ctx = logging.ContextWithCorrelationID(ctx, logging.GenerateCorrelationID())
	ctx = logging.ContextWithRequestID(ctx, logging.GenerateRequestID())
	logger := h.logger.WithContext(ctx).WithField("tool", "explainMove")

	logger.Info("Handling explainMove request")

	// Ensure engine is running
	if !h.engine.IsRunning() {
		logger.Debug("Starting KataGo engine")
		if err := h.engine.Start(ctx); err != nil {
			logger.Error("Failed to start engine: %v", err)
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
		return nil, fmt.Errorf("missing required parameter 'sgf'")
	}
	sgf, ok := sgfVal.(string)
	if !ok {
		return nil, fmt.Errorf("sgf must be a string")
	}

	// Get move to explain
	moveVal, ok := argsMap["move"]
	if !ok {
		return nil, fmt.Errorf("missing required parameter 'move'")
	}
	move, ok := moveVal.(string)
	if !ok {
		return nil, fmt.Errorf("move must be a string")
	}

	// Parse SGF
	parser := katago.NewSGFParser(sgf)
	position, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse SGF: %w", err)
	}

	// Get explanation
	logger.Info("Explaining move", "move", move)
	explanation, err := h.engine.ExplainMove(ctx, position, move)
	if err != nil {
		logger.Error("Failed to explain move: %v", err)
		return nil, fmt.Errorf("failed to explain move: %w", err)
	}
	logger.Debug("Move explanation completed", "winrate", explanation.Winrate)

	// Format result
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Move Explanation: %s\n\n", move))
	sb.WriteString(fmt.Sprintf("%s\n\n", explanation.Explanation))

	// Stats
	sb.WriteString("## Statistics\n")
	sb.WriteString(fmt.Sprintf("- Win rate: %.1f%%\n", explanation.Winrate*100))
	sb.WriteString(fmt.Sprintf("- Score lead: %.1f points\n", explanation.ScoreLead))
	sb.WriteString(fmt.Sprintf("- Engine visits: %d\n\n", explanation.Visits))

	// Strategic info
	sb.WriteString("## Strategic Analysis\n")
	sb.WriteString(fmt.Sprintf("- Board region: %s\n", explanation.Strategic.BoardRegion))
	sb.WriteString(fmt.Sprintf("- Urgency: %s\n", explanation.Strategic.Urgency))
	if len(explanation.Strategic.Purpose) > 0 {
		sb.WriteString(fmt.Sprintf("- Purpose: %s\n", strings.Join(explanation.Strategic.Purpose, ", ")))
	}

	// Pros and cons
	if len(explanation.Pros) > 0 {
		sb.WriteString("\n## Pros\n")
		for _, pro := range explanation.Pros {
			sb.WriteString(fmt.Sprintf("- %s\n", pro))
		}
	}

	if len(explanation.Cons) > 0 {
		sb.WriteString("\n## Cons\n")
		for _, con := range explanation.Cons {
			sb.WriteString(fmt.Sprintf("- %s\n", con))
		}
	}

	// Alternatives
	if len(explanation.Alternatives) > 0 {
		sb.WriteString("\n## Better Alternatives\n")
		for _, alt := range explanation.Alternatives {
			sb.WriteString(fmt.Sprintf("- **%s** (%.1f%% WR): %s\n",
				alt.Move, alt.Winrate*100, alt.Reasoning))
		}
	}

	return mcp.NewToolResultText(sb.String()), nil
}
