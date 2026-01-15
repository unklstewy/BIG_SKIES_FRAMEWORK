# Coordinator Development Guide

This guide establishes the standard pattern for developing new coordinators in the BIG_SKIES_FRAMEWORK.

## MQTT Client Creation Pattern

All coordinators **MUST** use the centralized `CreateMQTTClient()` factory function to ensure consistency and maintainability.

### ✅ Correct Pattern

```go
package coordinators

import (
    "context"
    "fmt"
    "github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
    "go.uber.org/zap"
)

// ExampleCoordinator demonstrates the standard pattern
type ExampleCoordinator struct {
    *BaseCoordinator
    config *ExampleCoordinatorConfig
}

type ExampleCoordinatorConfig struct {
    BaseConfig
    BrokerURL string `json:"broker_url"`
    // ... other coordinator-specific config
}

// NewExampleCoordinator creates a new example coordinator instance
func NewExampleCoordinator(config *ExampleCoordinatorConfig, logger *zap.Logger) (*ExampleCoordinator, error) {
    if config == nil {
        return nil, fmt.Errorf("config cannot be nil")
    }
    
    // ✅ Use CreateMQTTClient factory function
    // This ensures consistent MQTT configuration across all coordinators
    mqttClient, err := CreateMQTTClient(config.BrokerURL, mqtt.CoordinatorExample, logger)
    if err != nil {
        return nil, fmt.Errorf("failed to create MQTT client: %w", err)
    }
    
    // Create base coordinator with the MQTT client
    base := NewBaseCoordinator(mqtt.CoordinatorExample, mqttClient, logger)
    
    ec := &ExampleCoordinator{
        BaseCoordinator: base,
        config:          config,
    }
    
    // Register health check
    ec.RegisterHealthCheck(ec)
    
    return ec, nil
}

// Start begins coordinator operations
func (ec *ExampleCoordinator) Start(ctx context.Context) error {
    // Start base coordinator
    if err := ec.BaseCoordinator.Start(ctx); err != nil {
        return err
    }
    
    // Subscribe to topics
    // ... coordinator-specific subscription logic
    
    // ✅ Always start health publishing as a goroutine
    go ec.StartHealthPublishing(ctx)
    
    ec.GetLogger().Info("Example coordinator started successfully")
    return nil
}

// ... rest of coordinator implementation
```

### ❌ Incorrect Pattern (DO NOT USE)

```go
// ❌ WRONG: Creating MQTT client directly in coordinator
mqttConfig := &mqtt.Config{
    BrokerURL:            brokerURL,
    ClientID:             "example-coordinator",  // ❌ Duplicates configuration
    KeepAlive:            30 * time.Second,       // ❌ Copy-paste prone
    ConnectTimeout:       10 * time.Second,       // ❌ Inconsistent if changed
    AutoReconnect:        true,                   // ❌ Harder to maintain
    MaxReconnectInterval: 5 * time.Minute,        // ❌ No single source of truth
}
mqttClient, err := mqtt.NewClient(mqttConfig, logger)
// ...
```

## CreateMQTTClient Factory Function

**Location:** `internal/coordinators/base.go`

**Signature:**
```go
func CreateMQTTClient(brokerURL, clientID string, logger *zap.Logger) (*mqtt.Client, error)
```

**Parameters:**
- `brokerURL`: MQTT broker URL (e.g., "localhost:1883" or "tcp://mqtt-broker:1883")
  - If empty, defaults to `"tcp://mqtt-broker:1883"`
- `clientID`: Unique identifier for the coordinator (use constants from `pkg/mqtt`)
- `logger`: zap logger instance for logging

**Returns:**
- Configured `*mqtt.Client` ready to use
- Error if client creation fails

**Configuration Defaults:**
- KeepAlive: 30 seconds
- ConnectTimeout: 10 seconds
- AutoReconnect: true
- MaxReconnectInterval: 5 minutes

**Example:**
```go
mqttClient, err := CreateMQTTClient("localhost:1883", mqtt.CoordinatorExample, logger)
if err != nil {
    return nil, fmt.Errorf("failed to create MQTT client: %w", err)
}
```

## Health Publishing

All coordinators inherit health publishing capability from `BaseCoordinator`. Ensure your `Start()` method includes:

```go
// Start health status publishing
go ec.StartHealthPublishing(ctx)
```

This will:
- Publish health status every 30 seconds
- Use the coordinator's `HealthCheck()` method to get status
- Automatically handle context cancellation for graceful shutdown

## Configuration Constants

Use these constants from `pkg/mqtt` for `clientID`:
- `mqtt.CoordinatorMessage` = "message"
- `mqtt.CoordinatorSecurity` = "security"
- `mqtt.CoordinatorDataStore` = "datastore"
- `mqtt.CoordinatorApplication` = "application"
- `mqtt.CoordinatorPlugin` = "plugin"
- `mqtt.CoordinatorTelescope` = "telescope"
- `mqtt.CoordinatorUIElement` = "uielement"

## Benefits of This Pattern

✅ **Consistency** - All coordinators use identical MQTT configuration  
✅ **Maintainability** - Single source of truth for MQTT defaults  
✅ **Scalability** - Easy to add new coordinators with minimal code  
✅ **Testability** - Configuration can be easily mocked/overridden in tests  
✅ **Reliability** - Changes to MQTT configuration apply globally  

## Testing

When writing tests for new coordinators:

1. Do **NOT** create real MQTT clients
2. Use mock clients that implement the same interface
3. Focus on coordinator-specific logic, not MQTT communication

Example from existing tests:
```go
type mockMQTTClient struct {
    // ... test double implementation
}

func TestNewCoordinator(t *testing.T) {
    mockClient := newMockMQTTClient()
    // ... test using mock client
}
```

## Summary

**The standard pattern for all new coordinators:**

1. Define config struct inheriting from `BaseConfig`
2. In `New*Coordinator()`:
   - Validate config
   - Call `CreateMQTTClient(config.BrokerURL, mqtt.CoordinatorName, logger)`
   - Call `NewBaseCoordinator(mqtt.CoordinatorName, mqttClient, logger)`
3. In `Start()`:
   - Call `BaseCoordinator.Start(ctx)`
   - Subscribe to topics
   - `go ec.StartHealthPublishing(ctx)`
4. Implement `Check()` for health checks
5. Never call `mqtt.NewClient()` directly

This ensures code reuse, consistency, and maintainability across the framework.
