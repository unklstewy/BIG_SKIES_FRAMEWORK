// Package healthcheck provides interfaces and types for health monitoring.
package healthcheck

import (
	"context"
	"time"
)

// Status represents the health status of a component.
type Status string

const (
	// StatusHealthy indicates the component is functioning normally
	StatusHealthy Status = "healthy"
	// StatusDegraded indicates the component is functioning but with issues
	StatusDegraded Status = "degraded"
	// StatusUnhealthy indicates the component is not functioning properly
	StatusUnhealthy Status = "unhealthy"
	// StatusUnknown indicates the health status cannot be determined
	StatusUnknown Status = "unknown"
)

// Result contains the health check result for a component.
type Result struct {
	// ComponentName identifies the component being checked
	ComponentName string `json:"component"`
	// Status is the health status
	Status Status `json:"status"`
	// Message provides additional context about the health status
	Message string `json:"message,omitempty"`
	// Timestamp when the check was performed
	Timestamp time.Time `json:"timestamp"`
	// Duration of the health check
	Duration time.Duration `json:"duration"`
	// Details contains component-specific health information
	Details map[string]interface{} `json:"details,omitempty"`
}

// Checker is the interface that components must implement for health checking.
type Checker interface {
	// Check performs a health check and returns the result
	Check(ctx context.Context) *Result
	// Name returns the name of the component being checked
	Name() string
}

// CheckerFunc is an adapter to allow ordinary functions to be used as Checkers.
type CheckerFunc func(ctx context.Context) *Result

// Check calls the underlying function.
func (f CheckerFunc) Check(ctx context.Context) *Result {
	return f(ctx)
}

// Name returns a default name for function-based checkers.
func (f CheckerFunc) Name() string {
	return "checker-func"
}

// AggregatedResult contains health check results from multiple components.
type AggregatedResult struct {
	// OverallStatus is the aggregated health status
	OverallStatus Status `json:"status"`
	// Components contains individual component health results
	Components map[string]*Result `json:"components"`
	// Timestamp when the aggregation was performed
	Timestamp time.Time `json:"timestamp"`
}

// IsHealthy returns true if the overall status is healthy.
func (ar *AggregatedResult) IsHealthy() bool {
	return ar.OverallStatus == StatusHealthy
}

// IsDegraded returns true if the overall status is degraded.
func (ar *AggregatedResult) IsDegraded() bool {
	return ar.OverallStatus == StatusDegraded
}

// IsUnhealthy returns true if the overall status is unhealthy.
func (ar *AggregatedResult) IsUnhealthy() bool {
	return ar.OverallStatus == StatusUnhealthy
}

// DetermineOverallStatus calculates the overall status from component results.
func DetermineOverallStatus(results map[string]*Result) Status {
	if len(results) == 0 {
		return StatusUnknown
	}

	hasUnhealthy := false
	hasDegraded := false

	for _, result := range results {
		switch result.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		case StatusUnknown:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return StatusUnhealthy
	}
	if hasDegraded {
		return StatusDegraded
	}
	return StatusHealthy
}
