package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
)

// MQTTProxy forwards ASCOM API requests to devices via the BigSkies MQTT message bus.
// This integrates the ASCOM reflector with the existing BigSkies telescope coordinator,
// allowing ASCOM clients to control devices managed by the BigSkies framework.
//
// The MQTT proxy uses a request-response pattern:
// 1. Publish request to "ascom/request/{device_type}/{device_number}/{method}"
// 2. Wait for response on "ascom/response/{request_id}"
//
// This implementation provides timeout handling, automatic retries, and connection management.
type MQTTProxy struct {
	// config contains the MQTT proxy configuration
	config *MQTTProxyConfig

	// logger provides structured logging
	logger *zap.Logger

	// mqttClient is the MQTT client for message bus communication
	mqttClient *mqtt.Client

	// connected indicates whether the proxy is connected to MQTT
	connected atomic.Bool

	// pendingRequests tracks outstanding requests waiting for responses
	pendingRequests sync.Map // map[string]chan *MQTTResponse

	// metrics tracks proxy performance and health
	metrics ProxyMetrics

	// metricsMu protects access to metrics
	metricsMu sync.RWMutex

	// stopChan signals goroutines to stop
	stopChan chan struct{}

	// wg tracks active goroutines
	wg sync.WaitGroup
}

// MQTTProxyConfig contains configuration specific to MQTT proxies.
type MQTTProxyConfig struct {
	// Base configuration (timeout, retry, etc.)
	ProxyConfig

	// BrokerURL is the MQTT broker address.
	// Example: "tcp://localhost:1883" or "ssl://mqtt.bigskies.local:8883"
	BrokerURL string

	// Username for MQTT authentication (optional)
	Username string

	// Password for MQTT authentication (optional)
	Password string

	// QoS is the MQTT quality of service level (0, 1, or 2)
	QoS byte

	// TopicPrefix is the base topic for ASCOM messages.
	// Default: "ascom"
	TopicPrefix string

	// ResponseTimeout is how long to wait for MQTT responses.
	// If zero, defaults to ProxyConfig.Timeout.
	ResponseTimeout time.Duration
}

// MQTTRequest represents an ASCOM request sent via MQTT.
type MQTTRequest struct {
	// RequestID uniquely identifies this request for response correlation
	RequestID string `json:"request_id"`

	// DeviceType is the ASCOM device type (telescope, camera, etc.)
	DeviceType string `json:"device_type"`

	// DeviceNumber is the device instance number
	DeviceNumber int `json:"device_number"`

	// Method is the ASCOM API method name
	Method string `json:"method"`

	// HTTPMethod is the HTTP method (GET or PUT)
	HTTPMethod string `json:"http_method"`

	// Parameters are the method parameters
	Parameters map[string]string `json:"parameters"`

	// Timestamp is when the request was created
	Timestamp time.Time `json:"timestamp"`
}

// MQTTResponse represents an ASCOM response received via MQTT.
type MQTTResponse struct {
	// RequestID correlates this response with the original request
	RequestID string `json:"request_id"`

	// Value is the response value (can be any JSON type)
	Value interface{} `json:"value"`

	// ErrorNumber is the ASCOM error code (0 = success)
	ErrorNumber int `json:"error_number"`

	// ErrorMessage is the error description
	ErrorMessage string `json:"error_message"`

	// Timestamp is when the response was created
	Timestamp time.Time `json:"timestamp"`
}

