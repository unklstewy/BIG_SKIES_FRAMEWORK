# BIG SKIES Framework - Plugin SDK

## Overview

The BIG SKIES Framework uses a Docker-based plugin architecture where plugins are self-contained Docker containers that extend the framework's functionality. Plugins communicate with coordinators via MQTT messages and can provide additional services, device drivers, data processors, or UI elements.

## Architecture

### Plugin Model

```
┌─────────────────────────────────────┐
│      BIG SKIES Framework            │
│                                      │
│  ┌──────────────────────────────┐  │
│  │  Plugin Coordinator          │  │
│  │  - Plugin registry           │  │
│  │  - Lifecycle management      │  │
│  │  - Health monitoring         │  │
│  └──────────────────────────────┘  │
│              ↕ MQTT                 │
│  ┌──────────────────────────────┐  │
│  │  MQTT Broker (Mosquitto)     │  │
│  └──────────────────────────────┘  │
└─────────────────────────────────────┘
              ↕ MQTT
┌─────────────────────────────────────┐
│         Plugin Container             │
│                                      │
│  ┌──────────────────────────────┐  │
│  │  Plugin Service              │  │
│  │  - MQTT client               │  │
│  │  - Health reporting          │  │
│  │  - Business logic            │  │
│  └──────────────────────────────┘  │
│                                      │
│  ┌──────────────────────────────┐  │
│  │  Optional Components         │  │
│  │  - HTTP/REST API             │  │
│  │  - Database                  │  │
│  │  - External integrations     │  │
│  └──────────────────────────────┘  │
└─────────────────────────────────────┘
```

### Plugin Types

1. **Device Plugins** - Hardware device drivers and simulators
   - Example: ASCOM Alpaca Simulators, ADS-B receivers, weather stations

2. **Data Processing Plugins** - Data analysis and transformation
   - Example: Image processing, astrometry, plate solving

3. **Integration Plugins** - External service integration
   - Example: Cloud storage, notification services, databases

4. **UI Extension Plugins** - Additional UI components
   - Example: Custom dashboards, visualization tools

## Plugin Structure

### Required Files

```
my-plugin/
├── plugin.json          # Plugin manifest (REQUIRED)
├── Dockerfile          # Container definition (REQUIRED)
├── docker-compose.yml  # Optional: for multi-container plugins
├── README.md           # Plugin documentation
└── src/                # Plugin source code
    ├── main.go         # Or main.py, main.js, etc.
    └── ...
```

### Plugin Manifest (plugin.json)

```json
{
  "id": "unique-plugin-guid",
  "name": "My Plugin Name",
  "version": "1.0.0",
  "description": "Brief description of plugin functionality",
  "author": "Your Name or Organization",
  "license": "MIT",
  "homepage": "https://github.com/yourorg/your-plugin",
  
  "capabilities": {
    "provides_devices": false,
    "provides_ui": false,
    "requires_database": false,
    "requires_internet": false
  },
  
  "mqtt": {
    "topics_subscribe": [
      "bigskies/plugin/myplugin/#"
    ],
    "topics_publish": [
      "bigskies/plugin/myplugin/status",
      "bigskies/plugin/myplugin/data"
    ]
  },
  
  "docker": {
    "image": "my-plugin:latest",
    "ports": [
      "8080:8080"
    ],
    "environment": {
      "PLUGIN_CONFIG": "/config/plugin.conf",
      "LOG_LEVEL": "info"
    },
    "volumes": [
      "/var/lib/bigskies/plugins/myplugin:/data"
    ],
    "networks": [
      "bigskies-network"
    ]
  },
  
  "ui_elements": [
    {
      "id": "my-plugin-panel",
      "type": "panel",
      "title": "My Plugin Panel",
      "description": "Control panel for my plugin",
      "icon": "dashboard",
      "url": "/plugin/myplugin/panel"
    }
  ],
  
  "health_check": {
    "topic": "bigskies/plugin/myplugin/health",
    "interval_seconds": 30
  },
  
  "dependencies": {
    "coordinators": ["message", "security"],
    "min_framework_version": "1.0.0"
  }
}
```

