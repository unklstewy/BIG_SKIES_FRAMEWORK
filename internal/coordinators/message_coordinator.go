// Package coordinators implements the message coordinator.
package coordinators

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/healthcheck"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
	"go.uber.org/zap"
)

// MessageCoordinator manages the MQTT message bus and health monitoring.
type MessageCoordinator struct {
	*BaseCoordinator
	config           *MessageCoordinatorConfig
	subscribedTopics map[string]bool
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
	
	// Create MQTT client configuration
	mqttConfig := &mqtt.Config{
		BrokerURL:            fmt.Sprintf("%s:%d", config.BrokerURL, config.BrokerPort),
		ClientID:             "message-coordinator",
		KeepAlive:            30 * time.Second,
		ConnectTimeout:       10 * time.Second,
		AutoReconnect:        true,
		MaxReconnectInterval: 5 * time.Minute,
	}
	
	mqttClient, err := mqtt.NewClient(mqttConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MQTT client: %w", err)
	}
	
	base := NewBaseCoordinator("message-coordinator", mqttClient, logger)
	
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
	
	var msg mqtt.Message
	if err := msg.UnmarshalPayload(&payload); err != nil {
		mc.GetLogger().Error("Failed to unmarshal health message", zap.Error(err))
		return err
	}
	
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
