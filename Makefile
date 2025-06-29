.PHONY: all build test lint security bench clean help

# Variables
BINARY_NAME := pcf-mcp
VERSION := $(shell git describe --tags --always --dirty)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE) -w -s"
GOFLAGS := -trimpath

# Default target
all: lint security test build

# Build binary
build:
	@echo "Building $(BINARY_NAME)..."
	CGO_ENABLED=0 go build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/pcf-mcp/main.go

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Security scanning
security:
	@echo "Running security scan..."
	@./scripts/security-scan.sh

# Performance benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem -benchtime=10s ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f gosec.sarif
	rm -f cpu.prof mem.prof
	go clean -cache

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install github.com/sonatype-nexus-community/nancy@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

# Production build with all checks
production: lint security test
	@echo "Building production binary..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		$(GOFLAGS) $(LDFLAGS) \
		-tags netgo,osusergo \
		-o bin/$(BINARY_NAME)-linux-amd64 \
		cmd/pcf-mcp/main.go

# Docker production build
docker-production:
	@echo "Building production Docker image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(BINARY_NAME):$(VERSION) \
		-t $(BINARY_NAME):latest \
		--no-cache \
		.

# Help
help:
	@echo "Available targets:"
	@echo "  all              - Run lint, security, test, and build"
	@echo "  build            - Build the binary"
	@echo "  test             - Run tests with coverage"
	@echo "  lint             - Run golangci-lint"
	@echo "  security         - Run security scans"
	@echo "  bench            - Run benchmarks"
	@echo "  clean            - Clean build artifacts"
	@echo "  install-tools    - Install development tools"
	@echo "  production       - Production build with all checks"
	@echo "  docker-production - Build production Docker image"
	@echo "  help             - Show this help message"