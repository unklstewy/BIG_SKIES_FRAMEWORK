# BIG SKIES FRAMEWORK

The BIG SKIES FRAMEWORK is a plugin-extensible backend framework for telescope operations (terrestrial and astronomy). Built with Go, it uses a microservices architecture with Docker containers as plugin extensions, coordinated via an MQTT message bus.

## Features

- **Plugin Architecture**: Extensible via Docker containers
- **Microservices**: Coordinator-based architecture for modularity
- **Message Bus**: MQTT with JSON interchange format
- **Security**: mTLS support, role-based access control (RBAC)
- **Database**: PostgreSQL with containerized deployment
- **ASCOM Integration**: Full ASCOM Alpaca interface support

## Architecture

The system consists of 7 main coordinators:

1. **message-coordinator** - Message bus and health monitoring
2. **security-coordinator** - Security model, roles, accounts, mTLS/SSL
3. **data-store-coordinator** - PostgreSQL database management
4. **application-svc-coordinator** - Microservice tracking and monitoring
5. **plugin-coordinator** - Plugin lifecycle management
6. **telescope-coordinator** - ASCOM-Alpaca interface and telescope configs
7. **ui-element-coordinator** - UI provisioning from plugin APIs

See `docs/architecture/big_skies_architecture_gojs.json` for detailed architecture.

## Quick Start

### Prerequisites

- Go 1.25.5 or later
- Docker and Docker Compose
- Make

### Installation

1. Clone the repository:
```bash
git clone git@github.com:unklstewy/BIG_SKIES_FRAMEWORK.git
cd BIG_SKIES_FRAMEWORK
```

2. Install development tools:
```bash
make install-tools
```

3. Install dependencies:
```bash
go mod download
```

### Development

```bash
make help           # Show all available commands
make build          # Build all services
make test           # Run tests
make lint           # Run linters
make fmt            # Format code
```

### Running Services

```bash
make docker-up      # Start all services
make docker-logs    # View logs
make docker-down    # Stop all services
```

## Project Status

**Current Phase**: Telescope Coordinator Implementation ✅

- [x] Project initialization and directory structure
- [x] Core dependencies added (MQTT, PostgreSQL, Docker SDK, etc.)
- [x] Build tooling (Makefile, linters, formatters)
- [x] Base coordinator pattern and health check infrastructure
- [x] Message coordinator (MQTT broker management)
- [x] ASCOM engine with Alpaca client
- [x] Telescope coordinator with multi-tenant support
- [x] Docker Compose orchestration
- [x] Unit tests for coordinators and engines
- [ ] Integration tests with ASCOM simulator
- [ ] Remaining coordinator implementations
- [ ] Plugin SDK and lifecycle management
- [ ] CI/CD pipeline

**Completed Coordinators**:
- ✅ Message Coordinator - MQTT broker management and health monitoring
- ✅ Telescope Coordinator - ASCOM Alpaca integration, device management, multi-tenant configs

**In Progress**:
- Integration testing with ASCOM Alpaca Simulator

See `docs/coordinators/` for coordinator-specific documentation.

## Technology Stack

- **Language**: Go
- **Message Bus**: MQTT (Eclipse Paho)
- **Database**: PostgreSQL
- **Containers**: Docker
- **HTTP Router**: Gin
- **Configuration**: Viper
- **Logging**: Zap
- **Testing**: Testify

## Contributing

All commits should include co-author attribution:
```
Co-Authored-By: Warp <agent@warp.dev>
```

See GitHub Issues for current tasks and milestones.

## Documentation

- `WARP.md` - Development guidelines for AI assistants
- `INITSTATE.MD` - Initial project requirements
- `docs/architecture/` - Architecture specifications
- `next_steps.txt` - Implementation roadmap

## License

See LICENSE file for details.
