package ascomserver

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NewServer creates a new ASCOM Alpaca server instance with the given configuration.
// The server must be started with Start() before it will accept requests.
//
// Parameters:
//   - config: Server configuration (will be validated)
//   - logger: Structured logger for server operations (if nil, a no-op logger is used)
//
// Returns:
//   - *Server: Initialized server instance ready to be started
//   - error: Configuration validation error, if any
func NewServer(config *Config, logger *zap.Logger) (*Server, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Validate configuration and set defaults
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create the server instance
	server := &Server{
		config:             config,
		logger:             logger.With(zap.String("component", "ascom_server")),
		devices:            make(map[string]*VirtualDevice),
		stopCh:             make(chan struct{}),
		transactionCounter: 0,
	}

	// Initialize virtual devices from configuration.
	// Each device in the config becomes a virtual device that can be accessed via the API.
	for _, deviceConfig := range config.Devices {
		if err := server.registerDevice(deviceConfig); err != nil {
			return nil, fmt.Errorf("failed to register device %s-%d: %w",
				deviceConfig.Type, deviceConfig.Number, err)
		}
	}

	server.logger.Info("ASCOM Alpaca server created",
		zap.Int("device_count", len(server.devices)),
		zap.String("listen_address", config.Server.ListenAddress))

	return server, nil
}

// registerDevice creates a virtual device from configuration and adds it to the server.
// This internal method is called during server initialization.
func (s *Server) registerDevice(config DeviceConfig) error {
	// Generate a unique ID if not provided
	uniqueID := config.UniqueID
	if uniqueID == "" {
		// Create a UUID based on device type and number for stability across restarts
		uniqueID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%s-%d", config.Type, config.Number))).String()
	}

	// Create the virtual device
	device := &VirtualDevice{
		DeviceType:       config.Type,
		DeviceNumber:     config.Number,
		Name:             config.Name,
		Description:      config.Description,
		DriverInfo:       fmt.Sprintf("BigSkies ASCOM Reflector - %s Driver", config.Type),
		DriverVersion:    "1.0.0",
		InterfaceVersion: getInterfaceVersion(config.Type),
		UniqueID:         uniqueID,
		BackendConfig: BackendDeviceConfig{
			Mode: config.Backend.Mode,
			NetworkConfig: &NetworkBackendConfig{
				ServerURL:          config.Backend.NetworkURL,
				RemoteDeviceType:   config.Backend.NetworkDeviceType,
				RemoteDeviceNumber: config.Backend.NetworkDeviceNumber,
				Timeout:            s.config.Backend.Network.DefaultTimeout,
				RetryAttempts:      s.config.Backend.Network.DefaultRetryAttempts,
			},
			MQTTConfig: &MQTTBackendConfig{
				Broker:      s.config.Backend.MQTT.Broker,
				TelescopeID: config.Backend.MQTTTelescopeID,
				ClientID:    s.config.Backend.MQTT.ClientID,
				Username:    s.config.Backend.MQTT.Username,
				Password:    s.config.Backend.MQTT.Password,
				Timeout:     s.config.Backend.MQTT.Timeout,
				QoS:         s.config.Backend.MQTT.QoS,
			},
		},
		Connected:  false,
		LastUpdate: time.Now(),
		StateCache: nil, // Will be populated when device state is queried
	}

	// Add device to registry using a unique key
	key := DeviceKey(config.Type, config.Number)
	s.devices[key] = device

	s.logger.Info("Virtual device registered",
		zap.String("device_type", device.DeviceType),
		zap.Int("device_number", device.DeviceNumber),
		zap.String("name", device.Name),
		zap.String("unique_id", device.UniqueID))

	return nil
}

// getInterfaceVersion returns the ASCOM interface version number for a device type.
// Different device types have different interface versions as defined by the ASCOM standard.
//
// Current ASCOM interface versions (as of 2024):
//   - Telescope: 3
//   - Camera: 3
//   - Dome: 2
//   - Focuser: 3
//   - FilterWheel: 2
//   - Rotator: 2
//   - Switch: 2
//   - SafetyMonitor: 1
//   - ObservingConditions: 1
//   - CoverCalibrator: 1
func getInterfaceVersion(deviceType string) int {
	versions := map[string]int{
		"telescope":           3,
		"camera":              3,
		"dome":                2,
		"focuser":             3,
		"filterwheel":         2,
		"rotator":             2,
		"switch":              2,
		"safetymonitor":       1,
		"observingconditions": 1,
		"covercalibrator":     1,
	}

	if version, exists := versions[deviceType]; exists {
		return version
	}
	return 1 // Default to version 1 for unknown types
}

