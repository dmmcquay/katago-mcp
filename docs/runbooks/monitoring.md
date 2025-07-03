# Monitoring and Alerting

This guide covers setting up comprehensive monitoring and alerting for the KataGo MCP server.

## Overview

The KataGo MCP server exposes metrics via Prometheus and provides health endpoints for monitoring service health, performance, and operational status.

## Metrics Endpoints

### Health Endpoints
- **Health Check**: `GET /health` - Basic service health
- **Readiness Check**: `GET /ready` - Service readiness for traffic
- **Metrics**: `GET /metrics` - Prometheus metrics

### Key Metrics Categories

1. **Engine Metrics**
   - Engine uptime and status
   - Analysis request duration
   - Query success/failure rates
   - Engine restart events

2. **Cache Metrics**
   - Cache hit/miss rates
   - Cache size and utilization
   - Eviction events
   - TTL expiration statistics

3. **System Metrics**
   - Memory usage
   - CPU utilization
   - Disk I/O
   - Network connections

4. **Application Metrics**
   - Request rates
   - Error rates
   - Response times
   - Concurrent connections

## Prometheus Configuration

### 1. Prometheus Setup

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "katago_mcp_rules.yml"

scrape_configs:
  - job_name: 'katago-mcp'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 5s
    metrics_path: /metrics
    
alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

### 2. Alert Rules

Create `/etc/prometheus/katago_mcp_rules.yml`:

```yaml
groups:
- name: katago_mcp_alerts
  rules:
  
  # Engine Health
  - alert: KataGoEngineDown
    expr: katago_engine_up == 0
    for: 30s
    labels:
      severity: critical
    annotations:
      summary: "KataGo engine is down"
      description: "KataGo engine has been down for more than 30 seconds"

  - alert: KataGoEngineRestarting
    expr: increase(katago_engine_restarts_total[5m]) > 2
    for: 0s
    labels:
      severity: warning
    annotations:
      summary: "KataGo engine restarting frequently"
      description: "KataGo engine has restarted {{ $value }} times in the last 5 minutes"

  # Performance
  - alert: HighAnalysisLatency
    expr: histogram_quantile(0.95, katago_analysis_duration_seconds) > 30
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "High analysis latency detected"
      description: "95th percentile analysis latency is {{ $value }}s"

  - alert: HighErrorRate
    expr: rate(katago_requests_total{status="error"}[5m]) / rate(katago_requests_total[5m]) > 0.1
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "High error rate detected"
      description: "Error rate is {{ $value | humanizePercentage }}"

  # Cache Performance
  - alert: LowCacheHitRate
    expr: katago_cache_hit_rate < 0.5
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Low cache hit rate"
      description: "Cache hit rate is {{ $value | humanizePercentage }}"

  - alert: CacheMemoryHigh
    expr: katago_cache_size_bytes / katago_cache_max_size_bytes > 0.9
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "Cache memory usage high"
      description: "Cache is using {{ $value | humanizePercentage }} of available memory"

  # System Resources
  - alert: HighMemoryUsage
    expr: process_resident_memory_bytes > 8e9  # 8GB
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High memory usage"
      description: "Memory usage is {{ $value | humanizeBytes }}"

  - alert: ServiceDown
    expr: up{job="katago-mcp"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "KataGo MCP service is down"
      description: "KataGo MCP service has been down for more than 1 minute"
```

## Grafana Dashboard

### 1. Dashboard JSON

Create a comprehensive Grafana dashboard:

