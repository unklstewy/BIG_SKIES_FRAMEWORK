// Package mqtt defines message envelope structures for MQTT communication.
package mqtt

import (
	"encoding/json"
	"time"
)

// MessageType represents the type of message being sent.
type MessageType string

const (
	// MessageTypeCommand represents a command message
	MessageTypeCommand MessageType = "command"
	// MessageTypeEvent represents an event message
	MessageTypeEvent MessageType = "event"
	// MessageTypeRequest represents a request message
	MessageTypeRequest MessageType = "request"
	// MessageTypeResponse represents a response message
	MessageTypeResponse MessageType = "response"
	// MessageTypeStatus represents a status update
	MessageTypeStatus MessageType = "status"
)

// Message is the envelope structure for all MQTT messages in the framework.
type Message struct {
	// ID is a unique identifier for this message
	ID string `json:"id"`
	// Type indicates the message type
	Type MessageType `json:"type"`
	// Source identifies the sender (e.g., "coordinator:message")
	Source string `json:"source"`
	// Timestamp when the message was created
	Timestamp time.Time `json:"timestamp"`
	// CorrelationID links related messages (e.g., request/response)
	CorrelationID string `json:"correlation_id,omitempty"`
	// Payload contains the actual message data as JSON
	Payload json.RawMessage `json:"payload"`
}

// NewMessage creates a new message with the given parameters.
func NewMessage(msgType MessageType, source string, payload interface{}) (*Message, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &Message{
		ID:        GenerateMessageID(),
		Type:      msgType,
		Source:    source,
		Timestamp: time.Now().UTC(),
		Payload:   payloadBytes,
	}, nil
}

// UnmarshalPayload deserializes the payload into the provided structure.
func (m *Message) UnmarshalPayload(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}

// CommandMessage represents a command to be executed.
type CommandMessage struct {
	Command string                 `json:"command"`
	Args    map[string]interface{} `json:"args,omitempty"`
}

// EventMessage represents an event notification.
type EventMessage struct {
	Event string                 `json:"event"`
	Data  map[string]interface{} `json:"data,omitempty"`
}

// StatusMessage represents a status update.
type StatusMessage struct {
	State   string                 `json:"state"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ResponseMessage represents a response to a request.
type ResponseMessage struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// GenerateMessageID generates a unique message ID.
// In production, use UUID or similar. This is a simple implementation.
func GenerateMessageID() string {
	return time.Now().Format("20060102150405.000000")
}