// Start starts the ASCOM Alpaca server and begins accepting requests.
// This method blocks until the server is shut down or an error occurs.
//
// The server performs the following startup sequence:
//  1. Start the UDP discovery service
//  2. Initialize the HTTP router with all middleware and endpoints
//  3. Start the HTTP server on the configured address
//
// Parameters:
//   - ctx: Context for server lifecycle management
//
// Returns an error if the server fails to start or encounters a fatal error.
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting ASCOM Alpaca server")

	// Extract the port from the listen address for the discovery service.
	// The discovery service needs to know which port to advertise.
	apiPort := extractPort(s.config.Server.ListenAddress)
	if apiPort == 0 {
		apiPort = DefaultAPIPort
	}

	// Start the UDP discovery service.
	// This allows ASCOM clients to find this server on the network.
	s.discovery = NewDiscoveryService(
		s.config.Server.DiscoveryPort,
		apiPort,
		s.logger)

	if err := s.discovery.Start(); err != nil {
		return fmt.Errorf("failed to start discovery service: %w", err)
	}

	s.logger.Info("Discovery service started",
		zap.Int("udp_port", s.config.Server.DiscoveryPort),
		zap.Int("api_port", apiPort))

	// Initialize the HTTP router
	router := s.setupRouter()

	// Create the HTTP server with configured timeouts
	httpServer := &http.Server{
		Addr:         s.config.Server.ListenAddress,
		Handler:      router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		IdleTimeout:  s.config.Server.IdleTimeout,
	}

	// Start the HTTP server in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)

	serverErrors := make(chan error, 1)

	go func() {
		defer wg.Done()

		s.logger.Info("HTTP server starting",
			zap.String("address", httpServer.Addr))

		// Start HTTPS or HTTP based on TLS configuration
		if s.config.TLS.Enabled {
			serverErrors <- httpServer.ListenAndServeTLS(
				s.config.TLS.CertFile,
				s.config.TLS.KeyFile)
		} else {
			serverErrors <- httpServer.ListenAndServe()
		}
	}()

	// Wait for shutdown signal or context cancellation
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server error: %w", err)
		}
	case <-ctx.Done():
		s.logger.Info("Shutdown signal received")
	case <-s.stopCh:
		s.logger.Info("Server stop requested")
	}

	// Graceful shutdown
	s.logger.Info("Shutting down server")

	// Create a timeout context for graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown the HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("Error during HTTP server shutdown", zap.Error(err))
	}

	// Stop the discovery service
	s.discovery.Stop()

	// Wait for HTTP server goroutine to finish
	wg.Wait()

	s.logger.Info("Server shutdown complete")
	return nil
}

// Stop initiates a graceful shutdown of the server.
// This closes the stop channel, which signals the Start() method to begin shutdown.
func (s *Server) Stop() {
	close(s.stopCh)
}

// setupRouter initializes the Gin router with all middleware and routes.
// This creates the complete HTTP routing table for the ASCOM Alpaca API.
func (s *Server) setupRouter() *gin.Engine {
	// Set Gin mode based on logging configuration
	if s.config.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create the router
	router := gin.New()

	// Add global middleware in order of execution.
	// Order matters - these are called in sequence for each request.

	// 1. Error handler (must be first to catch panics in other middleware)
	router.Use(ErrorHandlerMiddleware(s.logger))

	// 2. Logging middleware (logs all requests)
	router.Use(LoggingMiddleware(s.logger))

	// 3. CORS middleware (handles cross-origin requests)
	if s.config.CORS.Enabled {
		router.Use(CORSMiddleware(s.config.CORS))
	}

	// 4. Authentication middleware (if enabled)
	router.Use(AuthMiddleware(s.config.Authentication))

	// 5. Transaction ID tracking (required by ASCOM Alpaca spec)
	router.Use(TransactionMiddleware(&s.transactionCounter))

	// Register management API routes.
	// These provide server-level information (not device-specific).
	managementAPI := NewManagementAPI(s)
	managementAPI.RegisterRoutes(router.Group(""))

	// TODO: Register device-specific API routes.
	// These will be added in Phase 3+ (telescope, camera, dome, etc.)
	// Each device type will have its own handler that implements the
	// ASCOM standard endpoints for that device type.
	//
	// Example (to be implemented):
	//   telescopeAPI := NewTelescopeAPI(s)
	//   telescopeAPI.RegisterRoutes(router.Group(""))
	//
	//   cameraAPI := NewCameraAPI(s)
	//   cameraAPI.RegisterRoutes(router.Group(""))
	//
	// For now, the server provides management endpoints only.
	// Device endpoints will return "not implemented" errors.

	s.logger.Info("HTTP router configured",
		zap.Bool("cors_enabled", s.config.CORS.Enabled),
		zap.Bool("auth_enabled", s.config.Authentication.Enabled),
		zap.Bool("tls_enabled", s.config.TLS.Enabled))

	return router
}

// extractPort extracts the port number from a listen address string.
// Handles formats like ":8080", "0.0.0.0:8080", "localhost:8080"
//
// Returns 0 if the port cannot be extracted.
func extractPort(address string) int {
	// Find the last colon (IPv6 addresses may have multiple colons)
	colonIndex := -1
	for i := len(address) - 1; i >= 0; i-- {
		if address[i] == ':' {
			colonIndex = i
			break
		}
	}

	if colonIndex == -1 {
		return 0
	}

	// Parse the port number
	portStr := address[colonIndex+1:]
	var port int
	_, err := fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		return 0
	}

	return port
}

// GetDevice retrieves a virtual device by type and number.
// This is used internally by device handlers to access device configuration.
//
// Returns nil if the device is not found.
func (s *Server) GetDevice(deviceType string, deviceNumber int) *VirtualDevice {
	key := DeviceKey(deviceType, deviceNumber)
	return s.devices[key]
}
