package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/logging"
)

func TestNewChecker(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test", "debug"))
	checker := NewChecker(logger, "1.0.0", "abc123")

	if checker == nil {
		t.Fatal("Expected non-nil checker")
	}
	if checker.version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", checker.version)
	}
	if checker.gitCommit != "abc123" {
		t.Errorf("Expected git commit abc123, got %s", checker.gitCommit)
	}
}

func TestRegisterCheck(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test", "debug"))
	checker := NewChecker(logger, "1.0.0", "abc123")

	// Register a check
	checkCalled := false
	checker.RegisterCheck("test", func(ctx context.Context) error {
		checkCalled = true
		return nil
	})

	// Verify check is registered
	if len(checker.checks) != 1 {
		t.Errorf("Expected 1 check, got %d", len(checker.checks))
	}

	// Run the check
	response := checker.CheckHealth(context.Background())
	if !checkCalled {
		t.Error("Expected check to be called")
	}
	if response.Status != StatusHealthy {
		t.Errorf("Expected healthy status, got %s", response.Status)
	}
}

func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name           string
		checks         map[string]error
		expectedStatus Status
		expectedComps  int
	}{
		{
			name:           "no checks",
			checks:         map[string]error{},
			expectedStatus: StatusHealthy,
			expectedComps:  0,
		},
		{
			name: "all healthy",
			checks: map[string]error{
				"database": nil,
				"engine":   nil,
			},
			expectedStatus: StatusHealthy,
			expectedComps:  2,
		},
		{
			name: "one unhealthy",
			checks: map[string]error{
				"database": nil,
				"engine":   errors.New("engine not running"),
			},
			expectedStatus: StatusUnhealthy,
			expectedComps:  2,
		},
		{
			name: "all unhealthy",
			checks: map[string]error{
				"database": errors.New("connection failed"),
				"engine":   errors.New("engine not running"),
			},
			expectedStatus: StatusUnhealthy,
			expectedComps:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewLoggerAdapter(logging.NewLogger("test", "debug"))
			checker := NewChecker(logger, "1.0.0", "abc123")

			// Register checks
			for name, err := range tt.checks {
				checkErr := err // Capture for closure
				checker.RegisterCheck(name, func(ctx context.Context) error {
					return checkErr
				})
			}

			// Run health check
			response := checker.CheckHealth(context.Background())

			// Verify response
			if response.Status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, response.Status)
			}
			if len(response.Components) != tt.expectedComps {
				t.Errorf("Expected %d components, got %d", tt.expectedComps, len(response.Components))
			}

			// Verify component statuses
			for _, comp := range response.Components {
				expectedErr := tt.checks[comp.Name]
				if expectedErr == nil && comp.Status != StatusHealthy {
					t.Errorf("Expected component %s to be healthy", comp.Name)
				} else if expectedErr != nil && comp.Status != StatusUnhealthy {
					t.Errorf("Expected component %s to be unhealthy", comp.Name)
				}
			}
		})
	}
}

func TestCheckHealthTimeout(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test", "debug"))
	checker := NewChecker(logger, "1.0.0", "abc123")

	// Register a check that takes too long
	checker.RegisterCheck("slow", func(ctx context.Context) error {
		select {
		case <-time.After(10 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	// Run health check with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	response := checker.CheckHealth(ctx)
	duration := time.Since(start)

	// Should complete within reasonable time (not 10 seconds)
	if duration > 6*time.Second {
		t.Errorf("Check took too long: %v", duration)
	}

	// Should have one unhealthy component
	if len(response.Components) != 1 {
		t.Fatalf("Expected 1 component, got %d", len(response.Components))
	}
	if response.Components[0].Status != StatusUnhealthy {
		t.Error("Expected component to be unhealthy due to timeout")
	}
}

func TestLivenessHandler(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test", "debug"))
	checker := NewChecker(logger, "1.0.0", "abc123")

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Call handler
	handler := checker.LivenessHandler()
	handler(rec, req)

	// Check response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Parse response
	var response Response
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response
	if response.Status != StatusHealthy {
		t.Errorf("Expected healthy status, got %s", response.Status)
	}
	if response.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", response.Version)
	}
	if response.GitCommit != "abc123" {
		t.Errorf("Expected git commit abc123, got %s", response.GitCommit)
	}
}

func TestReadinessHandler(t *testing.T) {
	tests := []struct {
		name           string
		checks         map[string]error
		expectedCode   int
		expectedStatus Status
	}{
		{
			name:           "no checks - healthy",
			checks:         map[string]error{},
			expectedCode:   http.StatusOK,
			expectedStatus: StatusHealthy,
		},
		{
			name: "all healthy",
			checks: map[string]error{
				"database": nil,
				"engine":   nil,
			},
			expectedCode:   http.StatusOK,
			expectedStatus: StatusHealthy,
		},
		{
			name: "one unhealthy",
			checks: map[string]error{
				"database": nil,
				"engine":   errors.New("not running"),
			},
			expectedCode:   http.StatusServiceUnavailable,
			expectedStatus: StatusUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewLoggerAdapter(logging.NewLogger("test", "debug"))
			checker := NewChecker(logger, "1.0.0", "abc123")

			// Register checks
			for name, err := range tt.checks {
				checkErr := err
				checker.RegisterCheck(name, func(ctx context.Context) error {
					return checkErr
				})
			}

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			rec := httptest.NewRecorder()

			// Call handler
			handler := checker.ReadinessHandler()
			handler(rec, req)

			// Check response code
			if rec.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, rec.Code)
			}

			// Parse response
			var response Response
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Verify response
			if response.Status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, response.Status)
			}
			if len(response.Components) != len(tt.checks) {
				t.Errorf("Expected %d components, got %d", len(tt.checks), len(response.Components))
			}
		})
	}
}

func TestConcurrentHealthChecks(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test", "debug"))
	checker := NewChecker(logger, "1.0.0", "abc123")

	// Register multiple checks with delays
	for i := 0; i < 5; i++ {
		checker.RegisterCheck(string(rune('a'+i)), func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	}

	// Measure time for parallel execution
	start := time.Now()
	response := checker.CheckHealth(context.Background())
	duration := time.Since(start)

	// Should complete faster than sequential (50ms)
	if duration > 30*time.Millisecond {
		t.Errorf("Checks took too long, might not be parallel: %v", duration)
	}

	// All should be healthy
	if response.Status != StatusHealthy {
		t.Errorf("Expected healthy status, got %s", response.Status)
	}
	if len(response.Components) != 5 {
		t.Errorf("Expected 5 components, got %d", len(response.Components))
	}
}
