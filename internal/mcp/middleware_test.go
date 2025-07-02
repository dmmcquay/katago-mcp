package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/config"
	"github.com/dmmcquay/katago-mcp/internal/logging"
	"github.com/dmmcquay/katago-mcp/internal/metrics"
	"github.com/dmmcquay/katago-mcp/internal/ratelimit"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestMiddleware(t *testing.T) {
	cfg := &logging.Config{
		Level:   "debug",
		Format:  logging.FormatText,
		Service: "test",
		Version: "test",
		Prefix:  "[TEST] ",
	}
	logger := logging.NewLoggerFromConfig(cfg)
	metricsCollector := metrics.NewCollector()

	t.Run("WrapTool", func(t *testing.T) {
		middleware := NewMiddleware(logger, metricsCollector, nil)

		// Create a test handler
		var called bool
		handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			called = true
			return mcp.NewToolResultText("success"), nil
		}

		// Wrap the handler
		wrapped := middleware.WrapTool("testTool", handler)

		// Call it
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{},
		}
		result, err := wrapped(context.Background(), req)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !called {
			t.Error("Handler was not called")
		}
		if result == nil {
			t.Error("Expected result, got nil")
		}
	})

	t.Run("RateLimiting", func(t *testing.T) {
		// Create a rate limiter with very low limits
		cfg := &config.RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 60, // 1 per second
			BurstSize:      2,
		}
		limiter := ratelimit.NewLimiter(cfg, logger)
		middleware := NewMiddleware(logger, metricsCollector, limiter)

		// Create a handler
		handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("success"), nil
		}

		wrapped := middleware.WrapTool("testTool", handler)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{},
		}

		// First two calls should succeed (burst)
		for i := 0; i < 2; i++ {
			result, err := wrapped(context.Background(), req)
			if err != nil {
				t.Errorf("Call %d: Expected no error, got %v", i+1, err)
			}
			if result == nil {
				t.Errorf("Call %d: Expected result, got nil", i+1)
			}
		}

		// Third call should be rate limited
		result, err := wrapped(context.Background(), req)
		if err == nil {
			t.Error("Expected rate limit error, got nil")
		}
		if result != nil {
			t.Error("Expected nil result when rate limited")
		}
		if !contains(err.Error(), "rate limit exceeded") {
			t.Errorf("Expected rate limit error, got: %v", err)
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		middleware := NewMiddleware(logger, metricsCollector, nil)

		// Create a handler that returns an error
		expectedErr := errors.New("test error")
		handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return nil, expectedErr
		}

		wrapped := middleware.WrapTool("testTool", handler)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{},
		}

		result, err := wrapped(context.Background(), req)
		if err != expectedErr {
			t.Errorf("Expected %v, got %v", expectedErr, err)
		}
		if result != nil {
			t.Error("Expected nil result on error")
		}
	})

	t.Run("ClientIDExtraction", func(t *testing.T) {
		middleware := NewMiddleware(logger, metricsCollector, nil)

		// Test with context client ID
		handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// In real implementation, we'd capture this from logs
			return mcp.NewToolResultText("success"), nil
		}

		wrapped := middleware.WrapTool("testTool", handler)

		// With client ID in context
		ctx := context.WithValue(context.Background(), "clientID", "test-client")
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{},
		}
		_, _ = wrapped(ctx, req)

		// With client ID in arguments
		req.Params.Arguments = map[string]interface{}{
			"clientID": "arg-client",
		}
		_, _ = wrapped(context.Background(), req)

		// Without client ID (should default to "anonymous")
		req.Params.Arguments = nil
		_, _ = wrapped(context.Background(), req)
	})

	t.Run("Retry", func(t *testing.T) {
		middleware := NewMiddleware(logger, metricsCollector, nil)

		t.Run("SuccessAfterRetry", func(t *testing.T) {
			callCount := 0
			handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				callCount++
				if callCount < 3 {
					return nil, errors.New("temporary error")
				}
				return mcp.NewToolResultText("success"), nil
			}

			wrapped := middleware.WrapToolWithRetry("testTool", handler, 3)
			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{},
			}

			result, err := wrapped(context.Background(), req)
			if err != nil {
				t.Errorf("Expected success after retry, got %v", err)
			}
			if result == nil {
				t.Error("Expected result, got nil")
			}
			if callCount != 3 {
				t.Errorf("Expected 3 calls, got %d", callCount)
			}
		})

		t.Run("MaxRetriesExceeded", func(t *testing.T) {
			handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return nil, errors.New("persistent error")
			}

			wrapped := middleware.WrapToolWithRetry("testTool", handler, 2)
			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{},
			}

			result, err := wrapped(context.Background(), req)
			if err == nil {
				t.Error("Expected error after max retries")
			}
			if result != nil {
				t.Error("Expected nil result on error")
			}
			if !contains(err.Error(), "failed after 2 retries") {
				t.Errorf("Expected retry exhaustion error, got: %v", err)
			}
		})

		t.Run("NoRetryOnRateLimit", func(t *testing.T) {
			callCount := 0
			handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				callCount++
				return nil, errors.New("rate limit exceeded for tool test")
			}

			wrapped := middleware.WrapToolWithRetry("testTool", handler, 3)
			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{},
			}

			_, err := wrapped(context.Background(), req)
			if err == nil {
				t.Error("Expected rate limit error")
			}
			if callCount != 1 {
				t.Errorf("Expected 1 call (no retry), got %d", callCount)
			}
		})

		t.Run("RetryBackoff", func(t *testing.T) {
			callTimes := []time.Time{}

			handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				callTimes = append(callTimes, time.Now())
				return nil, errors.New("error")
			}

			wrapped := middleware.WrapToolWithRetry("testTool", handler, 2)
			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{},
			}

			_, _ = wrapped(context.Background(), req)

			// Verify backoff timing
			if len(callTimes) != 3 { // Initial + 2 retries
				t.Errorf("Expected 3 calls, got %d", len(callTimes))
			}

			// First retry should have ~100ms backoff
			if len(callTimes) >= 2 {
				firstBackoff := callTimes[1].Sub(callTimes[0])
				if firstBackoff < 90*time.Millisecond || firstBackoff > 110*time.Millisecond {
					t.Errorf("Expected ~100ms first backoff, got %v", firstBackoff)
				}
			}

			// Second retry should have ~200ms backoff
			if len(callTimes) >= 3 {
				secondBackoff := callTimes[2].Sub(callTimes[1])
				if secondBackoff < 190*time.Millisecond || secondBackoff > 210*time.Millisecond {
					t.Errorf("Expected ~200ms second backoff, got %v", secondBackoff)
				}
			}
		})
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr || len(substr) == 0 ||
		(len(s) >= len(substr) && s[:len(substr)] == substr) ||
		(len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 1; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

