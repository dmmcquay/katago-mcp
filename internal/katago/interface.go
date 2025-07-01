package katago

import (
	"context"
)

// EngineInterface defines the interface for a KataGo engine.
// This allows for mocking in tests.
type EngineInterface interface {
	// Start starts the engine process
	Start(ctx context.Context) error

	// Stop stops the engine process
	Stop() error

	// IsRunning returns whether the engine is running
	IsRunning() bool

	// Ping checks if the engine is responsive
	Ping(ctx context.Context) error

	// Analyze analyzes a position
	Analyze(ctx context.Context, req *AnalysisRequest) (*AnalysisResult, error)

	// AnalyzeSGF analyzes a position from SGF
	AnalyzeSGF(ctx context.Context, sgf string, moveNum int) (*AnalysisResult, error)

	// ReviewGame reviews a complete game for mistakes
	ReviewGame(ctx context.Context, sgf string, thresholds *MistakeThresholds) (*GameReview, error)

	// EstimateTerritory estimates territory ownership
	EstimateTerritory(ctx context.Context, position *Position, threshold float64) (*TerritoryEstimate, error)

	// ExplainMove explains why a move is good or bad
	ExplainMove(ctx context.Context, position *Position, move string) (*MoveExplanation, error)
}

// Ensure Engine implements EngineInterface.
var _ EngineInterface = (*Engine)(nil)
