# justfile for PCF-MCP Server project
# This file defines common development tasks and automation

# Default task - show available commands
default:
    @just --list

# Run all tests with coverage
test:
    go test -v -race -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# Run tests without coverage
test-quick:
    go test -v ./...

# Build the binary
build:
    CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/pcf-mcp cmd/pcf-mcp/main.go

# Run golangci-lint
lint:
    golangci-lint run

# Format code
fmt:
    go fmt ./...
    gofumpt -w .

# Run the application (stdio mode)
run:
    go run cmd/pcf-mcp/main.go

# Run the server in HTTP mode
run-http:
    go run cmd/pcf-mcp/main.go --server-transport http --server-port 8080

# Run the server in HTTP mode with authentication
run-http-auth:
    #!/usr/bin/env bash
    TOKEN=$(openssl rand -base64 32)
    echo "Authentication token: Bearer $TOKEN"
    echo ""
    go run cmd/pcf-mcp/main.go --server-transport http --server-port 8080 \
        --server-auth-required true --server-auth-token "$TOKEN"

# Build Docker image
docker:
    docker build -t pcf-mcp:latest .

# Clean build artifacts
clean:
    rm -rf bin/
    rm -f coverage.out coverage.html
    go clean -cache

# Install development dependencies
deps:
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    go install mvdan.cc/gofumpt@latest
    go mod download
    go mod tidy

# Create a new git tag
tag version:
    git tag -a v{{version}} -m "Release version {{version}}"
    git push --tags

# Run security scan
security:
    ./scripts/security-scan.sh
    
# Generate documentation
docs:
    go doc -all > docs/godoc.txt

# Run integration tests
test-integration:
    INTEGRATION_TESTS=true go test -v -tags=integration ./tests/...

# Run stress tests
test-stress:
    STRESS_TEST=true go test -v -tags=integration -run TestStressTest ./tests/...

# Run end-to-end tests
test-e2e:
    INTEGRATION_TESTS=true go test -v -tags=integration -run TestEndToEnd ./tests/...

# Check for dependency updates
check-updates:
    go list -u -m all

# Vendor dependencies
vendor:
    go mod vendor

# Run benchmarks
bench:
    go test -bench=. -benchmem ./...

# Show test coverage in terminal
cover:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

# Validate go.mod and go.sum
mod-verify:
    go mod verify

# Download dependencies
mod-download:
    go mod download

# Update dependencies
mod-update:
    go get -u ./...
    go mod tidy