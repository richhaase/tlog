# tlog development tasks

.PHONY: build install test test-coverage fmt vet lint staticcheck tidy clean check help

# Default target
help:
	@echo "Available targets:"
	@echo "  build          Build the tlog binary with version information"
	@echo "  install        Install tlog to GOBIN with version information"
	@echo "  test           Run all unit tests"
	@echo "  test-coverage  Run tests with coverage report"
	@echo "  fmt            Format Go source code"
	@echo "  vet            Run go vet"
	@echo "  lint           Run golangci-lint v2"
	@echo "  staticcheck    Run staticcheck"
	@echo "  tidy           Tidy go modules"
	@echo "  clean          Clean build artifacts and test cache"
	@echo "  check          Run all quality checks (fmt, lint, vet, staticcheck, test)"

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Build the tlog binary with version information
build:
	@echo "Building tlog with version information..."
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/tlog ./cmd/tlog
	@echo "Built versioned tlog binary to bin/ (version: $(VERSION))"

# Install tlog to GOBIN with version information
install:
	go install -ldflags "$(LDFLAGS)" ./cmd/tlog

# Run all unit tests
test:
	@echo "Running unit tests..."
	go test ./...
	@echo "Unit tests passed!"

# Run tests with coverage
test-coverage:
	@echo "Running unit tests with coverage..."
	go clean -testcache
	go test -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out > coverage.txt
	@awk 'END{printf "Total coverage: %s\n", $$3}' coverage.txt
	go tool cover -html=coverage.out -o coverage.html
	@echo "Unit tests passed! Coverage report: coverage.html (see also coverage.txt)"

# Format Go source code
fmt:
	@echo "Formatting Go source code..."
	go fmt ./...
	@echo "Formatting complete!"

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "Vet passed!"

# Run golangci-lint v2
lint:
	@echo "Running golangci-lint v2..."
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0 run --timeout=10m ./...
	@echo "Linting passed!"

# Run staticcheck
staticcheck:
	@echo "Running staticcheck..."
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...
	@echo "Staticcheck passed!"

# Tidy go modules
tidy:
	@echo "Tidying go modules..."
	go mod tidy
	@echo "Modules tidied!"

# Clean build artifacts and test cache
clean:
	@echo "Cleaning build artifacts and caches..."
	rm -rf bin
	rm -f coverage.out coverage.html coverage.txt
	go clean
	go clean -testcache
	@echo "Build artifacts and test cache cleaned"

# Run all quality checks
check: fmt lint vet staticcheck test
