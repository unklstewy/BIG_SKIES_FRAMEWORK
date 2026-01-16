// Package coordinators provides coordinator implementations.
package coordinators

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/engines/ascom"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/ascomserver"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
)

// ASCOMCoordinator manages ASCOM Alpaca device interface and integrates with BigSkies framework.
// It provides an ASCOM-compliant REST API that acts as a facade over the BigSkies telescope
// and device coordinators, enabling standard astronomy software (N.I.N.A., PHD2, etc.) to
// control BigSkies-managed hardware.
//
// The coordinator:
// - Exposes ASCOM Alpaca REST API (HTTP) and UDP discovery
// - Translates ASCOM API calls to BigSkies MQTT messages
// - Routes requests to telescope-coordinator for device control
// - Stores device configurations in PostgreSQL via datastore-coordinator
// - Integrates with security-coordinator for authentication/authorization
type ASCOMCoordinator struct {
	*BaseCoordinator
	ascomEngine    *ascom.Engine           // ASCOM protocol engine
	db             *pgxpool.Pool           // Database connection
	config         *ASCOMConfig            // Coordinator configuration
	httpServer     *http.Server            // HTTP server for ASCOM API
	discoveryService *ascomserver.DiscoveryService // UDP discovery service
	deviceRegistry map[string]*ASCOMDevice // Registered ASCOM devices
}

// ASCOMConfig holds configuration for the ASCOM coordinator.
type ASCOMConfig struct {
	BaseConfig
	DatabaseURL           string        `json:"database_url"`
	HTTPListenAddress     string        `json:"http_listen_address"`      // Default: "0.0.0.0:11111"
	DiscoveryPort         int           `json:"discovery_port"`           // Default: 32227
	HealthCheckInterval   time.Duration `json:"health_check_interval"`
	ReadTimeout           time.Duration `json:"read_timeout"`
	WriteTimeout          time.Duration `json:"write_timeout"`
	IdleTimeout           time.Duration `json:"idle_timeout"`
	EnableCORS            bool          `json:"enable_cors"`              // Allow cross-origin requests
	MaxRequestSize        int64         `json:"max_request_size"`         // Maximum request body size
	ServerName            string        `json:"server_name"`              // ASCOM server name
	ServerDescription     string        `json:"server_description"`
	Manufacturer          string        `json:"manufacturer"`
}

// ASCOMDevice represents an ASCOM device configuration stored in the database.
type ASCOMDevice struct {
	ID               string                 `json:"id"`                // UUID
	DeviceType       string                 `json:"device_type"`       // telescope, camera, dome, etc.
	DeviceNumber     int                    `json:"device_number"`     // Device instance number
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	UniqueID         string                 `json:"unique_id"`         // ASCOM unique device ID
	BackendMode      string                 `json:"backend_mode"`      // mqtt, network, hybrid
	BackendConfig    map[string]interface{} `json:"backend_config"`    // Backend-specific configuration
	OrganizationID   string                 `json:"organization_id"`   // Multi-tenant isolation
	CreatedBy        string                 `json:"created_by"`        // User who created device
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	Enabled          bool                   `json:"enabled"`           // Whether device is active
}

