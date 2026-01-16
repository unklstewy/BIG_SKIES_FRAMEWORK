# ASCOM Coordinator Deployment Guide

## Overview

The ASCOM Coordinator provides an ASCOM Alpaca-compliant REST API interface for controlling BigSkies-managed telescope hardware. It acts as a bridge between standard astronomy software (N.I.N.A., PHD2, etc.) and the BigSkies framework.

## Architecture

```
ASCOM Client (N.I.N.A.) → ASCOM Coordinator → MQTT Bus → Telescope Coordinator → Hardware
     (HTTP/UDP)              (Container)        (Broker)     (Container)
```

The ASCOM coordinator:
- Exposes ASCOM Alpaca REST API on port 11111 (HTTP)
- Provides UDP discovery on port 32227
- Translates ASCOM API calls to BigSkies MQTT messages
- Routes requests to telescope-coordinator for device control
- Stores device configurations in PostgreSQL
- Integrates with security-coordinator for authentication

## Docker Deployment

### Start the ASCOM Coordinator

```bash
# Start all services including ASCOM coordinator
cd deployments/docker-compose
docker-compose up -d ascom-coordinator

# View logs
docker-compose logs -f ascom-coordinator

# Check health
docker-compose ps ascom-coordinator
```

### Stop the ASCOM Coordinator

```bash
docker-compose stop ascom-coordinator
```

### Rebuild After Changes

```bash
docker-compose build ascom-coordinator
docker-compose up -d ascom-coordinator
```

## Configuration

### Environment Variables

Set in `.env` file or docker-compose environment:

```bash
# Database
POSTGRES_PASSWORD=your_secure_password

# Logging
LOG_LEVEL=info  # debug, info, warn, error

# Security (optional)
JWT_SECRET=your_jwt_secret
```

### Command-Line Flags

Configured in docker-compose.yml `command` section:

```yaml
command: [
  "--broker-url=tcp://mqtt-broker:1883",          # MQTT broker URL
  "--database-url=postgres://...",                # PostgreSQL connection
  "--http-address=0.0.0.0:11111",                 # HTTP listen address
  "--discovery-port=32227",                       # UDP discovery port
  "--health-interval=30s",                        # Health check interval
  "--enable-cors=true",                           # Enable CORS
  "--server-name=BigSkies ASCOM Alpaca Server",   # ASCOM server name
  "--manufacturer=BigSkies",                      # Manufacturer name
  "--log-level=${LOG_LEVEL:-info}"                # Log level
]
```

## Service Dependencies

The ASCOM coordinator depends on:

1. **PostgreSQL** (`postgres`) - Device configuration storage
   - Must be healthy before starting
   - Loads ascom_schema.sql on first run

2. **MQTT Broker** (`mqtt-broker`) - Message bus communication
   - Must be started before coordinator

3. **Telescope Coordinator** (`telescope-coordinator`) - Backend device control
   - Must be running to process ASCOM requests

## Ports

| Port  | Protocol | Purpose                          |
|-------|----------|----------------------------------|
| 11111 | TCP/HTTP | ASCOM Alpaca REST API            |
| 32227 | UDP      | ASCOM Alpaca discovery service   |

### Accessing the API

From host machine:
```bash
# Test API versions endpoint
curl http://localhost:11111/management/apiversions

# Test server description
curl http://localhost:11111/management/v1/description

# Test configured devices
curl http://localhost:11111/management/v1/configureddevices
```

From ASCOM client software:
- Configure ASCOM Remote Device: `http://<host-ip>:11111`
- Discovery should auto-detect on port 32227

## Health Checks

The coordinator includes health monitoring:

### Docker Health Check
```bash
# Check container health
docker inspect bigskies-ascom-coordinator --format='{{.State.Health.Status}}'
```

Health check endpoint: `http://localhost:11111/management/apiversions`

### MQTT Health Publishing

The coordinator publishes health status every 30 seconds:
- Topic: `bigskies/coordinator/ascom/health/status`
- Includes: uptime, running state, MQTT connection status

### Manual Health Check

```bash
# Via HTTP API
curl http://localhost:11111/management/apiversions

# Via logs
docker-compose logs ascom-coordinator | grep -i health
```

## Database Schema

The ASCOM coordinator uses three tables:

1. **ascom_devices** - Device configurations
2. **ascom_device_state** - Cached device state
3. **ascom_sessions** - Client session tracking

Schema is automatically loaded from `/configs/sql/ascom_schema.sql`.

### Managing Devices

Devices are stored in PostgreSQL and loaded on startup:

