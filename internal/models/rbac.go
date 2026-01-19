// Package models provides data structures for the BIG SKIES Framework.
package models

import (
	"sync"
	"time"
)

// TopicProtectionRule defines which topics require RBAC validation and what permissions are needed.
type TopicProtectionRule struct {
	ID           string    `json:"id" db:"id"`
	TopicPattern string    `json:"topic_pattern" db:"topic_pattern"` // e.g., "bigskies/coordinator/telescope/+/slew"
	Resource     string    `json:"resource" db:"resource"`           // e.g., "telescope"
	Action       string    `json:"action" db:"action"`               // e.g., "control"
	Enabled      bool      `json:"enabled" db:"enabled"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// RBACValidationRequest is sent to security coordinator for permission validation.
type RBACValidationRequest struct {
	CorrelationID string      `json:"correlation_id"`
	UserID        string      `json:"user_id"`
	Resource      string      `json:"resource"`
	Action        string      `json:"action"`
	Context       UserContext `json:"context"`
	Timestamp     time.Time   `json:"timestamp"`
}

// RBACValidationResponse is received from security coordinator.
type RBACValidationResponse struct {
	CorrelationID string    `json:"correlation_id"`
	Allowed       bool      `json:"allowed"`
	Reason        string    `json:"reason,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// PendingMessage represents a message waiting for RBAC validation.
type PendingMessage struct {
	ID            string      `json:"id"`
	OriginalTopic string      `json:"original_topic"`
	Payload       []byte      `json:"payload"`
	UserContext   UserContext `json:"user_context"`
	CorrelationID string      `json:"correlation_id"`
	ReceivedAt    time.Time   `json:"received_at"`
	ExpiresAt     time.Time   `json:"expires_at"`
}

// UserContext contains authentication and authorization data extracted from messages.
type UserContext struct {
	UserID   string                 `json:"user_id"`
	Username string                 `json:"username"`
	Token    string                 `json:"token,omitempty"` // JWT token if present
	Roles    []string               `json:"roles,omitempty"`
	Groups   []string               `json:"groups,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Additional context
}

// RBACMetrics tracks performance and health metrics for RBAC operations.
type RBACMetrics struct {
	// Message processing metrics
	MessagesProcessed  int64 `json:"messages_processed"`
	MessagesValidated  int64 `json:"messages_validated"`
	MessagesRejected   int64 `json:"messages_rejected"`
	MessagesForwarded  int64 `json:"messages_forwarded"`
	ValidationTimeouts int64 `json:"validation_timeouts"`
	// Queue metrics
	CurrentQueueDepth int   `json:"current_queue_depth"`
	MaxQueueDepth     int   `json:"max_queue_depth"`
	QueueOverflows    int64 `json:"queue_overflows"`
	// Performance metrics
	AvgValidationTime time.Duration `json:"avg_validation_time"`
	MinValidationTime time.Duration `json:"min_validation_time"`
	MaxValidationTime time.Duration `json:"max_validation_time"`
	// Error metrics
	ValidationErrors  int64 `json:"validation_errors"`
	CoordinatorErrors int64 `json:"coordinator_errors"`
	// Health status
	LastHealthCheck time.Time    `json:"last_health_check"`
	IsHealthy       bool         `json:"is_healthy"`
	mu              sync.RWMutex `json:"-"`
}

// RecordMessageProcessed increments the message processed counter.
func (m *RBACMetrics) RecordMessageProcessed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MessagesProcessed++
}

// RecordMessageValidated increments the message validated counter.
func (m *RBACMetrics) RecordMessageValidated() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MessagesValidated++
}

// RecordMessageRejected increments the message rejected counter.
func (m *RBACMetrics) RecordMessageRejected() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MessagesRejected++
}

// RecordMessageForwarded increments the message forwarded counter.
func (m *RBACMetrics) RecordMessageForwarded() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MessagesForwarded++
}

// RecordValidationTimeout increments the validation timeout counter.
func (m *RBACMetrics) RecordValidationTimeout() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ValidationTimeouts++
}

// RecordQueueDepth updates the current queue depth and max queue depth.
func (m *RBACMetrics) RecordQueueDepth(depth int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CurrentQueueDepth = depth
	if depth > m.MaxQueueDepth {
		m.MaxQueueDepth = depth
	}
}

// RecordQueueOverflow increments the queue overflow counter.
func (m *RBACMetrics) RecordQueueOverflow() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.QueueOverflows++
}

// RecordValidationTime records a validation time measurement.
func (m *RBACMetrics) RecordValidationTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update min/max
	if m.MinValidationTime == 0 || duration < m.MinValidationTime {
		m.MinValidationTime = duration
	}
	if duration > m.MaxValidationTime {
		m.MaxValidationTime = duration
	}

	// Update average (simple moving average)
	if m.MessagesValidated > 0 {
		totalTime := m.AvgValidationTime * time.Duration(m.MessagesValidated-1)
		m.AvgValidationTime = (totalTime + duration) / time.Duration(m.MessagesValidated)
	}
}

// RecordValidationError increments the validation error counter.
func (m *RBACMetrics) RecordValidationError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ValidationErrors++
}

// RecordCoordinatorError increments the coordinator error counter.
func (m *RBACMetrics) RecordCoordinatorError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CoordinatorErrors++
}

// UpdateHealthStatus updates the health check timestamp and status.
func (m *RBACMetrics) UpdateHealthStatus(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LastHealthCheck = time.Now()
	m.IsHealthy = healthy
}

// GetMetrics returns a copy of the current metrics.
func (m *RBACMetrics) GetMetrics() RBACMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m
}
