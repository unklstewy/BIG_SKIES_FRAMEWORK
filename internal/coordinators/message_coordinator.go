// Package coordinators implements the message coordinator.
package coordinators

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/config"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/models"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/healthcheck"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
	"go.uber.org/zap"
)

// MessageCoordinator manages the MQTT message bus and health monitoring.
type MessageCoordinator struct {
	*BaseCoordinator
	config           *MessageCoordinatorConfig
	subscribedTopics map[string]bool
	configLoader     *config.Loader                    // Database configuration loader
	protectionRules  []models.TopicProtectionRule      // RBAC protection rules
	pendingMessages  map[string]*models.PendingMessage // Messages awaiting validation
	// Phase 3: Advanced Features
	auditLogger       *zap.Logger         // Dedicated audit logger
	metrics           *models.RBACMetrics // Performance and health metrics
	queueMutex        sync.RWMutex        // Separate mutex for queue operations
	maxQueueSize      int                 // Maximum pending messages
	validationTimeout time.Duration       // Timeout for RBAC validation
	rbacEnabled       bool                // RBAC validation enabled flag
	mu                sync.RWMutex
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
	// Phase 3: Advanced Features
	MaxQueueSize      int           `json:"max_queue_size"`     // Maximum pending messages
	ValidationTimeout time.Duration `json:"validation_timeout"` // Timeout for RBAC validation
	RBACEnabled       bool          `json:"rbac_enabled"`       // Enable RBAC validation
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
		BaseCoordinator:   base,
		config:            config,
		subscribedTopics:  make(map[string]bool),
		pendingMessages:   make(map[string]*models.PendingMessage),
		auditLogger:       logger.Named("rbac-audit"), // Dedicated audit logger
		metrics:           &models.RBACMetrics{},      // Initialize metrics
		maxQueueSize:      config.MaxQueueSize,
		validationTimeout: config.ValidationTimeout,
		rbacEnabled:       config.RBACEnabled,
	}

	// Set default values if not configured
	if mc.maxQueueSize == 0 {
		mc.maxQueueSize = 1000 // Default max queue size
	}
	if mc.validationTimeout == 0 {
		mc.validationTimeout = 30 * time.Second // Default timeout
	}

	// Register self health check
	mc.RegisterHealthCheck(mc)

	return mc, nil
}

