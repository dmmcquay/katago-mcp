# Production-ready Dockerfile for katago-mcp
# Multi-stage build for optimal security and size

# Build stage
FROM golang:1.23-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    git \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version injection
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# Build the binary with version information
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.GitCommit=${GIT_COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o katago-mcp ./cmd/katago-mcp

# KataGo base stage - get pre-built KataGo binary
FROM ghcr.io/dmmcquay/katago-base:v1.14.1 AS katago-base

# Final production stage
FROM debian:bookworm-slim

# Install only runtime dependencies
RUN apt-get update && apt-get install -y \
    libzip4 \
    libboost-filesystem-dev \
    libgoogle-perftools4 \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user for security
RUN groupadd -r katago && useradd -r -g katago -u 1000 -d /app -s /bin/false katago

# Create directories with proper permissions
RUN mkdir -p /app/config /app/models && \
    chown -R katago:katago /app

# Copy KataGo binary from base image
COPY --from=katago-base /usr/local/bin/katago /usr/local/bin/katago
RUN chmod +x /usr/local/bin/katago

# Copy built application binary
COPY --from=builder /app/katago-mcp /usr/local/bin/katago-mcp
RUN chmod +x /usr/local/bin/katago-mcp

# Copy configuration files
COPY config.production.json /app/config/config.json
COPY docker/katago-artifacts/test-config.cfg /app/config/analysis.cfg

# Switch to non-root user
USER katago
WORKDIR /app

# Set environment variables for production
ENV KATAGO_MCP_CONFIG=/app/config/config.json
ENV KATAGO_BINARY_PATH=/usr/local/bin/katago
ENV KATAGO_CONFIG_PATH=/app/config/analysis.cfg
ENV KATAGO_HTTP_PORT=8080

# Expose HTTP port for health checks
EXPOSE 8080

# Add health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Add metadata labels
LABEL org.opencontainers.image.title="KataGo MCP Server" \
      org.opencontainers.image.description="MCP server for KataGo Go analysis engine" \
      org.opencontainers.image.source="https://github.com/dmmcquay/katago-mcp" \
      org.opencontainers.image.authors="dmmcquay" \
      org.opencontainers.image.licenses="MIT"

# Start the application
CMD ["/usr/local/bin/katago-mcp"]