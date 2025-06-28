# Multi-stage Dockerfile for PCF-MCP Server
# This Dockerfile creates a minimal, secure container image

# Build stage
FROM golang:1.23-alpine AS builder

# Install ca-certificates for HTTPS support
RUN apk add --no-cache ca-certificates git

# Create non-root user
RUN adduser -D -u 10001 pcfmcp

WORKDIR /build

# Copy go mod files for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with static linking
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o pcf-mcp cmd/pcf-mcp/main.go

# Final stage - scratch image
FROM scratch

# Copy binary from builder
COPY --from=builder /build/pcf-mcp /pcf-mcp

# Copy ca-certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy passwd file for non-root user
COPY --from=builder /etc/passwd /etc/passwd

# Use non-root user
USER pcfmcp

# Expose MCP server port
EXPOSE 8080

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/pcf-mcp", "health"]

# Run the binary
ENTRYPOINT ["/pcf-mcp"]