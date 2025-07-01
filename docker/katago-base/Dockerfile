# Base image with KataGo pre-built
FROM golang:1.23-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    git \
    libeigen3-dev \
    libzip-dev \
    zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

# Clone and build KataGo with Eigen backend
ARG KATAGO_VERSION=v1.14.1
RUN git clone --branch ${KATAGO_VERSION} --depth 1 https://github.com/lightvector/KataGo.git /katago
WORKDIR /katago/cpp
RUN cmake . -DUSE_BACKEND=EIGEN -DCMAKE_BUILD_TYPE=Release
RUN make -j$(nproc)

# Final stage - minimal runtime image
FROM debian:bookworm-slim

# Install runtime dependencies only
RUN apt-get update && apt-get install -y \
    libzip4 \
    libboost-filesystem-dev \
    libgoogle-perftools4 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy KataGo binary
COPY --from=builder /katago/cpp/katago /usr/local/bin/katago
RUN chmod +x /usr/local/bin/katago

# Verify it works
RUN /usr/local/bin/katago version

LABEL org.opencontainers.image.description="KataGo v1.14.1 with Eigen backend for CPU-only analysis"
LABEL org.opencontainers.image.source="https://github.com/dmmcquay/katago-mcp"