## Development Requirements

### 1. MQTT Communication

All plugins MUST:
- Connect to MQTT broker at `mqtt-broker:1883` (inside Docker network)
- Publish health status every 30 seconds to `bigskies/plugin/{plugin-id}/health`
- Subscribe to control topics: `bigskies/plugin/{plugin-id}/control/#`

### 2. Health Reporting

Health status message format:
```json
{
  "id": "timestamp-id",
  "source": "plugin:{plugin-id}",
  "type": "status",
  "timestamp": "2026-01-16T00:00:00Z",
  "payload": {
    "component": "{plugin-name}",
    "status": "healthy|degraded|unhealthy",
    "message": "Status description",
    "details": {
      "running": true,
      "uptime_seconds": 12345,
      "custom_metrics": {}
    }
  }
}
```

### 3. Docker Requirements

- **Base Image**: Use official, minimal base images (Alpine, Distroless, etc.)
- **Size**: Keep images small (<500MB preferred)
- **Non-root User**: Run as non-root user inside container
- **Health Check**: Implement Docker HEALTHCHECK instruction
- **Logging**: Log to stdout/stderr for Docker log collection
- **Secrets**: Accept secrets via environment variables, never hardcode

### 4. Network Communication

- **MQTT**: Connect via `bigskies-network` Docker network
- **HTTP APIs**: Expose on documented ports, use reverse proxy if needed
- **Service Discovery**: Use DNS names within Docker network

## Plugin Lifecycle

### 1. Installation

```
POST bigskies/coordinator/plugin/install
{
  "plugin_id": "unique-plugin-guid",
  "source": "docker-registry/image:tag",
  "config": { ... }
}
```

Plugin Coordinator:
1. Downloads plugin manifest
2. Validates manifest schema
3. Pulls Docker image
4. Registers plugin in registry
5. Responds with success/failure

### 2. Starting

```
POST bigskies/coordinator/plugin/start
{
  "plugin_id": "unique-plugin-guid"
}
```

Plugin Coordinator:
1. Creates Docker container from image
2. Injects environment variables
3. Mounts volumes
4. Connects to bigskies-network
5. Starts container
6. Waits for health check
7. Responds when plugin is ready

### 3. Running

During operation:
- Plugin publishes health status every 30 seconds
- Plugin subscribes to control messages
- Plugin Coordinator monitors health
- Framework logs plugin activity

### 4. Stopping

```
POST bigskies/coordinator/plugin/stop
{
  "plugin_id": "unique-plugin-guid"
}
```

Plugin Coordinator:
1. Sends shutdown signal to container
2. Waits for graceful shutdown (30s timeout)
3. Force kills if necessary
4. Removes container (keeps image)
5. Responds with success/failure

### 5. Uninstallation

```
POST bigskies/coordinator/plugin/remove
{
  "plugin_id": "unique-plugin-guid"
}
```

Plugin Coordinator:
1. Stops plugin if running
2. Removes Docker image
3. Cleans up volumes (if specified)
4. Removes from registry
5. Responds with success/failure

## Security Considerations

### Authentication

Plugins requiring authentication should:
1. Subscribe to `bigskies/coordinator/security/plugin/{plugin-id}/token`
2. Receive JWT token from Security Coordinator
3. Include token in protected requests
4. Validate token expiration and refresh as needed

### Permissions

Plugins should request minimum necessary permissions:
- Read: View configurations and status
- Write: Modify configurations
- Control: Execute device commands
- Configure: Modify system settings

### Data Access

- **User Data**: Access only with explicit permission
- **Telescope Data**: Limited to authorized telescopes
- **System Data**: Read-only unless elevated permissions

## Best Practices

### 1. Configuration Management

- Use environment variables for configuration
- Support configuration file mounting
- Provide sensible defaults
- Document all configuration options

### 2. Error Handling

- Log errors with context
- Return meaningful error messages
- Implement retry logic for transient failures
- Report errors via health status

### 3. Resource Management

