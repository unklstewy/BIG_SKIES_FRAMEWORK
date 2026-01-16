.PHONY: help build test test-unit test-integration test-coverage lint fmt clean install-tools
.PHONY: docker-build docker-up docker-down docker-logs plugin-ascom-build plugin-ascom-up plugin-ascom-down plugin-ascom-logs

# Default target
help:
	@echo "BIG SKIES Framework - Makefile Commands"
	@echo "========================================"
	@echo "  make build            - Build all services"
	@echo "  make test             - Run all tests (unit + integration)"
	@echo "  make test-unit        - Run unit tests only"
	@echo "  make test-integration - Run integration tests (requires services)"
	@echo "  make test-coverage    - Run tests with coverage report"
	@echo "  make lint             - Run linters"
	@echo "  make fmt              - Format code"
	@echo "  make clean            - Clean build artifacts"
	@echo "  make install-tools    - Install development tools"
	@echo "  make docker-build     - Build Docker images"
	@echo "  make docker-up        - Start services with docker-compose"
	@echo "  make docker-down      - Stop services"
	@echo "  make docker-logs      - View service logs"
	@echo ""
	@echo "Plugin Commands:"
	@echo "  make plugin-ascom-build - Build ASCOM Alpaca Simulator plugin"
	@echo "  make plugin-ascom-up    - Start ASCOM plugin"
	@echo "  make plugin-ascom-down  - Stop ASCOM plugin"
	@echo "  make plugin-ascom-logs  - View ASCOM plugin logs"

# Build all services
build:
	@echo "Building all services..."
	@go build -v -o bin/ ./cmd/...

# Run all tests (unit + integration)
test:
	@echo "Running all tests..."
	@go test -v ./...

# Run unit tests only (skip integration tests)
test-unit:
	@echo "Running unit tests..."
	@go test -v -short ./...

# Run integration tests only (requires services to be running)
test-integration:
	@echo "Running integration tests..."
	@echo "Checking if services are running..."
	@docker ps --filter "name=bigskies-mqtt" --format "{{.Names}}" | grep -q bigskies-mqtt || \
		(echo "ERROR: Services not running. Start with 'make docker-up' first." && exit 1)
	@echo "Services detected, running integration tests..."
	@go test -v ./test/integration/... -count=1

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

# Plugin-specific targets
plugin-ascom-build:
	@echo "Building ASCOM Alpaca Simulator plugin..."
	@docker-compose -f deployments/docker-compose/docker-compose.yml build ascom-alpaca-simulator

plugin-ascom-up:
	@echo "Starting ASCOM Alpaca Simulator plugin..."
	@docker-compose -f deployments/docker-compose/docker-compose.yml up -d ascom-alpaca-simulator
	@echo "Plugin started. Access at http://localhost:32323"

plugin-ascom-down:
	@echo "Stopping ASCOM Alpaca Simulator plugin..."
	@docker-compose -f deployments/docker-compose/docker-compose.yml stop ascom-alpaca-simulator

plugin-ascom-logs:
	@docker-compose -f deployments/docker-compose/docker-compose.yml logs -f ascom-alpaca-simulator
