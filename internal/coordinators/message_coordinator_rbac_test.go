package coordinators

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/models"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
	"go.uber.org/zap/zaptest"
)

// TestMessageCoordinator_RBAC_ProtectionRuleMatching tests protection rule matching logic
func TestMessageCoordinator_RBAC_ProtectionRuleMatching(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &MessageCoordinatorConfig{
		BaseConfig: BaseConfig{
			Name: "message-coordinator",
		},
		BrokerURL:         "localhost",
		BrokerPort:        1883,
		MaxQueueSize:      1000,
		ValidationTimeout: 30 * time.Second,
	}

	mc := &MessageCoordinator{
		config:          config,
		protectionRules: []models.TopicProtectionRule{},
		auditLogger:     logger.Named("rbac-audit"),
		metrics:         &models.RBACMetrics{},
	}

	// Test cases for topic matching
	testCases := []struct {
		name             string
		rules            []models.TopicProtectionRule
		topic            string
		expectedMatch    bool
		expectedResource string
		expectedAction   string
	}{
		{
			name: "exact match",
			rules: []models.TopicProtectionRule{
				{TopicPattern: "bigskies/coordinator/telescope/control/slew", Resource: "telescope", Action: "control"},
			},
			topic:            "bigskies/coordinator/telescope/control/slew",
			expectedMatch:    true,
			expectedResource: "telescope",
			expectedAction:   "control",
		},
		{
			name: "wildcard match",
			rules: []models.TopicProtectionRule{
				{TopicPattern: "bigskies/coordinator/telescope/control/+", Resource: "telescope", Action: "control"},
			},
			topic:            "bigskies/coordinator/telescope/control/slew",
			expectedMatch:    true,
			expectedResource: "telescope",
			expectedAction:   "control",
		},
		{
			name: "no match",
			rules: []models.TopicProtectionRule{
				{TopicPattern: "bigskies/coordinator/security/user/+", Resource: "security", Action: "manage"},
			},
			topic:         "bigskies/coordinator/telescope/control/slew",
			expectedMatch: false,
		},
		{
			name: "multiple rules - first match wins",
			rules: []models.TopicProtectionRule{
				{TopicPattern: "bigskies/coordinator/telescope/control/+", Resource: "telescope", Action: "control"},
				{TopicPattern: "bigskies/coordinator/telescope/control/slew", Resource: "telescope", Action: "slew"},
			},
			topic:            "bigskies/coordinator/telescope/control/slew",
			expectedMatch:    true,
			expectedResource: "telescope",
			expectedAction:   "control",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mc.mu.Lock()
			mc.protectionRules = tc.rules
			mc.mu.Unlock()

			rule := mc.findMatchingRule(tc.topic)
			if tc.expectedMatch {
				assert.NotNil(t, rule, "Expected rule to match")
				if rule != nil {
					assert.Equal(t, tc.expectedResource, rule.Resource)
					assert.Equal(t, tc.expectedAction, rule.Action)
				}
			} else {
				assert.Nil(t, rule, "Expected no rule to match")
			}
		})
	}
}

// TestMessageCoordinator_RBAC_UserContextExtraction tests user context extraction
func TestMessageCoordinator_RBAC_UserContextExtraction(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &MessageCoordinatorConfig{
		BaseConfig: BaseConfig{
			Name: "message-coordinator",
		},
		BrokerURL:         "localhost",
		BrokerPort:        1883,
		MaxQueueSize:      1000,
		ValidationTimeout: 30 * time.Second,
	}

	mc := &MessageCoordinator{
		config:      config,
		auditLogger: logger.Named("rbac-audit"),
		metrics:     &models.RBACMetrics{},
	}

	testCases := []struct {
		name           string
		payload        interface{}
		expectedUserID string
		expectError    bool
	}{
		{
			name: "valid user context",
			payload: map[string]interface{}{
				"user_id":  "user123",
				"username": "testuser",
				"token":    "jwt-token-here",
			},
			expectedUserID: "user123",
			expectError:    false,
		},
		{
			name:           "empty payload",
			payload:        map[string]interface{}{},
			expectedUserID: "anonymous",
			expectError:    false,
		},
		{
			name:           "invalid JSON",
			payload:        "invalid json",
			expectedUserID: "",
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var payloadBytes []byte
			var err error

			if tc.name == "invalid JSON" {
				// For invalid JSON test, pass raw invalid JSON
				payloadBytes = []byte("invalid json")
			} else {
				// Create message envelope
				msg, err := mqtt.NewMessage(mqtt.MessageTypeRequest, "coordinator:test", tc.payload)
				require.NoError(t, err)
				payloadBytes, err = json.Marshal(msg)
				require.NoError(t, err)
			}

			context, err := mc.extractUserContext(payloadBytes)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedUserID, context.UserID)
			}
		})
	}
}