// NewMQTTProxy creates a new MQTT proxy instance.
//
// Parameters:
//   - config: MQTT proxy configuration
//   - logger: Structured logger (if nil, a no-op logger is used)
//
// Returns a configured MQTTProxy ready to connect.
func NewMQTTProxy(config *MQTTProxyConfig, logger *zap.Logger) (*MQTTProxy, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Validate configuration
	if config.BrokerURL == "" {
		return nil, fmt.Errorf("broker URL is required")
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}
	if config.ResponseTimeout == 0 {
		config.ResponseTimeout = config.Timeout
	}
	if config.TopicPrefix == "" {
		config.TopicPrefix = "ascom"
	}

	// Create MQTT client configuration
	mqttConfig := &mqtt.Config{
		BrokerURL:            config.BrokerURL,
		ClientID:             fmt.Sprintf("ascom-proxy-%s-%d", config.DeviceType, config.DeviceNumber),
		Username:             config.Username,
		Password:             config.Password,
		KeepAlive:            60 * time.Second,
		ConnectTimeout:       10 * time.Second,
		AutoReconnect:        true,
		MaxReconnectInterval: 60 * time.Second,
	}

	// Create MQTT client
	mqttClient, err := mqtt.NewClient(mqttConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MQTT client: %w", err)
	}

	proxy := &MQTTProxy{
		config:     config,
		logger:     logger.With(zap.String("component", "mqtt_proxy")),
		mqttClient: mqttClient,
		metrics: ProxyMetrics{
			ConnectionState: "disconnected",
		},
		stopChan: make(chan struct{}),
	}

	proxy.connected.Store(false)

	return proxy, nil
}

// Connect establishes a connection to the MQTT broker and subscribes to response topics.
func (m *MQTTProxy) Connect(ctx context.Context) error {
	m.logger.Info("Connecting to MQTT broker",
		zap.String("broker_url", m.config.BrokerURL),
		zap.String("device_type", m.config.DeviceType),
		zap.Int("device_number", m.config.DeviceNumber))

	// Connect to MQTT broker
	if err := m.mqttClient.Connect(); err != nil {
		m.updateConnectionState("error", err.Error())
		return ProxyError{
			Operation: "Connect",
			Backend:   m.config.BrokerURL,
			Message:   "MQTT connection failed",
			Err:       err,
		}
	}

	// Subscribe to response topic pattern
	// We use a wildcard to receive all responses for this proxy
	responseTopic := fmt.Sprintf("%s/response/+", m.config.TopicPrefix)
	if err := m.mqttClient.Subscribe(responseTopic, m.config.QoS, m.handleResponse); err != nil {
		m.mqttClient.Disconnect()
		m.updateConnectionState("error", err.Error())
		return ProxyError{
			Operation: "Connect",
			Backend:   m.config.BrokerURL,
			Message:   "failed to subscribe to response topic",
			Err:       err,
		}
	}

	m.connected.Store(true)
	m.updateConnectionState("connected", "")

	// Start cleanup goroutine for timed-out requests
	m.wg.Add(1)
	go m.cleanupTimedOutRequests()

	m.logger.Info("Successfully connected to MQTT broker")
	return nil
}

// Disconnect closes the MQTT connection and cleans up resources.
func (m *MQTTProxy) Disconnect(ctx context.Context) error {
	m.logger.Info("Disconnecting from MQTT broker")

	m.connected.Store(false)
	m.updateConnectionState("disconnected", "")

	// Signal goroutines to stop
	close(m.stopChan)

	// Cancel all pending requests
	m.pendingRequests.Range(func(key, value interface{}) bool {
		if ch, ok := value.(chan *MQTTResponse); ok {
			close(ch)
		}
		m.pendingRequests.Delete(key)
		return true
	})

	// Wait for goroutines to finish
	m.wg.Wait()

	// Disconnect from MQTT
	m.mqttClient.Disconnect()

	return nil
}

// IsConnected returns true if the proxy is connected to the MQTT broker.
func (m *MQTTProxy) IsConnected() bool {
	return m.connected.Load() && m.mqttClient.IsConnected()
}