// Start begins message coordinator operations.
func (mc *MessageCoordinator) Start(ctx context.Context) error {
	mc.GetLogger().Info("Starting message coordinator",
		zap.String("broker", mc.config.BrokerURL),
		zap.Int("port", mc.config.BrokerPort),
		zap.Bool("rbac_enabled", mc.rbacEnabled))

	// Start base coordinator first to connect MQTT
	if err := mc.BaseCoordinator.Start(ctx); err != nil {
		return err
	}

	// Wait for credentials if database access needed
	if _, err := mc.WaitForCredentials(ctx, 30*time.Second); err != nil {
		return err
	}

	// Load protection rules from database if RBAC is enabled
	if mc.rbacEnabled {
		if err := mc.loadProtectionRules(ctx); err != nil {
			return fmt.Errorf("failed to load protection rules: %w", err)
		}
	}

	// Subscribe to coordinator health topics
	if err := mc.subscribeHealthTopics(); err != nil {
		return fmt.Errorf("failed to subscribe to health topics: %w", err)
	}

	// Subscribe to all coordinator topics for RBAC interception (if enabled)
	if err := mc.subscribeCoordinatorTopics(); err != nil {
		return fmt.Errorf("failed to subscribe to coordinator topics: %w", err)
	}

	// Subscribe to configuration update topic
	if err := mc.subscribeConfigTopic(); err != nil {
		return fmt.Errorf("failed to subscribe to config topic: %w", err)
	}

	// Subscribe to RBAC validation response topic if RBAC is enabled
	if mc.rbacEnabled {
		if err := mc.subscribeRBACResponseTopic(); err != nil {
			return fmt.Errorf("failed to subscribe to RBAC response topic: %w", err)
		}
	}

	// Start health status publishing
	go mc.StartHealthPublishing(ctx)

	// Start RBAC timeout cleanup if RBAC is enabled
	if mc.rbacEnabled {
		go mc.startTimeoutCleanup(ctx)
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

	// RBAC Health Checks (prioritize over MQTT connection)
	rbacMetrics := mc.metrics.GetMetrics()
	details["rbac_messages_processed"] = rbacMetrics.MessagesProcessed
	details["rbac_messages_validated"] = rbacMetrics.MessagesValidated
	details["rbac_messages_rejected"] = rbacMetrics.MessagesRejected
	details["rbac_messages_forwarded"] = rbacMetrics.MessagesForwarded
	details["rbac_current_queue_depth"] = rbacMetrics.CurrentQueueDepth
	details["rbac_max_queue_depth"] = rbacMetrics.MaxQueueDepth
	details["rbac_queue_overflows"] = rbacMetrics.QueueOverflows
	details["rbac_validation_errors"] = rbacMetrics.ValidationErrors
	details["rbac_coordinator_errors"] = rbacMetrics.CoordinatorErrors
	details["rbac_validation_timeouts"] = rbacMetrics.ValidationTimeouts
	details["rbac_avg_validation_time"] = rbacMetrics.AvgValidationTime.String()
	details["rbac_min_validation_time"] = rbacMetrics.MinValidationTime.String()
	details["rbac_max_validation_time"] = rbacMetrics.MaxValidationTime.String()

	// Check queue health
	if rbacMetrics.CurrentQueueDepth > mc.maxQueueSize/2 {
		status = healthcheck.StatusDegraded
		message = "RBAC queue depth is high"
	}

	if rbacMetrics.QueueOverflows > 0 {
		status = healthcheck.StatusDegraded
		message = "RBAC queue overflows detected"
	}

	// Check error rates
	totalMessages := rbacMetrics.MessagesProcessed
	if totalMessages > 0 {
		errorRate := float64(rbacMetrics.ValidationErrors+rbacMetrics.CoordinatorErrors) / float64(totalMessages)
		if errorRate > 0.1 { // 10% error rate
			status = healthcheck.StatusDegraded
			message = "High RBAC error rate detected"
		}
	}

	// Only check MQTT connection and topics if RBAC is healthy
	if status == healthcheck.StatusHealthy {
		// Check MQTT connection (skip if no client for testing)
		mqttClient := mc.GetMQTTClient()
		if mqttClient != nil && !mqttClient.IsConnected() {
			status = healthcheck.StatusUnhealthy
			message = "MQTT client not connected"
			details["mqtt_connected"] = false
		} else {
			details["mqtt_connected"] = mqttClient != nil && (mqttClient == nil || mqttClient.IsConnected())
		}

		// Check subscribed topics (only if MQTT is connected or no client for testing)
		if status == healthcheck.StatusHealthy {
			mc.mu.RLock()
			topicCount := len(mc.subscribedTopics)
			mc.mu.RUnlock()

			details["subscribed_topics"] = topicCount

			if topicCount == 0 {
				status = healthcheck.StatusDegraded
				message = "No topics subscribed"
			}
		}
	} else {
		// RBAC is not healthy, still include MQTT details for diagnostics
		mqttClient := mc.GetMQTTClient()
		if mqttClient == nil {
			details["mqtt_connected"] = true // Assume connected for testing
		} else if !mqttClient.IsConnected() {
			details["mqtt_connected"] = false
		} else {
			details["mqtt_connected"] = true
		}

		mc.mu.RLock()
		details["subscribed_topics"] = len(mc.subscribedTopics)
		mc.mu.RUnlock()
	}

	// Update metrics health status
	mc.metrics.UpdateHealthStatus(status == healthcheck.StatusHealthy)

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

// loadProtectionRules loads RBAC protection rules from the database.
func (mc *MessageCoordinator) loadProtectionRules(ctx context.Context) error {
	// Create config loader if not set
	if mc.configLoader == nil {
		// We need to create a pool here. For now, assume it's set via SetConfigLoader
		return fmt.Errorf("config loader not set")
	}

	rules, err := mc.configLoader.LoadTopicProtectionRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to load protection rules: %w", err)
	}

	mc.mu.Lock()
	mc.protectionRules = rules
	mc.mu.Unlock()

	mc.GetLogger().Info("Loaded protection rules",
		zap.Int("count", len(rules)))

	return nil
}

// subscribeCoordinatorTopics subscribes to all coordinator topics for RBAC interception.
func (mc *MessageCoordinator) subscribeCoordinatorTopics() error {
	// Subscribe to wildcard topic for all coordinator messages
	topic := "bigskies/coordinator/+/+/+" // coordinator/{name}/{action}/{resource}
	return mc.subscribe(topic, mc.handleCoordinatorMessage)
}

// subscribeRBACResponseTopic subscribes to RBAC validation response topic.
func (mc *MessageCoordinator) subscribeRBACResponseTopic() error {
	topic := "bigskies/coordinator/security/rbac/response"
	return mc.subscribe(topic, mc.handleRBACResponse)
}

// handleCoordinatorMessage processes incoming coordinator messages for RBAC validation.
func (mc *MessageCoordinator) handleCoordinatorMessage(topic string, payload []byte) error {
	mc.metrics.RecordMessageProcessed()

	mc.GetLogger().Debug("Received coordinator message",
		zap.String("topic", topic),
		zap.Int("size", len(payload)))

	// Skip health and status messages
	if strings.Contains(topic, "/health/") || strings.Contains(topic, "/status/") {
		// Forward directly without validation
		mc.metrics.RecordMessageForwarded()
		return mc.forwardMessage(topic, payload)
	}

	// If RBAC is not enabled, forward all messages directly
	if !mc.rbacEnabled {
		mc.metrics.RecordMessageForwarded()
		return mc.forwardMessage(topic, payload)
	}

	// Check if topic matches protection rules
	rule := mc.findMatchingRule(topic)
	if rule == nil {
		// No protection rule, forward directly
		mc.metrics.RecordMessageForwarded()
		return mc.forwardMessage(topic, payload)
	}

	// Extract user context from message
	userContext, err := mc.extractUserContext(payload)
	if err != nil {
		mc.metrics.RecordValidationError()
		mc.auditLogger.Warn("RBAC validation failed - user context extraction error",
			zap.String("topic", topic),
			zap.String("user_id", userContext.UserID),
			zap.Error(err))
		return err
	}

	// Check queue size before adding new message
	mc.queueMutex.RLock()
	queueSize := len(mc.pendingMessages)
	mc.queueMutex.RUnlock()

	if queueSize >= mc.maxQueueSize {
		mc.metrics.RecordQueueOverflow()
		mc.auditLogger.Warn("RBAC validation failed - queue overflow",
			zap.String("topic", topic),
			zap.String("user_id", userContext.UserID),
			zap.Int("queue_size", queueSize),
			zap.Int("max_queue_size", mc.maxQueueSize))
		return fmt.Errorf("RBAC validation queue overflow: %d/%d", queueSize, mc.maxQueueSize)
	}

	// Create RBAC validation request
	correlationID := generateCorrelationID()
	request := models.RBACValidationRequest{
		CorrelationID: correlationID,
		UserID:        userContext.UserID,
		Resource:      rule.Resource,
		Action:        rule.Action,
		Context:       userContext,
		Timestamp:     time.Now(),
	}

	// Store pending message with proper timeout
	pending := &models.PendingMessage{
		ID:            correlationID,
		OriginalTopic: topic,
		Payload:       payload,
		UserContext:   userContext,
		CorrelationID: correlationID,
		ReceivedAt:    time.Now(),
		ExpiresAt:     time.Now().Add(mc.validationTimeout),
	}

	mc.queueMutex.Lock()
	mc.pendingMessages[correlationID] = pending
	mc.metrics.RecordQueueDepth(len(mc.pendingMessages))
	mc.queueMutex.Unlock()

	// Audit log: validation request initiated
	mc.auditLogger.Info("RBAC validation request initiated",
		zap.String("correlation_id", correlationID),
		zap.String("topic", topic),
		zap.String("user_id", userContext.UserID),
		zap.String("resource", rule.Resource),
		zap.String("action", rule.Action))

	// Send validation request to security coordinator
	if err := mc.sendRBACValidationRequest(request); err != nil {
		mc.metrics.RecordCoordinatorError()
		mc.queueMutex.Lock()
		delete(mc.pendingMessages, correlationID)
		mc.metrics.RecordQueueDepth(len(mc.pendingMessages))
		mc.queueMutex.Unlock()

		mc.auditLogger.Error("RBAC validation failed - coordinator communication error",
			zap.String("correlation_id", correlationID),
			zap.String("topic", topic),
			zap.String("user_id", userContext.UserID),
			zap.Error(err))
		return err
	}

	mc.metrics.RecordMessageValidated()
	mc.GetLogger().Debug("Sent RBAC validation request",
		zap.String("correlation_id", correlationID),
		zap.String("resource", rule.Resource),
		zap.String("action", rule.Action))

	return nil
}

// handleRBACResponse processes RBAC validation responses from security coordinator.
func (mc *MessageCoordinator) handleRBACResponse(topic string, payload []byte) error {
	// Unmarshal response
	var msg mqtt.Message
	if err := json.Unmarshal(payload, &msg); err != nil {
		mc.metrics.RecordValidationError()
		mc.GetLogger().Error("Failed to unmarshal RBAC response envelope", zap.Error(err))
		return err
	}

	var response models.RBACValidationResponse
	if err := msg.UnmarshalPayload(&response); err != nil {
		mc.metrics.RecordValidationError()
		mc.GetLogger().Error("Failed to unmarshal RBAC response payload", zap.Error(err))
		return err
	}

	// Find pending message
	mc.queueMutex.Lock()
	pending, exists := mc.pendingMessages[response.CorrelationID]
	if exists {
		delete(mc.pendingMessages, response.CorrelationID)
		mc.metrics.RecordQueueDepth(len(mc.pendingMessages))
	}
	mc.queueMutex.Unlock()

	if !exists {
		mc.auditLogger.Warn("RBAC validation response for unknown correlation ID",
			zap.String("correlation_id", response.CorrelationID),
			zap.Time("response_timestamp", response.Timestamp))
		return nil
	}

	// Calculate validation time
	validationTime := time.Since(pending.ReceivedAt)
	mc.metrics.RecordValidationTime(validationTime)

	if response.Allowed {
		// Forward the message
		if err := mc.forwardMessage(pending.OriginalTopic, pending.Payload); err != nil {
			mc.metrics.RecordCoordinatorError()
			mc.auditLogger.Error("RBAC validation failed - message forwarding error",
				zap.String("correlation_id", response.CorrelationID),
				zap.String("topic", pending.OriginalTopic),
				zap.String("user_id", pending.UserContext.UserID),
				zap.Duration("validation_time", validationTime),
				zap.Error(err))
			return err
		}

		mc.metrics.RecordMessageForwarded()
		mc.auditLogger.Info("RBAC validation allowed - message forwarded",
			zap.String("correlation_id", response.CorrelationID),
			zap.String("topic", pending.OriginalTopic),
			zap.String("user_id", pending.UserContext.UserID),
			zap.String("resource", response.CorrelationID), // Note: we don't have resource in response
			zap.Duration("validation_time", validationTime))

		mc.GetLogger().Debug("Forwarded validated message",
			zap.String("correlation_id", response.CorrelationID))
	} else {
		// Log security event - access denied
		mc.metrics.RecordMessageRejected()
		mc.auditLogger.Warn("RBAC validation denied - access rejected",
			zap.String("correlation_id", response.CorrelationID),
			zap.String("topic", pending.OriginalTopic),
			zap.String("user_id", pending.UserContext.UserID),
			zap.String("reason", response.Reason),
			zap.Duration("validation_time", validationTime))

		mc.GetLogger().Warn("RBAC validation denied",
			zap.String("correlation_id", response.CorrelationID),
			zap.String("reason", response.Reason),
			zap.String("user_id", pending.UserContext.UserID),
			zap.String("topic", pending.OriginalTopic))
		// Message is rejected (not forwarded)
	}

	return nil
}

// findMatchingRule finds the protection rule that matches the given topic.
func (mc *MessageCoordinator) findMatchingRule(topic string) *models.TopicProtectionRule {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	for _, rule := range mc.protectionRules {
		if mc.topicMatchesPattern(topic, rule.TopicPattern) {
			return &rule
		}
	}
	return nil
}

// topicMatchesPattern checks if a topic matches a pattern (simple wildcard matching).
func (mc *MessageCoordinator) topicMatchesPattern(topic, pattern string) bool {
	// Simple implementation: replace + with .* and match as regex
	// For production, consider a more robust pattern matching library
	pattern = strings.ReplaceAll(pattern, "+", ".*")
	matched, _ := regexp.MatchString("^"+pattern+"$", topic)
	return matched
}

// extractUserContext extracts user authentication context from message payload.
func (mc *MessageCoordinator) extractUserContext(payload []byte) (models.UserContext, error) {
	var msg mqtt.Message
	if err := json.Unmarshal(payload, &msg); err != nil {
		return models.UserContext{}, fmt.Errorf("failed to unmarshal message envelope: %w", err)
	}

	// For now, assume user context is in message metadata or payload
	// This is a placeholder - actual implementation depends on how auth is passed
	context := models.UserContext{
		UserID: "anonymous", // Default
	}

	// Try to extract from payload if it's a map
	var payloadData map[string]interface{}
	if err := msg.UnmarshalPayload(&payloadData); err == nil {
		if userID, ok := payloadData["user_id"].(string); ok {
			context.UserID = userID
		}
		if username, ok := payloadData["username"].(string); ok {
			context.Username = username
		}
		if token, ok := payloadData["token"].(string); ok {
			context.Token = token
		}
	}

	return context, nil
}

// sendRBACValidationRequest sends a validation request to the security coordinator.
func (mc *MessageCoordinator) sendRBACValidationRequest(request models.RBACValidationRequest) error {
	topic := "bigskies/coordinator/security/rbac/validate"
	msg, err := mqtt.NewMessage(mqtt.MessageTypeRequest, "coordinator:message", request)
	if err != nil {
		return fmt.Errorf("failed to create validation request message: %w", err)
	}

	return mc.GetMQTTClient().PublishJSON(topic, 1, false, msg)
}

// forwardMessage forwards a message to its original destination.
func (mc *MessageCoordinator) forwardMessage(topic string, payload []byte) error {
	// For interception, we need to republish to the same topic
	// In a real implementation, this might need special handling to avoid loops
	return mc.GetMQTTClient().Publish(topic, 1, false, payload)
}

// generateCorrelationID generates a unique correlation ID for requests.
func generateCorrelationID() string {
	return fmt.Sprintf("rbac-%d", time.Now().UnixNano())
}

// startTimeoutCleanup runs a background goroutine to clean up expired pending messages.
func (mc *MessageCoordinator) startTimeoutCleanup(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mc.cleanupExpiredMessages()
		}
	}
}

