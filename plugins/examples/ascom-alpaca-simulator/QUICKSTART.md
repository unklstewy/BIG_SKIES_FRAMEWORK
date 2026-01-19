# ASCOM Alpaca Simulator Plugin - Quick Start

## Build and Run

### Option 1: From Project Root (Recommended)

```bash
# Build
make plugin-ascom-build

# Start
make plugin-ascom-up

# Stop
make plugin-ascom-down

# View logs
make plugin-ascom-logs
```

### Option 2: From Plugin Directory

```bash
cd plugins/examples/ascom-alpaca-simulator

# Build
docker-compose build

# Start
docker-compose up -d

# Stop
docker-compose down

# View logs
docker-compose logs -f
```

## Access

- **Web UI**: http://localhost:32323
- **API Docs**: http://localhost:32323/swagger
- **API Base**: http://localhost:32323/api/v1

## Verify Plugin is Running

```bash
# Check health
curl http://localhost:32323/api/v1/management/apiversions

# Check container status
docker ps | grep ascom-alpaca-simulator
```

## Network Requirements

The plugin connects to the main BIG SKIES Framework via Docker network:
- **Network name**: `big_skies_framework_bigskies`
- **MQTT broker**: Accessible at `mqtt-broker:1883` on this network

Ensure the main framework is running first to create this network:
```bash
# From project root
make docker-up
```

## Troubleshooting

### Network not found
```
Error: network big_skies_framework_bigskies not found
```
**Solution**: Start the main framework first to create the network:
```bash
make docker-up
```

### Port already in use
```
Error: port 32323 already allocated
```
**Solution**: Another service is using port 32323. Stop it or change the port in `docker-compose.yml`:
```yaml
ports:
  - "33323:80"  # Use different external port
```

### MQTT connection failed
Check that MQTT broker is accessible:
```bash
docker exec bigskies-ascom-alpaca-simulator ping -c 3 mqtt-broker
```

## Load Telescope Configuration

### Via Script
```bash
cd plugins/examples/ascom-alpaca-simulator
./configs/deploy-config.sh s50 altaz
```

### Via MQTT
```bash
mosquitto_pub -h localhost -p 1883 \
  -t bigskies/plugin/f7e8d9c6-b5a4-3210-9876-543210fedcba/config/load \
  -m '{"command": "load_config", "model": "s50", "mount_type": "altaz"}'
```

See full documentation in [README.md](README.md)
