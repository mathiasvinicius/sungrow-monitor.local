# Build stage
FROM golang:1.22-bookworm AS builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy source code first
COPY . .

# Download dependencies and generate go.sum
RUN go mod tidy

# Build the binary
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o sungrow-monitor ./cmd/sungrow-monitor

# Runtime stage
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -m -s /bin/bash appuser

# Create directories
RUN mkdir -p /data /etc/sungrow-monitor /app/web && chown -R appuser:appuser /data /app

# Copy binary from builder
COPY --from=builder /app/sungrow-monitor /usr/local/bin/sungrow-monitor

# Copy default config
COPY config.yaml /etc/sungrow-monitor/config.yaml

# Copy web assets
COPY web/ /app/web/

# Switch to non-root user
USER appuser

WORKDIR /app

EXPOSE 8080

ENTRYPOINT ["sungrow-monitor"]
CMD ["serve", "--config", "/etc/sungrow-monitor/config.yaml"]
