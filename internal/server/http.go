package server

import (
	"context"
	"net/http"
	"time"

	"github.com/dmmcquay/katago-mcp/internal/health"
	"github.com/dmmcquay/katago-mcp/internal/logging"
	"github.com/dmmcquay/katago-mcp/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HTTPServer provides HTTP endpoints for health checks and metrics.
type HTTPServer struct {
	server     *http.Server
	logger     logging.ContextLogger
	checker    *health.Checker
	prometheus *metrics.PrometheusCollector
}

// NewHTTPServer creates a new HTTP server for health checks and metrics.
func NewHTTPServer(addr string, logger logging.ContextLogger, checker *health.Checker) *HTTPServer {
	prometheus := metrics.NewPrometheusCollector()

	mux := http.NewServeMux()

	// Register health endpoints
	mux.HandleFunc("/health", checker.LivenessHandler())
	mux.HandleFunc("/ready", checker.ReadinessHandler())

	// Register metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Apply middleware
	handler := PrometheusMiddleware(prometheus)(mux)

	return &HTTPServer{
		server: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger:     logger,
		checker:    checker,
		prometheus: prometheus,
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
