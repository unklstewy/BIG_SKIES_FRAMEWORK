# BIG SKIES Framework - Quick Start Guide

## Prerequisites

- Docker and Docker Compose installed
- Make (for convenience commands)
- Go 1.21+ (for local development)

**First time setup? Run the automated installer:**
```bash
./scripts/setup-dev-environment.sh
```
This will install all required tools (Git, Make, Go, Docker, Docker Compose, Go dev tools).

## Quick Start (5 minutes)

### 1. Credentials are already configured! ✅

The `.pgpass` file has been created with development credentials and copied to the Docker shared volume.

### 2. Build Docker images

```bash
make docker-build
```

This will:
- Ensure `.pgpass` is in the shared volume
- Build all coordinator Docker images
- Pull PostgreSQL and MQTT broker images

### 3. Start all services

```bash
make docker-up
```

This will start:
- PostgreSQL database
- MQTT broker (Mosquitto)
- Bootstrap coordinator (runs migrations, distributes credentials)
- All 7 coordinators in dependency order

### 4. Verify everything is running

```bash
make docker-ps
```

You should see all services with status "Up":
- `bigskies-postgres`
- `bigskies-mqtt`
- `bigskies-bootstrap`
- `bigskies-datastore`
- `bigskies-security`
- `bigskies-message`
- `bigskies-application`
- `bigskies-plugin`
- `bigskies-telescope`
- `bigskies-uielement`

### 5. View logs

```bash
# All services
make docker-logs

# Specific service
docker logs -f bigskies-bootstrap
docker logs -f bigskies-security
```

### 6. Stop services

```bash
make docker-down
```

## Common Commands

### Development

```bash
make build              # Build Go binaries locally
make test               # Run tests
make fmt                # Format code
make lint               # Run linters
```

### Docker

```bash
make docker-build       # Build images
make docker-up          # Start services
make docker-down        # Stop services
make docker-restart     # Restart all services
make docker-logs        # View logs (follow mode)
make docker-ps          # View service status
make docker-purge       # Purge everything (DESTRUCTIVE - asks for confirmation)
```

### Credentials

```bash
# Update .pgpass and copy to Docker volume
./scripts/update-pgpass.sh

# Edit credentials
nano .pgpass
./scripts/update-pgpass.sh
make docker-restart
```

## Service Ports

| Service | Port | Description |
|---------|------|-------------|
| PostgreSQL | 5432 | Database |
| MQTT (TCP) | 1883 | Message broker |
| MQTT (WebSocket) | 9001 | WebSocket interface |

## Architecture

```
┌──────────────────┐
│   PostgreSQL     │  (Database)
└────────┬─────────┘
         │
┌────────┴─────────┐
│  MQTT Broker     │  (Message Bus)
└────────┬─────────┘
         │
┌────────┴─────────────────────────┐
│  Bootstrap Coordinator           │  (Credentials + Migrations)
└────────┬─────────────────────────┘
         │
         ├─► DataStore Coordinator
         ├─► Security Coordinator
         ├─► Message Coordinator
         ├─► Application Coordinator
         ├─► Plugin Coordinator
         ├─► Telescope Coordinator
         └─► UIElement Coordinator
```

## MQTT Topics (for testing)

### Subscribe to all health messages
```bash
docker exec bigskies-mqtt mosquitto_sub -v -t 'bigskies/coordinator/+/health'
```

### Subscribe to bootstrap credentials
```bash
docker exec bigskies-mqtt mosquitto_sub -v -t 'bigskies/coordinator/bootstrap/credentials'
```

### Request credentials
```bash
docker exec bigskies-mqtt mosquitto_pub -t 'bigskies/coordinator/bootstrap/request' -m 'test'
```

## Troubleshooting

### Bootstrap coordinator won't start

**Check logs:**
```bash
docker logs bigskies-bootstrap
```

**Common issues:**
- `.pgpass` file missing: Run `./scripts/update-pgpass.sh`
- PostgreSQL not ready: Wait for health check, or check `docker logs bigskies-postgres`
- MQTT not ready: Check `docker logs bigskies-mqtt`

### Coordinators timeout waiting for credentials

**Check bootstrap is publishing:**
```bash
docker exec bigskies-mqtt mosquitto_sub -t 'bigskies/coordinator/bootstrap/credentials' -C 1
```

**Manually trigger republish:**
```bash
docker restart bigskies-bootstrap
```

### Database connection refused

**Check PostgreSQL is running:**
```bash
docker ps | grep postgres
docker logs bigskies-postgres
```

**Test connection:**
```bash
docker exec bigskies-postgres psql -U bigskies -d bigskies -c "SELECT version();"
```

### Need to reset everything

**Option 1: Using make docker-purge (recommended)**
```bash
make docker-purge       # Removes containers, volumes, images, and build cache
./scripts/update-pgpass.sh
make docker-build
make docker-up
```

**Option 2: Manual cleanup**
```bash
make docker-down
docker volume rm bigskies_postgres_data bigskies_mqtt_data bigskies_shared_secrets
./scripts/update-pgpass.sh
make docker-up
```

## Next Steps

1. **Review architecture**: `docs/architecture/COORDINATOR_ENGINE_ARCHITECTURE.md`
2. **Bootstrap setup**: `docs/setup/BOOTSTRAP_SETUP.md`
3. **Configure coordinators**: Edit configuration in PostgreSQL `coordinator_config` table
4. **Add plugins**: Follow plugin development guide (TBD)
5. **Connect ASCOM devices**: Configure telescope coordinator
6. **Build frontend**: Choose from Flutter, Unity, or Python GTK (on hold)

## Development Workflow

### Adding a new coordinator

1. Create `cmd/my-coordinator/main.go`
2. Use `BaseCoordinator.WaitForCredentials()` for database access
3. Add to `docker-compose.yml` with proper dependencies
4. Generate Dockerfile: Edit `scripts/generate-dockerfiles.sh`, run it
5. Build and test: `make docker-build && make docker-up`

### Modifying database schema

1. Create migration file: `configs/sql/my_migration.sql`
2. Add to `configs/bootstrap.yaml` migrations list
3. Restart bootstrap: `docker restart bigskies-bootstrap`
4. Check logs: `docker logs bigskies-bootstrap | grep migration`

### Testing MQTT messages

Use `mosquitto_pub` and `mosquitto_sub` inside the MQTT container:

```bash
# Subscribe
docker exec bigskies-mqtt mosquitto_sub -v -t 'bigskies/#'

# Publish
docker exec bigskies-mqtt mosquitto_pub -t 'bigskies/test' -m '{"hello":"world"}'
```

## Support

- Architecture guide: `docs/architecture/COORDINATOR_ENGINE_ARCHITECTURE.md`
- Bootstrap guide: `docs/setup/BOOTSTRAP_SETUP.md`
- Project README: `README.md`
- WARP guidelines: `WARP.md`

---

**Happy coding!** 🚀✨