```json
{
  "dashboard": {
    "id": null,
    "title": "KataGo MCP Server",
    "tags": ["katago", "mcp"],
    "timezone": "browser",
    "refresh": "5s",
    "time": {
      "from": "now-1h",
      "to": "now"
    },
    "panels": [
      {
        "id": 1,
        "title": "Engine Status",
        "type": "stat",
        "targets": [
          {
            "expr": "katago_engine_up",
            "legendFormat": "Engine Status"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "color": {
              "mode": "thresholds"
            },
            "thresholds": {
              "steps": [
                {"color": "red", "value": 0},
                {"color": "green", "value": 1}
              ]
            },
            "mappings": [
              {"options": {"0": {"text": "DOWN"}}, "type": "value"},
              {"options": {"1": {"text": "UP"}}, "type": "value"}
            ]
          }
        }
      },
      {
        "id": 2,
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(katago_requests_total[1m])",
            "legendFormat": "Requests/sec"
          }
        ]
      },
      {
        "id": 3,
        "title": "Analysis Latency",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.50, katago_analysis_duration_seconds)",
            "legendFormat": "p50"
          },
          {
            "expr": "histogram_quantile(0.95, katago_analysis_duration_seconds)",
            "legendFormat": "p95"
          },
          {
            "expr": "histogram_quantile(0.99, katago_analysis_duration_seconds)",
            "legendFormat": "p99"
          }
        ]
      },
      {
        "id": 4,
        "title": "Cache Hit Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "katago_cache_hit_rate",
            "legendFormat": "Hit Rate"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "percentunit",
            "color": {
              "mode": "thresholds"
            },
            "thresholds": {
              "steps": [
                {"color": "red", "value": 0},
                {"color": "yellow", "value": 0.5},
                {"color": "green", "value": 0.8}
              ]
            }
          }
        }
      },
      {
        "id": 5,
        "title": "Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "process_resident_memory_bytes",
            "legendFormat": "RSS Memory"
          },
          {
            "expr": "katago_cache_size_bytes",
            "legendFormat": "Cache Memory"
          }
        ],
        "yAxes": [
          {
            "unit": "bytes"
          }
        ]
      }
    ]
  }
}
```

### 2. Key Dashboard Panels

**Engine Health Panel**:
- Engine up/down status
- Engine version
- Last restart time
- Health check success rate

**Performance Panel**:
- Request rate (requests/second)
- Analysis latency percentiles (p50, p95, p99)
- Error rates by type
- Concurrent connections

**Cache Panel**:
- Hit/miss rates
- Cache size utilization
- Eviction events
- TTL expiration events

**System Panel**:
- Memory usage (RSS, heap, cache)
- CPU utilization
- Disk I/O
- Network connections

## Alertmanager Configuration

### 1. Alertmanager Setup

```yaml
# alertmanager.yml
global:
  smtp_smarthost: 'localhost:587'
  smtp_from: 'alerts@yourcompany.com'

route:
  group_by: ['alertname', 'severity']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'web.hook'
  routes:
  - match:
      severity: critical
    receiver: 'critical-alerts'
  - match:
      severity: warning
    receiver: 'warning-alerts'

receivers:
- name: 'web.hook'
  webhook_configs:
  - url: 'http://127.0.0.1:5001/'

- name: 'critical-alerts'
  email_configs:
  - to: 'oncall@yourcompany.com'
    subject: '[CRITICAL] KataGo MCP Alert'
    body: |
      Alert: {{ .GroupLabels.alertname }}
      Severity: {{ .GroupLabels.severity }}
      
      {{ range .Alerts }}
      Description: {{ .Annotations.description }}
      {{ end }}
  slack_configs:
  - api_url: 'YOUR_SLACK_WEBHOOK_URL'
    channel: '#alerts'
    title: 'KataGo MCP Critical Alert'
    text: '{{ .CommonAnnotations.summary }}'

- name: 'warning-alerts'
  email_configs:
  - to: 'team@yourcompany.com'
    subject: '[WARNING] KataGo MCP Alert'
    body: |
      Alert: {{ .GroupLabels.alertname }}
      
      {{ range .Alerts }}
      Description: {{ .Annotations.description }}
      {{ end }}
```

## Log Monitoring

### 1. Log Aggregation with ELK Stack

**Filebeat Configuration**:
```yaml
# filebeat.yml
filebeat.inputs:
- type: journald
  id: katago-mcp-logs
  include_matches:
    - "_SYSTEMD_UNIT=katago-mcp.service"

output.elasticsearch:
  hosts: ["localhost:9200"]
  index: "katago-mcp-%{+yyyy.MM.dd}"

setup.template.name: "katago-mcp"
setup.template.pattern: "katago-mcp-*"
```