// Get executes a GET request via MQTT.
func (m *MQTTProxy) Get(ctx context.Context, method string, params map[string]string) (interface{}, error) {
	startTime := time.Now()

	m.logger.Debug("Executing MQTT GET request",
		zap.String("method", method),
		zap.Any("params", params))

	// Execute request with retries
	var response interface{}
	err := m.executeWithRetry(ctx, func() error {
		var execErr error
		response, execErr = m.executeRequest(ctx, "GET", method, params)
		return execErr
	})

	// Update metrics
	m.updateMetrics(startTime, err == nil)

	if err != nil {
		return nil, err
	}

	return response, nil
}

// Put executes a PUT request via MQTT.
func (m *MQTTProxy) Put(ctx context.Context, method string, params map[string]string) (interface{}, error) {
	startTime := time.Now()

	m.logger.Debug("Executing MQTT PUT request",
		zap.String("method", method),
		zap.Any("params", params))

	// Execute request with retries
	var response interface{}
	err := m.executeWithRetry(ctx, func() error {
		var execErr error
		response, execErr = m.executeRequest(ctx, "PUT", method, params)
		return execErr
	})

	// Update metrics
	m.updateMetrics(startTime, err == nil)

	if err != nil {
		return nil, err
	}

	return response, nil
}

// HealthCheck verifies MQTT connectivity.
func (m *MQTTProxy) HealthCheck(ctx context.Context) error {
	m.logger.Debug("Performing MQTT health check")

	if !m.mqttClient.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	// Try a simple request to verify the backend is responding
	// Query the "connected" property which all ASCOM devices must support
	_, err := m.executeRequest(ctx, "GET", "connected", map[string]string{})
	return err
}

// GetMetrics returns current proxy metrics.
func (m *MQTTProxy) GetMetrics() *ProxyMetrics {
	m.metricsMu.RLock()
	defer m.metricsMu.RUnlock()

	// Return a copy of the metrics
	metricsCopy := m.metrics
	return &metricsCopy
}

// executeRequest sends a request via MQTT and waits for the response.
func (m *MQTTProxy) executeRequest(ctx context.Context, httpMethod, method string, params map[string]string) (interface{}, error) {
	if !m.IsConnected() {
		return nil, ErrNotConnected
	}

	// Generate unique request ID
	requestID := uuid.New().String()

	// Create request
	request := &MQTTRequest{
		RequestID:    requestID,
		DeviceType:   m.config.DeviceType,
		DeviceNumber: m.config.DeviceNumber,
		Method:       method,
		HTTPMethod:   httpMethod,
		Parameters:   params,
		Timestamp:    time.Now(),
	}

	// Create response channel
	responseChan := make(chan *MQTTResponse, 1)
	m.pendingRequests.Store(requestID, responseChan)
	defer m.pendingRequests.Delete(requestID)

	// Publish request
	requestTopic := fmt.Sprintf("%s/request/%s/%d/%s",
		m.config.TopicPrefix,
		m.config.DeviceType,
		m.config.DeviceNumber,
		method)

	if err := m.mqttClient.PublishJSON(requestTopic, m.config.QoS, false, request); err != nil {
		close(responseChan)
		return nil, fmt.Errorf("failed to publish request: %w", err)
	}

	m.logger.Debug("Published MQTT request",
		zap.String("request_id", requestID),
		zap.String("topic", requestTopic))

	// Wait for response or timeout
	select {
	case response, ok := <-responseChan:
		if !ok {
			return nil, fmt.Errorf("response channel closed")
		}

		// Check for ASCOM errors
		if response.ErrorNumber != 0 {
			return nil, fmt.Errorf("ASCOM error %d: %s", response.ErrorNumber, response.ErrorMessage)
		}

		return response.Value, nil

	case <-time.After(m.config.ResponseTimeout):
		close(responseChan)
		return nil, ErrTimeout

	case <-ctx.Done():
		close(responseChan)
		return nil, ctx.Err()
	}
}

