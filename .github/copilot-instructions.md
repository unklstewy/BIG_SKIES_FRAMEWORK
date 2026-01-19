# BIG SKIES Framework - AI Coding Assistant Instructions

## Project Overview
BIG SKIES is a plugin-extensible backend framework for telescope operations using a microservices architecture. Coordinators communicate via MQTT message bus, delegating technical operations to specialized engines.

## Architecture Fundamentals
- **Coordinators** own MQTT subscriptions and manage domain-specific logic
- **Engines** handle technical capabilities but **NEVER** touch MQTT (strictly enforced)
- **BaseCoordinator** provides common lifecycle, health checks, and credential management
- All components implement health checks with periodic MQTT publishing
- **Database-driven configuration** with migration patterns from struct-based config

## Critical Patterns & Conventions

### Coordinator Implementation
```go
type MyCoordinator struct {
    *BaseCoordinator
    config *MyCoordinatorConfig
    // domain-specific fields
}

func NewMyCoordinator(config *MyCoordinatorConfig, logger *zap.Logger) (*MyCoordinator, error) {
    mqttClient, _ := CreateMQTTClient(config.BrokerURL, mqtt.CoordinatorMyName, logger)
    base := NewBaseCoordinator(mqtt.CoordinatorMyName, mqttClient, logger)

    mc := &MyCoordinator{BaseCoordinator: base, config: config}
    mc.RegisterHealthCheck(mc) // Register self health check
    return mc, nil
}

func (mc *MyCoordinator) Start(ctx context.Context) error {
    // Wait for credentials if database access needed
    if _, err := mc.WaitForCredentials(ctx, 30*time.Second); err != nil {
        return err
    }

    // Start base coordinator
    if err := mc.BaseCoordinator.Start(ctx); err != nil {
        return err
    }

    // Start health publishing
    go mc.StartHealthPublishing(ctx)
    return nil
}
```

### MQTT Communication Patterns

**Topic Structure**:
```
bigskies/coordinator/{coordinator_name}/{action}/{resource}[/{detail}]
```

**Message Types**:
- **Command**: Synchronous operations (`bigskies/coordinator/security/auth/login`)
- **Event**: Fire-and-forget notifications (`bigskies/coordinator/service/event/heartbeat`)
- **Status**: Periodic broadcasts (`bigskies/coordinator/health/telescope-coordinator`)
- **Request-Response**: Correlated operations with message IDs

**Message Envelope**:
```json
{
  "message_id": "uuid",
  "type": "command|event|status|request|response",
  "source": "coordinator:name",
  "timestamp": "ISO-8601",
  "payload": { ... }
}
```

### Database Access Pattern
```go
// In Start() method - Wait for bootstrap credentials
creds, err := mc.WaitForCredentials(ctx, 30*time.Second)
if err != nil {
    return err
}

// Get connection URL
dbURL, err := mc.GetDatabaseURL()
if err != nil {
    return err
}

// Use pgxpool for connections
pool, err := pgxpool.New(ctx, dbURL)
```

### Configuration Management

**Database-Driven Pattern** (preferred):
```go
// Load from database with defaults
coordConfig, err := configLoader.LoadCoordinatorConfig(ctx, "my-coordinator")
brokerURL, _ := coordConfig.GetString("broker_url", "tcp://mqtt-broker:1883")
maxConns, _ := coordConfig.GetInt("max_connections", 10)

// Runtime config updates
func (c *MyCoordinator) handleConfigUpdate(topic string, payload []byte) error {
    newConfig, err := c.configLoader.LoadCoordinatorConfig(ctx, c.Name())
    if err != nil {
        return err
    }
    // Apply changes thread-safely
}
```

### Health Check Implementation
```go
func (mc *MyCoordinator) Check(ctx context.Context) *healthcheck.Result {
    status := healthcheck.StatusHealthy
    message := "Component is healthy"
    details := map[string]interface{}{}

    // Component-specific checks

    return &healthcheck.Result{
        ComponentName: "my-coordinator",
        Status: status,
        Message: message,
        Timestamp: time.Now(),
        Details: details,
    }
}
```

## Development Workflows

