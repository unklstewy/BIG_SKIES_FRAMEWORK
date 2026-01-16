// Package ascomserver provides a native ASCOM Alpaca REST API server implementation.
// This server acts as a reflector/proxy, accepting incoming ASCOM Alpaca requests from
// clients (such as N.I.N.A., TheSkyX, etc.) and forwarding them to backend telescope
// hardware via network, MQTT, or direct connections.
//
// The ASCOM Alpaca protocol is a RESTful HTTP API standard for astronomical equipment
// developed by the ASCOM Initiative (https://ascom-standards.org/).
package ascomserver

import (
	"time"

	"go.uber.org/zap"
)

// Constants for ASCOM Alpaca protocol compliance.
const (
	// AlpacaAPIVersion is the supported Alpaca API version.
	// Version 1 is the current stable release of the ASCOM Alpaca protocol.
	AlpacaAPIVersion = 1

	// AlpacaDiscoveryMessage is the UDP broadcast message used for device discovery.
	// Clients send this message to discover ASCOM Alpaca servers on the network.
	AlpacaDiscoveryMessage = "alpacadiscovery1"

	// DefaultDiscoveryPort is the standard UDP port for ASCOM Alpaca discovery broadcasts.
	// Servers listen on this port for discovery requests.
	DefaultDiscoveryPort = 32227

	// DefaultAPIPort is the default HTTP port for the ASCOM Alpaca REST API.
	// This is the standard port, but can be configured differently.
	DefaultAPIPort = 11111

	// DefaultServerName is the default name reported in the management API.
	DefaultServerName = "BigSkies ASCOM Reflector"

	// DefaultManufacturer is the default manufacturer name.
	DefaultManufacturer = "BigSkies Framework"

	// DefaultLocation is the default location string.
	DefaultLocation = "Observatory"

	// MaxTransactionID is the maximum value for transaction IDs before wrapping.
	// Transaction IDs are used to correlate requests and responses.
	MaxTransactionID = 2147483647 // Max int32

	// RequestTimeout is the default timeout for backend device requests.
	RequestTimeout = 30 * time.Second

	// DiscoveryTimeout is how long to wait for discovery responses.
	DiscoveryTimeout = 5 * time.Second
)

// Server represents the main ASCOM Alpaca server instance.
// It manages HTTP API endpoints, UDP discovery, and backend device connections.
type Server struct {
	// config holds the server configuration including listen addresses,
	// authentication settings, backend connection details, etc.
	config *Config

	// logger provides structured logging for all server operations.
	logger *zap.Logger

	// devices is a registry of virtual ASCOM devices exposed by this server.
	// Each device is mapped by its unique key: "{device_type}-{device_number}"
	// Example: "telescope-0", "camera-0", "dome-0"
	devices map[string]*VirtualDevice

	// discovery manages the UDP discovery service that allows ASCOM clients
	// to find this server on the network via broadcast messages.
	discovery *DiscoveryService

	// stopCh is used to signal shutdown to all background goroutines.
	stopCh chan struct{}

	// transactionCounter generates unique transaction IDs for each API request.
	// This is used to track and correlate requests/responses in logs and debugging.
	transactionCounter int32
}