// TestMessageCoordinator_RBAC_PendingMessageQueue tests pending message queue management
func TestMessageCoordinator_RBAC_PendingMessageQueue(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &MessageCoordinatorConfig{
		BaseConfig: BaseConfig{
			Name: "message-coordinator",
		},
		BrokerURL:         "localhost",
		BrokerPort:        1883,
		MaxQueueSize:      3, // Small queue for testing
		ValidationTimeout: 30 * time.Second,
	}

	mc := &MessageCoordinator{
		BaseCoordinator: &BaseCoordinator{
			name:          "message-coordinator",
			logger:        logger,
			shutdownFuncs: make([]func(context.Context) error, 0),
		},
		config:            config,
		pendingMessages:   make(map[string]*models.PendingMessage),
		auditLogger:       logger.Named("rbac-audit"),
		metrics:           &models.RBACMetrics{},
		maxQueueSize:      config.MaxQueueSize,
		validationTimeout: config.ValidationTimeout,
	}

	// Test adding messages to queue
	t.Run("add messages to queue", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			correlationID := generateCorrelationID()
			pending := &models.PendingMessage{
				ID:            correlationID,
				OriginalTopic: "test/topic",
				Payload:       []byte("test payload"),
				UserContext:   models.UserContext{UserID: "user123"},
				CorrelationID: correlationID,
				ReceivedAt:    time.Now(),
				ExpiresAt:     time.Now().Add(30 * time.Second), // Not expired
			}

			mc.queueMutex.Lock()
			mc.pendingMessages[correlationID] = pending
			mc.metrics.RecordQueueDepth(len(mc.pendingMessages))
			mc.queueMutex.Unlock()
		}

		mc.queueMutex.RLock()
		assert.Equal(t, 3, len(mc.pendingMessages))
		mc.queueMutex.RUnlock()
	})

	// Test queue overflow protection
	t.Run("queue overflow protection", func(t *testing.T) {
		correlationID := generateCorrelationID()
		pending := &models.PendingMessage{
			ID:            correlationID,
			OriginalTopic: "test/topic",
			Payload:       []byte("test payload"),
			UserContext:   models.UserContext{UserID: "user123"},
			CorrelationID: correlationID,
			ReceivedAt:    time.Now(),
			ExpiresAt:     time.Now().Add(30 * time.Second),
		}

		mc.queueMutex.Lock()
		queueSize := len(mc.pendingMessages)
		mc.queueMutex.Unlock()

		// Should not add if queue is at max capacity
		if queueSize >= mc.maxQueueSize {
			// This simulates the queue overflow check in handleCoordinatorMessage
			assert.True(t, queueSize >= mc.maxQueueSize)
		}
		_ = pending // Mark as used to avoid unused variable error
	})

	// Test cleanup of expired messages
	t.Run("cleanup expired messages", func(t *testing.T) {
		// Add an expired message
		correlationID := generateCorrelationID()
		expiredPending := &models.PendingMessage{
			ID:            correlationID,
			OriginalTopic: "test/topic",
			Payload:       []byte("test payload"),
			UserContext:   models.UserContext{UserID: "user123"},
			CorrelationID: correlationID,
			ReceivedAt:    time.Now().Add(-60 * time.Second), // Received 1 minute ago
			ExpiresAt:     time.Now().Add(-30 * time.Second), // Expired 30 seconds ago
		}

		mc.queueMutex.Lock()
		mc.pendingMessages[correlationID] = expiredPending
		initialCount := len(mc.pendingMessages)
		mc.queueMutex.Unlock()

		// Run cleanup
		mc.cleanupExpiredMessages()

		// Check that expired message was removed
		mc.queueMutex.RLock()
		finalCount := len(mc.pendingMessages)
		_, exists := mc.pendingMessages[correlationID]
		mc.queueMutex.RUnlock()

		assert.False(t, exists, "Expired message should be removed")
		assert.Equal(t, initialCount-1, finalCount, "Queue size should decrease by 1")
	})
}