**Log Parsing**:
```json
{
  "mappings": {
    "properties": {
      "timestamp": {"type": "date"},
      "level": {"type": "keyword"},
      "message": {"type": "text"},
      "correlation_id": {"type": "keyword"},
      "component": {"type": "keyword"},
      "duration": {"type": "float"},
      "error": {"type": "text"}
    }
  }
}
```

### 2. Key Log Metrics to Monitor

- Error rates by log level
- Analysis request patterns
- Engine restart events
- Cache performance patterns
- Security-related events

## Health Check Scripts

### 1. Basic Health Check

```bash
#!/bin/bash
# health_check.sh

HEALTH_URL="http://localhost:8080/health"
READY_URL="http://localhost:8080/ready"

# Check health endpoint
if ! curl -sf "$HEALTH_URL" > /dev/null; then
    echo "CRITICAL: Health check failed"
    exit 2
fi

# Check readiness endpoint
if ! curl -sf "$READY_URL" > /dev/null; then
    echo "WARNING: Readiness check failed"
    exit 1
fi

echo "OK: Service is healthy and ready"
exit 0
```

### 2. Comprehensive Service Check

```bash
#!/bin/bash
# service_check.sh

set -e

SERVICE_NAME="katago-mcp"
TIMEOUT=30

echo "Checking $SERVICE_NAME service..."

# Check systemd service status
if ! systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "CRITICAL: Service $SERVICE_NAME is not active"
    exit 2
fi

# Check process is running
if ! pgrep -f katago-mcp > /dev/null; then
    echo "CRITICAL: No katago-mcp process found"
    exit 2
fi

# Check health endpoints
timeout "$TIMEOUT" curl -sf http://localhost:8080/health || {
    echo "CRITICAL: Health endpoint failed"
    exit 2
}

timeout "$TIMEOUT" curl -sf http://localhost:8080/ready || {
    echo "WARNING: Ready endpoint failed"
    exit 1
}

# Check metrics endpoint
timeout "$TIMEOUT" curl -sf http://localhost:9090/metrics | grep -q katago_engine_up || {
    echo "WARNING: Metrics endpoint failed"
    exit 1
}

echo "OK: All checks passed"
exit 0
```

## Automated Monitoring Setup

### 1. Deployment Script

```bash
#!/bin/bash
# deploy_monitoring.sh

set -e

echo "Setting up KataGo MCP monitoring..."

# Install Prometheus
if ! command -v prometheus &> /dev/null; then
    echo "Installing Prometheus..."
    wget https://github.com/prometheus/prometheus/releases/latest/download/prometheus-*-linux-amd64.tar.gz
    tar xvf prometheus-*-linux-amd64.tar.gz
    sudo cp prometheus-*/prometheus /usr/local/bin/
    sudo cp prometheus-*/promtool /usr/local/bin/
fi

# Install Grafana
if ! command -v grafana-server &> /dev/null; then
    echo "Installing Grafana..."
    wget -q -O - https://packages.grafana.com/gpg.key | sudo apt-key add -
    echo "deb https://packages.grafana.com/oss/deb stable main" | sudo tee /etc/apt/sources.list.d/grafana.list
    sudo apt update && sudo apt install grafana
fi

# Configure Prometheus
sudo cp prometheus.yml /etc/prometheus/
sudo cp katago_mcp_rules.yml /etc/prometheus/

# Start services
sudo systemctl enable --now prometheus
sudo systemctl enable --now grafana-server

echo "Monitoring setup complete!"
echo "Prometheus: http://localhost:9090"
echo "Grafana: http://localhost:3000 (admin/admin)"
```

## Next Steps

After setting up monitoring:
1. Configure [log analysis](log-analysis.md) for detailed troubleshooting
2. Set up [performance tuning](performance-tuning.md) based on metrics
3. Review [incident response](incident-response.md) procedures
4. Plan [capacity scaling](scaling.md) based on usage patterns