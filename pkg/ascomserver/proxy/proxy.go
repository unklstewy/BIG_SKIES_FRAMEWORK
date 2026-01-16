// Package proxy provides backend connection implementations for the ASCOM Alpaca server.
// This package defines interfaces and implementations for forwarding ASCOM commands
// to actual telescope hardware via different connection methods (network, MQTT, direct serial).
package proxy

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// DeviceProxy is the interface that all backend proxy implementations must satisfy.
// A proxy is responsible for forwarding ASCOM API requests to actual hardware
// and returning the responses back to the ASCOM server.
//
// Different proxy implementations handle different connection types:
//   - NetworkProxy: Forwards requests to remote ASCOM Alpaca servers via HTTP
//   - MQTTProxy: Sends commands via MQTT to BigSkies telescope coordinator
//   - DirectProxy: Communicates directly with hardware via serial/USB (future)
//
// All proxies must implement these methods to provide a consistent interface
// regardless of the underlying connection mechanism.
type DeviceProxy interface {
	// Connect establishes a connection to the backend device/service.
	// This should perform any necessary initialization, authentication,
	// and connection establishment. It should be idempotent - calling
	// Connect multiple times should not cause issues.
	//
	// Returns an error if the connection cannot be established.
	Connect(ctx context.Context) error

	// Disconnect closes the connection to the backend device/service.
	// This should cleanly release all resources and close connections.
	// It should be safe to call even if not connected.
	//
	// Returns an error if disconnection fails, though the connection
	// will still be considered disconnected.
	Disconnect(ctx context.Context) error

	// IsConnected returns true if the proxy is currently connected
	// to the backend device/service. This should reflect the actual
	// connection state, not just whether Connect was called.
	IsConnected() bool

	// Get executes a GET request (read operation) on the backend device.
	// This is used for ASCOM endpoints that query device state or properties.
	//
	// Parameters:
	//   - ctx: Context for timeout and cancellation
	//   - method: ASCOM method name (e.g., "rightascension", "altitude")
	//   - params: Query parameters to include with the request
	//
	// Returns:
	//   - value: The value returned by the backend (type varies by method)
	//   - error: Any error that occurred during the request
	//
	// Example:
	//   value, err := proxy.Get(ctx, "rightascension", map[string]string{
	//     "ClientID": "1",
	//     "ClientTransactionID": "123",
	//   })
	Get(ctx context.Context, method string, params map[string]string) (interface{}, error)

	// Put executes a PUT request (write operation) on the backend device.
	// This is used for ASCOM endpoints that change device state or settings.
	//
	// Parameters:
	//   - ctx: Context for timeout and cancellation
	//   - method: ASCOM method name (e.g., "slewtocoordinates", "tracking")
	//   - params: Form parameters to include with the request
	//
	// Returns:
	//   - value: The value returned by the backend (often nil for commands)
	//   - error: Any error that occurred during the request
	//
	// Example:
	//   _, err := proxy.Put(ctx, "tracking", map[string]string{
	//     "Tracking": "true",
	//     "ClientID": "1",
	//     "ClientTransactionID": "123",
	//   })
	Put(ctx context.Context, method string, params map[string]string) (interface{}, error)

	// HealthCheck performs a health check on the backend connection.
	// This should verify that the connection is alive and the backend
	// is responding. It's used by the connection pool to detect and
	// recover from connection failures.
	//
	// Returns an error if the health check fails.
	HealthCheck(ctx context.Context) error

	// GetMetrics returns current metrics about the proxy's performance.
	// This includes request counts, error rates, latency, etc.
	// Used for monitoring and debugging.
	GetMetrics() *ProxyMetrics
}

// ProxyMetrics contains performance and health metrics for a proxy.
// These metrics help monitor the health and performance of backend connections.
type ProxyMetrics struct {
	// TotalRequests is the total number of requests sent through this proxy.
	TotalRequests int64

	// SuccessfulRequests is the number of requests that completed successfully.
	SuccessfulRequests int64

	// FailedRequests is the number of requests that returned an error.
	FailedRequests int64

	// LastRequestTime is the timestamp of the most recent request.
	LastRequestTime time.Time

	// LastSuccessTime is the timestamp of the most recent successful request.
	LastSuccessTime time.Time

	// LastFailureTime is the timestamp of the most recent failed request.
	LastFailureTime time.Time

	// AverageLatency is the average request latency in milliseconds.
	AverageLatency float64

	// ConnectionState describes the current connection state.
	ConnectionState string // "connected", "disconnected", "error", "reconnecting"

	// LastError is the most recent error message, if any.
	LastError string
}

