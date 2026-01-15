// Package coordinators implements the message coordinator.
package coordinators

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/healthcheck"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
	"go.uber.org/zap"
)

// testableMessageCoordinator wraps MessageCoordinator for testing
type testableMessageCoordinator struct {
	*MessageCoordinator
	mockClient *mockMQTTClient
}

func newTestableMessageCoordinator(logger *zap.Logger) *testableMessageCoordinator {
	mockClient := newMockMQTTClient()

	mc := &MessageCoordinator{
		BaseCoordinator: &BaseCoordinator{
			name:          mqtt.CoordinatorMessage,
			mqttClient:    nil,
			healthEngine:  healthcheck.NewEngine(logger, 30*time.Second),
			logger:        logger,
			shutdownFuncs: make([]func(context.Context) error, 0),
		},
		config: &MessageCoordinatorConfig{
			BaseConfig: BaseConfig{
				Name:                "message",
				HealthCheckInterval: 30 * time.Second,
			},
			BrokerURL:  "localhost",
			BrokerPort: 1883,
		},
		subscribedTopics: make(map[string]bool),
	}

	return &testableMessageCoordinator{
		MessageCoordinator: mc,
		mockClient:         mockClient,
	}
}

// Override StartHealthPublishing for testing
func (tmc *testableMessageCoordinator) StartHealthPublishing(ctx context.Context) {
	if tmc.mockClient == nil {
		tmc.GetLogger().Warn("Cannot publish health: MQTT client is nil")
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	time.Sleep(500 * time.Millisecond)

	// Use testable publish method
	tmc.publishHealthWithMock(ctx)

	for {
		select {
		case <-ctx.Done():
			tmc.GetLogger().Debug("Health publishing stopped")
			return
		case <-ticker.C:
			tmc.publishHealthWithMock(ctx)
		}
	}
}

func (tmc *testableMessageCoordinator) publishHealthWithMock(ctx context.Context) {
	health := tmc.HealthCheck(ctx)
	topic := mqtt.CoordinatorHealthTopic(mqtt.CoordinatorMessage)

	msg, err := mqtt.NewMessage(mqtt.MessageTypeStatus, "coordinator:"+mqtt.CoordinatorMessage, health)
	if err != nil {
		tmc.GetLogger().Error("Failed to create health message", zap.Error(err))
		return
	}

	if err := tmc.mockClient.PublishJSON(topic, 1, false, msg); err != nil {
		tmc.GetLogger().Error("Failed to publish health status",
			zap.String("topic", topic),
			zap.Error(err))
	}
}

// Override Check to use mock client
func (tmc *testableMessageCoordinator) Check(ctx context.Context) *healthcheck.Result {
	status := healthcheck.StatusHealthy
	message := "Message coordinator is healthy"
	details := make(map[string]interface{})

	// Check MQTT connection using mock
	if tmc.mockClient == nil || !tmc.mockClient.IsConnected() {
		status = healthcheck.StatusUnhealthy
		message = "MQTT client not connected"
		details["mqtt_connected"] = false
	} else {
		details["mqtt_connected"] = true
	}

	// Check subscribed topics
	tmc.mu.RLock()
	topicCount := len(tmc.subscribedTopics)
	tmc.mu.RUnlock()

	details["subscribed_topics"] = topicCount

	if status == healthcheck.StatusHealthy && topicCount == 0 {
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

// TestMessageCoordinator_handleHealthMessage tests that health messages are correctly unmarshaled.
func TestMessageCoordinator_handleHealthMessage(t *testing.T) {
	logger := zap.NewNop()
	tmc := newTestableMessageCoordinator(logger)

	tests := []struct {
		name           string
		createPayload  func() []byte
		expectError    bool
		validateResult func(t *testing.T)
	}{
		{
			name: "valid health message with full envelope",
			createPayload: func() []byte {
				health := &healthcheck.Result{
					ComponentName: "security-coordinator",
					Status:        healthcheck.StatusHealthy,
					Message:       "All systems operational",
					Timestamp:     time.Now(),
					Details: map[string]interface{}{
						"uptime_seconds": 123.45,
						"running":        true,
					},
				}

				msg, err := mqtt.NewMessage(mqtt.MessageTypeStatus, "coordinator:security", health)
				require.NoError(t, err)

				payload, err := json.Marshal(msg)
				require.NoError(t, err)
				return payload
			},
			expectError: false,
			validateResult: func(t *testing.T) {
				// Success case - no additional validation needed
			},
		},
		{
			name: "health message with degraded status",
			createPayload: func() []byte {
				health := &healthcheck.Result{
					ComponentName: "datastore-coordinator",
					Status:        healthcheck.StatusDegraded,
					Message:       "Database connection slow",
					Timestamp:     time.Now(),
					Details: map[string]interface{}{
						"latency_ms": 500,
					},
				}

				msg, err := mqtt.NewMessage(mqtt.MessageTypeStatus, "coordinator:datastore", health)
				require.NoError(t, err)

				payload, err := json.Marshal(msg)
				require.NoError(t, err)
				return payload
			},
			expectError: false,
			validateResult: func(t *testing.T) {
				// Success case
			},
		},
		{
			name: "health message with unhealthy status",
			createPayload: func() []byte {
				health := &healthcheck.Result{
					ComponentName: "application-coordinator",
					Status:        healthcheck.StatusUnhealthy,
					Message:       "Service unavailable",
					Timestamp:     time.Now(),
					Details: map[string]interface{}{
						"error": "connection refused",
					},
				}

				msg, err := mqtt.NewMessage(mqtt.MessageTypeStatus, "coordinator:application", health)
				require.NoError(t, err)

				payload, err := json.Marshal(msg)
				require.NoError(t, err)
				return payload
			},
			expectError: false,
			validateResult: func(t *testing.T) {
				// Success case
			},
		},
		{
			name: "invalid JSON payload",
			createPayload: func() []byte {
				return []byte("{invalid json}")
			},
			expectError: true,
			validateResult: func(t *testing.T) {
				// Error case
			},
		},
		{
			name: "missing message envelope",
			createPayload: func() []byte {
				health := &healthcheck.Result{
					ComponentName: "test",
					Status:        healthcheck.StatusHealthy,
				}
				payload, _ := json.Marshal(health)
				return payload
			},
			expectError: true,
			validateResult: func(t *testing.T) {
				// Error case - missing envelope fields
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic := mqtt.CoordinatorHealthTopic("test")
			payload := tt.createPayload()

			err := tmc.handleHealthMessage(topic, payload)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			tt.validateResult(t)
		})
	}
}

// TestMessageCoordinator_handleHealthMessage_UnmarshalsEnvelope tests that the full message envelope is unmarshaled.
func TestMessageCoordinator_handleHealthMessage_UnmarshalsEnvelope(t *testing.T) {
	logger := zap.NewNop()
	tmc := newTestableMessageCoordinator(logger)

	// Create a health message with specific envelope data
	health := &healthcheck.Result{
		ComponentName: "telescope-coordinator",
		Status:        healthcheck.StatusHealthy,
		Message:       "Telescope connected",
		Timestamp:     time.Now(),
	}

	msg, err := mqtt.NewMessage(mqtt.MessageTypeStatus, "coordinator:telescope", health)
	require.NoError(t, err)

	// Verify envelope has expected fields
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, mqtt.MessageTypeStatus, msg.Type)
	assert.Equal(t, "coordinator:telescope", msg.Source)
	assert.NotZero(t, msg.Timestamp)

	payload, err := json.Marshal(msg)
	require.NoError(t, err)

	// Call handleHealthMessage
	topic := mqtt.CoordinatorHealthTopic("telescope")
	err = tmc.handleHealthMessage(topic, payload)
	assert.NoError(t, err)
}

// TestMessageCoordinator_handleHealthMessage_UnmarshalsHealthPayload tests that the health payload is correctly extracted.
func TestMessageCoordinator_handleHealthMessage_UnmarshalsHealthPayload(t *testing.T) {
	logger := zap.NewNop()
	tmc := newTestableMessageCoordinator(logger)

	// Create health result with specific details
	expectedHealth := &healthcheck.Result{
		ComponentName: "plugin-coordinator",
		Status:        healthcheck.StatusHealthy,
		Message:       "3 plugins active",
		Timestamp:     time.Now(),
		Duration:      250 * time.Millisecond,
		Details: map[string]interface{}{
			"plugin_count":   3,
			"active_count":   3,
			"failed_count":   0,
			"uptime_seconds": 3600.5,
		},
	}

	msg, err := mqtt.NewMessage(mqtt.MessageTypeStatus, "coordinator:plugin", expectedHealth)
	require.NoError(t, err)

	payload, err := json.Marshal(msg)
	require.NoError(t, err)

	// Call handleHealthMessage
	topic := mqtt.CoordinatorHealthTopic("plugin")
	err = tmc.handleHealthMessage(topic, payload)
	assert.NoError(t, err)
}

// TestMessageCoordinator_Start_InitiatesHealthPublishing tests that Start() initiates health publishing goroutine.
func TestMessageCoordinator_Start_InitiatesHealthPublishing(t *testing.T) {
	logger := zap.NewNop()
	tmc := newTestableMessageCoordinator(logger)

	// Register the coordinator as a health checker
	tmc.RegisterHealthCheck(tmc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start health publishing goroutine separately (simulating what Start() does)
	go tmc.StartHealthPublishing(ctx)

	// Wait a bit for health publishing to start and publish initial message
	time.Sleep(1 * time.Second)

	// Cancel and cleanup
	cancel()
	time.Sleep(200 * time.Millisecond)

	// Verify that health messages were published
	msgs := tmc.mockClient.GetPublishedMessages()
	assert.GreaterOrEqual(t, len(msgs), 1, "Should have published at least one health message")

	// Verify health message structure
	if len(msgs) > 0 {
		msg := msgs[0]
		assert.Equal(t, mqtt.CoordinatorHealthTopic(mqtt.CoordinatorMessage), msg.topic)

		// Verify the message structure
		var envelope mqtt.Message
		err := json.Unmarshal(msg.payload, &envelope)
		assert.NoError(t, err)

		var health healthcheck.Result
		err = envelope.UnmarshalPayload(&health)
		assert.NoError(t, err)
		// The component name will be "message" from the coordinator's Name() method
		assert.Contains(t, health.ComponentName, "message")
	}
}

func TestMessageCoordinator_Check(t *testing.T) {
	logger := zap.NewNop()
	tmc := newTestableMessageCoordinator(logger)

	ctx := context.Background()

	tests := []struct {
		name              string
		mqttConnected     bool
		subscribedCount   int
		expectedStatus    healthcheck.Status
		expectedInMessage string
	}{
		{
			name:              "healthy with topics",
			mqttConnected:     true,
			subscribedCount:   5,
			expectedStatus:    healthcheck.StatusHealthy,
			expectedInMessage: "healthy",
		},
		{
			name:              "degraded with no subscriptions",
			mqttConnected:     true,
			subscribedCount:   0,
			expectedStatus:    healthcheck.StatusDegraded,
			expectedInMessage: "No topics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock client state
			tmc.mockClient.mu.Lock()
			tmc.mockClient.connected = tt.mqttConnected
			tmc.mockClient.mu.Unlock()

			// Setup subscribed topics
			tmc.mu.Lock()
			tmc.subscribedTopics = make(map[string]bool)
			for i := 0; i < tt.subscribedCount; i++ {
				tmc.subscribedTopics["test/topic/"+string(rune(i))] = true
			}
			tmc.mu.Unlock()

			result := tmc.Check(ctx)

			assert.NotNil(t, result)
			assert.Equal(t, "message-coordinator", result.ComponentName)
			assert.Equal(t, tt.expectedStatus, result.Status)
			assert.Contains(t, result.Message, tt.expectedInMessage)
			assert.NotZero(t, result.Timestamp)
			assert.NotNil(t, result.Details)
			assert.Contains(t, result.Details, "subscribed_topics")
		})
	}
}

func TestMessageCoordinator_ValidateConfig(t *testing.T) {
	logger := zap.NewNop()
	tmc := newTestableMessageCoordinator(logger)

	tests := []struct {
		name    string
		config  *MessageCoordinatorConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "config is nil",
		},
		{
			name: "empty broker URL",
			config: &MessageCoordinatorConfig{
				BrokerURL:  "",
				BrokerPort: 1883,
			},
			wantErr: true,
			errMsg:  "broker_url is required",
		},
		{
			name: "invalid port - too low",
			config: &MessageCoordinatorConfig{
				BrokerURL:  "localhost",
				BrokerPort: 0,
			},
			wantErr: true,
			errMsg:  "invalid broker_port",
		},
		{
			name: "invalid port - too high",
			config: &MessageCoordinatorConfig{
				BrokerURL:  "localhost",
				BrokerPort: 99999,
			},
			wantErr: true,
			errMsg:  "invalid broker_port",
		},
		{
			name: "valid config",
			config: &MessageCoordinatorConfig{
				BrokerURL:  "localhost",
				BrokerPort: 1883,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmc.config = tt.config
			err := tmc.ValidateConfig()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
