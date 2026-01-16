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

**Current Phase**: Backend Implementation Complete ✅

- [x] Project initialization and directory structure
- [x] Core dependencies added (MQTT, PostgreSQL, Docker SDK, etc.)
- [x] Build tooling (Makefile, linters, formatters)
- [x] Base coordinator pattern and health check infrastructure
- [x] All 7 coordinators implemented and operational
- [x] Docker Compose orchestration
- [x] Unit tests for coordinators and engines
- [x] Integration tests for all coordinators
- [ ] CI/CD pipeline (GitHub Actions)
- [ ] Plugin SDK documentation and examples
- [ ] Advanced ASCOM simulator integration

**Completed Coordinators** (7/7):
- ✅ Message Coordinator - MQTT broker management and health monitoring
- ✅ Security Coordinator - JWT auth, RBAC, user/role management, TLS/mTLS
- ✅ DataStore Coordinator - PostgreSQL connection pool management
- ✅ Application Coordinator - Service registry and health monitoring
- ✅ Plugin Coordinator - Plugin lifecycle management (install, verify, remove)
- ✅ UI Element Coordinator - Plugin UI element registry and provisioning
- ✅ Telescope Coordinator - ASCOM Alpaca integration, device management, multi-tenant configs

**Next Steps**:
- CI/CD pipeline setup
- Plugin SDK development and examples
- Enhanced documentation and architecture diagrams

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
