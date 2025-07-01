#!/bin/bash
set -e

echo "🔍 Validating Docker image setup..."

# Check if we're in CI or local
if [ -n "$CI" ]; then
    echo "Running in CI environment"
else
    echo "Running locally (some tests may fail on ARM64)"
fi

# Run validation checks
docker run --rm --platform linux/amd64 katago-mcp-e2e bash -c '
    set -e
    
    echo "=== Checking environment ==="
    echo "Architecture: $(uname -m)"
    echo "Go version: $(go version)"
    
    echo ""
    echo "=== Checking KataGo files ==="
    if [ -f /usr/local/bin/katago ]; then
        echo "✅ KataGo binary exists"
        ls -lh /usr/local/bin/katago
    else
        echo "❌ KataGo binary missing"
        exit 1
    fi
    
    if [ -f /katago/model.bin.gz ]; then
        echo "✅ Model file exists"
        ls -lh /katago/model.bin.gz
    else
        echo "❌ Model file missing"
        exit 1
    fi
    
    if [ -f /katago/config.cfg ]; then
        echo "✅ Config file exists"
        echo "Config contents:"
        head -5 /katago/config.cfg
    else
        echo "❌ Config file missing"
        exit 1
    fi
    
    echo ""
    echo "=== Checking Go project ==="
    if [ -f /app/go.mod ]; then
        echo "✅ Go module exists"
        head -3 /app/go.mod
    else
        echo "❌ Go module missing"
        exit 1
    fi
    
    echo ""
    echo "=== Checking e2e test files ==="
    if [ -d /app/e2e ]; then
        echo "✅ E2E test directory exists"
        ls /app/e2e/*.go | head -5
    else
        echo "❌ E2E test directory missing"
        exit 1
    fi
    
    echo ""
    echo "✅ All Docker image components are in place!"
'

echo ""
echo "🎉 Docker image validation complete!"
echo ""
echo "Note: Full e2e tests may fail on ARM64 Macs due to emulation issues."
echo "The tests will run correctly in CI (linux/amd64)."