// handleResponse processes incoming MQTT response messages.
func (m *MQTTProxy) handleResponse(topic string, payload []byte) error {
	m.logger.Debug("Received MQTT response", zap.String("topic", topic))

	// Parse response
	var response MQTTResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		m.logger.Error("Failed to parse MQTT response",
			zap.String("topic", topic),
			zap.Error(err))
		return err
	}

	// Look up pending request
	value, ok := m.pendingRequests.Load(response.RequestID)
	if !ok {
		m.logger.Warn("Received response for unknown request",
			zap.String("request_id", response.RequestID))
		return nil
	}

	// Send response to waiting goroutine
	if ch, ok := value.(chan *MQTTResponse); ok {
		select {
		case ch <- &response:
			m.logger.Debug("Delivered response to waiting goroutine",
				zap.String("request_id", response.RequestID))
		default:
			m.logger.Warn("Response channel full or closed",
				zap.String("request_id", response.RequestID))
		}
	}

	return nil
}

// cleanupTimedOutRequests periodically removes timed-out pending requests.
// This prevents memory leaks from requests that never receive responses.
func (m *MQTTProxy) cleanupTimedOutRequests() {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Cleanup logic could be enhanced to track request timestamps
			// For now, the executeRequest function handles timeouts directly
			m.logger.Debug("Cleanup check completed")

		case <-m.stopChan:
			m.logger.Debug("Stopping cleanup goroutine")
			return
		}
	}
}

// executeWithRetry executes an operation with automatic retry on failure.
func (m *MQTTProxy) executeWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	// Try the operation (initial attempt + retries)
	for attempt := 0; attempt <= m.config.RetryAttempts; attempt++ {
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
		m.logger.Warn("MQTT request attempt failed",
			zap.Int("attempt", attempt+1),
			zap.Int("max_attempts", m.config.RetryAttempts+1),
			zap.Error(lastErr))

		// Don't retry on the last attempt
		if attempt < m.config.RetryAttempts {
			// Wait before retrying (exponential backoff)
			delay := m.config.RetryDelay * time.Duration(1<<uint(attempt))
			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// All attempts failed
	return m.handleError("Request", lastErr)
}

// handleError wraps an error in a ProxyError with context.
func (m *MQTTProxy) handleError(operation string, err error) error {
	return ProxyError{
		Operation: operation,
		Backend:   m.config.BrokerURL,
		Message:   "MQTT request failed",
		Err:       err,
	}
}

// updateMetrics updates proxy performance metrics.
func (m *MQTTProxy) updateMetrics(startTime time.Time, success bool) {
	m.metricsMu.Lock()
	defer m.metricsMu.Unlock()

	// Update request counts
	m.metrics.TotalRequests++
	if success {
		m.metrics.SuccessfulRequests++
		m.metrics.LastSuccessTime = time.Now()
	} else {
		m.metrics.FailedRequests++
		m.metrics.LastFailureTime = time.Now()
	}

	m.metrics.LastRequestTime = time.Now()

	// Update average latency (exponential moving average)
	latency := time.Since(startTime).Milliseconds()
	if m.metrics.AverageLatency == 0 {
		m.metrics.AverageLatency = float64(latency)
	} else {
		// Exponential moving average with alpha = 0.2
		m.metrics.AverageLatency = 0.8*m.metrics.AverageLatency + 0.2*float64(latency)
	}
}

// updateConnectionState updates the connection state in metrics.
func (m *MQTTProxy) updateConnectionState(state string, errorMsg string) {
	m.metricsMu.Lock()
	defer m.metricsMu.Unlock()

	m.metrics.ConnectionState = state
	if errorMsg != "" {
		m.metrics.LastError = errorMsg
	}
}

// init registers the MQTT proxy factory.
func init() {
	RegisterProxyFactory("mqtt", func(config interface{}, logger *zap.Logger) (DeviceProxy, error) {
		mqttConfig, ok := config.(*MQTTProxyConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for MQTT proxy")
		}
		return NewMQTTProxy(mqttConfig, logger)
	})
}