### Building & Testing
- `make build` - Build all coordinators
- `make test` - Run unit tests
- `make test-integration` - Run integration tests (requires `make docker-up`)
- `make docker-build && make docker-up` - Full deployment
- `make lint` - Run golangci-lint, goimports, staticcheck

### Adding New Coordinator
1. Create `cmd/my-coordinator/main.go` with database config loading
2. Implement coordinator in `internal/coordinators/my_coordinator.go`
3. Add to `docker-compose.yml` with dependencies
4. Run `scripts/generate-dockerfiles.sh` to create Dockerfile
5. Add database config schema to `configs/sql/coordinator_config_schema.sql`
6. Update Makefile targets if needed

### Database Schema Changes
1. Add SQL migration to `configs/sql/`
2. Update `configs/bootstrap.yaml` migrations list
3. Restart bootstrap coordinator: `docker restart bigskies-bootstrap`

### Plugin Development
1. Create plugin directory under `plugins/`
2. Implement `plugin.json` manifest with MQTT topics and Docker config
3. Build plugin container with MQTT client
4. Plugin coordinator handles lifecycle via Docker API

### Configuration Migration (Struct → Database)
```go
// 1. Add to coordinator_config_schema.sql
INSERT INTO coordinator_config (coordinator_name, config_key, config_value)
VALUES ('my-coordinator', 'setting_name', '"default_value"');

// 2. Update main.go to load from database
configLoader := config.NewLoader(dbPool)
coordConfig, _ := configLoader.LoadCoordinatorConfig(ctx, "my-coordinator")

// 3. Add runtime update handler
func (c *MyCoordinator) SetConfigLoader(loader *config.Loader) {
    c.configLoader = loader
    // Subscribe to config updates
}
```

## Key Files & Directories
- `internal/coordinators/base.go` - Base coordinator with WaitForCredentials
- `docs/architecture/COORDINATOR_ENGINE_ARCHITECTURE.md` - Complete architecture guide
- `pkg/mqtt/` - MQTT client, topics, message types
- `pkg/healthcheck/` - Health monitoring with aggregation
- `internal/config/loader.go` - Database-driven configuration system
- `scripts/generate-dockerfiles.sh` - Template-based Dockerfile generation
- `deployments/docker-compose/docker-compose.yml` - Service orchestration
- `configs/sql/coordinator_config_schema.sql` - Configuration schema
- `plugins/examples/ascom-alpaca-simulator/` - Complete plugin example

## Integration Points
- **ASCOM Alpaca**: Astronomy protocol via `internal/engines/ascom/`
- **PostgreSQL**: Database via pgx/v5 pool with migrations
- **Docker**: Plugin containers via moby/docker client
- **MQTT**: Message bus via Eclipse Paho with QoS and reconnection
- **Multi-Framework UI**: GTK, Flutter, Unity, Qt, WPF, MFC, Blazor support

## Common Pitfalls
- **NEVER** let engines subscribe to MQTT topics (coordinators only)
- Always embed BaseCoordinator for lifecycle management
- Use WaitForCredentials before database operations (bootstrap dependency)
- Register health checks with the health engine
- Follow topic naming conventions exactly
- Use database-driven config, not struct-based
- Implement proper shutdown functions for cleanup
- Handle context cancellation in all operations
- Use structured logging with zap consistently

## Testing Approach
- Unit tests for individual components
- Integration tests require full docker-compose stack (`make docker-up`)
- Health checks tested via mock implementations
- MQTT testing via embedded broker or mocks
- Database tests use test fixtures
- Plugin tests use Docker containers

## ASCOM Device Types
- **Telescope**: Mount/OTA control (slew, track, park)
- **Camera**: Imaging capture and settings
- **Dome**: Observatory dome synchronization
- **Focuser**: Focus position control
- **FilterWheel**: Filter selection
- **Rotator**: Field rotation for alignment

## UI Framework Support
- **GTK**: Python GTK widgets
- **Flutter**: Material Design components
- **Unity**: Canvas and UI elements
- **Qt**: QWidgets and layouts
- **WPF**: XAML controls
- **MFC**: Windows dialogs and controls
- **Blazor**: WebAssembly components