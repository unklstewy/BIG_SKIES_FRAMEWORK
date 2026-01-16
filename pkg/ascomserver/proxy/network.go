package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// NetworkProxy forwards ASCOM API requests to a remote ASCOM Alpaca server via HTTP.
// This allows the ASCOM reflector to act as a proxy to existing ASCOM servers,
// enabling remote access, load balancing, or protocol translation.
//
// The NetworkProxy maintains an HTTP client with connection pooling and timeout management.
// It tracks metrics and provides health checking to detect backend failures.
type NetworkProxy struct {
	// config contains the network proxy configuration
	config *NetworkProxyConfig

	// logger provides structured logging
	logger *zap.Logger

	// httpClient is the HTTP client used for backend requests
	httpClient *http.Client

	// connected indicates whether the proxy is connected and healthy
	connected atomic.Bool

	// metrics tracks proxy performance and health
	metrics ProxyMetrics

	// metricsMu protects access to metrics
	metricsMu sync.RWMutex
}

// NetworkProxyConfig contains configuration specific to network proxies.
type NetworkProxyConfig struct {
	// Base configuration (timeout, retry, etc.)
	ProxyConfig

	// ServerURL is the base URL of the remote ASCOM Alpaca server.
	// Example: "http://192.168.1.100:11111" or "https://telescope.local:11111"
	ServerURL string

	// RemoteDeviceType is the device type on the remote server.
	// Usually the same as DeviceType, but allows for device type mapping.
	RemoteDeviceType string

	// RemoteDeviceNumber is the device number on the remote server.
	RemoteDeviceNumber int

	// ClientID is the ASCOM client ID to use in requests.
	// This identifies this proxy to the remote server.
	ClientID int32
}

// NewNetworkProxy creates a new network proxy instance.
//
// Parameters:
//   - config: Network proxy configuration
//   - logger: Structured logger (if nil, a no-op logger is used)
//
// Returns a configured NetworkProxy ready to connect.
func NewNetworkProxy(config *NetworkProxyConfig, logger *zap.Logger) (*NetworkProxy, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Validate configuration
	if config.ServerURL == "" {
		return nil, fmt.Errorf("server URL is required")
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}

	// Ensure RemoteDeviceType defaults to DeviceType if not specified
	if config.RemoteDeviceType == "" {
		config.RemoteDeviceType = config.DeviceType
	}

	// Create HTTP client with configured timeout and connection pooling
	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	proxy := &NetworkProxy{
		config:     config,
		logger:     logger.With(zap.String("component", "network_proxy")),
		httpClient: httpClient,
		metrics: ProxyMetrics{
			ConnectionState: "disconnected",
		},
	}

	proxy.connected.Store(false)

	return proxy, nil
}

// Connect establishes a connection to the remote ASCOM server.
// This performs a health check to verify the server is reachable and responding.
func (n *NetworkProxy) Connect(ctx context.Context) error {
	n.logger.Info("Connecting to remote ASCOM server",
		zap.String("server_url", n.config.ServerURL),
		zap.String("device_type", n.config.RemoteDeviceType),
		zap.Int("device_number", n.config.RemoteDeviceNumber))

	// Perform a health check to verify connectivity
	if err := n.HealthCheck(ctx); err != nil {
		n.updateConnectionState("error", err.Error())
		return ProxyError{
			Operation: "Connect",
			Backend:   n.config.ServerURL,
			Message:   "health check failed",
			Err:       err,
		}
	}

	n.connected.Store(true)
	n.updateConnectionState("connected", "")

	n.logger.Info("Successfully connected to remote ASCOM server")
	return nil
}

// Disconnect closes the connection to the remote server.
// For HTTP connections, this primarily means marking the proxy as disconnected.
func (n *NetworkProxy) Disconnect(ctx context.Context) error {
	n.logger.Info("Disconnecting from remote ASCOM server")

	n.connected.Store(false)
	n.updateConnectionState("disconnected", "")

	// Close idle connections in the HTTP client's transport
	n.httpClient.CloseIdleConnections()

	return nil
}

// IsConnected returns true if the proxy is connected to the backend.
func (n *NetworkProxy) IsConnected() bool {
	return n.connected.Load()
}

// Get executes a GET request on the remote ASCOM device.
// This retrieves a property or state value from the device.
func (n *NetworkProxy) Get(ctx context.Context, method string, params map[string]string) (interface{}, error) {
	startTime := time.Now()

	n.logger.Debug("Executing GET request",
		zap.String("method", method),
		zap.Any("params", params))

	// Build the URL for the request
	requestURL := n.buildURL(method)

	// Add query parameters
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, n.handleError("Get", err)
	}

	query := reqURL.Query()
	for key, value := range params {
		query.Add(key, value)
	}
	reqURL.RawQuery = query.Encode()

	// Execute the request with retries
	var response interface{}
	err = n.executeWithRetry(ctx, func() error {
		var execErr error
		response, execErr = n.doHTTPRequest(ctx, "GET", reqURL.String(), nil)
		return execErr
	})

	// Update metrics
	n.updateMetrics(startTime, err == nil)

	if err != nil {
		return nil, err
	}

	return response, nil
}

// Put executes a PUT request on the remote ASCOM device.
// This sets a property or executes a command on the device.
func (n *NetworkProxy) Put(ctx context.Context, method string, params map[string]string) (interface{}, error) {
	startTime := time.Now()

	n.logger.Debug("Executing PUT request",
		zap.String("method", method),
		zap.Any("params", params))

	// Build the URL for the request
	requestURL := n.buildURL(method)

	// Prepare form data
	formData := url.Values{}
	for key, value := range params {
		formData.Add(key, value)
	}

	// Execute the request with retries
	var response interface{}
	err := n.executeWithRetry(ctx, func() error {
		var execErr error
		response, execErr = n.doHTTPRequest(ctx, "PUT", requestURL, strings.NewReader(formData.Encode()))
		return execErr
	})

	// Update metrics
	n.updateMetrics(startTime, err == nil)

	if err != nil {
		return nil, err
	}

	return response, nil
}

