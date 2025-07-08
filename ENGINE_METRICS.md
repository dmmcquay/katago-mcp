# Engine Performance Metrics Implementation

## Overview

This document focuses specifically on extracting and exposing KataGo engine performance metrics for monitoring, optimization, and debugging.

## Available Performance Data

### 1. Query-Level Metrics (What we can measure)

```go
type QueryMetrics struct {
    QueryID       string  `json:"queryId"`
    TotalTime     float64 `json:"totalTime"`      // Total query time in seconds
    Visits        int     `json:"visits"`         // Number of search visits performed
    VisitsPerSec  float64 `json:"visitsPerSec"`   // Search efficiency (visits/totalTime)
    QueueWaitTime float64 `json:"queueWaitTime"`  // Time spent waiting in queue
}
```

### 2. Engine Process Metrics

```go
type ProcessMetrics struct {
    Uptime        time.Duration `json:"uptime"`
    TotalQueries  int64         `json:"totalQueries"`
    ActiveQueries int           `json:"activeQueries"`
    CPUPercent    float64       `json:"cpuPercent"`    // Process CPU usage
    MemoryMB      float64       `json:"memoryMB"`      // Process memory usage
    ThreadCount   int           `json:"threadCount"`   // Active threads
}
```

### 3. Performance Statistics

```go
type PerformanceStats struct {
    AvgQueryTime     float64 `json:"avgQueryTime"`     // Rolling average
    AvgVisitsPerSec  float64 `json:"avgVisitsPerSec"`  // Search efficiency
    P95QueryTime     float64 `json:"p95QueryTime"`     // 95th percentile
    P99QueryTime     float64 `json:"p99QueryTime"`     // 99th percentile
    QueriesPerMinute float64 `json:"queriesPerMinute"` // Throughput
}
```

## Implementation Approach

### Phase 1: Enhanced Query Metrics

Modify the engine to track per-query performance:

```go
func (e *Engine) sendQuery(query map[string]interface{}) (*Response, error) {
    metrics := &QueryMetrics{
        QueryID: query["id"].(string),
    }
    
    start := time.Now()
    // ... existing query logic ...
    
    // After receiving response:
    metrics.TotalTime = time.Since(start).Seconds()
    if resp.RootInfo.Visits > 0 {
        metrics.Visits = resp.RootInfo.Visits
        metrics.VisitsPerSec = float64(metrics.Visits) / metrics.TotalTime
    }
    
    // Store metrics for aggregation
    e.storeMetrics(metrics)
}
```

### Phase 2: Process Monitoring

Add process monitoring using the standard library:

```go
import (
    "runtime"
    "github.com/shirou/gopsutil/v3/process"
)

func (e *Engine) collectProcessMetrics() (*ProcessMetrics, error) {
    proc, err := process.NewProcess(int32(e.cmd.Process.Pid))
    if err != nil {
        return nil, err
    }
    
    cpuPercent, _ := proc.CPUPercent()
    memInfo, _ := proc.MemoryInfo()
    threads, _ := proc.NumThreads()
    
    return &ProcessMetrics{
        Uptime:        time.Since(e.startTime),
        TotalQueries:  atomic.LoadInt64(&e.totalQueries),
        ActiveQueries: len(e.pending),
        CPUPercent:    cpuPercent,
        MemoryMB:      float64(memInfo.RSS) / 1024 / 1024,
        ThreadCount:   int(threads),
    }, nil
}
```

### Phase 3: New Tool - getEngineMetrics

Add a dedicated MCP tool for performance monitoring:

```typescript
tool: "getEngineMetrics"
parameters: {
  detailed?: boolean  // Include per-query breakdown
  period?: string    // Time period: "1m", "5m", "1h"
}

response: {
  process: {
    uptime: "2h 15m 30s"
    totalQueries: 1523
    activeQueries: 3
    cpuPercent: 45.2
    memoryMB: 512.3
    threadCount: 8
  }
  performance: {
    avgQueryTime: 0.823      // seconds
    avgVisitsPerSec: 1205.4  // search efficiency
    p95QueryTime: 1.542
    p99QueryTime: 2.103
    queriesPerMinute: 24.5
  }
  recentQueries?: [  // if detailed=true
    {
      queryId: "q1234"
      totalTime: 0.752
      visits: 1000
      visitsPerSec: 1329.8
      timestamp: "2024-01-15T10:30:45Z"
    }
  ]
}
```

## Use Cases

### 1. Performance Monitoring Dashboard
```bash
# Get current performance metrics
mcp call getEngineMetrics '{"detailed": false}'

# Monitor efficiency over time
watch -n 5 'mcp call getEngineMetrics | jq .performance.avgVisitsPerSec'
```

### 2. Capacity Planning
- Track queries per minute to understand load
- Monitor memory usage growth over time
- Identify performance degradation patterns

### 3. Optimization
- Compare visits/second across different positions
- Identify slow queries (p99 times)
- Tune maxVisits based on actual performance

### 4. Debugging
- Correlate high CPU usage with specific query patterns
- Identify memory leaks (growing memory usage)
- Track active queries during hangs

## Configuration

```json
{
  "metrics": {
    "enableProcessMonitoring": true,
    "metricsRetentionMinutes": 60,
    "detailedMetricsLimit": 100
  }
}
```

## Benefits

1. **Real-time Monitoring** - See how KataGo is performing right now
2. **Historical Analysis** - Track performance trends over time
3. **Resource Planning** - Understand hardware requirements
4. **Problem Detection** - Identify performance issues early
5. **Optimization** - Data-driven tuning of analysis parameters

## Example Output

```json
{
  "process": {
    "uptime": "45m 30s",
    "totalQueries": 523,
    "activeQueries": 2,
    "cpuPercent": 78.5,
    "memoryMB": 892.4,
    "threadCount": 16
  },
  "performance": {
    "avgQueryTime": 1.235,
    "avgVisitsPerSec": 985.3,
    "p95QueryTime": 2.841,
    "p99QueryTime": 4.102,
    "queriesPerMinute": 18.7
  }
}
```

This would give you actionable insights like:
- "KataGo is analyzing ~985 positions per second"
- "95% of queries complete within 2.8 seconds"
- "Currently using 78% CPU and 892MB RAM"
- "Processing ~19 queries per minute"