// Package mqtt defines topic conventions for the BIG SKIES Framework.
package mqtt

import (
	"fmt"
	"strings"
)

// Topic naming conventions for BIG SKIES Framework.
// Format: bigskies/{component}/{action}/{resource}
const (
	// TopicPrefix is the root prefix for all framework topics
	TopicPrefix = "bigskies"

	// Component topics
	ComponentCoordinator = "coordinator"
	ComponentEngine      = "engine"
	ComponentService     = "service"
	ComponentPlugin      = "plugin"

	// Actions
	ActionCommand  = "cmd"
	ActionEvent    = "event"
	ActionStatus   = "status"
	ActionHealth   = "health"
	ActionConfig   = "config"
	ActionRequest  = "req"
	ActionResponse = "resp"

	// Coordinators
	CoordinatorMessage     = "message"
	CoordinatorSecurity    = "security"
	CoordinatorDataStore   = "datastore"
	CoordinatorApplication = "application"
	CoordinatorPlugin      = "plugin"
	CoordinatorTelescope   = "telescope"
	CoordinatorUIElement   = "uielement"
)

// TopicBuilder helps construct topic strings following conventions.
type TopicBuilder struct {
	parts []string
}

// NewTopicBuilder creates a new topic builder starting with the framework prefix.
func NewTopicBuilder() *TopicBuilder {
	return &TopicBuilder{
		parts: []string{TopicPrefix},
	}
}

// Component adds a component segment.
func (tb *TopicBuilder) Component(comp string) *TopicBuilder {
	tb.parts = append(tb.parts, ComponentCoordinator, comp)
	return tb
}

// Action adds an action segment.
func (tb *TopicBuilder) Action(action string) *TopicBuilder {
	tb.parts = append(tb.parts, action)
	return tb
}

// Resource adds a resource segment.
func (tb *TopicBuilder) Resource(resource string) *TopicBuilder {
	tb.parts = append(tb.parts, resource)
	return tb
}

// Build constructs the final topic string.
func (tb *TopicBuilder) Build() string {
	return strings.Join(tb.parts, "/")
}

// Common topic patterns

// CoordinatorHealthTopic returns the health check topic for a coordinator.
func CoordinatorHealthTopic(coordinator string) string {
	return NewTopicBuilder().
		Component(coordinator).
		Action(ActionHealth).
		Resource("status").
		Build()
}

// CoordinatorStatusTopic returns the status topic for a coordinator.
func CoordinatorStatusTopic(coordinator string) string {
	return NewTopicBuilder().
		Component(coordinator).
		Action(ActionStatus).
		Build()
}

// CoordinatorCommandTopic returns the command topic for a coordinator.
func CoordinatorCommandTopic(coordinator string) string {
	return NewTopicBuilder().
		Component(coordinator).
		Action(ActionCommand).
		Build()
}

// CoordinatorEventTopic returns the event topic for a coordinator.
func CoordinatorEventTopic(coordinator string, eventType string) string {
	return NewTopicBuilder().
		Component(coordinator).
		Action(ActionEvent).
		Resource(eventType).
		Build()
}

// ParseTopic extracts components from a topic string.
func ParseTopic(topic string) ([]string, error) {
	parts := strings.Split(topic, "/")
	if len(parts) < 2 || parts[0] != TopicPrefix {
		return nil, fmt.Errorf("invalid topic format: must start with %s", TopicPrefix)
	}
	return parts[1:], nil
}

// ValidateTopic checks if a topic follows framework conventions.
func ValidateTopic(topic string) bool {
	parts := strings.Split(topic, "/")
	return len(parts) >= 3 && parts[0] == TopicPrefix
}
