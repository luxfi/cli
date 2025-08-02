# Makefile for Lux CLI

# Variables
GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(GOPATH)/bin
PROJECT_NAME = lux
BINARY_NAME = lux
VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_DATE = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = -X 'github.com/luxfi/cli/cmd.Version=$(VERSION)' -extldflags '-Wl,-allow_commons'

# Default target
.PHONY: all
all: build

# Build the CLI binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) main.go
	@echo "Build complete: ./bin/$(BINARY_NAME)"

# Install the binary to GOBIN
.PHONY: install
install:
	@echo "Installing $(BINARY_NAME) to $(GOBIN)..."
	go install -ldflags "$(LDFLAGS)" .
	@echo "Installed to: $(GOBIN)/$(BINARY_NAME)"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint the code
.PHONY: lint
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run go mod tidy
.PHONY: tidy
tidy:
	@echo "Running go mod tidy..."
	go mod tidy

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean -cache
	@echo "Clean complete"

# Build for multiple platforms
.PHONY: build-all
build-all: build-linux build-darwin build-windows

.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-linux-arm64 main.go

.PHONY: build-darwin
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-darwin-arm64 main.go

.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-windows-amd64.exe main.go

# Development build (with race detector)
.PHONY: dev
dev:
	@echo "Building with race detector..."
	@mkdir -p bin
	go build -race -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) main.go

# Check for vulnerabilities
.PHONY: vuln-check
vuln-check:
	@echo "Checking for vulnerabilities..."
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# Generate documentation
.PHONY: docs
docs:
	@echo "Generating documentation..."
	go doc -all > docs.txt

# Show help
.PHONY: help
help:
	@echo "Lux CLI Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all           - Build the binary (default)"
	@echo "  build         - Build the binary"
	@echo "  install       - Install the binary to GOBIN"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  lint          - Run linters"
	@echo "  fmt           - Format code"
	@echo "  tidy          - Run go mod tidy"
	@echo "  clean         - Clean build artifacts"
	@echo "  build-all     - Build for all platforms"
	@echo "  dev           - Build with race detector"
	@echo "  vuln-check    - Check for vulnerabilities"
	@echo "  docs          - Generate documentation"
	@echo "  help          - Show this help message"

# Ensure dependencies are downloaded
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	go mod download