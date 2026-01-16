.PHONY: help build test test-unit test-integration test-coverage lint fmt clean install-tools
.PHONY: docker-build docker-up docker-down docker-logs docker-ps docker-restart docker-purge
.PHONY: db-backup db-restore db-status
.PHONY: plugin-ascom-build plugin-ascom-up plugin-ascom-down plugin-ascom-logs

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
	@echo "  make docker-build     - Build Docker images (ensures .pgpass is copied)"
	@echo "  make docker-up        - Start services with docker-compose"
	@echo "  make docker-down      - Stop services"
	@echo "  make docker-logs      - View service logs (follow mode)"
	@echo "  make docker-ps        - View service status"
	@echo "  make docker-restart   - Restart all services"
	@echo "  make docker-purge     - Purge containers, volumes, and build cache (DESTRUCTIVE)"
	@echo ""
	@echo "Database Commands:"
	@echo "  make db-backup        - Backup database to backups/database/"
	@echo "  make db-restore       - Restore database from backup (interactive)"
	@echo "  make db-status        - Show database status and connection info"
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
	@echo "Ensuring .pgpass is in shared volume..."
	@./scripts/update-pgpass.sh || echo "Warning: Could not update .pgpass (continuing anyway)"
	@docker-compose build

docker-up:
	@echo "Starting services..."
	@echo "Ensuring .pgpass is in shared volume..."
	@./scripts/update-pgpass.sh || echo "Warning: Could not update .pgpass (continuing anyway)"
	@docker-compose up -d
	@echo ""
	@echo "✅ Services started!"
	@echo "   View logs: make docker-logs"
	@echo "   Check status: docker-compose ps"

docker-down:
	@echo "Stopping services..."
	@docker-compose down

docker-logs:
	@docker-compose logs -f

docker-ps:
	@docker-compose ps

docker-restart:
	@echo "Restarting services..."
	@docker-compose restart

docker-purge:
	@echo "⚠️  WARNING: This will remove ALL containers, volumes, and build cache!"
	@echo ""
	@echo "💡 TIP: Backup your database first with 'make db-backup'"
	@echo "   - All coordinator containers"
	@echo "   - PostgreSQL data (database will be lost)"
	@echo "   - MQTT data"
	@echo "   - Shared secrets volume"
	@echo "   - Docker build cache"
	@echo ""
	@read -p "Are you sure? Type 'yes' to continue: " confirm && [ "$$confirm" = "yes" ] || (echo "Aborted." && exit 1)
	@echo ""
	@echo "Stopping and removing containers..."
	@docker-compose down -v
	@echo "Removing BIG SKIES images..."
	@docker images | grep bigskies | awk '{print $$3}' | xargs -r docker rmi -f 2>/dev/null || true
	@echo "Removing shared secrets volume..."
	@docker volume rm bigskies_shared_secrets 2>/dev/null || true
	@echo "Pruning build cache..."
	@docker builder prune -af
	@echo ""
	@echo "✅ Docker environment purged!"
	@echo "   To start fresh: ./scripts/update-pgpass.sh && make docker-build && make docker-up"

# Database management targets
db-backup:
	@./scripts/db-backup.sh

db-restore:
	@./scripts/db-restore.sh

db-status:
	@echo "Database Status"
	@echo "==============="
	@echo ""
	@echo "Container: bigskies-postgres"
	@docker ps --filter "name=bigskies-postgres" --format "  Status: {{.Status}}" 2>/dev/null || echo "  Status: Not running"
	@echo ""
	@echo "Connection Info:"
	@echo "  Host: localhost"
	@echo "  Port: 5432"
	@echo "  Database: bigskies"
	@echo "  User: bigskies"
	@echo ""
	@if docker ps --format '{{.Names}}' | grep -q "^bigskies-postgres$$"; then \
		echo "Database Size:"; \
		docker exec bigskies-postgres psql -U bigskies -d bigskies -c "SELECT pg_size_pretty(pg_database_size('bigskies')) as size;" -t 2>/dev/null | xargs echo "  Total:" || echo "  (unable to query)"; \
		echo ""; \
		echo "Tables:"; \
		docker exec bigskies-postgres psql -U bigskies -d bigskies -c "\\dt" 2>/dev/null || echo "  (unable to query)"; \
	else \
		echo "Database is not running. Start with: make docker-up"; \
	fi

# Plugin-specific targets (if ascom-alpaca-simulator is added to docker-compose.yml)
plugin-ascom-build:
	@echo "Building ASCOM Alpaca Simulator plugin..."
	@docker-compose build ascom-alpaca-simulator

plugin-ascom-up:
	@echo "Starting ASCOM Alpaca Simulator plugin..."
	@docker-compose up -d ascom-alpaca-simulator
	@echo "Plugin started. Access at http://localhost:32323"

plugin-ascom-down:
	@echo "Stopping ASCOM Alpaca Simulator plugin..."
	@docker-compose stop ascom-alpaca-simulator

plugin-ascom-logs:
	@docker-compose logs -f ascom-alpaca-simulator