// ProxyConfig contains common configuration for all proxy types.
// This base configuration is extended by specific proxy implementations.
type ProxyConfig struct {
	// DeviceType is the ASCOM device type (telescope, camera, etc.)
	DeviceType string

	// DeviceNumber is the device number (0-based)
	DeviceNumber int

	// Timeout is the request timeout duration.
	// If a request takes longer than this, it will be cancelled.
	Timeout time.Duration

	// RetryAttempts is the number of times to retry failed requests.
	// A value of 0 means no retries (only the initial attempt).
	RetryAttempts int

	// RetryDelay is the delay between retry attempts.
	// This can be used with exponential backoff strategies.
	RetryDelay time.Duration

	// Logger is the structured logger for this proxy.
	Logger *zap.Logger
}

// ProxyFactory is a function type that creates a new proxy instance.
// This allows different proxy types to be created dynamically based on configuration.
type ProxyFactory func(config interface{}, logger *zap.Logger) (DeviceProxy, error)

// Registry maintains a registry of proxy factories.
// This allows proxy types to be registered and created by name.
var proxyFactories = make(map[string]ProxyFactory)

// RegisterProxyFactory registers a proxy factory for a given backend mode.
// This allows new proxy types to be added without modifying the core server code.
//
// Parameters:
//   - mode: Backend mode identifier (e.g., "network", "mqtt", "direct")
//   - factory: Factory function that creates proxy instances
//
// Example:
//
//	RegisterProxyFactory("network", func(config interface{}, logger *zap.Logger) (DeviceProxy, error) {
//	    netConfig := config.(*NetworkProxyConfig)
//	    return NewNetworkProxy(netConfig, logger)
//	})
func RegisterProxyFactory(mode string, factory ProxyFactory) {
	proxyFactories[mode] = factory
}

// CreateProxy creates a new proxy instance for the given mode and configuration.
// This is the primary way to instantiate proxies in the ASCOM server.
//
// Parameters:
//   - mode: Backend mode ("network", "mqtt", "direct")
//   - config: Mode-specific configuration (must match the mode)
//   - logger: Structured logger for the proxy
//
// Returns:
//   - DeviceProxy: The created proxy instance
//   - error: Error if the mode is not registered or creation fails
//
// Example:
//
//	proxy, err := CreateProxy("network", networkConfig, logger)
//	if err != nil {
//	    return fmt.Errorf("failed to create proxy: %w", err)
//	}
func CreateProxy(mode string, config interface{}, logger *zap.Logger) (DeviceProxy, error) {
	factory, exists := proxyFactories[mode]
	if !exists {
		return nil, ErrUnknownProxyMode{Mode: mode}
	}

	return factory(config, logger)
}

// ProxyError is a custom error type for proxy-related errors.
// This allows callers to distinguish between different types of failures.
type ProxyError struct {
	// Operation is the operation that failed (e.g., "Connect", "Get", "Put")
	Operation string

	// Backend is a description of the backend (e.g., "http://telescope:11111")
	Backend string

	// Message is a human-readable error message
	Message string

	// Err is the underlying error, if any
	Err error
}

// Error implements the error interface for ProxyError.
func (e ProxyError) Error() string {
	if e.Err != nil {
		return e.Operation + " failed for " + e.Backend + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Operation + " failed for " + e.Backend + ": " + e.Message
}

// Unwrap returns the underlying error for error chain support.
func (e ProxyError) Unwrap() error {
	return e.Err
}

// ErrUnknownProxyMode is returned when an unknown proxy mode is requested.
type ErrUnknownProxyMode struct {
	Mode string
}

// Error implements the error interface for ErrUnknownProxyMode.
func (e ErrUnknownProxyMode) Error() string {
	return "unknown proxy mode: " + e.Mode
}

// Common proxy errors
var (
	// ErrNotConnected is returned when an operation requires a connection but none exists.
	ErrNotConnected = ProxyError{
		Operation: "Request",
		Message:   "proxy is not connected",
	}

	// ErrTimeout is returned when a request times out.
	ErrTimeout = ProxyError{
		Operation: "Request",
		Message:   "request timed out",
	}

	// ErrBackendUnavailable is returned when the backend service is not available.
	ErrBackendUnavailable = ProxyError{
		Operation: "Connect",
		Message:   "backend service unavailable",
	}
)
