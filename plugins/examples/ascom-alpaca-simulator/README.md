# ASCOM Alpaca Simulators Plugin

## Overview

This plugin integrates the complete ASCOM Alpaca Simulators suite into the BIG SKIES Framework, providing realistic simulations of astronomy devices for testing and development.

## Features

- **10 Device Simulators**:
  - Telescope
  - Camera
  - Dome
  - Focuser
  - Filter Wheel
  - Rotator
  - Switch
  - Safety Monitor
  - Observing Conditions
  - Cover Calibrator

- **Web-based UI**: Blazor web interface for configuration and control
- **ASCOM Alpaca API**: Full REST API compliance with ASCOM Alpaca specification
- **Swagger Documentation**: Interactive API documentation at `/swagger`
- **MQTT Integration**: Health reporting and control via BIG SKIES MQTT bus
- **Discovery Support**: ASCOM Alpaca discovery protocol

## Installation

### Prerequisites

- BIG SKIES Framework running with Docker
- Access to bigskies-network Docker network
- MQTT broker available at `mqtt-broker:1883`

### Build the Plugin

From the plugin directory:

```bash
cd plugins/examples/ascom-alpaca-simulator
docker build -t bigskies/ascom-alpaca-simulators:latest .
```

### Install via Plugin Coordinator

Using MQTT:

```bash
mosquitto_pub -h mqtt-broker \
  -t bigskies/coordinator/plugin/install \
  -m '{
    "plugin_id": "f7e8d9c6-b5a4-3210-9876-543210fedcba",
    "source": "bigskies/ascom-alpaca-simulators:latest"
  }'
```

## Pre-configured Telescope Profiles

The plugin includes pre-configured profiles for **Seestar** smart telescopes with realistic specifications:

### Available Configurations

- **Seestar S30**: 30mm f/5 APO, 150mm FL, Sony IMX662 (1920×1080)
- **Seestar S30 Pro**: 30mm f/5.3 APO, 160mm FL, Sony IMX585 (3840×2160 4K)
- **Seestar S50**: 50mm f/5 APO, 250mm FL, Sony IMX462 (1920×1080)

Each model includes configurations for three mount types:
- **Alt/Az**: Standard altitude-azimuth mount
- **Equatorial**: Polar-aligned equatorial mount
- **German Equatorial**: GEM mount with meridian flip

Each configuration includes complete device profiles:
- **Telescope**: Accurate optical specifications
- **Camera**: Real sensor specifications (resolution, pixel size)
- **Filter Wheel**: 3-position (UV/IR Cut, Duo-Band Hα/OIII, Dark Field)
- **Focuser**: Motorized focuser (3000 steps)
- **Switch**: Dew heater control

### Using Seestar Configurations

Configurations are located in `configs/` directory:

```bash
# Example: Deploy Seestar S50 with Alt/Az mount
cd plugins/examples/ascom-alpaca-simulator
cp -r configs/s50/altaz/* /tmp/ascom-config/alpaca/ascom-alpaca-simulator/
docker restart ascom-alpaca-simulator

# Or use the deployment script
./configs/deploy-config.sh s50 altaz
```

See `configs/README.md` for complete documentation and API examples.

### Dynamic Configuration via MQTT

The plugin includes a **configuration service** that allows you to dynamically load telescope configurations via MQTT without manual file copying or container restarts.

#### Available Commands

**Load Configuration**:
```bash
mosquitto_pub -h mqtt-broker \
  -t bigskies/plugin/f7e8d9c6-b5a4-3210-9876-543210fedcba/config/load \
  -m '{
    "command": "load_config",
    "model": "s50",
    "mount_type": "altaz",
    "request_id": "req-001"
  }'
```

**List Available Configurations**:
```bash
mosquitto_pub -h mqtt-broker \
  -t bigskies/plugin/f7e8d9c6-b5a4-3210-9876-543210fedcba/config/list \
  -m '{"command": "list_configs"}'
```

**Get Current Status**:
```bash
mosquitto_pub -h mqtt-broker \
  -t bigskies/plugin/f7e8d9c6-b5a4-3210-9876-543210fedcba/config/status \
  -m '{"command": "get_status"}'
```

#### Subscribing to Responses and Events

To monitor configuration changes:

```bash
# Subscribe to responses
mosquitto_sub -h mqtt-broker \
  -t 'bigskies/plugin/f7e8d9c6-b5a4-3210-9876-543210fedcba/config/response' \
  -v

# Subscribe to events
mosquitto_sub -h mqtt-broker \
  -t 'bigskies/plugin/f7e8d9c6-b5a4-3210-9876-543210fedcba/config/event' \
  -v
```

#### Configuration Service Features

- **Automatic backup**: Previous configuration is backed up before loading new one
- **Validation**: Model and mount type are validated before loading
- **State tracking**: Current configuration is stored in `/tmp/ascom-current-config.json`
- **Event publishing**: Configuration changes are published to MQTT event topic
- **Error handling**: Detailed error messages for troubleshooting

