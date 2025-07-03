# Configuration Management

This guide covers configuration management for the KataGo MCP server across different environments.

## Configuration Overview

The KataGo MCP server uses a hierarchical configuration system:

1. **Environment variables** (highest priority)
2. **JSON configuration file** (specified by `KATAGO_MCP_CONFIG`)
3. **Default values** (lowest priority)

## Configuration Structure

### Complete Configuration Example

```json
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
  },
  "rateLimit": {
    "enabled": true,
    "requestsPerSecond": 10,
    "burstSize": 20
  }
}
```

## Environment Variables

### Core Settings

```bash
# Configuration file location
export KATAGO_MCP_CONFIG="/etc/katago-mcp/config.json"

# Logging configuration
export KATAGO_MCP_LOG_LEVEL="info"           # debug, info, warn, error
export KATAGO_MCP_LOG_FORMAT="json"          # json, text

# KataGo binary and model paths
export KATAGO_BINARY_PATH="/usr/local/bin/katago"
export KATAGO_MODEL_PATH="/opt/katago/models/model.bin.gz"
export KATAGO_CONFIG_PATH="/opt/katago/config/analysis.cfg"

# Resource limits
export KATAGO_NUM_THREADS="4"
export KATAGO_MAX_VISITS="1000"
export KATAGO_MAX_TIME="10.0"

# Cache settings
export KATAGO_CACHE_ENABLED="true"
export KATAGO_CACHE_MAX_ITEMS="1000"
export KATAGO_CACHE_MAX_SIZE_BYTES="104857600"
export KATAGO_CACHE_TTL_SECONDS="3600"

# Service endpoints
export KATAGO_HEALTH_ADDRESS=":8080"
export KATAGO_METRICS_ADDRESS=":9090"

# Rate limiting
export KATAGO_RATE_LIMIT_ENABLED="true"
export KATAGO_RATE_LIMIT_RPS="10"
export KATAGO_RATE_LIMIT_BURST="20"
```

### Security Settings

```bash
# TLS configuration (if enabled)
export KATAGO_TLS_CERT_FILE="/etc/ssl/certs/katago-mcp.crt"
export KATAGO_TLS_KEY_FILE="/etc/ssl/private/katago-mcp.key"

# Authentication (if enabled)
export KATAGO_AUTH_ENABLED="true"
export KATAGO_AUTH_TOKEN="your-secret-token"

# CORS settings
export KATAGO_CORS_ENABLED="true"
export KATAGO_CORS_ORIGINS="https://yourdomain.com"
```

## Environment-Specific Configurations

### Development Environment

```json
{
  "server": {
    "name": "katago-mcp-dev",
    "version": "dev"
  },
  "katago": {
    "binaryPath": "/usr/local/bin/katago",
    "modelPath": "/opt/katago/models/small-model.bin.gz",
    "configPath": "/opt/katago/config/dev-analysis.cfg",
    "numThreads": 2,
    "maxVisits": 200,
    "maxTime": 5.0
  },
  "cache": {
    "enabled": true,
    "maxItems": 100,
    "maxSizeBytes": 10485760,
    "ttlSeconds": 600
  },
  "logging": {
    "level": "debug",
    "format": "text"
  },
  "metrics": {
    "enabled": true,
    "address": ":9090"
  },
  "health": {
    "enabled": true,
    "address": ":8080"
  }
}
```

### Staging Environment

```json
{
  "server": {
    "name": "katago-mcp-staging",
    "version": "staging"
  },
  "katago": {
    "binaryPath": "/usr/local/bin/katago",
    "modelPath": "/opt/katago/models/medium-model.bin.gz",
    "configPath": "/opt/katago/config/staging-analysis.cfg",
    "numThreads": 3,
    "maxVisits": 600,
    "maxTime": 8.0
  },
  "cache": {
    "enabled": true,
    "maxItems": 500,
    "maxSizeBytes": 52428800,
    "ttlSeconds": 1800
  },
  "logging": {
    "level": "info",
    "format": "json"
  },
  "metrics": {
    "enabled": true,
    "address": ":9090"
  },
  "health": {
    "enabled": true,
    "address": ":8080"
  },
  "rateLimit": {
    "enabled": true,
    "requestsPerSecond": 5,
    "burstSize": 10
  }
}
```

### Production Environment

