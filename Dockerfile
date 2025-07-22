# Multi-stage Docker build for O-RAN Near-RT RIC
# Based on security best practices and minimal attack surface

# Stage 1: Build stage with full development environment
FROM golang:1.21-alpine3.18 AS builder

# Install build dependencies including SCTP support for O-RAN E2 interface
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    gcc \
    musl-dev \
    linux-headers \
    lksctp-tools-dev \
    lksctp-tools \
    && update-ca-certificates

# Create non-root user for build
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with security flags
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o near-rt-ric \
    ./cmd/ric/main.go

# Build xApp manager
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o xapp-manager \
    ./cmd/xapp-manager/main.go

# Stage 2: Runtime stage with minimal base image including SCTP support
FROM alpine:3.18 AS runtime

# Install runtime SCTP libraries for O-RAN E2 interface
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    lksctp-tools \
    wget \
    && update-ca-certificates

# Create non-root user
RUN adduser -D -g '' appuser

# Import timezone data and CA certificates
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd

# Copy the binary from builder
COPY --from=builder /build/near-rt-ric /near-rt-ric
COPY --from=builder /build/xapp-manager /xapp-manager

# Copy configuration files
COPY --from=builder /build/configs /configs

# Use non-root user
USER appuser

# Expose ports
EXPOSE 8080 8443 830 2152

# Health check - Use wget instead of curl for minimal image
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
CMD ["/near-rt-ric"]

# Stage 3: Development image with debugging tools
FROM golang:1.21-alpine3.18 AS development

# Install development tools
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    gcc \
    musl-dev \
    linux-headers \
    curl \
    netcat-openbsd \
    tcpdump \
    strace \
    && update-ca-certificates

# Install Go debugging tools
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Create non-root user
RUN adduser -D -g '' developer

# Set working directory
WORKDIR /app

# Copy source code
COPY . .

# Download dependencies
RUN go mod download

# Build with debug symbols
RUN CGO_ENABLED=1 go build -gcflags="all=-N -l" -o near-rt-ric ./cmd/ric/main.go
RUN CGO_ENABLED=1 go build -gcflags="all=-N -l" -o xapp-manager ./cmd/xapp-manager/main.go

# Use non-root user
USER developer

# Expose debug port
EXPOSE 40000

# Default command for development
CMD ["./near-rt-ric"]