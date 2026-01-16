# Telescope Coordinator

The Telescope Coordinator manages telescope configurations, ASCOM Alpaca device integration, and multi-tenant telescope operations.

## Overview

The telescope coordinator provides:
- **ASCOM Alpaca Integration**: Full support for ASCOM Alpaca protocol devices (telescopes, cameras, domes, focusers, filter wheels)
- **Multi-Tenant Support**: Isolated telescope configurations per user/organization with role-based access control
- **Device Discovery**: Automatic discovery of ASCOM Alpaca devices on the network
- **Device Lifecycle**: Connection management, health monitoring, and automatic reconnection
- **Session Tracking**: Telescope usage sessions with start/end times and status tracking
- **Observatory Management**: Site configurations with coordinates and timezone information

## Architecture

### Components

1. **Telescope Coordinator** (`internal/coordinators/telescope_coordinator.go`)
   - MQTT message routing and handling
   - Configuration CRUD operations
   - Device connection management
   - Telescope control commands (slew, park, track, abort)
   - Session management

2. **ASCOM Engine** (`internal/engines/ascom/engine.go`)
   - Device registration and lifecycle management
   - Connection pool management
   - Health monitoring with failure detection
   - Automatic reconnection on failures
   - Thread-safe device access

3. **ASCOM Client** (`internal/engines/ascom/client.go`)
   - HTTP client for ASCOM Alpaca API
   - Device-specific operations:
     - Telescope: slew, park, track, home, abort
     - Camera: exposure, cooling, image download
     - Dome: shutter, azimuth, park
     - Focuser: position, temperature compensation
     - Filter wheel: position changes

### Database Schema

Located in `configs/sql/telescope_schema.sql`:

- **telescope_configurations**: Telescope config metadata (owner, site, mount type, capabilities)
- **telescope_devices**: ASCOM device registrations (device type, server URL, connection info)
- **telescope_permissions**: Fine-grained access control (read, write, control, configure)
- **observatory_sites**: Physical site information (coordinates, elevation, timezone)
- **telescope_sessions**: Usage tracking (start/end times, user, session type, notes)

### MQTT Topics

All topics are prefixed with `bigskies/coordinator/telescope/`

#### Configuration Management
- `config/create` - Create new telescope configuration
- `config/list` - List configurations for owner
- `config/get` - Get specific configuration
- `config/update` - Update configuration
- `config/delete` - Delete configuration

#### Device Management
- `device/discover` - Discover ASCOM devices on network
- `device/connect` - Connect to registered device
- `device/disconnect` - Disconnect from device

#### Telescope Control
- `control/slew` - Slew to RA/Dec coordinates
- `control/park` - Park telescope
- `control/unpark` - Unpark telescope
- `control/track` - Enable/disable tracking
- `control/abort` - Abort current slew operation

#### Status
- `status/get` - Get telescope status (position, tracking, parked, etc.)

#### Session Management
- `session/start` - Start telescope session
- `session/end` - End telescope session

All commands respond on `response/{subtopic}` (e.g., `response/config/create/response`)

## Configuration

Environment variables:
- `DATABASE_URL` - PostgreSQL connection string
- `MQTT_BROKER` - MQTT broker URL (default: `tcp://mqtt:1883`)
- `DISCOVERY_PORT` - ASCOM discovery port (default: `32227`)
- `HEALTH_CHECK_INTERVAL` - Health check interval (default: `30s`)

## Usage Examples

### Discover ASCOM Devices

```json
{
  "topic": "bigskies/coordinator/telescope/device/discover",
  "payload": {
    "port": 32227
  }
}
```

Response:
```json
{
  "success": true,
  "devices": [
    {
      "device_id": "alpaca-telescope-0",
      "device_type": "telescope",
      "device_number": 0,
      "name": "Simulator Telescope",
      "server_url": "http://192.168.1.100:11111",
      "uuid": "device-unique-id"
    }
  ]
}
```

### Create Telescope Configuration

```json
{
  "topic": "bigskies/coordinator/telescope/config/create",
  "payload": {
    "name": "My Observatory Telescope",
    "description": "Primary imaging telescope",
    "owner_id": "user-uuid",
    "owner_type": "user",
    "site_id": "site-uuid",
    "mount_type": "equatorial",
    "capabilities": {
      "has_camera": true,
      "has_dome": true,
      "has_focuser": true,
      "has_filter_wheel": true
    }
  }
}
```

