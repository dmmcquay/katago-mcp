package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/logging"
)

// Status represents the health status of a component.
type Status string

const (
	// StatusHealthy indicates the component is healthy.
	StatusHealthy Status = "healthy"
	// StatusUnhealthy indicates the component is unhealthy.
	StatusUnhealthy Status = "unhealthy"
	// StatusDegraded indicates the component is working but degraded.
	StatusDegraded Status = "degraded"
)

// Check represents a health check function.
type Check func(ctx context.Context) error

// Component represents a system component with health status.
type Component struct {
	Name        string                 `json:"name"`
	Status      Status                 `json:"status"`
	Message     string                 `json:"message,omitempty"`
	LastChecked time.Time              `json:"last_checked"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Response represents the health check response.
type Response struct {
	Status     Status      `json:"status"`
	Timestamp  time.Time   `json:"timestamp"`
	Components []Component `json:"components,omitempty"`
	Version    string      `json:"version,omitempty"`
	GitCommit  string      `json:"git_commit,omitempty"`
}

// Checker manages health checks for the application.
type Checker struct {
	logger    logging.ContextLogger
	checks    map[string]Check
	mu        sync.RWMutex
	version   string
	gitCommit string
}

// NewChecker creates a new health checker.
func NewChecker(logger logging.ContextLogger, version, gitCommit string) *Checker {
	return &Checker{
		logger:    logger,
		checks:    make(map[string]Check),
		version:   version,
		gitCommit: gitCommit,
	}
}

// RegisterCheck registers a health check for a component.
func (c *Checker) RegisterCheck(name string, check Check) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks[name] = check
}

// CheckHealth performs all registered health checks.
func (c *Checker) CheckHealth(ctx context.Context) Response {
	c.mu.RLock()
	defer c.mu.RUnlock()

	response := Response{
		Status:     StatusHealthy,
		Timestamp:  time.Now().UTC(),
		Version:    c.version,
		GitCommit:  c.gitCommit,
		Components: make([]Component, 0, len(c.checks)),
	}

	// If no checks registered, consider it healthy
	if len(c.checks) == 0 {
		return response
	}

	// Run all checks in parallel
	type result struct {
		name      string
		component Component
	}

	results := make(chan result, len(c.checks))
	var wg sync.WaitGroup

	for name, check := range c.checks {
		wg.Add(1)
		go func(name string, check Check) {
			defer wg.Done()

			component := Component{
				Name:        name,
				Status:      StatusHealthy,
				LastChecked: time.Now().UTC(),
			}

			// Create a timeout context for each check
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			if err := check(checkCtx); err != nil {
				component.Status = StatusUnhealthy
				component.Message = err.Error()
				c.logger.WithField("component", name).Error("Health check failed", "error", err)
			}

			results <- result{name: name, component: component}
		}(name, check)
	}

	// Wait for all checks to complete
	wg.Wait()
	close(results)

	// Collect results and determine overall status
	hasUnhealthy := false
	for res := range results {
		response.Components = append(response.Components, res.component)
		if res.component.Status == StatusUnhealthy {
			hasUnhealthy = true
		}
	}

	if hasUnhealthy {
		response.Status = StatusUnhealthy
	}

	return response
}

// LivenessHandler returns an HTTP handler for liveness checks.
func (c *Checker) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Simple liveness check - if we can handle requests, we're alive
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := Response{
			Status:    StatusHealthy,
			Timestamp: time.Now().UTC(),
			Version:   c.version,
			GitCommit: c.gitCommit,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			c.logger.Error("Failed to encode liveness response", "error", err)
		}
	}
}

// ReadinessHandler returns an HTTP handler for readiness checks.
func (c *Checker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Add correlation ID for tracing
		ctx = logging.ContextWithCorrelationID(ctx, logging.GenerateCorrelationID())
		logger := c.logger.WithContext(ctx)

		logger.Debug("Performing readiness check")

		response := c.CheckHealth(ctx)

		w.Header().Set("Content-Type", "application/json")

		// Set appropriate status code
		statusCode := http.StatusOK
		if response.Status != StatusHealthy {
			statusCode = http.StatusServiceUnavailable
		}

		w.WriteHeader(statusCode)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Error("Failed to encode readiness response", "error", err)
		}
	}
}