// NewASCOMCoordinator creates a new ASCOM coordinator instance.
func NewASCOMCoordinator(config *ASCOMConfig, logger *zap.Logger) (*ASCOMCoordinator, error) {
	if config.Name == "" {
		config.Name = "ascom"
	}

	// Set defaults
	if config.HTTPListenAddress == "" {
		config.HTTPListenAddress = "0.0.0.0:11111"
	}
	if config.DiscoveryPort == 0 {
		config.DiscoveryPort = 32227
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 30 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 30 * time.Second
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 60 * time.Second
	}
	if config.MaxRequestSize == 0 {
		config.MaxRequestSize = 1 << 20 // 1MB
	}
	if config.ServerName == "" {
		config.ServerName = "BigSkies ASCOM Alpaca Server"
	}
	if config.ServerDescription == "" {
		config.ServerDescription = "ASCOM Alpaca interface for BigSkies Framework"
	}
	if config.Manufacturer == "" {
		config.Manufacturer = "BigSkies"
	}

	// Connect to database
	dbConfig, err := pgxpool.ParseConfig(config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	db, err := pgxpool.NewWithConfig(context.Background(), dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test database connection
	if err := db.Ping(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create MQTT client
	brokerURL := ""
	if config.MQTTConfig != nil {
		brokerURL = config.MQTTConfig.BrokerURL
	}
	mqttClient, err := CreateMQTTClient(brokerURL, mqtt.CoordinatorASCOM, logger)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create MQTT client: %w", err)
	}

	baseCoord := NewBaseCoordinator(mqtt.CoordinatorASCOM, mqttClient, logger)

	// Initialize ASCOM engine
	ascomEngine := ascom.NewEngine(logger, config.HealthCheckInterval)

	coord := &ASCOMCoordinator{
		BaseCoordinator: baseCoord,
		ascomEngine:     ascomEngine,
		db:              db,
		config:          config,
		deviceRegistry:  make(map[string]*ASCOMDevice),
	}

	// Register health checks
	coord.RegisterHealthCheck(ascomEngine)

	// Register shutdown functions
	coord.RegisterShutdownFunc(func(ctx context.Context) error {
		if coord.httpServer != nil {
			logger.Info("Shutting down HTTP server")
			shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			if err := coord.httpServer.Shutdown(shutdownCtx); err != nil {
				logger.Error("HTTP server shutdown error", zap.Error(err))
			}
		}
		return nil
	})

	coord.RegisterShutdownFunc(func(ctx context.Context) error {
		if coord.discoveryService != nil {
			logger.Info("Stopping discovery service")
			coord.discoveryService.Stop()
		}
		return nil
	})

	coord.RegisterShutdownFunc(func(ctx context.Context) error {
		ascomEngine.Stop()
		return nil
	})

	coord.RegisterShutdownFunc(func(ctx context.Context) error {
		db.Close()
		logger.Info("Closed database connection")
		return nil
	})

	return coord, nil
}

// Start begins coordinator operations and starts the ASCOM Alpaca server.
func (c *ASCOMCoordinator) Start(ctx context.Context) error {
	if err := c.BaseCoordinator.Start(ctx); err != nil {
		return err
	}

	// Load device configurations from database
	if err := c.loadDevices(ctx); err != nil {
		return fmt.Errorf("failed to load devices: %w", err)
	}

	// Start ASCOM engine
	c.ascomEngine.Start(ctx)

	// Subscribe to ASCOM coordinator topics
	topics := []string{
		mqtt.NewTopicBuilder().Component("ascom").Action("config").Resource("create").Build(),
		mqtt.NewTopicBuilder().Component("ascom").Action("config").Resource("update").Build(),
		mqtt.NewTopicBuilder().Component("ascom").Action("config").Resource("delete").Build(),
		mqtt.NewTopicBuilder().Component("ascom").Action("config").Resource("list").Build(),
		mqtt.NewTopicBuilder().Component("ascom").Action("config").Resource("get").Build(),
		mqtt.NewTopicBuilder().Component("ascom").Action("device").Resource("reload").Build(),
	}

	for _, topic := range topics {
		if err := c.GetMQTTClient().Subscribe(topic, 1, c.handleMessageWrapper); err != nil {
			c.GetLogger().Error("Failed to subscribe to topic",
				zap.String("topic", topic),
				zap.Error(err))
			return fmt.Errorf("failed to subscribe to %s: %w", topic, err)
		}
		c.GetLogger().Info("Subscribed to topic", zap.String("topic", topic))
	}

	// Start UDP discovery service
	apiPort := extractPort(c.config.HTTPListenAddress)
	if apiPort == 0 {
		apiPort = 11111
	}

	c.discoveryService = ascomserver.NewDiscoveryService(
		c.config.DiscoveryPort,
		apiPort,
		c.GetLogger())

	if err := c.discoveryService.Start(); err != nil {
		return fmt.Errorf("failed to start discovery service: %w", err)
	}

	c.GetLogger().Info("Discovery service started",
		zap.Int("udp_port", c.config.DiscoveryPort),
		zap.Int("api_port", apiPort))

	// Initialize HTTP router for ASCOM API
	router := c.setupRouter()

	// Create HTTP server
	c.httpServer = &http.Server{
		Addr:         c.config.HTTPListenAddress,
		Handler:      router,
		ReadTimeout:  c.config.ReadTimeout,
		WriteTimeout: c.config.WriteTimeout,
		IdleTimeout:  c.config.IdleTimeout,
	}

	// Start HTTP server in goroutine
	go func() {
		c.GetLogger().Info("Starting ASCOM HTTP server",
			zap.String("address", c.config.HTTPListenAddress))

		if err := c.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			c.GetLogger().Error("HTTP server error", zap.Error(err))
		}
	}()

	// Start health status publishing
	go c.BaseCoordinator.StartHealthPublishing(ctx)

	c.GetLogger().Info("ASCOM coordinator started",
		zap.Int("device_count", len(c.deviceRegistry)))

	return nil
}

// setupRouter creates and configures the HTTP router for ASCOM API.
func (c *ASCOMCoordinator) setupRouter() *gin.Engine {
	// Set Gin mode based on log level
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(c.loggingMiddleware())

	if c.config.EnableCORS {
		router.Use(c.corsMiddleware())
	}

	// Management API endpoints (ASCOM Alpaca standard)
	management := router.Group("/management")
	{
		management.GET("/apiversions", c.handleAPIVersions)
		management.GET("/v1/description", c.handleServerDescription)
		management.GET("/v1/configureddevices", c.handleConfiguredDevices)
	}

	// Device API endpoints will be registered dynamically based on loaded devices
	// This happens in registerDeviceRoutes() called from loadDevices()

	return router
}

// loadDevices loads device configurations from the database.
func (c *ASCOMCoordinator) loadDevices(ctx context.Context) error {
	c.GetLogger().Info("Loading device configurations from database")

	query := `
		SELECT id, device_type, device_number, name, description, unique_id,
		       backend_mode, backend_config, organization_id, created_by,
		       created_at, updated_at, enabled
		FROM ascom_devices
		WHERE enabled = true
		ORDER BY device_type, device_number
	`

	rows, err := c.db.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query devices: %w", err)
	}
	defer rows.Close()

	deviceCount := 0
	for rows.Next() {
		var device ASCOMDevice
		var backendConfigJSON []byte

		err := rows.Scan(
			&device.ID,
			&device.DeviceType,
			&device.DeviceNumber,
			&device.Name,
			&device.Description,
			&device.UniqueID,
			&device.BackendMode,
			&backendConfigJSON,
			&device.OrganizationID,
			&device.CreatedBy,
			&device.CreatedAt,
			&device.UpdatedAt,
			&device.Enabled,
		)
		if err != nil {
			c.GetLogger().Error("Failed to scan device row", zap.Error(err))
			continue
		}

		// Parse backend config JSON
		if err := json.Unmarshal(backendConfigJSON, &device.BackendConfig); err != nil {
			c.GetLogger().Error("Failed to parse backend config",
				zap.String("device_id", device.ID),
				zap.Error(err))
			continue
		}

		// Add to registry
		key := deviceKey(device.DeviceType, device.DeviceNumber)
		c.deviceRegistry[key] = &device

		c.GetLogger().Info("Loaded device",
			zap.String("id", device.ID),
			zap.String("type", device.DeviceType),
			zap.Int("number", device.DeviceNumber),
			zap.String("name", device.Name))

		deviceCount++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating device rows: %w", err)
	}

	c.GetLogger().Info("Device configurations loaded",
		zap.Int("count", deviceCount))

	return nil
}

