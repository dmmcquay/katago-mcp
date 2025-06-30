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

// ToolsHandler manages MCP tools for KataGo
type ToolsHandler struct {
	engine *katago.Engine
	logger *logging.Logger
}

// NewToolsHandler creates a new tools handler
func NewToolsHandler(engine *katago.Engine, logger *logging.Logger) *ToolsHandler {
	return &ToolsHandler{
		engine: engine,
		logger: logger,
	}
}

// RegisterTools registers all tools with the MCP server
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
}

// handleAnalyzePosition handles the analyzePosition tool
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

// handleGetEngineStatus handles the getEngineStatus tool
func (h *ToolsHandler) handleGetEngineStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status := "stopped"
	if h.engine.IsRunning() {
		status = "running"
	}

	info := fmt.Sprintf("KataGo engine status: %s", status)
	return mcp.NewToolResultText(info), nil
}

// handleStartEngine handles the startEngine tool
func (h *ToolsHandler) handleStartEngine(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.engine.IsRunning() {
		return mcp.NewToolResultText("KataGo engine is already running"), nil
	}

	if err := h.engine.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start engine: %w", err)
	}

	return mcp.NewToolResultText("KataGo engine started successfully"), nil
}

// handleStopEngine handles the stopEngine tool
func (h *ToolsHandler) handleStopEngine(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !h.engine.IsRunning() {
		return mcp.NewToolResultText("KataGo engine is not running"), nil
	}

	if err := h.engine.Stop(); err != nil {
		return nil, fmt.Errorf("failed to stop engine: %w", err)
	}

	return mcp.NewToolResultText("KataGo engine stopped successfully"), nil
}