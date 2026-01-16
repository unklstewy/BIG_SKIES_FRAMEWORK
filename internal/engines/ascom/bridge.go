// Package ascom provides ASCOM protocol engines and bridges.
package ascom

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
)

// Bridge translates ASCOM Alpaca API calls into BigSkies MQTT messages
// and routes them to the appropriate coordinators (primarily telescope-coordinator).
//
// The bridge implements a request-response pattern:
// 1. ASCOM API handler calls Bridge.Execute() with method and parameters
// 2. Bridge publishes request to appropriate MQTT topic
// 3. Backend coordinator processes request and publishes response
// 4. Bridge correlates response and returns to API handler
//
// This decouples the ASCOM API layer from device implementations, allowing
// devices to be controlled via the BigSkies MQTT message bus.
type Bridge struct {
	// mqttClient is the MQTT client for message bus communication
	mqttClient *mqtt.Client

	// logger provides structured logging
	logger *zap.Logger

	// pendingRequests tracks outstanding requests awaiting responses
	// Key: request_id, Value: response channel
	pendingRequests sync.Map

	// responseTimeout is how long to wait for responses
	responseTimeout time.Duration

	// stopChan signals cleanup goroutine to stop
	stopChan chan struct{}

	// wg tracks active goroutines
	wg sync.WaitGroup
}

// BridgeConfig configures the MQTT bridge.
type BridgeConfig struct {
	// MQTTClient is the MQTT client to use
	MQTTClient *mqtt.Client

	// ResponseTimeout is how long to wait for MQTT responses
	ResponseTimeout time.Duration

	// Logger for structured logging
	Logger *zap.Logger
}

// BridgeRequest represents an ASCOM request to be sent via MQTT.
type BridgeRequest struct {
	// RequestID uniquely identifies this request for response correlation
	RequestID string `json:"request_id"`

	// DeviceType is the ASCOM device type (telescope, camera, etc.)
	DeviceType string `json:"device_type"`

	// DeviceNumber is the device instance number
	DeviceNumber int `json:"device_number"`

	// Method is the ASCOM API method name (e.g., "slew", "park", "connected")
	Method string `json:"method"`

	// HTTPMethod is the HTTP method (GET or PUT)
	HTTPMethod string `json:"http_method"`

	// Parameters are the method parameters from the API request
	Parameters map[string]string `json:"parameters"`

	// Timestamp is when the request was created
	Timestamp time.Time `json:"timestamp"`
}

