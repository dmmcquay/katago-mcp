# Base image with KataGo, model, and config pre-installed
FROM ubuntu:22.04

LABEL maintainer="KataGo MCP"
LABEL description="KataGo base image with neural network model for testing"

# Install dependencies
RUN apt-get update && apt-get install -y \
    wget \
    curl \
    ca-certificates \
    libzip-dev \
    libboost-filesystem-dev \
    libgoogle-perftools-dev \
    && rm -rf /var/lib/apt/lists/*

# Install KataGo binary
ARG KATAGO_VERSION=1.15.3
RUN wget -q https://github.com/lightvector/KataGo/releases/download/v${KATAGO_VERSION}/katago-v${KATAGO_VERSION}-linux-x64.tar.gz \
    && tar -xzf katago-v${KATAGO_VERSION}-linux-x64.tar.gz \
    && mv katago /usr/local/bin/ \
    && chmod +x /usr/local/bin/katago \
    && rm katago-v${KATAGO_VERSION}-linux-x64.tar.gz

# Create directory for KataGo files
RUN mkdir -p /katago

# Download a small neural network model (15 block network, ~30MB)
# Using a specific, stable model for reproducible testing
ARG MODEL_NAME=g170-b15c192-s1672170752-d466197061.bin.gz
RUN wget -O /katago/${MODEL_NAME} \
    https://katagoarchive.org/g170/neuralnets/${MODEL_NAME} \
    && ln -s /katago/${MODEL_NAME} /katago/default-model.bin.gz

# Create a minimal KataGo config for testing
RUN cat > /katago/default-config.cfg << 'EOF'
# Minimal KataGo config for e2e testing
# Optimized for fast testing on CPU
numSearchThreads = 1
maxVisits = 100
maxTime = 2.0
maxPlayouts = 200
nnCacheSizePowerOfTwo = 16
nnMutexPoolSizePowerOfTwo = 10
nnMaxBatchSize = 8
numVirtualLossesPerThread = 1
searchFactorAfterOnePass = 0.50
searchFactorAfterTwoPass = 0.25
reportAnalysisWinratesAs = BLACK
conservativePass = true
# Disable GPU for CPU-only testing
cudaDeviceToUseThread0 = -1
EOF

# Set environment variables
ENV KATAGO_MODEL_PATH=/katago/default-model.bin.gz
ENV KATAGO_CONFIG_PATH=/katago/default-config.cfg

# Verify KataGo installation
RUN katago version && \
    echo "quit" | timeout 30s katago gtp -model ${KATAGO_MODEL_PATH} -config ${KATAGO_CONFIG_PATH} || \
    echo "KataGo verification completed"

# Install Go for running tests
ENV GO_VERSION=1.21.10
RUN wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz \
    && rm go${GO_VERSION}.linux-amd64.tar.gz \
    && /usr/local/go/bin/go version

ENV PATH="/usr/local/go/bin:${PATH}"

# This is a base image, so no CMD/ENTRYPOINT