```json
{
  "server": {
    "name": "katago-mcp-production",
    "version": "1.0.0"
  },
  "katago": {
    "binaryPath": "/usr/local/bin/katago",
    "modelPath": "/opt/katago/models/production-model.bin.gz",
    "configPath": "/opt/katago/config/production-analysis.cfg",
    "numThreads": 6,
    "maxVisits": 1200,
    "maxTime": 12.0
  },
  "cache": {
    "enabled": true,
    "maxItems": 2000,
    "maxSizeBytes": 209715200,
    "ttlSeconds": 7200
  },
  "logging": {
    "level": "warn",
    "format": "json"
  },
  "metrics": {
    "enabled": true,
    "address": ":9090"
  },
  "health": {
    "enabled": true,
    "address": ":8080"
  },
  "rateLimit": {
    "enabled": true,
    "requestsPerSecond": 20,
    "burstSize": 40
  }
}
```

## KataGo Configuration

### Analysis Configuration Template

Create environment-specific KataGo configurations:

```bash
# Generate base configuration
katago genconfig -model /opt/katago/models/your-model.bin.gz -output base-analysis.cfg

# Customize for environment
cp base-analysis.cfg dev-analysis.cfg
```

### Development Settings (dev-analysis.cfg)

```ini
# Lower resource usage for development
numSearchThreads = 1
maxPlayouts = 100
maxTime = 3.0
maxVisits = 200

# Smaller neural network cache
nnCacheSizePowerOfTwo = 16
nnMaxBatchSize = 4

# Faster analysis
conservativePass = false
wideRootNoise = 0.04
```

### Production Settings (production-analysis.cfg)

```ini
# Optimized for production performance
numSearchThreads = 4
maxPlayouts = 400
maxTime = 10.0
maxVisits = 1000

# Larger neural network cache
nnCacheSizePowerOfTwo = 20
nnMaxBatchSize = 16

# Balanced analysis quality
conservativePass = true
wideRootNoise = 0.02
```

## Configuration Validation

### Validation Script

```bash
#!/bin/bash
# validate_config.sh

CONFIG_FILE="${1:-/etc/katago-mcp/config.json}"

echo "Validating configuration: $CONFIG_FILE"

# Check JSON syntax
if ! jq . "$CONFIG_FILE" > /dev/null 2>&1; then
    echo "ERROR: Invalid JSON syntax"
    exit 1
fi

# Validate required fields
REQUIRED_FIELDS=(
    ".katago.binaryPath"
    ".katago.modelPath"
    ".katago.configPath"
)

for field in "${REQUIRED_FIELDS[@]}"; do
    if ! jq -e "$field" "$CONFIG_FILE" > /dev/null 2>&1; then
        echo "ERROR: Missing required field: $field"
        exit 1
    fi
done

# Check file existence
BINARY_PATH=$(jq -r '.katago.binaryPath' "$CONFIG_FILE")
MODEL_PATH=$(jq -r '.katago.modelPath' "$CONFIG_FILE")
KATAGO_CONFIG_PATH=$(jq -r '.katago.configPath' "$CONFIG_FILE")

if [[ ! -x "$BINARY_PATH" ]]; then
    echo "ERROR: KataGo binary not found or not executable: $BINARY_PATH"
    exit 1
fi

if [[ ! -f "$MODEL_PATH" ]]; then
    echo "ERROR: Model file not found: $MODEL_PATH"
    exit 1
fi

if [[ ! -f "$KATAGO_CONFIG_PATH" ]]; then
    echo "ERROR: KataGo config file not found: $KATAGO_CONFIG_PATH"
    exit 1
fi

# Validate numeric ranges
MAX_VISITS=$(jq -r '.katago.maxVisits' "$CONFIG_FILE")
if [[ "$MAX_VISITS" -lt 1 || "$MAX_VISITS" -gt 10000 ]]; then
    echo "WARNING: maxVisits should be between 1 and 10000"
fi

echo "Configuration validation passed"
```

### Test Configuration

```bash
# Test with actual KataGo
katago-mcp --config /etc/katago-mcp/config.json --validate

# Test KataGo directly
katago analysis \
  -config /opt/katago/config/analysis.cfg \
  -model /opt/katago/models/model.bin.gz \
  < /dev/null
```

## Configuration Templates

### Template Generation Script