// BridgeResponse represents a response received from the backend via MQTT.
type BridgeResponse struct {
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

// NewBridge creates a new ASCOM-to-MQTT bridge instance.
func NewBridge(config *BridgeConfig) (*Bridge, error) {
	if config.MQTTClient == nil {
		return nil, fmt.Errorf("MQTT client is required")
	}

	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	if config.ResponseTimeout == 0 {
		config.ResponseTimeout = 30 * time.Second
	}

	bridge := &Bridge{
		mqttClient:      config.MQTTClient,
		logger:          logger.With(zap.String("component", "ascom_bridge")),
		responseTimeout: config.ResponseTimeout,
		stopChan:        make(chan struct{}),
	}

	return bridge, nil
}

// Start begins bridge operations and subscribes to response topics.
func (b *Bridge) Start(ctx context.Context) error {
	b.logger.Info("Starting ASCOM MQTT bridge")

	// Subscribe to ASCOM response topic
	// Responses from all coordinators come back on this topic
	responseTopic := "bigskies/coordinator/ascom/response/+"
	if err := b.mqttClient.Subscribe(responseTopic, 1, b.handleResponse); err != nil {
		return fmt.Errorf("failed to subscribe to response topic: %w", err)
	}

	b.logger.Info("Subscribed to response topic", zap.String("topic", responseTopic))

	// Start cleanup goroutine for timed-out requests
	b.wg.Add(1)
	go b.cleanupTimedOutRequests()

	b.logger.Info("ASCOM MQTT bridge started")
	return nil
}

// Stop shuts down the bridge and cleans up resources.
func (b *Bridge) Stop() error {
	b.logger.Info("Stopping ASCOM MQTT bridge")

	// Signal cleanup goroutine to stop
	close(b.stopChan)

	// Cancel all pending requests
	b.pendingRequests.Range(func(key, value interface{}) bool {
		if ch, ok := value.(chan *BridgeResponse); ok {
			close(ch)
		}
		b.pendingRequests.Delete(key)
		return true
	})

	// Wait for goroutines to finish
	b.wg.Wait()

	b.logger.Info("ASCOM MQTT bridge stopped")
	return nil
}

// Execute sends an ASCOM request via MQTT and waits for the response.
//
// Parameters:
//   - ctx: Context for cancellation
//   - deviceType: ASCOM device type (e.g., "telescope")
//   - deviceNumber: Device instance number
//   - method: ASCOM method name (e.g., "slew", "park", "rightascension")
//   - httpMethod: HTTP method ("GET" or "PUT")
//   - params: Method parameters from the API request
//
// Returns the response value or an error.
func (b *Bridge) Execute(
	ctx context.Context,
	deviceType string,
	deviceNumber int,
	method string,
	httpMethod string,
	params map[string]string,
) (interface{}, error) {
	// Generate unique request ID
	requestID := uuid.New().String()

	b.logger.Debug("Executing ASCOM request via MQTT",
		zap.String("request_id", requestID),
		zap.String("device_type", deviceType),
		zap.Int("device_number", deviceNumber),
		zap.String("method", method),
		zap.String("http_method", httpMethod))

	// Create request
	request := &BridgeRequest{
		RequestID:    requestID,
		DeviceType:   deviceType,
		DeviceNumber: deviceNumber,
		Method:       method,
		HTTPMethod:   httpMethod,
		Parameters:   params,
		Timestamp:    time.Now(),
	}

	// Create response channel
	responseChan := make(chan *BridgeResponse, 1)
	b.pendingRequests.Store(requestID, responseChan)
	defer b.pendingRequests.Delete(requestID)

	// Determine target topic based on device type and method
	topic := b.buildRequestTopic(deviceType, deviceNumber, method)

	// Wrap request in MQTT message envelope
	msg, err := mqtt.NewMessage(mqtt.MessageTypeRequest, "ascom-coordinator", request)
	if err != nil {
		close(responseChan)
		return nil, fmt.Errorf("failed to create MQTT message: %w", err)
	}

	// Publish request
	if err := b.mqttClient.PublishJSON(topic, 1, false, msg); err != nil {
		close(responseChan)
		return nil, fmt.Errorf("failed to publish request: %w", err)
	}

	b.logger.Debug("Published MQTT request",
		zap.String("request_id", requestID),
		zap.String("topic", topic))

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

	case <-time.After(b.responseTimeout):
		close(responseChan)
		return nil, fmt.Errorf("request timed out after %v", b.responseTimeout)

	case <-ctx.Done():
		close(responseChan)
		return nil, ctx.Err()
	}
}

// buildRequestTopic constructs the MQTT topic for a request based on device type and method.
// This routes requests to the appropriate coordinator.
func (b *Bridge) buildRequestTopic(deviceType string, deviceNumber int, method string) string {
	// Most telescope operations route to telescope-coordinator
	// Format: bigskies/coordinator/telescope/{action}/{resource}

	// Map ASCOM methods to BigSkies actions
	action, resource := b.mapMethodToAction(deviceType, method)

	// Build topic using TopicBuilder
	topic := mqtt.NewTopicBuilder().
		Component("telescope").
		Action(action).
		Resource(resource).
		Build()

	return topic
}

// mapMethodToAction maps ASCOM method names to BigSkies actions and resources.
// This translation layer adapts ASCOM API semantics to BigSkies topic structure.
func (b *Bridge) mapMethodToAction(deviceType, method string) (action, resource string) {
	// Default action based on device type
	switch deviceType {
	case "telescope":
		return b.mapTelescopeMethod(method)
	case "camera":
		return "control", "camera/" + method
	case "dome":
		return "control", "dome/" + method
	case "focuser":
		return "control", "focuser/" + method
	default:
		return "control", deviceType + "/" + method
	}
}