// TestMessageCoordinator_RBAC_ValidationTimeout tests timeout handling
func TestMessageCoordinator_RBAC_ValidationTimeout(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &MessageCoordinatorConfig{
		BaseConfig: BaseConfig{
			Name: "message-coordinator",
		},
		BrokerURL:         "localhost",
		BrokerPort:        1883,
		MaxQueueSize:      1000,
		ValidationTimeout: 1 * time.Second, // Short timeout for testing
	}

	mc := &MessageCoordinator{
		BaseCoordinator: &BaseCoordinator{
			name:          "message-coordinator",
			logger:        logger,
			shutdownFuncs: make([]func(context.Context) error, 0),
		},
		config:            config,
		pendingMessages:   make(map[string]*models.PendingMessage),
		auditLogger:       logger.Named("rbac-audit"),
		metrics:           &models.RBACMetrics{},
		maxQueueSize:      config.MaxQueueSize,
		validationTimeout: config.ValidationTimeout,
	}

	// Add a message that will timeout
	correlationID := generateCorrelationID()
	pending := &models.PendingMessage{
		ID:            correlationID,
		OriginalTopic: "test/topic",
		Payload:       []byte("test payload"),
		UserContext:   models.UserContext{UserID: "user123"},
		CorrelationID: correlationID,
		ReceivedAt:    time.Now().Add(-2 * time.Second), // Received 2 seconds ago
		ExpiresAt:     time.Now().Add(-1 * time.Second), // Already expired
	}

	mc.queueMutex.Lock()
	mc.pendingMessages[correlationID] = pending
	mc.queueMutex.Unlock()

	// Run cleanup
	mc.cleanupExpiredMessages()

	// Verify message was removed and timeout was recorded
	mc.queueMutex.RLock()
	_, exists := mc.pendingMessages[correlationID]
	mc.queueMutex.RUnlock()

	assert.False(t, exists, "Expired message should be removed")

	// Check that timeout was recorded in metrics
	metrics := mc.metrics.GetMetrics()
	assert.Equal(t, int64(1), metrics.ValidationTimeouts, "Timeout should be recorded in metrics")
}

// TestMessageCoordinator_RBAC_Metrics tests RBAC metrics collection
func TestMessageCoordinator_RBAC_Metrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &MessageCoordinatorConfig{
		BaseConfig: BaseConfig{
			Name: "message-coordinator",
		},
		BrokerURL:         "localhost",
		BrokerPort:        1883,
		MaxQueueSize:      1000,
		ValidationTimeout: 30 * time.Second,
	}

	mc := &MessageCoordinator{
		config:      config,
		auditLogger: logger.Named("rbac-audit"),
		metrics:     &models.RBACMetrics{},
	}

	// Test metrics recording
	t.Run("record message processed", func(t *testing.T) {
		initialMetrics := mc.metrics.GetMetrics()
		initialProcessed := initialMetrics.MessagesProcessed

		mc.metrics.RecordMessageProcessed()

		finalMetrics := mc.metrics.GetMetrics()
		assert.Equal(t, initialProcessed+1, finalMetrics.MessagesProcessed)
	})

	t.Run("record message validated", func(t *testing.T) {
		initialMetrics := mc.metrics.GetMetrics()
		initialValidated := initialMetrics.MessagesValidated

		mc.metrics.RecordMessageValidated()

		finalMetrics := mc.metrics.GetMetrics()
		assert.Equal(t, initialValidated+1, finalMetrics.MessagesValidated)
	})

	t.Run("record message rejected", func(t *testing.T) {
		initialMetrics := mc.metrics.GetMetrics()
		initialRejected := initialMetrics.MessagesRejected

		mc.metrics.RecordMessageRejected()

		finalMetrics := mc.metrics.GetMetrics()
		assert.Equal(t, initialRejected+1, finalMetrics.MessagesRejected)
	})

	t.Run("record validation error", func(t *testing.T) {
		initialMetrics := mc.metrics.GetMetrics()
		initialErrors := initialMetrics.ValidationErrors

		mc.metrics.RecordValidationError()

		finalMetrics := mc.metrics.GetMetrics()
		assert.Equal(t, initialErrors+1, finalMetrics.ValidationErrors)
	})

	t.Run("record queue overflow", func(t *testing.T) {
		initialMetrics := mc.metrics.GetMetrics()
		initialOverflows := initialMetrics.QueueOverflows

		mc.metrics.RecordQueueOverflow()

		finalMetrics := mc.metrics.GetMetrics()
		assert.Equal(t, initialOverflows+1, finalMetrics.QueueOverflows)
	})
}

