# Docker Deployment Guide

This guide covers deploying katago-mcp using Docker in production environments.

## Production Dockerfile Features

The production `Dockerfile` includes:

- **Multi-stage build** - Optimized for security and size
- **Non-root user** - Runs as user `katago` (UID 1000) for security
- **Minimal base image** - Uses debian:bookworm-slim for smaller attack surface
- **Health checks** - Built-in Docker health checks via HTTP endpoints
- **Version injection** - Build-time version information
- **KataGo integration** - Pre-built KataGo binary from base image

## Quick Start

### Build the Image

```bash
# Using the build script (recommended)
./docker-build.sh --tag v0.1.0

# Or manually
docker build -t katago-mcp:v0.1.0 \
  --build-arg GIT_COMMIT=$(git rev-parse HEAD) \
  --build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  .
```

### Run with Docker

```bash
# Basic run
docker run --rm -p 8080:8080 katago-mcp:v0.1.0

# With custom configuration
docker run --rm -p 8080:8080 \
  -v $(pwd)/config.production.json:/app/config/config.json:ro \
  katago-mcp:v0.1.0
```

### Run with Docker Compose

```bash
# Production deployment
docker-compose -f docker-compose.production.yml up -d

# Check status
docker-compose -f docker-compose.production.yml ps
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KATAGO_MCP_CONFIG` | `/app/config/config.json` | Path to configuration file |
| `KATAGO_BINARY_PATH` | `/usr/local/bin/katago` | Path to KataGo binary |
| `KATAGO_CONFIG_PATH` | `/app/config/analysis.cfg` | Path to KataGo config |
| `KATAGO_HTTP_PORT` | `8080` | HTTP health check port |
| `KATAGO_MCP_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `KATAGO_LOG_FORMAT` | `json` | Log format (json, text) |

### Volume Mounts

| Host Path | Container Path | Purpose |
|-----------|----------------|---------|
| `./config/` | `/app/config/` | Configuration files |
| `./models/` | `/app/models/` | KataGo neural network models |
| `./logs/` | `/app/logs/` | Application logs (optional) |

## Health Checks

The container exposes health check endpoints:

- `GET /health` - Liveness probe (server health)
- `GET /ready` - Readiness probe (KataGo engine health)

Example health check response:
```json
{
  "status": "healthy",
  "timestamp": "2025-07-02T12:00:00Z",
  "checks": {
    "katago": "healthy"
  },
  "version": "0.1.0",
  "gitCommit": "abc123"
}
```

## Security Features

### Container Security

- **Non-root user** - Runs as UID 1000 with no shell access
- **Read-only filesystem** - Container filesystem is read-only
- **No new privileges** - Prevents privilege escalation
- **Minimal dependencies** - Only required runtime libraries
- **Security scanning** - Automated vulnerability scans in CI

### Network Security

- **Single port exposure** - Only port 8080 for health checks
- **No external dependencies** - Fully self-contained
- **MCP over stdio** - Primary communication via stdin/stdout

## Resource Requirements

### Minimum Requirements

- **CPU**: 1 core
- **Memory**: 1GB RAM
- **Storage**: 500MB (without custom models)

### Recommended for Production

- **CPU**: 2-4 cores
- **Memory**: 2-4GB RAM
- **Storage**: 2GB (with multiple models)

### KataGo-Specific Requirements

- **CPU**: Modern x86_64 processor with AVX2 support
- **Memory**: Additional 500MB-1GB per concurrent analysis
- **Models**: 50-200MB per neural network model

## Kubernetes Deployment

### Basic Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: katago-mcp
spec:
  replicas: 2
  selector:
    matchLabels:
      app: katago-mcp
  template:
    metadata:
      labels:
        app: katago-mcp
    spec:
      containers:
      - name: katago-mcp
        image: ghcr.io/dmmcquay/katago-mcp:v0.1.0
        ports:
        - containerPort: 8080
        resources:
          requests:
            cpu: 1000m
            memory: 1Gi
          limits:
            cpu: 2000m
            memory: 2Gi
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: katago-mcp-service
spec:
  selector:
    app: katago-mcp
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
```

## Monitoring and Observability

### Health Monitoring

```bash
# Check container health
docker inspect katago-mcp --format='{{.State.Health.Status}}'

# Check application health
curl http://localhost:8080/health

# Check readiness
curl http://localhost:8080/ready
```

### Logs

```bash
# View container logs
docker logs katago-mcp

# Follow logs
docker logs -f katago-mcp

# With docker-compose
docker-compose -f docker-compose.production.yml logs -f
```

### Metrics

The application outputs structured JSON logs suitable for:

- **ELK Stack** - Elasticsearch, Logstash, Kibana
- **Loki** - Grafana Loki for log aggregation
- **Prometheus** - Log-based metrics with promtail

## Troubleshooting

### Common Issues

#### Container Won't Start

```bash
# Check logs for errors
docker logs katago-mcp

# Common causes:
# - Missing KataGo model files
# - Invalid configuration
# - Insufficient memory
```

#### Health Checks Failing

```bash
# Test health endpoint manually
curl -v http://localhost:8080/health

# Check KataGo process
docker exec katago-mcp ps aux | grep katago
```

#### Performance Issues

```bash
# Check resource usage
docker stats katago-mcp

# Monitor KataGo analysis times
docker logs katago-mcp | grep "analysis completed"
```

### Debug Mode

Enable debug logging:

```bash
docker run --rm -p 8080:8080 \
  -e KATAGO_MCP_LOG_LEVEL=debug \
  katago-mcp:v0.1.0
```

## CI/CD Integration

The production Dockerfile is built automatically in CI:

- **Automated builds** - On every push to main
- **Security scanning** - Trivy vulnerability scans
- **Multi-arch support** - linux/amd64 and linux/arm64
- **Registry publishing** - Pushed to GitHub Container Registry

### Build Arguments

| Argument | Description | Example |
|----------|-------------|---------|
| `GIT_COMMIT` | Git commit hash | `abc123def456` |
| `BUILD_TIME` | Build timestamp | `2025-07-02T12:00:00Z` |

## Best Practices

1. **Use specific tags** - Avoid `latest` in production
2. **Resource limits** - Set appropriate CPU/memory limits
3. **Health checks** - Configure proper liveness/readiness probes
4. **Security scanning** - Regularly scan images for vulnerabilities
5. **Log aggregation** - Collect and analyze structured logs
6. **Backup models** - Keep neural network models in persistent storage
7. **Rolling updates** - Use rolling deployments for zero downtime

## Support

For issues related to Docker deployment:

1. Check container logs for errors
2. Verify health endpoints are accessible
3. Ensure sufficient resources are allocated
4. Review configuration file syntax
5. Test with minimal configuration first