- Implement graceful shutdown (handle SIGTERM)
- Clean up resources on exit
- Monitor memory and CPU usage
- Implement rate limiting if exposing APIs

### 4. Logging

- Use structured logging (JSON preferred)
- Include correlation IDs for request tracing
- Log at appropriate levels (debug, info, warn, error)
- Avoid logging sensitive information

### 5. Testing

- Provide unit tests for business logic
- Include integration tests with MQTT broker
- Test Docker container builds
- Document manual testing procedures

## Example: Minimal Plugin in Go

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
    pluginID = "my-plugin-guid"
    pluginName = "my-plugin"
)

type HealthStatus struct {
    ID        string    `json:"id"`
    Source    string    `json:"source"`
    Type      string    `json:"type"`
    Timestamp time.Time `json:"timestamp"`
    Payload   struct {
        Component string `json:"component"`
        Status    string `json:"status"`
        Message   string `json:"message"`
        Details   map[string]interface{} `json:"details"`
    } `json:"payload"`
}

func main() {
    brokerURL := os.Getenv("MQTT_BROKER")
    if brokerURL == "" {
        brokerURL = "tcp://mqtt-broker:1883"
    }
    
    opts := mqtt.NewClientOptions()
    opts.AddBroker(brokerURL)
    opts.SetClientID(pluginID)
    opts.SetCleanSession(true)
    
    client := mqtt.NewClient(opts)
    if token := client.Connect(); token.Wait() && token.Error() != nil {
        log.Fatal("Failed to connect to MQTT:", token.Error())
    }
    
    log.Println("Plugin started, connected to MQTT")
    
    // Subscribe to control topics
    if token := client.Subscribe("bigskies/plugin/"+pluginID+"/control/#", 1, handleMessage); token.Wait() && token.Error() != nil {
        log.Fatal("Failed to subscribe:", token.Error())
    }
    
    // Start health reporting
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go publishHealth(ctx, client)
    
    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    
    log.Println("Shutting down...")
    client.Disconnect(250)
}

func publishHealth(ctx context.Context, client mqtt.Client) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            status := HealthStatus{
                ID:        time.Now().Format("20060102150405.000000"),
                Source:    "plugin:" + pluginID,
                Type:      "status",
                Timestamp: time.Now(),
            }
            status.Payload.Component = pluginName
            status.Payload.Status = "healthy"
            status.Payload.Message = "Plugin is operational"
            status.Payload.Details = map[string]interface{}{
                "running": true,
            }
            
            payload, _ := json.Marshal(status)
            client.Publish("bigskies/plugin/"+pluginID+"/health", 1, false, payload)
        }
    }
}

func handleMessage(client mqtt.Client, msg mqtt.Message) {
    log.Printf("Received message on %s: %s", msg.Topic(), msg.Payload())
    // Handle control messages
}
```

### Dockerfile for Go Plugin

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o plugin .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/plugin .
USER 1000:1000
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
ENTRYPOINT ["./plugin"]
```

## Publishing Plugins

### 1. Package Plugin

```bash
docker build -t my-plugin:1.0.0 .
docker tag my-plugin:1.0.0 registry.example.com/my-plugin:1.0.0
docker push registry.example.com/my-plugin:1.0.0
```

### 2. Create Release

Include in release:
- `plugin.json` manifest
- `README.md` documentation
- `CHANGELOG.md` version history
- Docker image reference
- Installation instructions

### 3. Register with Framework

Provide installation command:
```bash
# Via MQTT
mosquitto_pub -h mqtt-broker -t bigskies/coordinator/plugin/install \
  -m '{"plugin_id":"my-plugin-guid","source":"registry.example.com/my-plugin:1.0.0"}'
```

## Support and Resources

- **Framework Documentation**: `docs/`
- **Example Plugins**: `plugins/examples/`
- **MQTT Topics Reference**: `docs/architecture/mqtt_message_flows_gojs.json`
- **GitHub Issues**: Report bugs and request features
- **Developer Forum**: Community support and discussions

## License

Plugins can use any OSI-approved open source license or proprietary licenses. The license must be specified in `plugin.json` and included in the plugin distribution.
