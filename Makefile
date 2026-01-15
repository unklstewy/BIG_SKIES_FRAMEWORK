.PHONY: help build test lint fmt clean install-tools

# Default target
help:
	@echo "BIG SKIES Framework - Makefile Commands"
	@echo "========================================"
	@echo "  make build         - Build all services"
	@echo "  make test          - Run all tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make lint          - Run linters"
	@echo "  make fmt           - Format code"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make install-tools - Install development tools"
	@echo "  make docker-build  - Build Docker images"
	@echo "  make docker-up     - Start services with docker-compose"
	@echo "  make docker-down   - Stop services"

# Build all services
build:
	@echo "Building all services..."
	@go build -v -o bin/ ./cmd/...

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linters
lint:
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run 'make install-tools' first."; \
		exit 1; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@go clean

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@./scripts/install-tools.sh

# Docker targets
docker-build:
	@echo "Building Docker images..."
	@docker-compose -f deployments/docker-compose/docker-compose.yml build

docker-up:
	@echo "Starting services..."
	@docker-compose -f deployments/docker-compose/docker-compose.yml up -d

docker-down:
	@echo "Stopping services..."
	@docker-compose -f deployments/docker-compose/docker-compose.yml down

docker-logs:
	@docker-compose -f deployments/docker-compose/docker-compose.yml logs -f
