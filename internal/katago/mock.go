package katago

import (
	"context"
	"fmt"
	"sync"
)

// MockEngine is a mock implementation of EngineInterface for testing.
type MockEngine struct {
	mu             sync.Mutex
	running        bool
	pingErr        error
	analyzeResp    *AnalysisResult
	analyzeErr     error
	startErr       error
	stopErr        error
	pingCallCount  int
	startCallCount int
	stopCallCount  int
}

// NewMockEngine creates a new mock engine.
func NewMockEngine() *MockEngine {
	return &MockEngine{}
}

// SetRunning sets the running state of the mock engine.
func (m *MockEngine) SetRunning(running bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = running
}

// SetPingError sets the error to return from Ping.
func (m *MockEngine) SetPingError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingErr = err
}

// SetAnalyzeResponse sets the response to return from Analyze.
func (m *MockEngine) SetAnalyzeResponse(resp *AnalysisResult, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.analyzeResp = resp
	m.analyzeErr = err
}

// SetStartError sets the error to return from Start.
func (m *MockEngine) SetStartError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startErr = err
}

// GetPingCallCount returns the number of times Ping was called.
func (m *MockEngine) GetPingCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pingCallCount
}

// Start implements EngineInterface.
func (m *MockEngine) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startCallCount++
	if m.startErr != nil {
		return m.startErr
	}
	m.running = true
	return nil
}

// Stop implements EngineInterface.
func (m *MockEngine) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopCallCount++
	if m.stopErr != nil {
		return m.stopErr
	}
	m.running = false
	return nil
}

// IsRunning implements EngineInterface.
func (m *MockEngine) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// Ping implements EngineInterface.
func (m *MockEngine) Ping(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingCallCount++
	if !m.running {
		return fmt.Errorf("engine not running")
	}
	return m.pingErr
}

// Analyze implements EngineInterface.
func (m *MockEngine) Analyze(ctx context.Context, req *AnalysisRequest) (*AnalysisResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return nil, fmt.Errorf("engine not running")
	}
	return m.analyzeResp, m.analyzeErr
}

// AnalyzeSGF implements EngineInterface.
func (m *MockEngine) AnalyzeSGF(ctx context.Context, sgf string, moveNum int) (*AnalysisResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return nil, fmt.Errorf("engine not running")
	}
	return m.analyzeResp, m.analyzeErr
}

// ReviewGame implements EngineInterface.
func (m *MockEngine) ReviewGame(ctx context.Context, sgf string, thresholds *MistakeThresholds) (*GameReview, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return nil, fmt.Errorf("engine not running")
	}
	// Return a simple review
	return &GameReview{
		Summary: ReviewSummary{
			TotalMoves:    10,
			BlackAccuracy: 90.0,
			WhiteAccuracy: 85.0,
		},
		Mistakes: []Mistake{},
	}, nil
}

// EstimateTerritory implements EngineInterface.
func (m *MockEngine) EstimateTerritory(ctx context.Context, position *Position, threshold float64) (*TerritoryEstimate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return nil, fmt.Errorf("engine not running")
	}
	// Return a simple estimate
	return &TerritoryEstimate{
		BlackTerritory: 40,
		WhiteTerritory: 41,
		DamePoints:     0,
		ScoreEstimate:  -1.5,
		ScoreString:    "W+1.5",
	}, nil
}

// ExplainMove implements EngineInterface.
func (m *MockEngine) ExplainMove(ctx context.Context, position *Position, move string) (*MoveExplanation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return nil, fmt.Errorf("engine not running")
	}
	// Return a simple explanation
	return &MoveExplanation{
		Move:        move,
		Explanation: "This is a good move",
		Winrate:     0.55,
		ScoreLead:   0.5,
		Visits:      100,
	}, nil
}