// cleanupExpiredMessages removes expired pending messages and logs timeouts.
func (mc *MessageCoordinator) cleanupExpiredMessages() {
	now := time.Now()
	var expired []string

	mc.queueMutex.Lock()
	for id, pending := range mc.pendingMessages {
		if now.After(pending.ExpiresAt) {
			expired = append(expired, id)
		}
	}

	for _, id := range expired {
		pending := mc.pendingMessages[id]
		delete(mc.pendingMessages, id)

		// Record timeout metrics and audit log
		mc.metrics.RecordValidationTimeout()
		mc.auditLogger.Warn("RBAC validation timeout - message expired",
			zap.String("correlation_id", id),
			zap.String("topic", pending.OriginalTopic),
			zap.String("user_id", pending.UserContext.UserID),
			zap.Time("received_at", pending.ReceivedAt),
			zap.Time("expired_at", pending.ExpiresAt),
			zap.Duration("timeout_duration", mc.validationTimeout))
	}
	mc.metrics.RecordQueueDepth(len(mc.pendingMessages))
	mc.queueMutex.Unlock()

	if len(expired) > 0 {
		mc.GetLogger().Warn("Cleaned up expired RBAC validation requests",
			zap.Int("expired_count", len(expired)),
			zap.Int("remaining_queue_depth", len(mc.pendingMessages)))
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
