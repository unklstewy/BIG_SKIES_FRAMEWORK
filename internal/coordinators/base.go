// Package coordinators provides base coordinator implementation.
package coordinators

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/api"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/healthcheck"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
	"go.uber.org/zap"
)

// BaseCoordinator provides common functionality for all coordinators.
type BaseCoordinator struct {
	name          string
	mqttClient    *mqtt.Client
	healthEngine  *healthcheck.Engine
	logger        *zap.Logger
	config        interface{}
	running       bool
	mu            sync.RWMutex
	startTime     time.Time
	shutdownFuncs []func(context.Context) error
}

// BaseConfig holds common configuration for coordinators.
type BaseConfig struct {
	// Name is the coordinator instance name
	Name string `json:"name"`
	// MQTTConfig for message bus connection
	MQTTConfig *mqtt.Config `json:"mqtt"`
	// HealthCheckInterval for periodic health checks
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	// LogLevel for the coordinator
	LogLevel string `json:"log_level"`
}

// NewBaseCoordinator creates a new base coordinator instance.
func NewBaseCoordinator(name string, mqttClient *mqtt.Client, logger *zap.Logger) *BaseCoordinator {
	if logger == nil {
		logger = zap.NewNop()
	}

	healthEngine := healthcheck.NewEngine(logger, 3*time.Second)

	return &BaseCoordinator{
		name:          name,
		mqttClient:    mqttClient,
		healthEngine:  healthEngine,
		logger:        logger.With(zap.String("coordinator", name)),
		shutdownFuncs: make([]func(context.Context) error, 0),
	}
}

// Name returns the coordinator name.
func (bc *BaseCoordinator) Name() string {
	return bc.name
}

// IsRunning returns true if the coordinator is running.
func (bc *BaseCoordinator) IsRunning() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.running
}

// setRunning updates the running state.
func (bc *BaseCoordinator) setRunning(running bool) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.running = running
	if running {
		bc.startTime = time.Now()
	}
}

// Start begins coordinator operations.
func (bc *BaseCoordinator) Start(ctx context.Context) error {
	if bc.IsRunning() {
		return fmt.Errorf("coordinator %s is already running", bc.name)
	}

	bc.logger.Info("Starting coordinator")

	// Connect MQTT if available
	if bc.mqttClient != nil && !bc.mqttClient.IsConnected() {
		if err := bc.mqttClient.Connect(); err != nil {
			return fmt.Errorf("failed to connect MQTT: %w", err)
		}
	}

	// Start health check engine
	go bc.healthEngine.Start(ctx)

	bc.setRunning(true)
	bc.logger.Info("Coordinator started successfully")

	return nil
}

// Stop shuts down the coordinator.
func (bc *BaseCoordinator) Stop(ctx context.Context) error {
	if !bc.IsRunning() {
		return nil
	}

	bc.logger.Info("Stopping coordinator")

	// Execute shutdown functions in reverse order
	for i := len(bc.shutdownFuncs) - 1; i >= 0; i-- {
		if err := bc.shutdownFuncs[i](ctx); err != nil {
			bc.logger.Error("Shutdown function failed", zap.Error(err))
		}
	}

	// Stop health engine
	bc.healthEngine.Stop()

	// Disconnect MQTT
	if bc.mqttClient != nil && bc.mqttClient.IsConnected() {
		bc.mqttClient.Disconnect()
	}

	bc.setRunning(false)
	bc.logger.Info("Coordinator stopped")

	return nil
}

// HealthCheck returns the coordinator's health status.
func (bc *BaseCoordinator) HealthCheck(ctx context.Context) *healthcheck.Result {
	status := healthcheck.StatusHealthy
	message := "Coordinator is healthy"

	if !bc.IsRunning() {
		status = healthcheck.StatusUnhealthy
		message = "Coordinator is not running"
	} else if bc.mqttClient != nil && !bc.mqttClient.IsConnected() {
		status = healthcheck.StatusDegraded
		message = "MQTT client not connected"
	}

	uptime := time.Since(bc.startTime)

	return &healthcheck.Result{
		ComponentName: bc.name,
		Status:        status,
		Message:       message,
		Timestamp:     time.Now(),
		Details: map[string]interface{}{
			"uptime_seconds": uptime.Seconds(),
			"running":        bc.IsRunning(),
			"mqtt_connected": bc.mqttClient != nil && bc.mqttClient.IsConnected(),
		},
	}
}

