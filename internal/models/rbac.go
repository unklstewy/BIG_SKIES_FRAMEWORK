// Package models provides data structures for the BIG SKIES Framework.
package models

import (
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