#### Response Format

**Success Response**:
```json
{
  "request_id": "req-001",
  "command": "load_config",
  "success": true,
  "message": "Configuration loaded successfully",
  "data": {
    "model": "s50",
    "mount_type": "altaz",
    "loaded_at": "2026-01-16T02:00:00Z",
    "loaded_by": "mqtt:req-001",
    "config_path": "/app/configs/s50/altaz"
  },
  "timestamp": "2026-01-16T02:00:00Z"
}
```

**Error Response**:
```json
{
  "request_id": "req-001",
  "command": "load_config",
  "success": false,
  "message": "Invalid model: s60",
  "timestamp": "2026-01-16T02:00:00Z"
}
```

## Configuration

### Environment Variables

- `PLUGIN_ID` - Plugin identifier (default: `f7e8d9c6-b5a4-3210-9876-543210fedcba`)
- `PLUGIN_NAME` - Plugin display name (default: `ASCOM Alpaca Simulators`)
- `MQTT_BROKER` - MQTT broker URL (default: `tcp://mqtt-broker:1883`)
- `LOG_LEVEL` - Logging level: debug, info, warn, error (default: `info`)
- `ASPNETCORE_URLS` - ASCOM server bind address (default: `http://+:80`)

### Ports

- **32323** - HTTP port for ASCOM Alpaca API and Web UI
- **32227** - UDP port for ASCOM discovery (optional)

### Volumes

No persistent volumes required. All configuration stored in memory.

## Usage

### Access Web UI

Once the plugin is running:

```
http://localhost:32323
```

Features:
- Device configuration pages
- Control panels for each simulator
- Setup wizards
- Real-time device status

### Access Swagger API Documentation

```
http://localhost:32323/swagger
```

### ASCOM Alpaca API Endpoints

Base URL: `http://localhost:32323/api/v1`

Examples:

```bash
# Get API versions
curl http://localhost:32323/api/v1/management/apiversions

# Get telescope information
curl http://localhost:32323/api/v1/telescope/0/description

# Get telescope position
curl http://localhost:32323/api/v1/telescope/0/rightascension
curl http://localhost:32323/api/v1/telescope/0/declination

# Slew telescope
curl -X PUT http://localhost:32323/api/v1/telescope/0/slewtocoordinatesasync \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "RightAscension=12.5&Declination=45.0&ClientID=1&ClientTransactionID=1"
```

### Integration with Telescope Coordinator

The BIG SKIES Telescope Coordinator can discover and control these simulators:

```bash
# Discover devices
mosquitto_pub -h mqtt-broker \
  -t bigskies/coordinator/telescope/device/discover \
  -m '{"port": 32323}'

# Connect to telescope
mosquitto_pub -h mqtt-broker \
  -t bigskies/coordinator/telescope/device/connect \
  -m '{
    "device_id": "ascom-telescope-0",
    "device_type": "telescope",
    "device_number": 0,
    "server_url": "http://localhost:32323"
  }'
```

## MQTT Topics

### Published Topics

- `bigskies/plugin/f7e8d9c6-b5a4-3210-9876-543210fedcba/health` - Health status (every 30s)
- `bigskies/plugin/f7e8d9c6-b5a4-3210-9876-543210fedcba/status` - Plugin status updates
- `bigskies/plugin/f7e8d9c6-b5a4-3210-9876-543210fedcba/discovery` - Device discovery events

### Subscribed Topics

- `bigskies/plugin/f7e8d9c6-b5a4-3210-9876-543210fedcba/control/#` - Control commands

## Health Monitoring

The plugin reports health status every 30 seconds via MQTT.

Health Status Values:
- **healthy** - Plugin operational, ASCOM API responding
- **degraded** - Plugin operational, but ASCOM API not responding
- **unhealthy** - Plugin shutting down or failed

Example health message:

```json
{
  "id": "20260116010530.123456",
  "source": "plugin:f7e8d9c6-b5a4-3210-9876-543210fedcba",
  "type": "status",
  "timestamp": "2026-01-16T01:05:30Z",
  "payload": {
    "component": "ASCOM Alpaca Simulators",
    "status": "healthy",
    "message": "Plugin operational, ASCOM API responding",
    "details": {
      "running": true,
      "uptime_seconds": 3600.5,
      "ascom_api_url": "http://localhost/api/v1/management/apiversions"
    }
  }
}
```

## Architecture