// RegisterShutdownFunc adds a function to be called during shutdown.
func (bc *BaseCoordinator) RegisterShutdownFunc(fn func(context.Context) error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.shutdownFuncs = append(bc.shutdownFuncs, fn)
}

// RegisterHealthCheck adds a health checker to the coordinator.
func (bc *BaseCoordinator) RegisterHealthCheck(checker healthcheck.Checker) {
	bc.healthEngine.Register(checker)
}

// GetHealthEngine returns the health check engine.
func (bc *BaseCoordinator) GetHealthEngine() *healthcheck.Engine {
	return bc.healthEngine
}

// GetMQTTClient returns the MQTT client.
func (bc *BaseCoordinator) GetMQTTClient() *mqtt.Client {
	return bc.mqttClient
}

// GetLogger returns the logger.
func (bc *BaseCoordinator) GetLogger() *zap.Logger {
	return bc.logger
}

// LoadConfig loads configuration into the coordinator.
func (bc *BaseCoordinator) LoadConfig(config interface{}) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.config = config
	return nil
}

// ValidateConfig validates the current configuration.
func (bc *BaseCoordinator) ValidateConfig() error {
	// Base validation - override in specific coordinators
	return nil
}

// GetConfig returns the current configuration.
func (bc *BaseCoordinator) GetConfig() interface{} {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.config
}

// StartHealthPublishing starts periodic health status publishing to MQTT.
// This should be called in the coordinator's Start() method as a goroutine.
func (bc *BaseCoordinator) StartHealthPublishing(ctx context.Context) {
	if bc.mqttClient == nil {
		bc.logger.Warn("Cannot publish health: MQTT client is nil")
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Small delay to allow MQTT subscriptions to be established
	time.Sleep(500 * time.Millisecond)

	// Publish initial health
	bc.publishHealth(ctx)

	for {
		select {
		case <-ctx.Done():
			bc.logger.Debug("Health publishing stopped")
			return
		case <-ticker.C:
			bc.publishHealth(ctx)
		}
	}
}

// publishHealth publishes a single health status update.
func (bc *BaseCoordinator) publishHealth(ctx context.Context) {
	if bc.mqttClient == nil {
		bc.logger.Debug("Skipping health publish: MQTT client is nil")
		return
	}

	health := bc.HealthCheck(ctx)
	topic := mqtt.CoordinatorHealthTopic(bc.name)

	// Wrap health result in MQTT message envelope
	msg, err := mqtt.NewMessage(mqtt.MessageTypeStatus, "coordinator:"+bc.name, health)
	if err != nil {
		bc.logger.Error("Failed to create health message",
			zap.Error(err))
		return
	}

	if err := bc.mqttClient.PublishJSON(topic, 1, false, msg); err != nil {
		bc.logger.Error("Failed to publish health status",
			zap.String("topic", topic),
			zap.Error(err))
	}
}

// CreateMQTTClient creates and configures an MQTT client for a coordinator.
// This centralizes MQTT configuration to ensure consistency across all coordinators.
func CreateMQTTClient(brokerURL, clientID string, logger *zap.Logger) (*mqtt.Client, error) {
	if brokerURL == "" {
		brokerURL = "tcp://mqtt-broker:1883"
	}

	mqttConfig := &mqtt.Config{
		BrokerURL:            brokerURL,
		ClientID:             clientID,
		KeepAlive:            30 * time.Second,
		ConnectTimeout:       10 * time.Second,
		AutoReconnect:        true,
		MaxReconnectInterval: 5 * time.Minute,
	}

	return mqtt.NewClient(mqttConfig, logger)
}

// Verify BaseCoordinator implements interfaces
var _ api.Coordinator = (*BaseCoordinator)(nil)
var _ api.Configurable = (*BaseCoordinator)(nil)