// HealthCheck verifies the remote server is accessible and responding.
// This queries the "connected" property which all ASCOM devices must support.
func (n *NetworkProxy) HealthCheck(ctx context.Context) error {
	n.logger.Debug("Performing health check")

	// Try to query the "connected" property as a basic health check.
	// All ASCOM devices must implement this property.
	requestURL := n.buildURL("connected")

	// Add minimal query parameters
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return err
	}

	query := reqURL.Query()
	query.Add("ClientID", fmt.Sprintf("%d", n.config.ClientID))
	query.Add("ClientTransactionID", "0")
	reqURL.RawQuery = query.Encode()

	// Execute health check request
	_, err = n.doHTTPRequest(ctx, "GET", reqURL.String(), nil)
	return err
}

// GetMetrics returns current proxy metrics.
func (n *NetworkProxy) GetMetrics() *ProxyMetrics {
	n.metricsMu.RLock()
	defer n.metricsMu.RUnlock()

	// Return a copy of the metrics
	metricsCopy := n.metrics
	return &metricsCopy
}

// buildURL constructs the full URL for an ASCOM API endpoint.
// Format: {ServerURL}/api/v1/{device_type}/{device_number}/{method}
func (n *NetworkProxy) buildURL(method string) string {
	return fmt.Sprintf("%s/api/v1/%s/%d/%s",
		n.config.ServerURL,
		n.config.RemoteDeviceType,
		n.config.RemoteDeviceNumber,
		method)
}

// doHTTPRequest executes an HTTP request and parses the ASCOM response.
// This handles the low-level HTTP communication and ASCOM protocol parsing.
func (n *NetworkProxy) doHTTPRequest(ctx context.Context, httpMethod, url string, body io.Reader) (interface{}, error) {
	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, httpMethod, url, body)
	if err != nil {
		return nil, err
	}

	// Set appropriate content type for PUT requests
	if httpMethod == "PUT" && body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// Execute the request
	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse ASCOM response
	var ascomResp struct {
		Value        interface{} `json:"Value"`
		ErrorNumber  int         `json:"ErrorNumber"`
		ErrorMessage string      `json:"ErrorMessage"`
	}

	if err := json.Unmarshal(respBody, &ascomResp); err != nil {
		return nil, fmt.Errorf("failed to parse ASCOM response: %w", err)
	}

	// Check for ASCOM errors
	if ascomResp.ErrorNumber != 0 {
		return nil, fmt.Errorf("ASCOM error %d: %s", ascomResp.ErrorNumber, ascomResp.ErrorMessage)
	}

	return ascomResp.Value, nil
}

// executeWithRetry executes an operation with automatic retry on failure.
// This implements exponential backoff for transient failures.
func (n *NetworkProxy) executeWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	// Try the operation (initial attempt + retries)
	for attempt := 0; attempt <= n.config.RetryAttempts; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the operation
		lastErr = operation()
		if lastErr == nil {
			return nil // Success!
		}

		// Log the failure
		n.logger.Warn("Request attempt failed",
			zap.Int("attempt", attempt+1),
			zap.Int("max_attempts", n.config.RetryAttempts+1),
			zap.Error(lastErr))

		// Don't retry on the last attempt
		if attempt < n.config.RetryAttempts {
			// Wait before retrying (exponential backoff)
			delay := n.config.RetryDelay * time.Duration(1<<uint(attempt))
			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// All attempts failed
	return n.handleError("Request", lastErr)
}

// handleError wraps an error in a ProxyError with context.
func (n *NetworkProxy) handleError(operation string, err error) error {
	return ProxyError{
		Operation: operation,
		Backend:   n.config.ServerURL,
		Message:   "request failed",
		Err:       err,
	}
}

// updateMetrics updates proxy performance metrics.
func (n *NetworkProxy) updateMetrics(startTime time.Time, success bool) {
	n.metricsMu.Lock()
	defer n.metricsMu.Unlock()

	// Update request counts
	n.metrics.TotalRequests++
	if success {
		n.metrics.SuccessfulRequests++
		n.metrics.LastSuccessTime = time.Now()
	} else {
		n.metrics.FailedRequests++
		n.metrics.LastFailureTime = time.Now()
	}

	n.metrics.LastRequestTime = time.Now()

	// Update average latency (simple moving average)
	latency := time.Since(startTime).Milliseconds()
	if n.metrics.AverageLatency == 0 {
		n.metrics.AverageLatency = float64(latency)
	} else {
		// Exponential moving average with alpha = 0.2
		n.metrics.AverageLatency = 0.8*n.metrics.AverageLatency + 0.2*float64(latency)
	}
}

// updateConnectionState updates the connection state in metrics.
func (n *NetworkProxy) updateConnectionState(state string, errorMsg string) {
	n.metricsMu.Lock()
	defer n.metricsMu.Unlock()

	n.metrics.ConnectionState = state
	if errorMsg != "" {
		n.metrics.LastError = errorMsg
	}
}

// init registers the network proxy factory.
func init() {
	RegisterProxyFactory("network", func(config interface{}, logger *zap.Logger) (DeviceProxy, error) {
		netConfig, ok := config.(*NetworkProxyConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for network proxy")
		}
		return NewNetworkProxy(netConfig, logger)
	})
}
