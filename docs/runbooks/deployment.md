# Deployment Guide

This guide covers deploying the KataGo MCP server in various environments.

## Prerequisites

### System Requirements
- **OS**: Linux (Ubuntu 20.04+ recommended), macOS, or Windows
- **CPU**: Multi-core CPU (4+ cores recommended for production)
- **RAM**: 8GB+ (depends on KataGo model size and cache configuration)
- **Storage**: 2GB+ for binaries and models
- **Network**: Outbound internet access for model downloads (initial setup)

### Dependencies
- **KataGo**: v1.13.0 or later
- **Neural Network Model**: Compatible with your KataGo version
- **Go**: 1.21+ (for building from source)

## Installation Methods

### Option 1: Pre-built Binaries (Recommended)

1. **Download the latest release**:
   ```bash
   # Replace VERSION with the latest release
   VERSION="v1.0.0"
   ARCH="linux-amd64"  # or darwin-amd64, windows-amd64
   
   curl -L -o katago-mcp \
     "https://github.com/dmmcquay/katago-mcp/releases/download/${VERSION}/katago-mcp-${ARCH}"
   
   chmod +x katago-mcp
   sudo mv katago-mcp /usr/local/bin/
   ```

2. **Verify installation**:
   ```bash
   katago-mcp --version
   ```

### Option 2: Docker Deployment

1. **Pull the image**:
   ```bash
   docker pull ghcr.io/dmmcquay/katago-mcp:latest
   ```

2. **Run with Docker**:
   ```bash
   docker run -d \
     --name katago-mcp \
     -p 8080:8080 \
     -v /path/to/config:/app/config:ro \
     -e KATAGO_CONFIG_PATH=/app/config/config.json \
     ghcr.io/dmmcquay/katago-mcp:latest
   ```

### Option 3: Build from Source

1. **Clone and build**:
   ```bash
   git clone https://github.com/dmmcquay/katago-mcp.git
   cd katago-mcp
   ./build.sh
   sudo cp katago-mcp /usr/local/bin/
   ```

## KataGo Setup

### 1. Install KataGo

**Ubuntu/Debian**:
```bash
# Using apt (if available)
sudo apt update && sudo apt install katago

# Or download from releases
wget https://github.com/lightvector/KataGo/releases/download/v1.14.1/katago-v1.14.1-linux-x64.tar.gz
tar -xzf katago-v1.14.1-linux-x64.tar.gz
sudo cp katago /usr/local/bin/
```

**macOS**:
```bash
# Using Homebrew
brew install katago

# Or download from releases
curl -L -o katago.tar.gz \
  https://github.com/lightvector/KataGo/releases/download/v1.14.1/katago-v1.14.1-macos-x64.tar.gz
tar -xzf katago.tar.gz
sudo cp katago /usr/local/bin/
```

### 2. Download Neural Network Model

```bash
# Create KataGo directory
sudo mkdir -p /opt/katago/models
cd /opt/katago/models

# Download a suitable model (choose based on your hardware)
# Small model (faster, weaker):
sudo wget https://media.katagoarchive.org/g170e-b6c96-s175395328-d26788732.bin.gz

# Medium model (balanced):
sudo wget https://media.katagoarchive.org/g170-b30c320x2-s4824661760-d1229536699.bin.gz

# Large model (slower, stronger):
sudo wget https://media.katagoarchive.org/g170-b40c256x2-s5095420928-d1229425124.bin.gz

# Set permissions
sudo chown -R katago:katago /opt/katago/
```

### 3. Generate KataGo Configuration

```bash
# Create analysis configuration
sudo mkdir -p /opt/katago/config
cd /opt/katago/config

# Generate config for your model
sudo katago genconfig \
  -model /opt/katago/models/g170-b30c320x2-s4824661760-d1229536699.bin.gz \
  -output analysis.cfg

# Optimize for your hardware
sudo katago benchmark -config analysis.cfg -model /opt/katago/models/g170-b30c320x2-s4824661760-d1229536699.bin.gz
```

## Service Configuration

### 1. Create Configuration File

