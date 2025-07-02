package health

import (
	"context"
	"fmt"
	"testing"

	"github.com/dmmcquay/katago-mcp/internal/katago"
	"github.com/dmmcquay/katago-mcp/internal/logging"
)

// TestHealthCheckWithMockEngine tests health checks using a mock KataGo engine.
func TestHealthCheckWithMockEngine(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test", "debug"))
	checker := NewChecker(logger, "1.0.0", "abc123")

	// Create mock engine
	mockEngine := katago.NewMockEngine()

	// Register health check that uses the mock engine
	checker.RegisterCheck("katago", func(ctx context.Context) error {
		return mockEngine.Ping(ctx)
	})

	tests := []struct {
		name           string
		engineRunning  bool
		pingError      error
		expectedStatus Status
	}{
		{
			name:           "engine running and healthy",
			engineRunning:  true,
			pingError:      nil,
			expectedStatus: StatusHealthy,
		},
		{
			name:           "engine not running",
			engineRunning:  false,
			pingError:      nil, // Ping will fail because engine is not running
			expectedStatus: StatusUnhealthy,
		},
		{
			name:           "engine running but ping fails",
			engineRunning:  true,
			pingError:      fmt.Errorf("network error"),
			expectedStatus: StatusUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mock
			mockEngine.SetRunning(tt.engineRunning)
			mockEngine.SetPingError(tt.pingError)

			// Perform health check
			response := checker.CheckHealth(context.Background())

			// Verify overall status
			if response.Status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, response.Status)
			}

			// Verify component status
			if len(response.Components) != 1 {
				t.Fatalf("Expected 1 component, got %d", len(response.Components))
			}

			katagoComponent := response.Components[0]
			if katagoComponent.Name != "katago" {
				t.Errorf("Expected component name 'katago', got %s", katagoComponent.Name)
			}

			expectedComponentStatus := StatusHealthy
			if !tt.engineRunning || tt.pingError != nil {
				expectedComponentStatus = StatusUnhealthy
			}

			if katagoComponent.Status != expectedComponentStatus {
				t.Errorf("Expected component status %s, got %s", expectedComponentStatus, katagoComponent.Status)
			}
		})
	}
}

// TestHealthCheckCallsEngine verifies that health check actually calls the engine.
func TestHealthCheckCallsEngine(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test", "debug"))
	checker := NewChecker(logger, "1.0.0", "abc123")

	// Create mock engine
	mockEngine := katago.NewMockEngine()
	mockEngine.SetRunning(true)

	// Register health check
	checker.RegisterCheck("katago", func(ctx context.Context) error {
		return mockEngine.Ping(ctx)
	})

	// Initial ping count should be 0
	if mockEngine.GetPingCallCount() != 0 {
		t.Errorf("Expected 0 ping calls initially, got %d", mockEngine.GetPingCallCount())
	}

	// Perform health check
	checker.CheckHealth(context.Background())

	// Verify ping was called
	if mockEngine.GetPingCallCount() != 1 {
		t.Errorf("Expected 1 ping call, got %d", mockEngine.GetPingCallCount())
	}

	// Perform another health check
	checker.CheckHealth(context.Background())

	// Verify ping was called again
	if mockEngine.GetPingCallCount() != 2 {
		t.Errorf("Expected 2 ping calls, got %d", mockEngine.GetPingCallCount())
	}
}
