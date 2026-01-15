# BIG SKIES Framework Deployment

This directory contains deployment configurations for the BIG SKIES Framework.

## Structure
- `docker/` - Dockerfiles for coordinators
- `docker-compose/` - Docker Compose orchestration
- `kubernetes/` - Kubernetes manifests (future)

## Docker Deployment

### Quick Start

```bash
cd deployments/docker-compose
cp .env.example .env
# Edit .env with your configuration
docker-compose up -d
```

### Services

- **mqtt-broker**: Eclipse Mosquitto (ports 1883, 9001)
- **postgres**: PostgreSQL 16 (port 5432)
- **message-coordinator**: Message bus management
- **datastore-coordinator**: Database management  
- **application-coordinator**: Service registry
- **plugin-coordinator**: Plugin lifecycle
- **uielement-coordinator**: UI element tracking

### Management

```bash
# View status
docker-compose ps

# View logs
docker-compose logs -f [service-name]

# Restart service
docker-compose restart [service-name]

# Stop all
docker-compose down

# Clean volumes
docker-compose down -v
```

See full documentation in this directory for detailed configuration and troubleshooting.
