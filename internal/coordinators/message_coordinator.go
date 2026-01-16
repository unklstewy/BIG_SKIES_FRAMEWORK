// Package coordinators implements the message coordinator.
package coordinators

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/config"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/healthcheck"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
	"go.uber.org/zap"
)

// MessageCoordinator manages the MQTT message bus and health monitoring.
type MessageCoordinator struct {
	*BaseCoordinator
	config           *MessageCoordinatorConfig
	subscribedTopics map[string]bool
	configLoader     *config.Loader // Database configuration loader
	mu               sync.RWMutex
}

// MessageCoordinatorConfig holds configuration for the message coordinator.
type MessageCoordinatorConfig struct {
	BaseConfig
	// BrokerURL is the MQTT broker address
	BrokerURL string `json:"broker_url"`
	// BrokerPort is the MQTT broker port
	BrokerPort int `json:"broker_port"`
	// MonitorInterval for health checks
	MonitorInterval time.Duration `json:"monitor_interval"`
	// MaxReconnectAttempts before declaring unhealthy
	MaxReconnectAttempts int `json:"max_reconnect_attempts"`
}

// NewMessageCoordinator creates a new message coordinator instance.
func NewMessageCoordinator(config *MessageCoordinatorConfig, logger *zap.Logger) (*MessageCoordinator, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Create MQTT client
	brokerURL := fmt.Sprintf("%s:%d", config.BrokerURL, config.BrokerPort)
	mqttClient, err := CreateMQTTClient(brokerURL, mqtt.CoordinatorMessage, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MQTT client: %w", err)
	}

	base := NewBaseCoordinator(mqtt.CoordinatorMessage, mqttClient, logger)

	mc := &MessageCoordinator{
		BaseCoordinator:  base,
		config:           config,
		subscribedTopics: make(map[string]bool),
	}

	// Register self health check
	mc.RegisterHealthCheck(mc)

	return mc, nil
}

// Start begins message coordinator operations.
func (mc *MessageCoordinator) Start(ctx context.Context) error {
	mc.GetLogger().Info("Starting message coordinator",
		zap.String("broker", mc.config.BrokerURL),
		zap.Int("port", mc.config.BrokerPort))

	// Start base coordinator
	if err := mc.BaseCoordinator.Start(ctx); err != nil {
		return err
	}

	// Subscribe to coordinator health topics
	if err := mc.subscribeHealthTopics(); err != nil {
		return fmt.Errorf("failed to subscribe to health topics: %w", err)
	}

	// Subscribe to configuration update topic
	if err := mc.subscribeConfigTopic(); err != nil {
		return fmt.Errorf("failed to subscribe to config topic: %w", err)
	}

	// Start health status publishing
	go mc.StartHealthPublishing(ctx)

	mc.GetLogger().Info("Message coordinator started successfully")
	return nil
}

// Stop shuts down the message coordinator.
func (mc *MessageCoordinator) Stop(ctx context.Context) error {
	mc.GetLogger().Info("Stopping message coordinator")

	// Unsubscribe from all topics
	mc.mu.RLock()
	topics := make([]string, 0, len(mc.subscribedTopics))
	for topic := range mc.subscribedTopics {
		topics = append(topics, topic)
	}
	mc.mu.RUnlock()

	for _, topic := range topics {
		if err := mc.GetMQTTClient().Unsubscribe(topic); err != nil {
			mc.GetLogger().Warn("Failed to unsubscribe", zap.String("topic", topic), zap.Error(err))
		}
	}

	// Stop base coordinator
	return mc.BaseCoordinator.Stop(ctx)
}

// subscribeHealthTopics subscribes to health check topics from all coordinators.
func (mc *MessageCoordinator) subscribeHealthTopics() error {
	coordinators := []string{
		mqtt.CoordinatorMessage,
		mqtt.CoordinatorSecurity,
		mqtt.CoordinatorDataStore,
		mqtt.CoordinatorApplication,
		mqtt.CoordinatorPlugin,
		mqtt.CoordinatorTelescope,
		mqtt.CoordinatorUIElement,
	}

	for _, coord := range coordinators {
		topic := mqtt.CoordinatorHealthTopic(coord)
		if err := mc.subscribe(topic, mc.handleHealthMessage); err != nil {
			return err
		}
	}

	return nil
}

// subscribe subscribes to a topic and tracks it.
func (mc *MessageCoordinator) subscribe(topic string, handler mqtt.MessageHandler) error {
	if err := mc.GetMQTTClient().Subscribe(topic, 1, handler); err != nil {
		return err
	}

	mc.mu.Lock()
	mc.subscribedTopics[topic] = true
	mc.mu.Unlock()

	return nil
}

// handleHealthMessage processes health check messages from coordinators.
func (mc *MessageCoordinator) handleHealthMessage(topic string, payload []byte) error {
	mc.GetLogger().Debug("Received health message",
		zap.String("topic", topic),
		zap.Int("size", len(payload)))

	// First unmarshal the entire MQTT message envelope
	var msg mqtt.Message
	if err := json.Unmarshal(payload, &msg); err != nil {
		mc.GetLogger().Error("Failed to unmarshal health message envelope", zap.Error(err))
		return err
	}

	// Then unmarshal the health check result from the payload
	var health healthcheck.Result
	if err := msg.UnmarshalPayload(&health); err != nil {
		mc.GetLogger().Error("Failed to unmarshal health payload", zap.Error(err))
		return err
	}

	mc.GetLogger().Debug("Processed health message",
		zap.String("component", health.ComponentName),
		zap.String("status", string(health.Status)))

	// TODO: Store and aggregate health data
	return nil
}

