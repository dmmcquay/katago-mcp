package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/health"
	"github.com/dmmcquay/katago-mcp/internal/logging"
)

func TestNewHTTPServer(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test", "debug"))
	checker := health.NewChecker(logger, "1.0.0", "abc123")

	server := NewHTTPServer(":8080", logger, checker)
	if server == nil {
		t.Fatal("Expected non-nil server")
	}
	if server.server.Addr != ":8080" {
		t.Errorf("Expected addr :8080, got %s", server.server.Addr)
	}
}

func TestHTTPServerStartStop(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test", "debug"))
	checker := health.NewChecker(logger, "1.0.0", "abc123")

	// Use a random port to avoid conflicts
	server := NewHTTPServer(":0", logger, checker)

	// Start server
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}
}

func TestHealthEndpoints(t *testing.T) {
	logger := logging.NewLoggerAdapter(logging.NewLogger("test", "debug"))
	checker := health.NewChecker(logger, "1.0.0", "abc123")

	// Register a check
	checker.RegisterCheck("test", func(ctx context.Context) error {
		return nil
	})

	server := NewHTTPServer(":18080", logger, checker)

	// Start server
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test health endpoint
	resp, err := http.Get("http://localhost:18080/health")
	if err != nil {
		t.Fatalf("Failed to get /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var healthResp health.Response
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if healthResp.Status != health.StatusHealthy {
		t.Errorf("Expected healthy status, got %s", healthResp.Status)
	}

	// Test ready endpoint
	resp, err = http.Get("http://localhost:18080/ready")
	if err != nil {
		t.Fatalf("Failed to get /ready: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var readyResp health.Response
	if err := json.NewDecoder(resp.Body).Decode(&readyResp); err != nil {
		t.Fatalf("Failed to decode ready response: %v", err)
	}

	if readyResp.Status != health.StatusHealthy {
		t.Errorf("Expected healthy status, got %s", readyResp.Status)
	}

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Stop(ctx)
}