// VirtualDevice represents a virtual ASCOM device exposed by the server.
// Each virtual device proxies requests to one or more backend devices and
// maintains cached state for performance optimization.
type VirtualDevice struct {
	// DeviceType is the ASCOM device type (telescope, camera, dome, etc.)
	DeviceType string

	// DeviceNumber is the ASCOM device number (0-based index).
	// Multiple devices of the same type are differentiated by this number.
	DeviceNumber int

	// Name is the human-readable device name reported to clients.
	Name string

	// Description is a detailed description of the device.
	Description string

	// DriverInfo contains information about the driver implementation.
	DriverInfo string

	// DriverVersion is the version string of the driver.
	DriverVersion string

	// InterfaceVersion is the ASCOM device interface version number.
	// Different device types have different interface versions.
	InterfaceVersion int

	// UniqueID is a globally unique identifier for this device instance.
	// This should persist across server restarts if possible.
	UniqueID string

	// BackendConfig specifies how to connect to the actual hardware.
	// This determines whether to use network proxy, MQTT, or direct connection.
	BackendConfig BackendDeviceConfig

	// Connected indicates whether the device is currently connected.
	// This is cached state that may be refreshed periodically.
	Connected bool

	// LastUpdate is the timestamp of the last state update from the backend.
	// Used to determine if cached data is stale.
	LastUpdate time.Time

	// StateCache stores device-specific state to reduce backend queries.
	// The structure varies by device type (telescope has RA/Dec, camera has temperature, etc.)
	StateCache interface{}
}

// BackendDeviceConfig specifies how to connect to backend hardware.
type BackendDeviceConfig struct {
	// Mode determines the connection type: "network", "mqtt", or "direct"
	Mode string

	// NetworkConfig is used when Mode is "network" - connects to remote Alpaca server
	NetworkConfig *NetworkBackendConfig

	// MQTTConfig is used when Mode is "mqtt" - integrates with BigSkies MQTT bus
	MQTTConfig *MQTTBackendConfig

	// DirectConfig is used when Mode is "direct" - direct serial/USB connection
	// This is for future implementation
	DirectConfig *DirectBackendConfig
}

// NetworkBackendConfig configures a network proxy to a remote ASCOM Alpaca server.
type NetworkBackendConfig struct {
	// ServerURL is the base URL of the remote ASCOM Alpaca server.
	// Example: "http://192.168.1.100:11111"
	ServerURL string

	// RemoteDeviceType is the device type on the remote server.
	// Usually matches the virtual device type, but may differ for device mapping.
	RemoteDeviceType string

	// RemoteDeviceNumber is the device number on the remote server.
	RemoteDeviceNumber int

	// Timeout is the HTTP request timeout for this backend.
	Timeout time.Duration

	// RetryAttempts is the number of retry attempts for failed requests.
	RetryAttempts int
}

// MQTTBackendConfig configures MQTT integration with BigSkies telescope coordinator.
type MQTTBackendConfig struct {
	// Broker is the MQTT broker address (host:port).
	Broker string

	// TelescopeID identifies which BigSkies telescope configuration to control.
	// This corresponds to telescope configs in the telescope coordinator.
	TelescopeID string

	// ClientID is the MQTT client identifier for this server.
	ClientID string

	// Username for MQTT authentication (optional).
	Username string

	// Password for MQTT authentication (optional).
	Password string

	// Timeout for MQTT request/response operations.
	Timeout time.Duration

	// QoS is the MQTT quality of service level (0, 1, or 2).
	QoS byte
}

// DirectBackendConfig configures direct serial/USB hardware connections.
// This is for future implementation to support direct telescope control.
type DirectBackendConfig struct {
	// Port is the serial port device path.
	// Example: "/dev/ttyUSB0" on Linux, "COM3" on Windows
	Port string

	// BaudRate is the serial communication baud rate.
	BaudRate int

	// Protocol specifies the telescope protocol (e.g., "LX200", "Meade", "Celestron")
	Protocol string

	// Timeout for serial communication operations.
	Timeout time.Duration
}

// DiscoveryService manages the UDP discovery protocol.
// It listens for "alpacadiscovery1" broadcasts and responds with server information.
type DiscoveryService struct {
	// port is the UDP port to listen on (typically 32227)
	port int

	// apiPort is the HTTP API port to advertise in discovery responses
	apiPort int

	// logger provides structured logging
	logger *zap.Logger

	// stopCh signals the discovery service to shut down
	stopCh chan struct{}
}

