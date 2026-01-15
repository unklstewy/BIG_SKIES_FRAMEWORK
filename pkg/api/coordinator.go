// Package api defines core interfaces for the BIG SKIES Framework.
package api

import (
	"context"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/healthcheck"
)

// Coordinator is the base interface that all coordinators must implement.
type Coordinator interface {
	// Name returns the unique name of the coordinator
	Name() string

	// Start initializes and starts the coordinator
	Start(ctx context.Context) error

	// Stop gracefully shuts down the coordinator
	Stop(ctx context.Context) error

	// HealthCheck returns the health status of the coordinator
	HealthCheck(ctx context.Context) *healthcheck.Result

	// IsRunning returns true if the coordinator is currently running
	IsRunning() bool
}

// Configurable is an interface for components that can be configured.
type Configurable interface {
	// LoadConfig loads configuration from the provided source
	LoadConfig(config interface{}) error

	// ValidateConfig validates the current configuration
	ValidateConfig() error

	// GetConfig returns the current configuration
	GetConfig() interface{}
}

// Lifecycle manages component lifecycle states.
type Lifecycle interface {
	// Initialize prepares the component for operation
	Initialize(ctx context.Context) error

	// Shutdown performs cleanup and releases resources
	Shutdown(ctx context.Context) error
}
