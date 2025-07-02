package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	prometheusOnce     sync.Once
	prometheusInstance *PrometheusCollector
)

// PrometheusCollector provides Prometheus metrics for the KataGo MCP server.
type PrometheusCollector struct {
	// MCP Tool metrics
	toolCallsTotal   *prometheus.CounterVec
	toolErrorsTotal  *prometheus.CounterVec
	toolDurationSecs *prometheus.HistogramVec

	// Rate limit metrics
	rateLimitHitsTotal   *prometheus.CounterVec
	rateLimitChecksTotal prometheus.Counter

	// KataGo engine metrics
	engineStatus        *prometheus.GaugeVec
	engineRestartsTotal prometheus.Counter
	engineHealthChecks  *prometheus.CounterVec
	engineQueryDuration *prometheus.HistogramVec

	// HTTP metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec

	// Resource metrics
	activeClients     prometheus.Gauge
	activeConnections prometheus.Gauge

	// Cache metrics
	cacheHitsTotal   prometheus.Counter
	cacheMissesTotal prometheus.Counter
	cacheSize        prometheus.Gauge
	cacheItems       prometheus.Gauge
}

// NewPrometheusCollector creates a new Prometheus metrics collector (singleton).
func NewPrometheusCollector() *PrometheusCollector {
	prometheusOnce.Do(func() {
		prometheusInstance = &PrometheusCollector{
			// MCP Tool metrics
			toolCallsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "katago_mcp_tool_calls_total",
					Help: "Total number of MCP tool calls",
				},
				[]string{"tool", "status"},
			),
			toolErrorsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "katago_mcp_tool_errors_total",
					Help: "Total number of MCP tool errors",
				},
				[]string{"tool", "error_type"},
			),
			toolDurationSecs: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "katago_mcp_tool_duration_seconds",
					Help:    "Duration of MCP tool calls in seconds",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"tool"},
			),

			// Rate limit metrics
			rateLimitHitsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "katago_mcp_rate_limit_hits_total",
					Help: "Total number of rate limit hits",
				},
				[]string{"client", "tool"},
			),
			rateLimitChecksTotal: promauto.NewCounter(
				prometheus.CounterOpts{
					Name: "katago_mcp_rate_limit_checks_total",
					Help: "Total number of rate limit checks",
				},
			),

			// KataGo engine metrics
			engineStatus: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "katago_engine_status",
					Help: "Status of the KataGo engine (1=running, 0=stopped)",
				},
				[]string{"version"},
			),
			engineRestartsTotal: promauto.NewCounter(
				prometheus.CounterOpts{
					Name: "katago_engine_restarts_total",
					Help: "Total number of KataGo engine restarts",
				},
			),
			engineHealthChecks: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "katago_engine_health_checks_total",
					Help: "Total number of KataGo engine health checks",
				},
				[]string{"status"},
			),
			engineQueryDuration: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "katago_engine_query_duration_seconds",
					Help:    "Duration of KataGo engine queries in seconds",
					Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10},
				},
				[]string{"query_type"},
			),

			// HTTP metrics
			httpRequestsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "katago_mcp_http_requests_total",
					Help: "Total number of HTTP requests",
				},
				[]string{"method", "path", "status"},
			),
			httpRequestDuration: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "katago_mcp_http_request_duration_seconds",
					Help:    "Duration of HTTP requests in seconds",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"method", "path"},
			),

			// Resource metrics
			activeClients: promauto.NewGauge(
				prometheus.GaugeOpts{
					Name: "katago_mcp_active_clients",
					Help: "Number of active MCP clients",
				},
			),
			activeConnections: promauto.NewGauge(
				prometheus.GaugeOpts{
					Name: "katago_mcp_active_connections",
					Help: "Number of active connections",
				},
			),

			// Cache metrics
			cacheHitsTotal: promauto.NewCounter(
				prometheus.CounterOpts{
					Name: "katago_mcp_cache_hits_total",
					Help: "Total number of cache hits",
				},
			),
			cacheMissesTotal: promauto.NewCounter(
				prometheus.CounterOpts{
					Name: "katago_mcp_cache_misses_total",
					Help: "Total number of cache misses",
				},
			),
			cacheSize: promauto.NewGauge(
				prometheus.GaugeOpts{
					Name: "katago_mcp_cache_size_bytes",
					Help: "Current cache size in bytes",
				},
			),
			cacheItems: promauto.NewGauge(
				prometheus.GaugeOpts{
					Name: "katago_mcp_cache_items",
					Help: "Current number of items in cache",
				},
			),
		}
	})
	return prometheusInstance
}

// RecordToolCall records a tool call metric.
func (p *PrometheusCollector) RecordToolCall(tool, status string, durationSecs float64) {
	p.toolCallsTotal.WithLabelValues(tool, status).Inc()
	p.toolDurationSecs.WithLabelValues(tool).Observe(durationSecs)

	if status == "error" {
		p.toolErrorsTotal.WithLabelValues(tool, "general").Inc()
	}
}

// RecordRateLimit records a rate limit event.
func (p *PrometheusCollector) RecordRateLimit(client, tool string, hit bool) {
	p.rateLimitChecksTotal.Inc()
	if hit {
		p.rateLimitHitsTotal.WithLabelValues(client, tool).Inc()
	}
}

// RecordEngineStatus records the current engine status.
func (p *PrometheusCollector) RecordEngineStatus(running bool, version string) {
	value := 0.0
	if running {
		value = 1.0
	}
	p.engineStatus.WithLabelValues(version).Set(value)
}

// RecordEngineRestart records an engine restart.
func (p *PrometheusCollector) RecordEngineRestart() {
	p.engineRestartsTotal.Inc()
}

// RecordEngineHealthCheck records a health check result.
func (p *PrometheusCollector) RecordEngineHealthCheck(success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	p.engineHealthChecks.WithLabelValues(status).Inc()
}

// RecordEngineQuery records an engine query duration.
func (p *PrometheusCollector) RecordEngineQuery(queryType string, durationSecs float64) {
	p.engineQueryDuration.WithLabelValues(queryType).Observe(durationSecs)
}

// RecordHTTPRequest records an HTTP request.
func (p *PrometheusCollector) RecordHTTPRequest(method, path, status string, durationSecs float64) {
	p.httpRequestsTotal.WithLabelValues(method, path, status).Inc()
	p.httpRequestDuration.WithLabelValues(method, path).Observe(durationSecs)
}

// SetActiveClients sets the number of active clients.
func (p *PrometheusCollector) SetActiveClients(count float64) {
	p.activeClients.Set(count)
}

// SetActiveConnections sets the number of active connections.
func (p *PrometheusCollector) SetActiveConnections(count float64) {
	p.activeConnections.Set(count)
}

// RecordCacheHit records a cache hit.
func (p *PrometheusCollector) RecordCacheHit() {
	p.cacheHitsTotal.Inc()
}

// RecordCacheMiss records a cache miss.
func (p *PrometheusCollector) RecordCacheMiss() {
	p.cacheMissesTotal.Inc()
}

// SetCacheStats sets the current cache statistics.
func (p *PrometheusCollector) SetCacheStats(items, sizeBytes float64) {
	p.cacheItems.Set(items)
	p.cacheSize.Set(sizeBytes)
}