// DiscoveryResponse is the JSON response sent to discovery broadcasts.
// This follows the ASCOM Alpaca discovery protocol specification.
type DiscoveryResponse struct {
	// AlpacaPort is the TCP port number where the Alpaca REST API is available.
	AlpacaPort int `json:"AlpacaPort"`
}

// APIResponse is the standard wrapper for all ASCOM Alpaca API responses.
// Every API endpoint must return this structure with appropriate fields populated.
type APIResponse struct {
	// Value contains the actual data returned by the request.
	// The type varies by endpoint (bool, int, float64, string, struct, etc.)
	Value interface{} `json:"Value,omitempty"`

	// ClientTransactionID echoes back the client's transaction ID.
	// This allows clients to correlate responses with requests.
	ClientTransactionID int32 `json:"ClientTransactionID"`

	// ServerTransactionID is a unique ID generated by the server for this request.
	// Used for debugging and log correlation.
	ServerTransactionID int32 `json:"ServerTransactionID"`

	// ErrorNumber indicates success (0) or error (non-zero).
	// ASCOM defines standard error codes for common failure scenarios.
	ErrorNumber int `json:"ErrorNumber"`

	// ErrorMessage provides a human-readable description of any error.
	// Empty string when ErrorNumber is 0 (success).
	ErrorMessage string `json:"ErrorMessage"`
}

// ASCOM standard error codes.
// These are defined by the ASCOM standard and should be used consistently.
const (
	// ErrorCodeSuccess indicates the operation completed successfully.
	ErrorCodeSuccess = 0x0000

	// ErrorCodeNotImplemented indicates the requested operation is not implemented.
	ErrorCodeNotImplemented = 0x0400

	// ErrorCodeInvalidValue indicates a parameter value is invalid or out of range.
	ErrorCodeInvalidValue = 0x0401

	// ErrorCodeValueNotSet indicates a value has not been set yet.
	ErrorCodeValueNotSet = 0x0402

	// ErrorCodeNotConnected indicates the device is not connected.
	ErrorCodeNotConnected = 0x0407

	// ErrorCodeInvalidWhileParked indicates the operation is invalid while parked.
	ErrorCodeInvalidWhileParked = 0x0408

	// ErrorCodeInvalidWhileSlaved indicates the operation is invalid while slaved.
	ErrorCodeInvalidWhileSlaved = 0x0409

	// ErrorCodeInvalidOperation indicates the requested operation is not valid.
	ErrorCodeInvalidOperation = 0x040B

	// ErrorCodeActionNotImplemented indicates the requested action is not implemented.
	ErrorCodeActionNotImplemented = 0x040C

	// ErrorCodeUnspecifiedError indicates an unspecified error occurred.
	ErrorCodeUnspecifiedError = 0x04FF
)

// NewAPIResponse creates a new API response with the given values.
// This helper ensures all required fields are populated correctly.
func NewAPIResponse(value interface{}, clientTxnID, serverTxnID int32, errNum int, errMsg string) *APIResponse {
	return &APIResponse{
		Value:               value,
		ClientTransactionID: clientTxnID,
		ServerTransactionID: serverTxnID,
		ErrorNumber:         errNum,
		ErrorMessage:        errMsg,
	}
}

// NewSuccessResponse creates a successful API response.
func NewSuccessResponse(value interface{}, clientTxnID, serverTxnID int32) *APIResponse {
	return NewAPIResponse(value, clientTxnID, serverTxnID, ErrorCodeSuccess, "")
}

// NewErrorResponse creates an error API response.
func NewErrorResponse(clientTxnID, serverTxnID int32, errNum int, errMsg string) *APIResponse {
	return NewAPIResponse(nil, clientTxnID, serverTxnID, errNum, errMsg)
}

// DeviceKey generates a unique key for a device based on type and number.
// This is used as the map key in the devices registry.
func DeviceKey(deviceType string, deviceNumber int) string {
	return deviceType + "-" + string(rune(deviceNumber+'0'))
}