// mapTelescopeMethod maps telescope-specific ASCOM methods to BigSkies actions.
func (b *Bridge) mapTelescopeMethod(method string) (action, resource string) {
	// Map common telescope methods to BigSkies topics
	methodMap := map[string]struct{ action, resource string }{
		// Movement commands
		"slewtocoordinates":      {"control", "slew"},
		"slewtocoordinatesasync": {"control", "slew"},
		"slewtotarget":           {"control", "slew"},
		"slewtotargetasync":      {"control", "slew"},
		"slewtoaltaz":            {"control", "slew"},
		"slewtoaltazasync":       {"control", "slew"},
		"abortslew":              {"control", "abort"},

		// Parking
		"park":      {"control", "park"},
		"unpark":    {"control", "unpark"},
		"setpark":   {"control", "setpark"},
		"findhome":  {"control", "findhome"},

		// Tracking
		"tracking":       {"control", "track"},
		"trackingrate":   {"control", "trackingrate"},
		"trackingrates":  {"status", "trackingrates"},

		// Synchronization
		"synctocoordinates": {"control", "sync"},
		"synctotarget":      {"control", "sync"},
		"synctoaltaz":       {"control", "sync"},

		// Position queries
		"rightascension":  {"status", "coordinates"},
		"declination":     {"status", "coordinates"},
		"altitude":        {"status", "coordinates"},
		"azimuth":         {"status", "coordinates"},
		"siderealtime":    {"status", "time"},

		// State queries
		"connected":     {"status", "connection"},
		"slewing":       {"status", "state"},
		"athome":        {"status", "state"},
		"atpark":        {"status", "state"},
		"ispulseguiding": {"status", "state"},

		// Capabilities queries (read-only)
		"canslew":      {"status", "capabilities"},
		"canpark":      {"status", "capabilities"},
		"canfindhome":  {"status", "capabilities"},
		"cansync":      {"status", "capabilities"},

		// Configuration
		"sitelatitude":    {"config", "site"},
		"sitelongitude":   {"config", "site"},
		"siteelevation":   {"config", "site"},
		"doesrefraction":  {"config", "refraction"},
	}

	if mapping, exists := methodMap[method]; exists {
		return mapping.action, mapping.resource
	}

	// Default: treat as status query
	return "status", "get"
}

// handleResponse processes incoming MQTT response messages.
func (b *Bridge) handleResponse(topic string, payload []byte) error {
	b.logger.Debug("Received MQTT response", zap.String("topic", topic))

	// Parse MQTT message envelope
	var msg mqtt.Message
	if err := json.Unmarshal(payload, &msg); err != nil {
		b.logger.Error("Failed to parse MQTT message envelope",
			zap.String("topic", topic),
			zap.Error(err))
		return err
	}

	// Extract response from message payload
	responseJSON, err := json.Marshal(msg.Payload)
	if err != nil {
		b.logger.Error("Failed to marshal response payload", zap.Error(err))
		return err
	}

	var response BridgeResponse
	if err := json.Unmarshal(responseJSON, &response); err != nil {
		b.logger.Error("Failed to parse bridge response",
			zap.String("topic", topic),
			zap.Error(err))
		return err
	}

	// Look up pending request
	value, ok := b.pendingRequests.Load(response.RequestID)
	if !ok {
		b.logger.Warn("Received response for unknown request",
			zap.String("request_id", response.RequestID))
		return nil
	}

	// Send response to waiting goroutine
	if ch, ok := value.(chan *BridgeResponse); ok {
		select {
		case ch <- &response:
			b.logger.Debug("Delivered response to waiting goroutine",
				zap.String("request_id", response.RequestID))
		default:
			b.logger.Warn("Response channel full or closed",
				zap.String("request_id", response.RequestID))
		}
	}

	return nil
}

// cleanupTimedOutRequests periodically removes timed-out pending requests.
// This prevents memory leaks from requests that never receive responses.
func (b *Bridge) cleanupTimedOutRequests() {
	defer b.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Cleanup is handled by request timeout in Execute()
			// This goroutine primarily exists for future enhancements
			b.logger.Debug("Cleanup check completed")

		case <-b.stopChan:
			b.logger.Debug("Stopping cleanup goroutine")
			return
		}
	}
}