// handleMessageWrapper wraps handleMessage to satisfy MessageHandler signature.
func (c *ASCOMCoordinator) handleMessageWrapper(topic string, payload []byte) error {
	c.handleMessage(topic, payload)
	return nil
}

// handleMessage routes MQTT messages to appropriate handlers.
func (c *ASCOMCoordinator) handleMessage(topic string, payload []byte) {
	c.GetLogger().Debug("Received message",
		zap.String("topic", topic),
		zap.Int("payload_size", len(payload)))

	ctx := context.Background()

	// Route based on topic
	switch topic {
	case "bigskies/coordinator/ascom/config/create":
		c.handleCreateDevice(ctx, payload)
	case "bigskies/coordinator/ascom/config/update":
		c.handleUpdateDevice(ctx, payload)
	case "bigskies/coordinator/ascom/config/delete":
		c.handleDeleteDevice(ctx, payload)
	case "bigskies/coordinator/ascom/config/list":
		c.handleListDevices(ctx, payload)
	case "bigskies/coordinator/ascom/config/get":
		c.handleGetDevice(ctx, payload)
	case "bigskies/coordinator/ascom/device/reload":
		c.handleReloadDevices(ctx, payload)
	default:
		c.GetLogger().Warn("Unhandled topic", zap.String("topic", topic))
	}
}

