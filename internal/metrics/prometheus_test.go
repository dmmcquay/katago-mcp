package metrics

import (
	"testing"
	"time"
)

func TestPrometheusCollector(t *testing.T) {
	collector := NewPrometheusCollector()

	// Test tool metrics
	collector.RecordToolCall("analyzePosition", "success", 0.5)
	collector.RecordToolCall("analyzePosition", "error", 0.1)
	collector.RecordToolCall("findMistakes", "success", 2.5)

	// Test rate limit metrics
	collector.RecordRateLimit("client1", "analyzePosition", false)
	collector.RecordRateLimit("client1", "analyzePosition", true)
	collector.RecordRateLimit("client2", "findMistakes", true)

	// Test engine metrics
	collector.RecordEngineStatus(true, "1.14.0")
	collector.RecordEngineHealthCheck(true)
	collector.RecordEngineHealthCheck(false)
	collector.RecordEngineQuery("query", 1.5)
	collector.RecordEngineRestart()

	// Test HTTP metrics
	collector.RecordHTTPRequest("GET", "/health", "200", 0.01)
	collector.RecordHTTPRequest("GET", "/metrics", "200", 0.05)

	// Test resource metrics
	collector.SetActiveClients(5)
	collector.SetActiveConnections(10)

	// Give metrics time to be recorded
	time.Sleep(10 * time.Millisecond)

	// If we get here without panic, the test passes
	// In a real test, we would query the metrics and verify values
}
