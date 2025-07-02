package server

import (
	"context"
	"net/http"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/health"
	"github.com/dmmcquay/katago-mcp/internal/logging"
)

// HTTPServer provides HTTP endpoints for health checks.
type HTTPServer struct {
	server  *http.Server
	logger  logging.ContextLogger
	checker *health.Checker
}

// NewHTTPServer creates a new HTTP server for health checks.
func NewHTTPServer(addr string, logger logging.ContextLogger, checker *health.Checker) *HTTPServer {
	mux := http.NewServeMux()

	// Register health endpoints
	mux.HandleFunc("/health", checker.LivenessHandler())
	mux.HandleFunc("/ready", checker.ReadinessHandler())

	return &HTTPServer{
		server: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger:  logger,
		checker: checker,
	}
}

// Start starts the HTTP server.
func (s *HTTPServer) Start() error {
	s.logger.Info("Starting HTTP health check server", "addr", s.server.Addr)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully stops the HTTP server.
func (s *HTTPServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP health check server")
	return s.server.Shutdown(ctx)
}
