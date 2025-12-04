# Multi-stage build for production-ready Go application
# Build stage with optimization and security
FROM golang:1.25-alpine AS builder

# Set build arguments
ARG VERSION=dev
ARG BUILD_TIME

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Create appuser for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Cache dependencies separately for better build performance
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build with security and optimization flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -a -installsuffix cgo \
    -o main ./cmd/server/main.go

# Production runtime stage with minimal footprint and security hardening
FROM alpine:3.20 AS production

# Security: Create non-root user with minimal privileges
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup -h /app -s /bin/sh

# Install only required runtime packages
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    curl \
    && rm -rf /var/cache/apk/* \
    && rm -rf /tmp/*

# Set working directory
WORKDIR /app

# Create directories first, then copy binary
RUN mkdir -p /app/logs /app/tmp /app/scripts /app/migrations

# Copy binary from builder stage with proper ownership
COPY --from=builder /app/main /app/main

# Set proper ownership and permissions
RUN chown -R appuser:appgroup /app \
    && chmod +x /app/main \
    && chmod -R 755 /app

# Switch to non-root user
USER appuser

# Expose application port
EXPOSE 8080

# Health check for container orchestration
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Set volumes for writable directories (for read-only root filesystem)
VOLUME ["/app/logs", "/app/tmp"]

# Default command
CMD ["./main"]