// PublishHealth publishes health check results to the message bus.
func (mc *MessageCoordinator) PublishHealth(ctx context.Context) error {
	result := mc.HealthCheck(ctx)

	topic := mqtt.CoordinatorHealthTopic(mqtt.CoordinatorMessage)
	msg, err := mqtt.NewMessage(mqtt.MessageTypeStatus, "coordinator:message", result)
	if err != nil {
		return fmt.Errorf("failed to create health message: %w", err)
	}

	return mc.GetMQTTClient().PublishJSON(topic, 1, false, msg)
}

// Check implements healthcheck.Checker interface.
func (mc *MessageCoordinator) Check(ctx context.Context) *healthcheck.Result {
	status := healthcheck.StatusHealthy
	message := "Message coordinator is healthy"
	details := make(map[string]interface{})

	// Check MQTT connection
	mqttClient := mc.GetMQTTClient()
	if mqttClient == nil || !mqttClient.IsConnected() {
		status = healthcheck.StatusUnhealthy
		message = "MQTT client not connected"
		details["mqtt_connected"] = false
	} else {
		details["mqtt_connected"] = true
	}

	// Check subscribed topics
	mc.mu.RLock()
	topicCount := len(mc.subscribedTopics)
	mc.mu.RUnlock()

	details["subscribed_topics"] = topicCount

	if topicCount == 0 {
		status = healthcheck.StatusDegraded
		message = "No topics subscribed"
	}

	return &healthcheck.Result{
		ComponentName: "message-coordinator",
		Status:        status,
		Message:       message,
		Timestamp:     time.Now(),
		Details:       details,
	}
}

// SetConfigLoader sets the configuration loader for runtime config updates.
func (mc *MessageCoordinator) SetConfigLoader(loader *config.Loader) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.configLoader = loader
}

// subscribeConfigTopic subscribes to configuration update messages.
func (mc *MessageCoordinator) subscribeConfigTopic() error {
	topic := "bigskies/coordinator/config/update/message-coordinator"
	return mc.subscribe(topic, mc.handleConfigUpdate)
}

// handleConfigUpdate processes runtime configuration update messages.
//
// Expected message payload:
//
//	{
//	  "config_key": "broker_port",
//	  "config_value": 1884
//	}
func (mc *MessageCoordinator) handleConfigUpdate(topic string, payload []byte) error {
	mc.GetLogger().Info("Received configuration update",
		zap.String("topic", topic),
		zap.Int("size", len(payload)))

	// Unmarshal the MQTT message envelope
	var msg mqtt.Message
	if err := json.Unmarshal(payload, &msg); err != nil {
		mc.GetLogger().Error("Failed to unmarshal config update envelope", zap.Error(err))
		return err
	}

	// Unmarshal the config update payload
	var update struct {
		ConfigKey   string      `json:"config_key"`
		ConfigValue interface{} `json:"config_value"`
	}
	if err := msg.UnmarshalPayload(&update); err != nil {
		mc.GetLogger().Error("Failed to unmarshal config update payload", zap.Error(err))
		return err
	}

	// Reload configuration from database
	if mc.configLoader == nil {
		mc.GetLogger().Warn("Config loader not set, cannot reload configuration")
		return fmt.Errorf("config loader not set")
	}

	ctx := context.Background()
	coordConfig, err := mc.configLoader.LoadCoordinatorConfig(ctx, "message-coordinator")
	if err != nil {
		mc.GetLogger().Error("Failed to reload configuration", zap.Error(err))
		return err
	}

	// Parse updated configuration
	brokerURL, err := coordConfig.GetString("broker_url", "localhost")
	if err != nil {
		mc.GetLogger().Error("Failed to parse broker_url", zap.Error(err))
		return err
	}
	brokerPort, err := coordConfig.GetInt("broker_port", 1883)
	if err != nil {
		mc.GetLogger().Error("Failed to parse broker_port", zap.Error(err))
		return err
	}
	monitorInterval, err := coordConfig.GetDuration("monitor_interval", 30*time.Second)
	if err != nil {
		mc.GetLogger().Error("Failed to parse monitor_interval", zap.Error(err))
		return err
	}
	maxReconnectAttempts, err := coordConfig.GetInt("max_reconnect_attempts", 5)
	if err != nil {
		mc.GetLogger().Error("Failed to parse max_reconnect_attempts", zap.Error(err))
		return err
	}

	// Update configuration (thread-safe)
	mc.mu.Lock()
	mc.config.BrokerURL = brokerURL
	mc.config.BrokerPort = brokerPort
	mc.config.MonitorInterval = monitorInterval
	mc.config.MaxReconnectAttempts = maxReconnectAttempts
	mc.mu.Unlock()

	mc.GetLogger().Info("Configuration reloaded successfully",
		zap.String("config_key", update.ConfigKey),
		zap.String("broker_url", brokerURL),
		zap.Int("broker_port", brokerPort),
		zap.Duration("monitor_interval", monitorInterval),
		zap.Int("max_reconnect_attempts", maxReconnectAttempts))

	return nil
}

// Name returns the coordinator name.
func (mc *MessageCoordinator) Name() string {
	return "message-coordinator"
}

// LoadConfig loads configuration.
func (mc *MessageCoordinator) LoadConfig(config interface{}) error {
	cfg, ok := config.(*MessageCoordinatorConfig)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	mc.config = cfg
	return mc.BaseCoordinator.LoadConfig(config)
}

// ValidateConfig validates the configuration.
func (mc *MessageCoordinator) ValidateConfig() error {
	if mc.config == nil {
		return fmt.Errorf("config is nil")
	}
	if mc.config.BrokerURL == "" {
		return fmt.Errorf("broker_url is required")
	}
	if mc.config.BrokerPort <= 0 || mc.config.BrokerPort > 65535 {
		return fmt.Errorf("invalid broker_port: %d", mc.config.BrokerPort)
	}
	return nil
}
