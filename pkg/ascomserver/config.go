package ascomserver

import (
	"fmt"
	"time"
)

// Config holds all configuration settings for the ASCOM Alpaca server.
// This structure can be populated from YAML/JSON configuration files,
// environment variables, or programmatically.
type Config struct {
	// Server contains HTTP server configuration.
	Server ServerConfig `json:"server" yaml:"server"`

	// Authentication contains authentication and authorization settings.
	Authentication AuthConfig `json:"authentication" yaml:"authentication"`

	// CORS contains Cross-Origin Resource Sharing settings.
	CORS CORSConfig `json:"cors" yaml:"cors"`

	// TLS contains TLS/SSL configuration for HTTPS.
	TLS TLSConfig `json:"tls" yaml:"tls"`

	// Backend contains backend device connection configuration.
	Backend BackendConfig `json:"backend" yaml:"backend"`

	// Logging contains logging configuration.
	Logging LoggingConfig `json:"logging" yaml:"logging"`

	// Devices is a list of virtual devices to expose via this server.
	// Each device proxies to backend hardware or services.
	Devices []DeviceConfig `json:"devices" yaml:"devices"`
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	// ListenAddress is the address to bind the HTTP server to.
	// Use "0.0.0.0" to listen on all interfaces, or specify a specific IP.
	// Format: "host:port" or ":port"
	// Example: "0.0.0.0:11111" or ":11111"
	ListenAddress string `json:"listen_address" yaml:"listen_address"`

	// DiscoveryPort is the UDP port for ASCOM Alpaca discovery.
	// Standard port is 32227. Must match what clients expect.
	DiscoveryPort int `json:"discovery_port" yaml:"discovery_port"`

	// ServerName is the name reported in the management API.
	// This appears in client software when browsing available servers.
	ServerName string `json:"server_name" yaml:"server_name"`

	// Manufacturer is the manufacturer name reported in the API.
	Manufacturer string `json:"manufacturer" yaml:"manufacturer"`

	// ManufacturerVersion is the version string for the manufacturer's software.
	ManufacturerVersion string `json:"manufacturer_version" yaml:"manufacturer_version"`

	// Location is a human-readable location string (e.g., "Backyard Observatory").
	Location string `json:"location" yaml:"location"`

	// ReadTimeout is the maximum duration for reading the entire request.
	// This prevents slow clients from holding connections open indefinitely.
	ReadTimeout time.Duration `json:"read_timeout" yaml:"read_timeout"`

	// WriteTimeout is the maximum duration for writing the response.
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`

	// IdleTimeout is the maximum duration to wait for the next request
	// when keep-alives are enabled.
	IdleTimeout time.Duration `json:"idle_timeout" yaml:"idle_timeout"`
}

// AuthConfig contains authentication and authorization settings.
type AuthConfig struct {
	// Enabled determines whether authentication is required.
	// If false, all requests are allowed without credentials.
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Username is the required username for HTTP Basic Authentication.
	// Only used if Enabled is true.
	Username string `json:"username" yaml:"username"`

	// Password is the required password for HTTP Basic Authentication.
	// Only used if Enabled is true.
	// TODO: In production, consider using hashed passwords or external auth providers.
	Password string `json:"password" yaml:"password"`

	// Realm is the authentication realm string shown in browser prompts.
	Realm string `json:"realm" yaml:"realm"`
}

// CORSConfig contains Cross-Origin Resource Sharing settings.
// CORS is important for web-based ASCOM clients that run in browsers.
type CORSConfig struct {
	// Enabled determines whether CORS headers are added to responses.
	Enabled bool `json:"enabled" yaml:"enabled"`

	// AllowedOrigins is a list of origins permitted to make cross-origin requests.
	// Use ["*"] to allow all origins (not recommended for production).
	// Example: ["http://localhost:3000", "https://observatory.example.com"]
	AllowedOrigins []string `json:"allowed_origins" yaml:"allowed_origins"`

	// AllowedMethods is a list of HTTP methods allowed for CORS requests.
	// Typically ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
	AllowedMethods []string `json:"allowed_methods" yaml:"allowed_methods"`

	// AllowedHeaders is a list of HTTP headers allowed in CORS requests.
	AllowedHeaders []string `json:"allowed_headers" yaml:"allowed_headers"`

	// AllowCredentials indicates whether credentials (cookies, auth) are allowed.
	AllowCredentials bool `json:"allow_credentials" yaml:"allow_credentials"`

	// MaxAge indicates how long (in seconds) browsers can cache preflight results.
	MaxAge int `json:"max_age" yaml:"max_age"`
}

// TLSConfig contains TLS/SSL configuration for HTTPS.
type TLSConfig struct {
	// Enabled determines whether TLS/SSL is enabled.
	// If true, the server will use HTTPS instead of HTTP.
	Enabled bool `json:"enabled" yaml:"enabled"`

	// CertFile is the path to the TLS certificate file (PEM format).
	CertFile string `json:"cert_file" yaml:"cert_file"`

	// KeyFile is the path to the TLS private key file (PEM format).
	KeyFile string `json:"key_file" yaml:"key_file"`

	// MinVersion specifies the minimum TLS version to accept.
	// Example: "1.2" or "1.3"
	MinVersion string `json:"min_version" yaml:"min_version"`
}

// BackendConfig contains backend connection configuration.
// This determines how the server connects to actual telescope hardware.
type BackendConfig struct {
	// Mode specifies the primary backend mode: "network", "mqtt", or "hybrid"
	// - "network": Proxy to remote ASCOM Alpaca servers via HTTP
	// - "mqtt": Integrate with BigSkies MQTT message bus
	// - "hybrid": Mix of network and MQTT backends per device
	Mode string `json:"mode" yaml:"mode"`

	// Network contains settings for network proxy mode.
	Network NetworkConfig `json:"network" yaml:"network"`

	// MQTT contains settings for MQTT integration mode.
	MQTT MQTTConfig `json:"mqtt" yaml:"mqtt"`
}

// NetworkConfig contains settings for network proxy backend.
type NetworkConfig struct {
	// Devices is a list of device mappings for network proxy mode.
	// Each virtual device is mapped to a remote ASCOM Alpaca server.
	Devices []NetworkDeviceMapping `json:"devices" yaml:"devices"`

	// DefaultTimeout is the default HTTP timeout for backend requests.
	DefaultTimeout time.Duration `json:"default_timeout" yaml:"default_timeout"`

	// DefaultRetryAttempts is the default number of retry attempts.
	DefaultRetryAttempts int `json:"default_retry_attempts" yaml:"default_retry_attempts"`
}

// NetworkDeviceMapping maps a virtual device to a remote ASCOM server.
type NetworkDeviceMapping struct {
	// DeviceType is the virtual device type exposed by this server.
	DeviceType string `json:"device_type" yaml:"device_type"`

	// DeviceNumber is the virtual device number.
	DeviceNumber int `json:"device_number" yaml:"device_number"`

	// ServerURL is the remote ASCOM Alpaca server URL.
	ServerURL string `json:"server_url" yaml:"server_url"`

	// RemoteDeviceType is the device type on the remote server.
	// If empty, uses the same as DeviceType.
	RemoteDeviceType string `json:"remote_device_type" yaml:"remote_device_type"`

	// RemoteDeviceNumber is the device number on the remote server.
	RemoteDeviceNumber int `json:"remote_device_number" yaml:"remote_device_number"`
}

// MQTTConfig contains settings for MQTT integration backend.
type MQTTConfig struct {
	// Broker is the MQTT broker address (host:port).
	// Example: "localhost:1883" or "mqtt.example.com:1883"
	Broker string `json:"broker" yaml:"broker"`

	// TelescopeID identifies the telescope configuration in BigSkies.
	// This must match a telescope ID configured in the telescope coordinator.
	TelescopeID string `json:"telescope_id" yaml:"telescope_id"`

	// ClientID is the MQTT client identifier for this server instance.
	// If empty, a random client ID will be generated.
	ClientID string `json:"client_id" yaml:"client_id"`

	// Username for MQTT broker authentication (optional).
	Username string `json:"username" yaml:"username"`

	// Password for MQTT broker authentication (optional).
	Password string `json:"password" yaml:"password"`

	// QoS is the MQTT quality of service level (0, 1, or 2).
	// 0 = at most once, 1 = at least once, 2 = exactly once
	QoS byte `json:"qos" yaml:"qos"`

	// Timeout is the timeout for MQTT request/response operations.
	Timeout time.Duration `json:"timeout" yaml:"timeout"`

	// KeepAlive is the MQTT keep-alive interval.
	KeepAlive time.Duration `json:"keep_alive" yaml:"keep_alive"`
}

// LoggingConfig contains logging configuration.
type LoggingConfig struct {
	// Level is the logging level: "debug", "info", "warn", "error"
	Level string `json:"level" yaml:"level"`

	// Format is the log format: "json" or "console"
	// "json" is recommended for production, "console" for development.
	Format string `json:"format" yaml:"format"`

	// OutputPaths is a list of output destinations.
	// Can be file paths or "stdout"/"stderr".
	OutputPaths []string `json:"output_paths" yaml:"output_paths"`

	// ErrorOutputPaths is a list of error output destinations.
	ErrorOutputPaths []string `json:"error_output_paths" yaml:"error_output_paths"`
}

// DeviceConfig defines a virtual device exposed by the server.
type DeviceConfig struct {
	// Type is the ASCOM device type (telescope, camera, dome, etc.)
	Type string `json:"type" yaml:"type"`

	// Number is the device number (0-based).
	Number int `json:"number" yaml:"number"`

	// Name is the human-readable device name.
	Name string `json:"name" yaml:"name"`

	// Description is a detailed device description.
	Description string `json:"description" yaml:"description"`

	// UniqueID is a unique identifier for this device instance.
	// If empty, one will be generated based on type and number.
	UniqueID string `json:"unique_id" yaml:"unique_id"`

	// Backend specifies how to connect to the actual hardware.
	Backend DeviceBackendConfig `json:"backend" yaml:"backend"`
}

// DeviceBackendConfig specifies backend connection for a specific device.
type DeviceBackendConfig struct {
	// Mode is the backend connection mode: "network", "mqtt", or "direct"
	Mode string `json:"mode" yaml:"mode"`

	// NetworkURL is the remote ASCOM server URL (for network mode).
	NetworkURL string `json:"network_url" yaml:"network_url"`

	// NetworkDeviceType is the remote device type (for network mode).
	NetworkDeviceType string `json:"network_device_type" yaml:"network_device_type"`

	// NetworkDeviceNumber is the remote device number (for network mode).
	NetworkDeviceNumber int `json:"network_device_number" yaml:"network_device_number"`

	// MQTTTelescopeID is the telescope ID (for MQTT mode).
	MQTTTelescopeID string `json:"mqtt_telescope_id" yaml:"mqtt_telescope_id"`
}

// Validate checks the configuration for errors and sets defaults.
// This should be called after loading configuration from file or environment.
func (c *Config) Validate() error {
	// Set server defaults
	if c.Server.ListenAddress == "" {
		c.Server.ListenAddress = fmt.Sprintf(":%d", DefaultAPIPort)
	}
	if c.Server.DiscoveryPort == 0 {
		c.Server.DiscoveryPort = DefaultDiscoveryPort
	}
	if c.Server.ServerName == "" {
		c.Server.ServerName = DefaultServerName
	}
	if c.Server.Manufacturer == "" {
		c.Server.Manufacturer = DefaultManufacturer
	}
	if c.Server.ManufacturerVersion == "" {
		c.Server.ManufacturerVersion = "1.0.0"
	}
	if c.Server.Location == "" {
		c.Server.Location = DefaultLocation
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30 * time.Second
	}
	if c.Server.IdleTimeout == 0 {
		c.Server.IdleTimeout = 60 * time.Second
	}

	// Set auth defaults
	if c.Authentication.Realm == "" {
		c.Authentication.Realm = "ASCOM Alpaca Server"
	}

	// Set CORS defaults
	if c.CORS.Enabled {
		if len(c.CORS.AllowedOrigins) == 0 {
			c.CORS.AllowedOrigins = []string{"*"}
		}
		if len(c.CORS.AllowedMethods) == 0 {
			c.CORS.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
		}
		if len(c.CORS.AllowedHeaders) == 0 {
			c.CORS.AllowedHeaders = []string{"*"}
		}
		if c.CORS.MaxAge == 0 {
			c.CORS.MaxAge = 3600 // 1 hour
		}
	}

	// Validate backend mode
	if c.Backend.Mode == "" {
		c.Backend.Mode = "network" // Default to network mode
	}
	switch c.Backend.Mode {
	case "network", "mqtt", "hybrid":
		// Valid modes
	default:
		return fmt.Errorf("invalid backend mode: %s (must be 'network', 'mqtt', or 'hybrid')", c.Backend.Mode)
	}

	// Set backend defaults
	if c.Backend.Network.DefaultTimeout == 0 {
		c.Backend.Network.DefaultTimeout = RequestTimeout
	}
	if c.Backend.Network.DefaultRetryAttempts == 0 {
		c.Backend.Network.DefaultRetryAttempts = 3
	}

	if c.Backend.MQTT.Timeout == 0 {
		c.Backend.MQTT.Timeout = RequestTimeout
	}
	if c.Backend.MQTT.KeepAlive == 0 {
		c.Backend.MQTT.KeepAlive = 60 * time.Second
	}
	if c.Backend.MQTT.QoS == 0 {
		c.Backend.MQTT.QoS = 1 // At least once delivery
	}

	// Set logging defaults
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
	if len(c.Logging.OutputPaths) == 0 {
		c.Logging.OutputPaths = []string{"stdout"}
	}
	if len(c.Logging.ErrorOutputPaths) == 0 {
		c.Logging.ErrorOutputPaths = []string{"stderr"}
	}

	// Validate devices
	if len(c.Devices) == 0 {
		return fmt.Errorf("at least one device must be configured")
	}

	deviceKeys := make(map[string]bool)
	for i, dev := range c.Devices {
		if dev.Type == "" {
			return fmt.Errorf("device %d: type is required", i)
		}
		if dev.Number < 0 {
			return fmt.Errorf("device %d: number must be non-negative", i)
		}

		// Check for duplicate device type+number combinations
		key := fmt.Sprintf("%s-%d", dev.Type, dev.Number)
		if deviceKeys[key] {
			return fmt.Errorf("duplicate device: %s (type=%s, number=%d)", key, dev.Type, dev.Number)
		}
		deviceKeys[key] = true

		// Set device defaults
		if dev.Name == "" {
			c.Devices[i].Name = fmt.Sprintf("%s #%d", dev.Type, dev.Number)
		}
		if dev.Description == "" {
			c.Devices[i].Description = fmt.Sprintf("BigSkies ASCOM %s", dev.Type)
		}

		// Validate backend configuration
		if dev.Backend.Mode == "" {
			c.Devices[i].Backend.Mode = c.Backend.Mode // Use global backend mode as default
		}
	}

	return nil
}

// DefaultConfig returns a configuration with sensible defaults.
// This can be used as a starting point for custom configurations.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			ListenAddress:       fmt.Sprintf(":%d", DefaultAPIPort),
			DiscoveryPort:       DefaultDiscoveryPort,
			ServerName:          DefaultServerName,
			Manufacturer:        DefaultManufacturer,
			ManufacturerVersion: "1.0.0",
			Location:            DefaultLocation,
			ReadTimeout:         30 * time.Second,
			WriteTimeout:        30 * time.Second,
			IdleTimeout:         60 * time.Second,
		},
		Authentication: AuthConfig{
			Enabled: false,
			Realm:   "ASCOM Alpaca Server",
		},
		CORS: CORSConfig{
			Enabled:          true,
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: false,
			MaxAge:           3600,
		},
		TLS: TLSConfig{
			Enabled: false,
		},
		Backend: BackendConfig{
			Mode: "network",
			Network: NetworkConfig{
				DefaultTimeout:       RequestTimeout,
				DefaultRetryAttempts: 3,
			},
			MQTT: MQTTConfig{
				QoS:       1,
				Timeout:   RequestTimeout,
				KeepAlive: 60 * time.Second,
			},
		},
		Logging: LoggingConfig{
			Level:            "info",
			Format:           "json",
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stderr"},
		},
		Devices: []DeviceConfig{},
	}
}
