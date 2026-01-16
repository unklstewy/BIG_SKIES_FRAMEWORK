# Bootstrap Coordinator Setup Guide

This guide explains how to set up and use the Bootstrap Coordinator to manage database credentials and migrations for the BIG SKIES Framework.

## Overview

The Bootstrap Coordinator is a special coordinator that:
1. **Loads database credentials** from a `.pgpass` file in a shared volume
2. **Runs database migrations** to set up the schema
3. **Publishes credentials** to other coordinators via MQTT (base64-encoded for minor obscurity)
4. **Stays running** to republish credentials when coordinators restart

## Architecture

```
┌─────────────────────────────────────────┐
│   Bootstrap Coordinator (Container)     │
│                                         │
│  1. Load .pgpass from /shared/secrets  │
│  2. Run database migrations            │
│  3. Publish credentials via MQTT       │
│  4. Listen for credential requests     │
└─────────────────────────────────────────┘
           │
           │ MQTT Topics:
           │  - bigskies/coordinator/bootstrap/credentials (publish)
           │  - bigskies/coordinator/bootstrap/request (subscribe)
           ↓
┌─────────────────────────────────────────┐
│   Other Coordinators (Containers)       │
│                                         │
│  1. Subscribe to bootstrap/credentials │
│  2. Decode base64 pgpass path          │
│  3. Load credentials from shared vol   │
│  4. Connect to database                │
│  5. Load configuration from DB         │
└─────────────────────────────────────────┘
```

## Shared Volume

All containers that need database access share a **tmpfs volume** mounted at `/shared/secrets`:

- **Type**: tmpfs (memory-based, doesn't persist to disk)
- **Permissions**: 0600, uid=1000
- **Contains**: `.pgpass` file with PostgreSQL credentials

### Why tmpfs?
- Credentials only exist in memory, never written to disk
- Automatically cleaned up when containers stop
- Better security than bind-mounted host files

## Setup Steps

### 1. Create .pgpass File

First, create a `.pgpass` file with your PostgreSQL credentials:

```bash
# Create the file
cat > .pgpass << EOF
localhost:5432:bigskies:bigskies:bigskies_dev_password
EOF

# Set proper permissions (REQUIRED)
chmod 0600 .pgpass
```

**Format**: `hostname:port:database:username:password`

**Wildcards supported**: `*:*:*:bigskies:password` matches any host/port/database

### 2. Copy .pgpass to Docker Volume

The `.pgpass` file needs to be copied into the `shared_secrets` volume before starting containers.

**Option A: Manual Copy (Development)**
```bash
# Create a temporary container to copy the file
docker run --rm -v bigskies_shared_secrets:/shared/secrets -v $(pwd):/host alpine sh -c "
  mkdir -p /shared/secrets &&
  cp /host/.pgpass /shared/secrets/.pgpass &&
  chmod 0600 /shared/secrets/.pgpass &&
  chown 1000:1000 /shared/secrets/.pgpass
"
```

**Option B: Init Container (Production)**
Add an init container to docker-compose.yml:
```yaml
services:
  init-secrets:
    image: alpine:latest
    volumes:
      - shared_secrets:/shared/secrets
      - ./secrets:/secrets:ro
    command: sh -c "cp /secrets/.pgpass /shared/secrets/.pgpass && chmod 0600 /shared/secrets/.pgpass && chown 1000:1000 /shared/secrets/.pgpass"
```

### 3. Configure Bootstrap Coordinator

Edit `configs/bootstrap.yaml`:

```yaml
database:
  host: postgres
  port: 5432
  database: bigskies
  sslmode: disable
  max_connections: 10
  connection_timeout: 10s

mqtt:
  broker_url: tcp://mqtt-broker
  broker_port: 1883
  client_id: bootstrap-coordinator
  reconnect_interval: 5s
  max_reconnect_attempts: 10

migrations:
  directory: configs/sql
  migrations:
    - name: coordinator_config_schema
      file: coordinator_config_schema.sql
      version: "1.0.0"
    - name: security_schema
      file: security_schema.sql
      version: "1.0.0"
    - name: telescope_schema
      file: telescope_schema.sql
      version: "1.0.0"
```

### 4. Start Services

```bash
# Start all services
docker-compose up -d

# View bootstrap coordinator logs
docker logs -f bigskies-bootstrap

# Check all coordinator health
docker-compose ps
```

## Coordinator Startup Sequence

The docker-compose.yml enforces the following startup order:

1. **postgres** + **mqtt-broker** (infrastructure)
2. **bootstrap-coordinator** (loads credentials, runs migrations)
3. **datastore-coordinator** (database connection manager)
4. **security-coordinator** + **message-coordinator** (core services)
5. **application-coordinator** (service registry)
6. **plugin-coordinator** + **telescope-coordinator** + **uielement-coordinator** (application services)

## MQTT Topics

### Published by Bootstrap Coordinator

**Topic**: `bigskies/coordinator/bootstrap/credentials`

**Payload**:
```json
{
  "pgpass_path": "L3NoYXJlZC9zZWNyZXRzLy5wZ3Bhc3M=",
  "version": "1.0"
}
```

- `pgpass_path`: Base64-encoded path to `.pgpass` file
- `version`: Protocol version for future compatibility

**Frequency**: 
- Immediately on startup
- Every 30 seconds (periodic republish)
- On request (see below)

### Subscribed by Bootstrap Coordinator

**Topic**: `bigskies/coordinator/bootstrap/request`

**Payload**: Coordinator name (e.g., `"datastore-coordinator"`)

**Purpose**: Allows coordinators to request credentials if they missed the initial publish

## Coordinator Integration

All coordinators inherit credential loading from `BaseCoordinator`. To use it:

```go
// In your coordinator's main.go

// 1. Create MQTT client
mqttClient, err := mqtt.NewClient(...)

// 2. Connect to MQTT
if err := mqttClient.Connect(); err != nil {
    log.Fatal(err)
}

// 3. Create base coordinator
baseCoord := coordinators.NewBaseCoordinator("my-coordinator", mqttClient, logger)

// 4. Wait for credentials (30 second timeout)
ctx := context.Background()
creds, err := baseCoord.WaitForCredentials(ctx, 30*time.Second)
if err != nil {
    log.Fatal("Failed to load credentials:", err)
}

// 5. Use credentials to connect to database
dbURL := creds.ConnectionString()
db, err := pgxpool.New(ctx, dbURL)
```

### Fallback Behavior

If the bootstrap coordinator is unavailable or the credential message is not received within the timeout:

1. ✅ **WaitForCredentials** returns an error
2. ⚠️ Coordinator startup fails (will be retried by Docker)
3. 🔄 Docker restart policy eventually succeeds when bootstrap is available

### Manual Override

For development, you can bypass the bootstrap coordinator by setting environment variables:

```bash
export DATABASE_URL="postgresql://bigskies:password@localhost:5432/bigskies"
```

Then modify your coordinator to check `DATABASE_URL` before calling `WaitForCredentials()`.

## Troubleshooting

### Bootstrap Coordinator Fails to Start

**Check logs**:
```bash
docker logs bigskies-bootstrap
```

**Common issues**:
- `.pgpass` file not found in `/shared/secrets`
- `.pgpass` file has wrong permissions (must be 0600)
- Cannot connect to PostgreSQL (check `postgres` service)
- Cannot connect to MQTT broker (check `mqtt-broker` service)

### Coordinators Timeout Waiting for Credentials

**Symptoms**: Coordinators log "timeout waiting for database credentials"

**Solutions**:
1. Check bootstrap coordinator is running: `docker ps | grep bootstrap`
2. Check MQTT broker is accessible: `docker exec bigskies-bootstrap nc -zv mqtt-broker 1883`
3. Manually request credentials:
   ```bash
   docker exec bigskies-bootstrap mosquitto_pub -t bigskies/coordinator/bootstrap/request -m "test"
   ```
4. Check bootstrap is publishing:
   ```bash
   docker exec bigskies-mqtt mosquitto_sub -t bigskies/coordinator/bootstrap/credentials -C 1
   ```

### Database Migrations Fail

**Check migration logs**:
```bash
docker logs bigskies-bootstrap | grep migration
```

**Common issues**:
- SQL syntax errors in migration files
- Migrations already applied (idempotent, should be safe)
- Database connection issues

**Rerun migrations**:
```bash
docker restart bigskies-bootstrap
```

**Skip migrations** (if already applied):
```bash
docker-compose up -d bootstrap-coordinator --scale bootstrap-coordinator=0
docker run --rm -v bigskies_shared_secrets:/shared/secrets bigskies-bootstrap --skip-migrations
```

## Security Considerations

### Development
- `.pgpass` stored in tmpfs (memory-only)
- Anonymous MQTT access enabled
- Plaintext credentials in `.pgpass`

### Production Recommendations
1. **Enable MQTT authentication**:
   ```yaml
   mqtt:
     username: coordinator_user
     password: ${MQTT_PASSWORD}  # From environment
   ```

2. **Use PostgreSQL SSL**:
   ```yaml
   database:
     sslmode: require
     sslcert: /app/certs/client-cert.pem
     sslkey: /app/certs/client-key.pem
   ```

3. **Encrypt .pgpass contents** (future enhancement):
   - Use age/sops for encryption at rest
   - Decrypt in bootstrap coordinator at runtime

4. **Rotate credentials**:
   - Update `.pgpass` file
   - Restart bootstrap coordinator
   - All coordinators will receive new credentials

5. **Audit access**:
   - Enable PostgreSQL connection logging
   - Monitor MQTT topic access

## Advanced Configuration

### Custom .pgpass Location

Override in docker-compose.yml:
```yaml
services:
  bootstrap-coordinator:
    command: [
      "--pgpass", "/custom/path/.pgpass",
      "--config", "/app/configs/bootstrap.yaml"
    ]
```

### Skip Migrations

Useful when migrations are managed externally:
```yaml
services:
  bootstrap-coordinator:
    command: [
      "--skip-migrations",
      "--config", "/app/configs/bootstrap.yaml"
    ]
```

### Publish-Only Mode

Only publish credentials, don't run migrations:
```yaml
services:
  bootstrap-coordinator:
    command: [
      "--publish-only",
      "--config", "/app/configs/bootstrap.yaml"
    ]
```

## Next Steps

- Configure your coordinators to use `WaitForCredentials()`
- Set up monitoring for bootstrap coordinator health
- Implement credential rotation procedures
- Enable MQTT authentication for production

## References

- [PostgreSQL .pgpass Documentation](https://www.postgresql.org/docs/current/libpq-pgpass.html)
- [MQTT QoS Levels](https://www.hivemq.com/blog/mqtt-essentials-part-6-mqtt-quality-of-service-levels/)
- [Docker Volumes](https://docs.docker.com/storage/volumes/)
