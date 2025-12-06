# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy source code first
COPY . .

# Download dependencies and generate go.sum
RUN go mod tidy

# Build the binary
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o sungrow-monitor ./cmd/sungrow-monitor

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

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
