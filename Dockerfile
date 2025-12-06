# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o sungrow-monitor ./cmd/sungrow-monitor

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

# Create data directory
RUN mkdir -p /data /etc/sungrow-monitor && chown appuser:appuser /data

# Copy binary from builder
COPY --from=builder /app/sungrow-monitor /usr/local/bin/sungrow-monitor

# Copy default config
COPY config.yaml /etc/sungrow-monitor/config.yaml

# Switch to non-root user
USER appuser

WORKDIR /data

EXPOSE 8080

ENTRYPOINT ["sungrow-monitor"]
CMD ["serve", "--config", "/etc/sungrow-monitor/config.yaml"]
