package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/logging"
	"github.com/dmmcquay/katago-mcp/internal/metrics"
	"github.com/dmmcquay/katago-mcp/internal/ratelimit"
	"github.com/mark3labs/mcp-go/mcp"
)

// Middleware wraps MCP tool handlers with common functionality like rate limiting, metrics, and logging.
type Middleware struct {
	logger      logging.ContextLogger
	metrics     *metrics.Collector
	rateLimiter *ratelimit.Limiter
}

// NewMiddleware creates a new middleware instance.
func NewMiddleware(logger logging.ContextLogger, metrics *metrics.Collector, rateLimiter *ratelimit.Limiter) *Middleware {
	return &Middleware{
		logger:      logger,
		metrics:     metrics,
		rateLimiter: rateLimiter,
	}
}

// ToolHandler is the function signature for MCP tool handlers.
type ToolHandler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)

// WrapTool wraps a tool handler with middleware functionality.
func (m *Middleware) WrapTool(toolName string, handler ToolHandler) ToolHandler {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()

		// Extract client ID from context or request
		clientID := extractClientID(ctx, request)

		// Log the request
		m.logger.Info("Tool request received",
			"tool", toolName,
			"client", clientID,
			"arguments", request.Params.Arguments,
		)

		// Check rate limits
		if m.rateLimiter != nil {
			allowed, err := m.rateLimiter.Allow(clientID, toolName)
			if !allowed {
				m.logger.Warn("Rate limit exceeded",
					"tool", toolName,
					"client", clientID,
					"error", err,
				)
				m.metrics.RecordToolCall(toolName, "rate_limited", time.Since(start))
				return nil, fmt.Errorf("rate limit exceeded for tool %s: %w", toolName, err)
			}
		}

		// Call the actual handler
		result, err := handler(ctx, request)

		// Record metrics
		status := "success"
		if err != nil {
			status = "error"
			m.logger.Error("Tool request failed",
				"tool", toolName,
				"client", clientID,
				"error", err,
				"duration", time.Since(start),
			)
		} else {
			m.logger.Info("Tool request completed",
				"tool", toolName,
				"client", clientID,
				"duration", time.Since(start),
			)
		}
		m.metrics.RecordToolCall(toolName, status, time.Since(start))

		return result, err
	}
}

// WrapToolWithRetry wraps a tool handler with retry logic in addition to standard middleware.
func (m *Middleware) WrapToolWithRetry(toolName string, handler ToolHandler, maxRetries int) ToolHandler {
	wrappedHandler := m.WrapTool(toolName, handler)

	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var lastErr error
		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				// Exponential backoff between retries
				// Safe conversion: attempt is always >= 1 and <= maxRetries (small number)
				shiftAmount := attempt - 1
				if shiftAmount > 10 { // Prevent overflow for large shift amounts
					shiftAmount = 10
				}
				backoff := time.Duration(1<<uint(shiftAmount)) * 100 * time.Millisecond // #nosec G115 -- shiftAmount is bounded
				m.logger.Debug("Retrying tool request",
					"tool", toolName,
					"attempt", attempt,
					"backoff", backoff,
				)
				time.Sleep(backoff)
			}

			result, err := wrappedHandler(ctx, request)
			if err == nil {
				return result, nil
			}

			// Don't retry rate limit errors
			if strings.Contains(err.Error(), "rate limit exceeded") {
				return nil, err
			}

			lastErr = err
		}

		return nil, fmt.Errorf("tool %s failed after %d retries: %w", toolName, maxRetries, lastErr)
	}
}

// extractClientID attempts to extract a client identifier from the context or request.
func extractClientID(ctx context.Context, request mcp.CallToolRequest) string {
	// First check context for client ID
	if clientID, ok := ctx.Value("clientID").(string); ok && clientID != "" {
		return clientID
	}

	// Check if arguments contain a client ID
	if request.Params.Arguments != nil {
		if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if clientID, ok := args["clientID"].(string); ok && clientID != "" {
				return clientID
			}
		}
	}

	// Default to "anonymous"
	return "anonymous"
}
