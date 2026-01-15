# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

BIG_SKIES_FRAMEWORK is a plugin-extensible backend framework for telescope operations (terrestrial and astronomy). The framework uses a microservices architecture with Docker containers as plugin extensions, coordinated via an MQTT message bus with JSON interchange format.

**Repository**: `git@github.com:unklstewy/BIG_SKIES_FRAMEWORK.git`

## Architecture

The system follows a coordinator-based architecture defined in `docs/architecture/big_skies_architecture_gojs.json`. Key coordinators:

- **message-coordinator**: Message bus and health monitoring
- **application-svc-coordinator**: Tracks and monitors microservices
- **security-coordinator**: Manages security model, roles, groups, accounts, mTLS/SSL (Let's Encrypt)
- **telescope-coordinator**: ASCOM-Alpaca interface and telescope configurations
- **plugin-coordinator**: Plugin lifecycle management (install, verify, version, update, remove by GUID)
- **ui-element-coordinator**: UI provisioning from plugin APIs
- **data-store-coordinator**: PostgreSQL database management

Each coordinator has:
- Health check engine (API, reporter, diagnostics)
- Setup wizard API for configuration
- Specific service engines for domain logic

## Development Requirements

### Language & Technology Stack
- **Primary language**: Go
- **Database**: PostgreSQL (containerized, required)
- **Message bus**: MQTT with JSON messages
- **Container platform**: Docker (all supporting applications must be containerized)
- **Security**: mTLS support required, role-based access control (RBAC)
- **External integration**: ASCOM Alpaca Simulator for testing (see https://github.com/ASCOMInitiative)

### Project Structure
```
.
├── cmd/                    # Application entry points (coordinator services)
├── internal/              # Private application code
│   ├── coordinators/     # Coordinator implementations
│   ├── engines/          # Engine implementations (health, security, etc.)
│   ├── services/         # Service implementations
│   ├── models/           # Internal data models
│   └── config/           # Configuration management
├── pkg/                   # Public libraries
│   ├── mqtt/             # MQTT client wrapper
│   ├── healthcheck/      # Health check interfaces
│   ├── plugin/           # Plugin SDK
│   └── api/              # Common API types
├── api/                   # API definitions and specs
├── configs/              # Configuration files
├── deployments/          # Docker and orchestration configs
├── scripts/              # Build and utility scripts
└── test/                 # Integration tests and fixtures
```

### Common Commands
**Build and Test**:
```bash
make help           # Show all available commands
make build          # Build all services
make test           # Run all tests
make test-coverage  # Run tests with coverage report
make fmt            # Format code
make lint           # Run linters
make clean          # Clean build artifacts
```

**Development Setup**:
```bash
make install-tools  # Install golangci-lint, goimports, staticcheck, gotestsum
```

**Docker**:
```bash
make docker-build   # Build Docker images
make docker-up      # Start services with docker-compose
make docker-down    # Stop services
make docker-logs    # View logs
```

**Dependencies**:
```bash
go mod tidy         # Clean up dependencies
go mod download     # Download dependencies
```

### Code Standards
- **Documentation**: All code must be heavily commented - every function, method, interface, and struct requires documentation explaining purpose and behavior
- **Security**: Follow modern PWA security-focused design principles
- **Architecture**: Maintain separation between coordinators and their engines/services

### CI/CD & Workflow
- Use GitHub Actions for CI/CD automation
- Track progress with GitHub Issues
- Document features and decisions in GitHub Wiki
- Include co-author attribution in commits: `Co-Authored-By: Warp <agent@warp.dev>`

## Frontend (On Hold)
Multiple frontend options are planned but on hold until backend completion:
- Flutter
- Unity Engine
- Python GTK

## Example Plugin Use Cases
### Terrestrial
- ADS-B data source for aircraft tracking and imaging
- Wildlife tracker with AI-based visual identification

### Astronomy
- DeepSky@Home: Distributed deep sky survey workloads
- Astro photography spatial VR viewer