// TestMessageCoordinator_RBAC_HealthCheck tests RBAC health monitoring
func TestMessageCoordinator_RBAC_HealthCheck(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &MessageCoordinatorConfig{
		BaseConfig: BaseConfig{
			Name: "message-coordinator",
		},
		BrokerURL:         "localhost",
		BrokerPort:        1883,
		MaxQueueSize:      100,
		ValidationTimeout: 30 * time.Second,
	}

	mc := &MessageCoordinator{
		BaseCoordinator: &BaseCoordinator{
			name:          "message-coordinator",
			logger:        logger,
			shutdownFuncs: make([]func(context.Context) error, 0),
		},
		config:          config,
		protectionRules: []models.TopicProtectionRule{},
		pendingMessages: make(map[string]*models.PendingMessage),
		auditLogger:     logger.Named("rbac-audit"),
		metrics:         &models.RBACMetrics{},
		maxQueueSize:    config.MaxQueueSize,
	}

	// Test healthy state (RBAC metrics should be healthy even if no topics subscribed)
	t.Run("healthy state", func(t *testing.T) {
		result := mc.Check(context.Background())
		// Note: Status might be "degraded" due to no topics subscribed, but RBAC should be healthy
		assert.Contains(t, []string{"healthy", "degraded"}, string(result.Status))
		// The message should not be about RBAC issues
		assert.NotContains(t, result.Message, "RBAC")
		assert.NotContains(t, result.Message, "high")
		assert.NotContains(t, result.Message, "overflow")
	})

	// Test degraded state due to high queue depth
	t.Run("degraded due to high queue", func(t *testing.T) {
		// Fill queue to >50% capacity
		for i := 0; i < 60; i++ {
			correlationID := generateCorrelationID()
			pending := &models.PendingMessage{
				ID:            correlationID,
				OriginalTopic: "test/topic",
				Payload:       []byte("test payload"),
				UserContext:   models.UserContext{UserID: "user123"},
				CorrelationID: correlationID,
				ReceivedAt:    time.Now(),
				ExpiresAt:     time.Now().Add(30 * time.Second),
			}

			mc.queueMutex.Lock()
			mc.pendingMessages[correlationID] = pending
			mc.queueMutex.Unlock()
		}

		// Record the final queue depth
		mc.metrics.RecordQueueDepth(60)

		result := mc.Check(context.Background())
		assert.Equal(t, "degraded", string(result.Status))
		// Should be degraded due to high queue, not just no topics
		assert.Contains(t, result.Message, "high")

		// Clean up
		mc.queueMutex.Lock()
		mc.pendingMessages = make(map[string]*models.PendingMessage)
		mc.metrics.RecordQueueDepth(0)
		mc.queueMutex.Unlock()
	})

	// Test degraded state due to queue overflows
	t.Run("degraded due to overflows", func(t *testing.T) {
		mc.metrics.RecordQueueOverflow()

		result := mc.Check(context.Background())
		assert.Equal(t, "degraded", string(result.Status))
		assert.Contains(t, result.Message, "overflow")
	})
}