```
┌────────────────────────────────────┐
│  ASCOM Alpaca Simulators Plugin   │
│                                     │
│  ┌──────────────────────────────┐ │
│  │  ASCOM Alpaca Simulators     │ │
│  │  (.NET 8 / ASP.NET Core)     │ │
│  │  - Blazor Web UI             │ │
│  │  - REST API                  │ │
│  │  - 10 Device Simulators      │ │
│  └──────────────────────────────┘ │
│              │                      │
│  ┌──────────────────────────────┐ │
│  │  Health Reporter (Go)        │ │
│  │  - MQTT client               │ │
│  │  - Health monitoring         │ │
│  │  - Control handler           │ │
│  └──────────────────────────────┘ │
│              │                      │
│  ┌──────────────────────────────┐ │
│  │  Config Service (Go)         │ │
│  │  - MQTT client               │ │
│  │  - Configuration loading     │ │
│  │  - File management           │ │
│  │  - State tracking            │ │
│  └──────────────────────────────┘ │
└────────────────────────────────────┘
           │
           ↓ MQTT
┌────────────────────────────────────┐
│    BIG SKIES Framework             │
│  - MQTT Broker                     │
│  - Plugin Coordinator              │
│  - Telescope Coordinator           │
└────────────────────────────────────┘
```

## Development

### Project Structure

```
ascom-alpaca-simulator/
├── plugin.json              # Plugin manifest
├── Dockerfile              # Multi-stage build
├── start.sh                # Startup script
├── README.md               # This file
├── configs/                # Pre-configured telescope profiles
│   ├── s30/                # Seestar S30 configurations
│   ├── s30-pro/            # Seestar S30 Pro configurations
│   └── s50/                # Seestar S50 configurations
├── health-reporter/        # Go health reporter
│   ├── main.go             # Health reporter code
│   └── go.mod              # Go dependencies
└── config-service/         # Go configuration service
    ├── main.go             # Service entry point
    ├── models.go           # Data structures
    ├── loader.go           # File operations
    ├── handler.go          # MQTT handlers
    └── go.mod              # Go dependencies
```

### Building Locally

```bash
# Build the Docker image
docker build -t bigskies/ascom-alpaca-simulators:latest .

# Run locally (outside framework)
docker run -it --rm \
  -p 32323:80 \
  -e MQTT_BROKER=tcp://host.docker.internal:1883 \
  bigskies/ascom-alpaca-simulators:latest

# Run within framework network
docker run -it --rm \
  --network bigskies-network \
  -p 32323:80 \
  bigskies/ascom-alpaca-simulators:latest
```

### Testing

```bash
# Test ASCOM API
curl http://localhost:32323/api/v1/management/apiversions

# Expected response:
{
  "Value": [1],
  "ErrorNumber": 0,
  "ErrorMessage": "",
  "ClientTransactionID": 0,
  "ServerTransactionID": 1
}

# Test health endpoint
curl http://localhost:32323/api/v1/management/apiversions

# Monitor MQTT health messages
mosquitto_sub -h mqtt-broker \
  -t 'bigskies/plugin/f7e8d9c6-b5a4-3210-9876-543210fedcba/health' \
  -v
```

## Troubleshooting

### Plugin not starting

Check logs:
```bash
docker logs bigskies-ascom-alpaca-simulators
```

### MQTT connection issues

Verify MQTT broker is accessible:
```bash
docker exec bigskies-ascom-alpaca-simulators ping mqtt-broker
```

### ASCOM API not responding

Check if ASCOM server is running:
```bash
docker exec bigskies-ascom-alpaca-simulators \
  curl http://localhost/api/v1/management/apiversions
```

### Port conflicts

Ensure port 32323 is not in use:
```bash
lsof -i :32323
```

### Configuration service issues

Check if config service is running:
```bash
docker exec bigskies-ascom-alpaca-simulators ps aux | grep config-service
```

View config service logs:
```bash
docker logs bigskies-ascom-alpaca-simulators 2>&1 | grep '\[Config\]'
```

Check current configuration state:
```bash
docker exec bigskies-ascom-alpaca-simulators cat /tmp/ascom-current-config.json
```

## Credits

- **ASCOM Alpaca Simulators**: [Daniel Van Noord](https://github.com/DanielVanNoord/ASCOM.Alpaca.Simulators)
- **ASCOM Initiative**: [ASCOM Standards](https://ascom-standards.org/)
- **BIG SKIES Framework**: Plugin integration and health reporting

## License

MIT License - See LICENSE file for details

The ASCOM Alpaca Simulators are licensed under their own MIT license.

## Support

- **Plugin Issues**: Open an issue in the BIG SKIES Framework repository
- **ASCOM Simulator Issues**: Report to [ASCOM.Alpaca.Simulators](https://github.com/DanielVanNoord/ASCOM.Alpaca.Simulators/issues)
- **Documentation**: See `docs/plugins/PLUGIN_SDK.md` for plugin development guide

## Version History

### 1.0.0 (2026-01-16)
- Initial release
- Integration with BIG SKIES Framework
- All 10 ASCOM device simulators
- MQTT health reporting
- Web UI access
- Swagger API documentation