```sql
-- Connect to database
docker exec -it bigskies-postgres psql -U bigskies -d bigskies

-- List ASCOM devices
SELECT id, device_type, device_number, name, enabled FROM ascom_devices;

-- Add a new telescope device
INSERT INTO ascom_devices (
    id, device_type, device_number, name, description, unique_id,
    backend_mode, backend_config, created_by, enabled
) VALUES (
    gen_random_uuid(),
    'telescope',
    0,
    'My Telescope',
    'Primary imaging telescope',
    gen_random_uuid()::text,
    'mqtt',
    '{"timeout_seconds": 30}',
    '00000000-0000-0000-0000-000000000001',  -- admin user
    true
);

-- Reload devices (restart coordinator or send MQTT reload message)
```

## Troubleshooting

### Coordinator Won't Start

1. Check dependencies are running:
   ```bash
   docker-compose ps postgres mqtt-broker telescope-coordinator
   ```

2. Check database connectivity:
   ```bash
   docker exec bigskies-postgres pg_isready -U bigskies
   ```

3. Check logs:
   ```bash
   docker-compose logs ascom-coordinator
   ```

### No Devices Available

1. Check database for devices:
   ```bash
   docker exec -it bigskies-postgres psql -U bigskies -d bigskies \
     -c "SELECT device_type, device_number, name, enabled FROM ascom_devices;"
   ```

2. Verify default device was created:
   - Default telescope device (ID: `50000000-0000-0000-0000-000000000001`)

3. Restart coordinator to reload devices:
   ```bash
   docker-compose restart ascom-coordinator
   ```

### ASCOM Client Can't Connect

1. Verify port is accessible:
   ```bash
   curl http://localhost:11111/management/apiversions
   ```

2. Check firewall allows port 11111 (TCP) and 32227 (UDP)

3. Try UDP discovery:
   ```bash
   # Send discovery packet (requires netcat)
   echo -n "alpacadiscovery1" | nc -u -w1 localhost 32227
   ```

4. Check CORS is enabled if using web client:
   ```bash
   curl -H "Origin: http://example.com" -I http://localhost:11111/management/apiversions
   ```

### MQTT Communication Issues

1. Check MQTT broker:
   ```bash
   docker-compose logs mqtt-broker
   ```

2. Monitor MQTT traffic:
   ```bash
   docker run --rm -it --network bigskies-network eclipse-mosquitto \
     mosquitto_sub -h mqtt-broker -t 'bigskies/#' -v
   ```

3. Check coordinator MQTT connection:
   ```bash
   docker-compose logs ascom-coordinator | grep -i mqtt
   ```

## Integration with BigSkies Framework

### Message Flow

1. **ASCOM Client → ASCOM Coordinator** (HTTP)
   ```
   GET http://localhost:11111/api/v1/telescope/0/rightascension
   ```

2. **ASCOM Coordinator → Telescope Coordinator** (MQTT)
   ```
   Topic: bigskies/coordinator/telescope/status/coordinates
   Payload: {request_id, device_type, device_number, method, params}
   ```

3. **Telescope Coordinator → ASCOM Coordinator** (MQTT)
   ```
   Topic: bigskies/coordinator/ascom/response/{request_id}
   Payload: {request_id, value, error_number, error_message}
   ```

4. **ASCOM Coordinator → ASCOM Client** (HTTP)
   ```
   {
     "Value": 12.5,
     "ClientTransactionID": 123,
     "ServerTransactionID": 456,
     "ErrorNumber": 0,
     "ErrorMessage": ""
   }
   ```

### Coordinator Integration

The ASCOM coordinator integrates with:

- **Security Coordinator**: Authentication/authorization (future)
- **Telescope Coordinator**: Device control operations
- **Datastore Coordinator**: Configuration persistence
- **Application Coordinator**: Service registry and health
- **Message Coordinator**: MQTT message routing

## Production Considerations

### Security

1. **Enable Authentication**: Integrate with security-coordinator
2. **Use HTTPS**: Add TLS termination (nginx/traefik)
3. **Restrict Access**: Firewall rules for ports 11111, 32227
4. **Database Security**: Use strong passwords, SSL connections

### Performance

1. **Connection Pooling**: Configure in ascom_coordinator.go
2. **Caching**: Device state cached in ascom_device_state table
3. **MQTT QoS**: Uses QoS 1 for reliable delivery
4. **Timeouts**: 30s default, configurable

### Monitoring

1. **Health Checks**: Docker health + HTTP endpoint
2. **MQTT Health**: Published every 30s
3. **Metrics**: Track in ascom_device_state table
4. **Logs**: Structured logging with zap

### Scaling

- Multiple ASCOM coordinators can run simultaneously
- Each coordinator loads all devices from database
- MQTT ensures message routing to correct backend
- Consider load balancer for multiple instances

## References

- ASCOM Alpaca Specification: https://ascom-standards.org/api/
- BigSkies Architecture: `/docs/architecture/`
- Database Schema: `/configs/sql/ascom_schema.sql`
- Coordinator Code: `/internal/coordinators/ascom_coordinator.go`
