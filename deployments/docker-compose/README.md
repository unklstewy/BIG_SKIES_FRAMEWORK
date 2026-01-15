# BIG SKIES Framework - Docker Compose Deployment

This directory contains the Docker Compose configuration for deploying the BIG SKIES Framework.

## Quick Start

```bash
# Start all services
docker compose up -d

# View logs
docker compose logs -f

# Stop all services
docker compose down

# Stop and remove volumes (WARNING: deletes all data)
docker compose down -v
```

## Services

The deployment includes the following services:

### Infrastructure Services
- **postgres**: PostgreSQL 16 database with automatic schema initialization
- **mqtt-broker**: Eclipse Mosquitto MQTT broker (ports 1883, 9001)

### Coordinator Services
- **message-coordinator**: Message bus and health monitoring
- **datastore-coordinator**: Database management
- **application-coordinator**: Application service tracking
- **plugin-coordinator**: Plugin lifecycle management
- **uielement-coordinator**: UI element provisioning
- **security-coordinator**: Security, authentication, and TLS management

## Database Initialization

The PostgreSQL database is automatically initialized on first startup with:
- Security schema (users, roles, groups, permissions, TLS certificates)
- Default admin user (username: `admin`, password: `bigskies_admin_2024`)
- Default roles: admin, operator, observer, developer
- Default groups: administrators, operators, observers

**IMPORTANT**: Change the default admin password immediately in production!

### How It Works

SQL schema files in `../../configs/sql/` are automatically mounted into the PostgreSQL container's `/docker-entrypoint-initdb.d/` directory. PostgreSQL runs these scripts automatically when initializing a fresh database.

The initialization only runs when the database is created for the first time. To reinitialize:

```bash
# Stop postgres and remove its volume
docker compose down postgres
docker volume rm docker-compose_postgres-data

# Start postgres again (will run initialization)
docker compose up -d postgres
```

## Environment Variables

Create a `.env` file in this directory to customize settings:

```bash
# PostgreSQL
POSTGRES_PASSWORD=your_secure_password

# Security
JWT_SECRET=your_jwt_secret_here

# Logging
LOG_LEVEL=info  # Options: debug, info, warn, error
```

## Network Configuration

All services communicate on the `bigskies-network` bridge network. External access:
- PostgreSQL: `localhost:5432`
- MQTT TCP: `localhost:1883`
- MQTT WebSocket: `localhost:9001`

## Volumes

Persistent data is stored in the following Docker volumes:
- `postgres-data`: Database files
- `mqtt-data`: MQTT broker persistence
- `mqtt-logs`: MQTT broker logs
- `plugin-data`: Installed plugins
- `cert-data`: TLS/SSL certificates

## Health Checks

All coordinator services include health checks that verify:
- Service binary is executable
- Dependencies are accessible

PostgreSQL includes a health check using `pg_isready`.

Services that depend on PostgreSQL wait for it to be healthy before starting.

## Development

### Building Services

```bash
# Build all services
docker compose build

# Build specific service
docker compose build security-coordinator

# Build without cache
docker compose build --no-cache
```

### Viewing Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f security-coordinator

# Last 100 lines
docker compose logs --tail=100 security-coordinator
```

### Testing Authentication

```bash
# Subscribe to responses
mosquitto_sub -h localhost -p 1883 -t "bigskies/coordinator/security/response/#" -v

# In another terminal, send login request
mosquitto_pub -h localhost -p 1883 \
  -t "bigskies/coordinator/security/auth/login" \
  -m '{"username":"admin","password":"bigskies_admin_2024"}'
```

## Troubleshooting

### Database Connection Issues

```bash
# Check if postgres is healthy
docker compose ps postgres

# View postgres logs
docker compose logs postgres

# Connect to postgres directly
docker exec -it bigskies-postgres psql -U bigskies -d bigskies
```

### MQTT Connection Issues

```bash
# Check mqtt broker status
docker compose ps mqtt-broker

# Test MQTT connectivity
mosquitto_pub -h localhost -p 1883 -t "test" -m "hello"
```

### Container Health Issues

```bash
# Check health status
docker ps --filter "name=bigskies-" --format "table {{.Names}}\t{{.Status}}"

# Inspect health check details
docker inspect bigskies-security-coordinator --format='{{json .State.Health}}' | python3 -m json.tool
```

## Production Considerations

Before deploying to production:

1. **Change default passwords**: Update `POSTGRES_PASSWORD` and admin user password
2. **Change JWT secret**: Set a strong `JWT_SECRET` value
3. **Use TLS**: Configure TLS for PostgreSQL and MQTT
4. **Backup strategy**: Implement regular backups of `postgres-data` volume
5. **Resource limits**: Add CPU and memory limits to services
6. **Monitoring**: Set up logging and monitoring infrastructure
7. **Network security**: Restrict port exposure and use firewalls