```bash
#!/bin/bash
# generate_config.sh

ENVIRONMENT="${1:-production}"
OUTPUT_FILE="${2:-config.json}"

case "$ENVIRONMENT" in
    "dev"|"development")
        THREADS=2
        VISITS=200
        TIME=5.0
        CACHE_ITEMS=100
        CACHE_SIZE=10485760
        CACHE_TTL=600
        LOG_LEVEL="debug"
        ;;
    "staging")
        THREADS=3
        VISITS=600
        TIME=8.0
        CACHE_ITEMS=500
        CACHE_SIZE=52428800
        CACHE_TTL=1800
        LOG_LEVEL="info"
        ;;
    "prod"|"production")
        THREADS=6
        VISITS=1200
        TIME=12.0
        CACHE_ITEMS=2000
        CACHE_SIZE=209715200
        CACHE_TTL=7200
        LOG_LEVEL="warn"
        ;;
    *)
        echo "Unknown environment: $ENVIRONMENT"
        echo "Usage: $0 [dev|staging|prod] [output_file]"
        exit 1
        ;;
esac

cat > "$OUTPUT_FILE" << EOF
{
  "server": {
    "name": "katago-mcp-$ENVIRONMENT",
    "version": "1.0.0"
  },
  "katago": {
    "binaryPath": "/usr/local/bin/katago",
    "modelPath": "/opt/katago/models/model.bin.gz",
    "configPath": "/opt/katago/config/analysis.cfg",
    "numThreads": $THREADS,
    "maxVisits": $VISITS,
    "maxTime": $TIME
  },
  "cache": {
    "enabled": true,
    "maxItems": $CACHE_ITEMS,
    "maxSizeBytes": $CACHE_SIZE,
    "ttlSeconds": $CACHE_TTL
  },
  "logging": {
    "level": "$LOG_LEVEL",
    "format": "json"
  },
  "metrics": {
    "enabled": true,
    "address": ":9090"
  },
  "health": {
    "enabled": true,
    "address": ":8080"
  }
}
EOF

echo "Generated $ENVIRONMENT configuration: $OUTPUT_FILE"
```

## Configuration Management Best Practices

### 1. Version Control

```bash
# Store configurations in git
git add configs/
git commit -m "Update production configuration"

# Use branches for different environments
git checkout -b config/staging
git checkout -b config/production
```

### 2. Configuration Secrets

```bash
# Use environment variables for secrets
export KATAGO_AUTH_TOKEN="$(cat /etc/secrets/auth-token)"
export KATAGO_DB_PASSWORD="$(cat /etc/secrets/db-password)"

# Or use external secret management
vault kv get -field=token secret/katago-mcp/auth
```

### 3. Configuration Deployment

```bash
#!/bin/bash
# deploy_config.sh

ENVIRONMENT="$1"
CONFIG_SOURCE="configs/$ENVIRONMENT.json"
CONFIG_TARGET="/etc/katago-mcp/config.json"

# Validate new configuration
./validate_config.sh "$CONFIG_SOURCE"

# Backup current configuration
cp "$CONFIG_TARGET" "/etc/katago-mcp/config.json.backup.$(date +%s)"

# Deploy new configuration
cp "$CONFIG_SOURCE" "$CONFIG_TARGET"
chown katago-mcp:katago-mcp "$CONFIG_TARGET"
chmod 640 "$CONFIG_TARGET"

# Restart service
systemctl restart katago-mcp

# Verify service started successfully
sleep 5
if systemctl is-active --quiet katago-mcp; then
    echo "Configuration deployed successfully"
else
    echo "Service failed to start, rolling back"
    cp "/etc/katago-mcp/config.json.backup."* "$CONFIG_TARGET"
    systemctl restart katago-mcp
    exit 1
fi
```

## Monitoring Configuration Changes

### Configuration Drift Detection

```bash
#!/bin/bash
# check_config_drift.sh

EXPECTED_CONFIG="/etc/katago-mcp/expected-config.json"
CURRENT_CONFIG="/etc/katago-mcp/config.json"

if ! diff -q "$EXPECTED_CONFIG" "$CURRENT_CONFIG" > /dev/null; then
    echo "ALERT: Configuration drift detected"
    diff "$EXPECTED_CONFIG" "$CURRENT_CONFIG"
    exit 1
fi

echo "Configuration matches expected state"
```

### Configuration Metrics

Track configuration changes in your monitoring system:
- Configuration file modification time
- Configuration validation status
- Service restart events after config changes

## Next Steps

After configuring your service:
1. Set up [monitoring](monitoring.md) for your environment
2. Test with [troubleshooting](troubleshooting.md) procedures
3. Plan [performance tuning](performance-tuning.md) based on your workload