```bash
sudo mkdir -p /etc/katago-mcp
sudo tee /etc/katago-mcp/config.json << 'EOF'
{
  "server": {
    "name": "katago-mcp-production",
    "version": "1.0.0"
  },
  "katago": {
    "binaryPath": "/usr/local/bin/katago",
    "modelPath": "/opt/katago/models/g170-b30c320x2-s4824661760-d1229536699.bin.gz",
    "configPath": "/opt/katago/config/analysis.cfg",
    "numThreads": 4,
    "maxVisits": 1000,
    "maxTime": 10.0
  },
  "cache": {
    "enabled": true,
    "maxItems": 1000,
    "maxSizeBytes": 104857600,
    "ttlSeconds": 3600
  },
  "logging": {
    "level": "info",
    "format": "json"
  },
  "metrics": {
    "enabled": true,
    "address": ":9090",
    "path": "/metrics"
  },
  "health": {
    "enabled": true,
    "address": ":8080",
    "readyPath": "/ready",
    "healthPath": "/health"
  }
}
EOF
```

### 2. Create System User

```bash
# Create dedicated user
sudo useradd --system --no-create-home --shell /bin/false katago-mcp
sudo usermod -a -G katago katago-mcp

# Set permissions
sudo chown -R katago-mcp:katago-mcp /etc/katago-mcp/
sudo chmod 640 /etc/katago-mcp/config.json
```

### 3. Create systemd Service

```bash
sudo tee /etc/systemd/system/katago-mcp.service << 'EOF'
[Unit]
Description=KataGo MCP Server
Documentation=https://github.com/dmmcquay/katago-mcp
After=network.target
Wants=network.target

[Service]
Type=exec
User=katago-mcp
Group=katago-mcp
ExecStart=/usr/local/bin/katago-mcp
Environment=KATAGO_MCP_CONFIG=/etc/katago-mcp/config.json
Environment=KATAGO_MCP_LOG_LEVEL=info
Environment=KATAGO_MCP_LOG_FORMAT=json
Restart=always
RestartSec=5
StartLimitInterval=0

# Security settings
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
PrivateDevices=true
ProtectHostname=true
ProtectClock=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectKernelLogs=true
ProtectControlGroups=true
RestrictRealtime=true
RestrictSUIDSGID=true
RemoveIPC=true
PrivateMounts=true

# Resource limits
LimitNOFILE=65536
MemoryMax=8G
TasksMax=4096

# Working directory
WorkingDirectory=/opt/katago-mcp
ReadWritePaths=/opt/katago-mcp /tmp

[Install]
WantedBy=multi-user.target
EOF
```

## Deployment Steps

### 1. Pre-deployment Validation

```bash
# Test KataGo installation
katago version

# Test model loading
katago benchmark -config /opt/katago/config/analysis.cfg \
  -model /opt/katago/models/g170-b30c320x2-s4824661760-d1229536699.bin.gz \
  -numthreads 1 -visits 10

# Validate configuration
katago-mcp --config /etc/katago-mcp/config.json --validate
```

### 2. Deploy Service

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable service
sudo systemctl enable katago-mcp

# Start service
sudo systemctl start katago-mcp

# Check status
sudo systemctl status katago-mcp
```

### 3. Post-deployment Verification

```bash
# Check service logs
sudo journalctl -u katago-mcp -f

# Test health endpoints
curl http://localhost:8080/health
curl http://localhost:8080/ready

# Test metrics endpoint
curl http://localhost:9090/metrics

# Test MCP functionality (if you have an MCP client)
# This would depend on your specific MCP client setup
```

### 4. Configure Log Rotation

```bash
sudo tee /etc/logrotate.d/katago-mcp << 'EOF'
/var/log/katago-mcp/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 katago-mcp katago-mcp
    postrotate
        systemctl reload katago-mcp
    endscript
}
EOF
```

## Environment-Specific Configurations

### Development Environment
- Lower resource limits
- Debug logging enabled
- Smaller cache sizes
- Faster KataGo settings (fewer visits)

### Staging Environment
- Production-like configuration
- Extended logging for testing
- Health check intervals for validation

### Production Environment
- Optimized KataGo settings
- Resource monitoring enabled
- Backup and recovery procedures
- Security hardening applied

## Rollback Procedures

### Service Rollback
```bash
# Stop current service
sudo systemctl stop katago-mcp

# Replace binary with previous version
sudo cp /backup/katago-mcp-previous /usr/local/bin/katago-mcp

# Start service
sudo systemctl start katago-mcp
```

### Configuration Rollback
```bash
# Restore previous configuration
sudo cp /backup/config.json.backup /etc/katago-mcp/config.json

# Restart service
sudo systemctl restart katago-mcp
```

## Next Steps

After successful deployment:
1. Set up [monitoring and alerting](monitoring.md)
2. Configure [log analysis](log-analysis.md)
3. Review [security hardening](security.md)
4. Plan [backup and recovery](backup-recovery.md)