// Placeholder message handlers (to be implemented)
func (c *ASCOMCoordinator) handleCreateDevice(ctx context.Context, payload []byte) {
	c.GetLogger().Info("handleCreateDevice called")
	// TODO: Implement device creation
}

func (c *ASCOMCoordinator) handleUpdateDevice(ctx context.Context, payload []byte) {
	c.GetLogger().Info("handleUpdateDevice called")
	// TODO: Implement device update
}

func (c *ASCOMCoordinator) handleDeleteDevice(ctx context.Context, payload []byte) {
	c.GetLogger().Info("handleDeleteDevice called")
	// TODO: Implement device deletion
}

func (c *ASCOMCoordinator) handleListDevices(ctx context.Context, payload []byte) {
	c.GetLogger().Info("handleListDevices called")
	// TODO: Implement device listing
}

func (c *ASCOMCoordinator) handleGetDevice(ctx context.Context, payload []byte) {
	c.GetLogger().Info("handleGetDevice called")
	// TODO: Implement device retrieval
}

func (c *ASCOMCoordinator) handleReloadDevices(ctx context.Context, payload []byte) {
	c.GetLogger().Info("Reloading devices from database")
	if err := c.loadDevices(ctx); err != nil {
		c.GetLogger().Error("Failed to reload devices", zap.Error(err))
	}
}

// HTTP endpoint handlers for ASCOM management API
func (c *ASCOMCoordinator) handleAPIVersions(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"Value":                []int{1},
		"ErrorNumber":          0,
		"ErrorMessage":         "",
	})
}

func (c *ASCOMCoordinator) handleServerDescription(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"Value": gin.H{
			"ServerName":        c.config.ServerName,
			"Manufacturer":      c.config.Manufacturer,
			"ManufacturerVersion": "1.0.0",
			"Location":          c.config.HTTPListenAddress,
		},
		"ErrorNumber":  0,
		"ErrorMessage": "",
	})
}

func (c *ASCOMCoordinator) handleConfiguredDevices(ctx *gin.Context) {
	devices := make([]gin.H, 0, len(c.deviceRegistry))

	for _, device := range c.deviceRegistry {
		devices = append(devices, gin.H{
			"DeviceName":   device.Name,
			"DeviceType":   device.DeviceType,
			"DeviceNumber": device.DeviceNumber,
			"UniqueID":     device.UniqueID,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"Value":        devices,
		"ErrorNumber":  0,
		"ErrorMessage": "",
	})
}

// Middleware
func (c *ASCOMCoordinator) loggingMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		path := ctx.Request.URL.Path

		ctx.Next()

		c.GetLogger().Info("HTTP request",
			zap.String("method", ctx.Request.Method),
			zap.String("path", path),
			zap.Int("status", ctx.Writer.Status()),
			zap.Duration("duration", time.Since(start)))
	}
}

func (c *ASCOMCoordinator) corsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(http.StatusOK)
			return
		}

		ctx.Next()
	}
}

// Utility functions
func deviceKey(deviceType string, deviceNumber int) string {
	return fmt.Sprintf("%s-%d", deviceType, deviceNumber)
}

func extractPort(address string) int {
	var port int
	fmt.Sscanf(address, "%*[^:]:%d", &port)
	return port
}
