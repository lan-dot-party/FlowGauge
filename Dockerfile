# FlowGauge Dockerfile
# Multi-stage build for minimal image size

# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w \
    -X github.com/lan-dot-party/flowgauge/pkg/version.Version=${VERSION} \
    -X github.com/lan-dot-party/flowgauge/pkg/version.Commit=${COMMIT} \
    -X github.com/lan-dot-party/flowgauge/pkg/version.BuildDate=${BUILD_DATE}" \
    -o flowgauge ./cmd/flowgauge

# Runtime stage
FROM alpine:3.23

# Labels
LABEL org.opencontainers.image.title="FlowGauge"
LABEL org.opencontainers.image.description="Bandwidth monitoring with Multi-WAN support"
LABEL org.opencontainers.image.source="https://github.com/lan-dot-party/flowgauge"
LABEL org.opencontainers.image.licenses="MIT"

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -S flowgauge && adduser -S flowgauge -G flowgauge

# Create directories
RUN mkdir -p /etc/flowgauge /var/lib/flowgauge && \
    chown -R flowgauge:flowgauge /var/lib/flowgauge

# Copy binary from builder
COPY --from=builder /build/flowgauge /usr/bin/flowgauge

# Copy example config
COPY configs/flowgauge.example.yaml /etc/flowgauge/config.yaml.example

# Set working directory
WORKDIR /var/lib/flowgauge

# Switch to non-root user
USER flowgauge

# Expose API port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
ENTRYPOINT ["flowgauge"]
CMD ["server"]