### Connect to Device

```json
{
  "topic": "bigskies/coordinator/telescope/device/connect",
  "payload": {
    "device_id": "alpaca-telescope-0"
  }
}
```

### Slew Telescope

```json
{
  "topic": "bigskies/coordinator/telescope/control/slew",
  "payload": {
    "telescope_id": "config-uuid",
    "right_ascension": 12.5,
    "declination": 45.0
  }
}
```

### Start Session

```json
{
  "topic": "bigskies/coordinator/telescope/session/start",
  "payload": {
    "telescope_id": "config-uuid",
    "user_id": "user-uuid",
    "session_type": "imaging",
    "notes": "M31 imaging session"
  }
}
```

## Multi-Tenant Access Control

The telescope coordinator enforces access control through:

1. **Ownership**: Configurations are owned by users or organizations
2. **Permissions**: Fine-grained permissions (read, write, control, configure) can be granted to users/groups
3. **Session Tracking**: All usage is logged with user attribution

Permission types:
- `read` - View configuration and status
- `write` - Modify configuration
- `control` - Control telescope (slew, park, track)
- `configure` - Modify device settings

## Health Monitoring

The coordinator provides health checks via:
- Base coordinator health check (running, MQTT connected)
- ASCOM engine health check (device status, connection health)

Health status published to: `bigskies/coordinator/telescope/health`

## Testing

### Unit Tests
Located in `internal/coordinators/telescope_coordinator_test.go` and `internal/engines/ascom/engine_test.go`

Run tests:
```bash
make test
# or
go test ./internal/coordinators ./internal/engines/ascom
```

### Integration Tests
Comprehensive integration tests are located in `test/integration/telescope_coordinator_test.go`:

```bash
make test-integration
```

Tests cover:
- Health status monitoring
- Configuration CRUD operations (create, read, update, delete, list)
- Device discovery and connection management
- Telescope control commands (slew, park, unpark, tracking, abort)
- Status retrieval
- Session management (start/end)
- Multi-operation workflows
- Rapid sequential requests

All integration tests require running services:
```bash
make docker-up    # Start all services
make test-integration
```

## Deployment

### Docker Compose
The telescope coordinator runs as a containerized service:

```yaml
telescope-coordinator:
  build:
    context: .
    dockerfile: deployments/docker/Dockerfile.telescope-coordinator
  environment:
    - DATABASE_URL=postgresql://user:pass@postgres:5432/bigskies
    - MQTT_BROKER=tcp://mqtt:1883
  depends_on:
    - postgres
    - mqtt
    - security-coordinator
```

Start services:
```bash
make docker-up
```

### Binary
Build and run standalone:
```bash
make build
./bin/telescope-coordinator \
  --db-url="postgresql://localhost:5432/bigskies" \
  --mqtt-broker="tcp://localhost:1883"
```

## Development

### Adding New Device Types
1. Add device-specific methods to `internal/engines/ascom/client.go`
2. Update `AlpacaDevice` model in `internal/models/alpaca.go`
3. Add tests in `internal/engines/ascom/client_test.go`
4. Update capabilities in telescope configuration schema

### Adding New Control Commands
1. Add MQTT handler in `internal/coordinators/telescope_coordinator.go`
2. Subscribe to new topic in `Start()` method
3. Add corresponding tests
4. Update this documentation

## Future Enhancements

- [ ] WebSocket support for real-time status updates
- [ ] Image acquisition and storage integration
- [ ] Automated scheduling and queue management
- [ ] Weather integration for safety monitoring
- [ ] Multiple telescope support per configuration
- [ ] Plate solving integration
- [ ] Auto-guiding support
- [ ] Advanced mount modeling

## References

- [ASCOM Alpaca API](https://ascom-standards.org/api/)
- [ASCOM Alpaca Simulators](https://github.com/ASCOMInitiative/ASCOM.Alpaca.Simulators)
- BIG SKIES Architecture: `docs/architecture/big_skies_architecture_gojs.json`
