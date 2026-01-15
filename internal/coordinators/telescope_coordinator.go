// Package coordinators provides coordinator implementations.
package coordinators

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/engines/ascom"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
	"go.uber.org/zap"
)

// TelescopeCoordinator manages telescope configurations and ASCOM device operations.
// It provides multi-tenant telescope management with RBAC integration.
type TelescopeCoordinator struct {
	*BaseCoordinator
	ascomEngine *ascom.Engine
	db          *pgxpool.Pool
	config      *TelescopeConfig
}

// TelescopeConfig holds configuration for the telescope coordinator.
type TelescopeConfig struct {
	BaseConfig
	DatabaseURL         string        `json:"database_url"`
	DiscoveryPort       int           `json:"discovery_port"` // Default 32227 for ASCOM Alpaca
	HealthCheckInterval time.Duration `json:"health_check_interval"`
}

// NewTelescopeCoordinator creates a new telescope coordinator instance.
func NewTelescopeCoordinator(config *TelescopeConfig, logger *zap.Logger) (*TelescopeCoordinator, error) {
	if config.Name == "" {
		config.Name = "telescope"
	}

	if config.DiscoveryPort == 0 {
		config.DiscoveryPort = 32227
	}

	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
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
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create MQTT client
	brokerURL := ""
	if config.MQTTConfig != nil {
		brokerURL = config.MQTTConfig.BrokerURL
	}
	mqttClient, err := CreateMQTTClient(brokerURL, mqtt.CoordinatorTelescope, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MQTT client: %w", err)
	}

	baseCoord := NewBaseCoordinator(mqtt.CoordinatorTelescope, mqttClient, logger)

	// Initialize ASCOM engine
	ascomEngine := ascom.NewEngine(logger, config.HealthCheckInterval)

	coord := &TelescopeCoordinator{
		BaseCoordinator: baseCoord,
		ascomEngine:     ascomEngine,
		db:              db,
		config:          config,
	}

	// Register health checks
	coord.RegisterHealthCheck(ascomEngine)

	// Register shutdown functions
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

// Start begins coordinator operations and subscribes to MQTT topics.
func (c *TelescopeCoordinator) Start(ctx context.Context) error {
	if err := c.BaseCoordinator.Start(ctx); err != nil {
		return err
	}

	// Start ASCOM engine
	c.ascomEngine.Start(ctx)

	// Subscribe to telescope topics
	topics := []string{
		mqtt.NewTopicBuilder().Component("telescope").Action("config").Resource("create").Build(),
		mqtt.NewTopicBuilder().Component("telescope").Action("config").Resource("update").Build(),
		mqtt.NewTopicBuilder().Component("telescope").Action("config").Resource("delete").Build(),
		mqtt.NewTopicBuilder().Component("telescope").Action("config").Resource("list").Build(),
		mqtt.NewTopicBuilder().Component("telescope").Action("config").Resource("get").Build(),
		mqtt.NewTopicBuilder().Component("telescope").Action("device").Resource("discover").Build(),
		mqtt.NewTopicBuilder().Component("telescope").Action("device").Resource("connect").Build(),
		mqtt.NewTopicBuilder().Component("telescope").Action("device").Resource("disconnect").Build(),
		mqtt.NewTopicBuilder().Component("telescope").Action("control").Resource("slew").Build(),
		mqtt.NewTopicBuilder().Component("telescope").Action("control").Resource("park").Build(),
		mqtt.NewTopicBuilder().Component("telescope").Action("control").Resource("unpark").Build(),
		mqtt.NewTopicBuilder().Component("telescope").Action("control").Resource("track").Build(),
		mqtt.NewTopicBuilder().Component("telescope").Action("control").Resource("abort").Build(),
		mqtt.NewTopicBuilder().Component("telescope").Action("status").Resource("get").Build(),
		mqtt.NewTopicBuilder().Component("telescope").Action("session").Resource("start").Build(),
		mqtt.NewTopicBuilder().Component("telescope").Action("session").Resource("end").Build(),
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

	// Start health status publishing
	go c.BaseCoordinator.StartHealthPublishing(ctx)

	c.GetLogger().Info("Telescope coordinator started")
	return nil
}

// handleMessageWrapper wraps handleMessage to satisfy MessageHandler signature.
func (c *TelescopeCoordinator) handleMessageWrapper(topic string, payload []byte) error {
	c.handleMessage(topic, payload)
	return nil
}

// handleMessage routes MQTT messages to appropriate handlers.
func (c *TelescopeCoordinator) handleMessage(topic string, payload []byte) {
	c.GetLogger().Debug("Received message",
		zap.String("topic", topic),
		zap.Int("payload_size", len(payload)))

	ctx := context.Background()

	// Route based on topic
	switch topic {
	case "bigskies/coordinator/telescope/config/create":
		c.handleCreateConfig(ctx, payload)
	case "bigskies/coordinator/telescope/config/update":
		c.handleUpdateConfig(ctx, payload)
	case "bigskies/coordinator/telescope/config/delete":
		c.handleDeleteConfig(ctx, payload)
	case "bigskies/coordinator/telescope/config/list":
		c.handleListConfigs(ctx, payload)
	case "bigskies/coordinator/telescope/config/get":
		c.handleGetConfig(ctx, payload)
	case "bigskies/coordinator/telescope/device/discover":
		c.handleDiscoverDevices(ctx, payload)
	case "bigskies/coordinator/telescope/device/connect":
		c.handleConnectDevice(ctx, payload)
	case "bigskies/coordinator/telescope/device/disconnect":
		c.handleDisconnectDevice(ctx, payload)
	case "bigskies/coordinator/telescope/control/slew":
		c.handleSlewTelescope(ctx, payload)
	case "bigskies/coordinator/telescope/control/park":
		c.handleParkTelescope(ctx, payload)
	case "bigskies/coordinator/telescope/control/unpark":
		c.handleUnparkTelescope(ctx, payload)
	case "bigskies/coordinator/telescope/control/track":
		c.handleSetTracking(ctx, payload)
	case "bigskies/coordinator/telescope/control/abort":
		c.handleAbortSlew(ctx, payload)
	case "bigskies/coordinator/telescope/status/get":
		c.handleGetStatus(ctx, payload)
	case "bigskies/coordinator/telescope/session/start":
		c.handleStartSession(ctx, payload)
	case "bigskies/coordinator/telescope/session/end":
		c.handleEndSession(ctx, payload)
	default:
		c.GetLogger().Warn("Unhandled topic", zap.String("topic", topic))
	}
}

// handleCreateConfig creates a new telescope configuration.
func (c *TelescopeCoordinator) handleCreateConfig(ctx context.Context, payload []byte) {
	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		OwnerID     string  `json:"owner_id"`
		OwnerType   string  `json:"owner_type"`
		SiteID      string  `json:"site_id"`
		MountType   string  `json:"mount_type"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal create config request", zap.Error(err))
		c.publishResponse("config/create/response", map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Validate and normalize mount type
	mountType := req.MountType
	if mountType == "altazimuth" {
		mountType = "altaz"
	}
	if mountType != "altaz" && mountType != "equatorial" && mountType != "dobsonian" {
		mountType = "equatorial" // default
	}

	// Handle site_id - use NULL if empty, invalid UUID, or site doesn't exist
	var siteID interface{}
	if req.SiteID != "" {
		// Validate UUID format
		if parsedUUID, err := uuid.Parse(req.SiteID); err == nil {
			// Check if site exists in database
			var siteExists bool
			checkSiteQuery := `SELECT EXISTS(SELECT 1 FROM observatory_sites WHERE id = $1)`
			err = c.db.QueryRow(ctx, checkSiteQuery, parsedUUID.String()).Scan(&siteExists)
			if err == nil && siteExists {
				siteID = parsedUUID.String()
			} else {
				// Site doesn't exist, use NULL
				siteID = nil
				c.GetLogger().Debug("Site ID not found, using NULL",
					zap.String("site_id", req.SiteID))
			}
		} else {
			// Invalid UUID format
			siteID = nil
		}
	} else {
		siteID = nil
	}

	// Ensure the owner user exists (for testing/development)
	// Check if user exists, if not create a stub user
	var userExists bool
	checkUserQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`
	err := c.db.QueryRow(ctx, checkUserQuery, req.OwnerID).Scan(&userExists)
	if err != nil {
		c.GetLogger().Error("Failed to check user existence", zap.Error(err))
		c.publishResponse("config/create/response", map[string]interface{}{
			"success": false,
			"error":   "Failed to validate user",
		})
		return
	}

	if !userExists {
		// Create a stub user for testing/development
		userInsertQuery := `
			INSERT INTO users (id, username, email, password_hash, enabled)
			VALUES ($1, $2, $3, $4, true)
			ON CONFLICT (id) DO NOTHING
		`
		_, err = c.db.Exec(ctx, userInsertQuery,
			req.OwnerID,
			"test_user_"+req.OwnerID[:8],
			"test_"+req.OwnerID[:8]+"@test.local",
			"$2a$10$stub")
		if err != nil {
			c.GetLogger().Warn("Failed to create stub user, continuing anyway",
				zap.Error(err))
		}
	}

	// Create telescope configuration
	configID := uuid.New().String()
	query := `
		INSERT INTO telescope_configurations 
		(id, name, description, owner_id, owner_type, site_id, mount_type, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true, NOW(), NOW())
	`

	_, err = c.db.Exec(ctx, query,
		configID, req.Name, req.Description, req.OwnerID, req.OwnerType,
		siteID, mountType)

	if err != nil {
		c.GetLogger().Error("Failed to create telescope configuration", zap.Error(err))
		c.publishResponse("config/create/response", map[string]interface{}{
			"success": false,
			"error":   "Failed to create configuration",
		})
		return
	}

	c.publishResponse("config/create/response", map[string]interface{}{
		"success": true,
		"id":      configID,
	})

	c.GetLogger().Info("Telescope configuration created",
		zap.String("config_id", configID),
		zap.String("name", req.Name))
}

// handleUpdateConfig updates an existing telescope configuration.
func (c *TelescopeCoordinator) handleUpdateConfig(ctx context.Context, payload []byte) {
	var req struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		MountType   string `json:"mount_type"`
		Enabled     bool   `json:"enabled"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal update config request", zap.Error(err))
		c.publishResponse("config/update/response", map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Validate and normalize mount type
	mountType := req.MountType
	if mountType == "altazimuth" {
		mountType = "altaz"
	}
	if mountType != "altaz" && mountType != "equatorial" && mountType != "dobsonian" {
		// Keep existing value if invalid
		mountType = req.MountType
	}

	query := `
		UPDATE telescope_configurations
		SET name = $2, description = $3, mount_type = $4, enabled = $5, updated_at = NOW()
		WHERE id = $1
	`

	result, err := c.db.Exec(ctx, query, req.ID, req.Name, req.Description, mountType, req.Enabled)
	if err != nil {
		c.GetLogger().Error("Failed to update telescope configuration", zap.Error(err))
		c.publishResponse("config/update/response", map[string]interface{}{
			"success": false,
			"error":   "Failed to update configuration",
		})
		return
	}

	if result.RowsAffected() == 0 {
		c.publishResponse("config/update/response", map[string]interface{}{
			"success": false,
			"error":   "Configuration not found",
		})
		return
	}

	c.publishResponse("config/update/response", map[string]interface{}{
		"success": true,
	})

	c.GetLogger().Info("Telescope configuration updated", zap.String("config_id", req.ID))
}

// handleDeleteConfig deletes a telescope configuration.
func (c *TelescopeCoordinator) handleDeleteConfig(ctx context.Context, payload []byte) {
	var req struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal delete config request", zap.Error(err))
		c.publishResponse("config/delete/response", map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Unregister from ASCOM engine first
	c.ascomEngine.UnregisterTelescope(req.ID)

	// Delete from database (cascade will handle related records)
	query := `DELETE FROM telescope_configurations WHERE id = $1`
	result, err := c.db.Exec(ctx, query, req.ID)
	if err != nil {
		c.GetLogger().Error("Failed to delete telescope configuration", zap.Error(err))
		c.publishResponse("config/delete/response", map[string]interface{}{
			"success": false,
			"error":   "Failed to delete configuration",
		})
		return
	}

	if result.RowsAffected() == 0 {
		c.publishResponse("config/delete/response", map[string]interface{}{
			"success": false,
			"error":   "Configuration not found",
		})
		return
	}

	c.publishResponse("config/delete/response", map[string]interface{}{
		"success": true,
	})

	c.GetLogger().Info("Telescope configuration deleted", zap.String("config_id", req.ID))
}

// handleListConfigs lists telescope configurations for a user.
func (c *TelescopeCoordinator) handleListConfigs(ctx context.Context, payload []byte) {
	var req struct {
		UserID string `json:"user_id"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal list configs request", zap.Error(err))
		c.publishResponse("config/list/response", map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Query configurations owned by user or accessible via permissions
	query := `
		SELECT DISTINCT tc.id, tc.name, tc.description, tc.owner_id, tc.owner_type, 
		       tc.site_id, tc.mount_type, tc.enabled, tc.created_at, tc.updated_at
		FROM telescope_configurations tc
		LEFT JOIN telescope_permissions tp ON tc.id = tp.telescope_id
		WHERE tc.owner_id = $1 
		   OR (tp.principal_id = $1 AND tp.principal_type = 'user')
		ORDER BY tc.name
	`

	rows, err := c.db.Query(ctx, query, req.UserID)
	if err != nil {
		c.GetLogger().Error("Failed to list telescope configurations", zap.Error(err))
		c.publishResponse("config/list/response", map[string]interface{}{
			"success": false,
			"error":   "Failed to list configurations",
		})
		return
	}
	defer rows.Close()

	configs := make([]map[string]interface{}, 0)
	for rows.Next() {
		var config struct {
			ID          string
			Name        string
			Description string
			OwnerID     string
			OwnerType   string
			SiteID      *string
			MountType   string
			Enabled     bool
			CreatedAt   time.Time
			UpdatedAt   time.Time
		}

		err := rows.Scan(&config.ID, &config.Name, &config.Description,
			&config.OwnerID, &config.OwnerType, &config.SiteID,
			&config.MountType, &config.Enabled, &config.CreatedAt, &config.UpdatedAt)
		if err != nil {
			c.GetLogger().Error("Failed to scan telescope configuration", zap.Error(err))
			continue
		}

		configs = append(configs, map[string]interface{}{
			"id":          config.ID,
			"name":        config.Name,
			"description": config.Description,
			"owner_id":    config.OwnerID,
			"owner_type":  config.OwnerType,
			"site_id":     config.SiteID,
			"mount_type":  config.MountType,
			"enabled":     config.Enabled,
			"created_at":  config.CreatedAt,
			"updated_at":  config.UpdatedAt,
		})
	}

	c.publishResponse("config/list/response", map[string]interface{}{
		"success": true,
		"configs": configs,
	})
}

// handleGetConfig gets a specific telescope configuration.
func (c *TelescopeCoordinator) handleGetConfig(ctx context.Context, payload []byte) {
	var req struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal get config request", zap.Error(err))
		c.publishResponse("config/get/response", map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	if req.ID == "" {
		c.publishResponse("config/get/response", map[string]interface{}{
			"success": false,
			"error":   "Configuration ID is required",
		})
		return
	}

	query := `
		SELECT id, name, description, owner_id, owner_type, site_id, mount_type, 
		       enabled, created_at, updated_at
		FROM telescope_configurations
		WHERE id = $1
	`

	var config struct {
		ID          string
		Name        string
		Description string
		OwnerID     string
		OwnerType   string
		SiteID      *string
		MountType   string
		Enabled     bool
		CreatedAt   time.Time
		UpdatedAt   time.Time
	}

	err := c.db.QueryRow(ctx, query, req.ID).Scan(
		&config.ID, &config.Name, &config.Description,
		&config.OwnerID, &config.OwnerType, &config.SiteID,
		&config.MountType, &config.Enabled, &config.CreatedAt, &config.UpdatedAt)

	if err != nil {
		c.GetLogger().Error("Failed to get telescope configuration", zap.Error(err))
		c.publishResponse("config/get/response", map[string]interface{}{
			"success": false,
			"error":   "Configuration not found",
		})
		return
	}

	c.publishResponse("config/get/response", map[string]interface{}{
		"success": true,
		"config": map[string]interface{}{
			"id":          config.ID,
			"name":        config.Name,
			"description": config.Description,
			"owner_id":    config.OwnerID,
			"owner_type":  config.OwnerType,
			"site_id":     config.SiteID,
			"mount_type":  config.MountType,
			"enabled":     config.Enabled,
			"created_at":  config.CreatedAt,
			"updated_at":  config.UpdatedAt,
		},
	})
}

// handleDiscoverDevices discovers ASCOM devices on the network.
func (c *TelescopeCoordinator) handleDiscoverDevices(ctx context.Context, payload []byte) {
	var req struct {
		Port int `json:"port"`
	}

	if err := json.Unmarshal(payload, &req); err != nil || req.Port == 0 {
		req.Port = c.config.DiscoveryPort
	}

	devices, err := c.ascomEngine.DiscoverDevices(ctx, req.Port)
	if err != nil {
		c.GetLogger().Error("Device discovery failed", zap.Error(err))
		c.publishResponse("device/discover/response", map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.publishResponse("device/discover/response", map[string]interface{}{
		"success": true,
		"devices": devices,
		"count":   len(devices),
	})

	c.GetLogger().Info("Device discovery completed",
		zap.Int("device_count", len(devices)))
}

// handleConnectDevice connects to an ASCOM device.
func (c *TelescopeCoordinator) handleConnectDevice(ctx context.Context, payload []byte) {
	var req struct {
		DeviceID string `json:"device_id"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal connect device request", zap.Error(err))
		c.publishResponse("device/connect/response", map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	err := c.ascomEngine.ConnectDevice(ctx, req.DeviceID)
	if err != nil {
		c.GetLogger().Error("Failed to connect device", zap.Error(err))
		c.publishResponse("device/connect/response", map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.publishResponse("device/connect/response", map[string]interface{}{
		"success": true,
	})

	c.GetLogger().Info("Device connected", zap.String("device_id", req.DeviceID))
}

// handleDisconnectDevice disconnects from an ASCOM device.
func (c *TelescopeCoordinator) handleDisconnectDevice(ctx context.Context, payload []byte) {
	var req struct {
		DeviceID string `json:"device_id"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal disconnect device request", zap.Error(err))
		c.publishResponse("device/disconnect/response", map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	err := c.ascomEngine.DisconnectDevice(ctx, req.DeviceID)
	if err != nil {
		c.GetLogger().Error("Failed to disconnect device", zap.Error(err))
		c.publishResponse("device/disconnect/response", map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.publishResponse("device/disconnect/response", map[string]interface{}{
		"success": true,
	})

	c.GetLogger().Info("Device disconnected", zap.String("device_id", req.DeviceID))
}

// handleSlewTelescope slews a telescope to coordinates.
func (c *TelescopeCoordinator) handleSlewTelescope(ctx context.Context, payload []byte) {
	var req struct {
		DeviceID         string  `json:"device_id"`
		TelescopeID      string  `json:"telescope_id"` // deprecated, use device_id
		RightAscension   float64 `json:"right_ascension"`
		Declination      float64 `json:"declination"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal slew request", zap.Error(err))
		c.publishResponse("control/slew/response", map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Support both device_id and telescope_id for backward compatibility
	deviceID := req.DeviceID
	if deviceID == "" {
		deviceID = req.TelescopeID
	}

	// Get telescope device
	device, err := c.ascomEngine.GetTelescopeDevice(deviceID, "telescope")
	if err != nil {
		c.GetLogger().Error("Failed to get telescope device", zap.Error(err))
		c.publishResponse("control/slew/response", map[string]interface{}{
			"success": false,
			"error":   "Telescope not found",
		})
		return
	}

	// Perform slew
	client := c.ascomEngine.GetClient()
	err = client.SlewToCoordinates(ctx, device, req.RightAscension, req.Declination)
	if err != nil {
		c.GetLogger().Error("Failed to slew telescope", zap.Error(err))
		c.publishResponse("control/slew/response", map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.publishResponse("control/slew/response", map[string]interface{}{
		"success": true,
	})

	c.GetLogger().Info("Telescope slew initiated",
		zap.String("device_id", deviceID),
		zap.Float64("ra", req.RightAscension),
		zap.Float64("dec", req.Declination))
}

// handleParkTelescope parks a telescope.
func (c *TelescopeCoordinator) handleParkTelescope(ctx context.Context, payload []byte) {
	var req struct {
		DeviceID    string `json:"device_id"`
		TelescopeID string `json:"telescope_id"` // deprecated, use device_id
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal park request", zap.Error(err))
		c.publishResponse("control/park/response", map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Support both device_id and telescope_id for backward compatibility
	deviceID := req.DeviceID
	if deviceID == "" {
		deviceID = req.TelescopeID
	}

	device, err := c.ascomEngine.GetTelescopeDevice(deviceID, "telescope")
	if err != nil {
		c.publishResponse("control/park/response", map[string]interface{}{
			"success": false,
			"error":   "Telescope not found",
		})
		return
	}

	client := c.ascomEngine.GetClient()
	err = client.Park(ctx, device)
	if err != nil {
		c.GetLogger().Error("Failed to park telescope", zap.Error(err))
		c.publishResponse("control/park/response", map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.publishResponse("control/park/response", map[string]interface{}{
		"success": true,
	})

	c.GetLogger().Info("Telescope parked", zap.String("device_id", deviceID))
}

// handleUnparkTelescope unparks a telescope.
func (c *TelescopeCoordinator) handleUnparkTelescope(ctx context.Context, payload []byte) {
	var req struct {
		DeviceID    string `json:"device_id"`
		TelescopeID string `json:"telescope_id"` // deprecated, use device_id
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal unpark request", zap.Error(err))
		c.publishResponse("control/unpark/response", map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Support both device_id and telescope_id for backward compatibility
	deviceID := req.DeviceID
	if deviceID == "" {
		deviceID = req.TelescopeID
	}

	device, err := c.ascomEngine.GetTelescopeDevice(deviceID, "telescope")
	if err != nil {
		c.publishResponse("control/unpark/response", map[string]interface{}{
			"success": false,
			"error":   "Telescope not found",
		})
		return
	}

	client := c.ascomEngine.GetClient()
	err = client.Unpark(ctx, device)
	if err != nil {
		c.GetLogger().Error("Failed to unpark telescope", zap.Error(err))
		c.publishResponse("control/unpark/response", map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.publishResponse("control/unpark/response", map[string]interface{}{
		"success": true,
	})

	c.GetLogger().Info("Telescope unparked", zap.String("device_id", deviceID))
}

// handleSetTracking enables or disables telescope tracking.
func (c *TelescopeCoordinator) handleSetTracking(ctx context.Context, payload []byte) {
	var req struct {
		DeviceID    string `json:"device_id"`
		TelescopeID string `json:"telescope_id"` // deprecated, use device_id
		Enabled     bool   `json:"enabled"`       // alias for tracking
		Tracking    bool   `json:"tracking"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal tracking request", zap.Error(err))
		c.publishResponse("control/track/response", map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Support both device_id and telescope_id for backward compatibility
	deviceID := req.DeviceID
	if deviceID == "" {
		deviceID = req.TelescopeID
	}

	// Support both enabled and tracking fields
	tracking := req.Tracking || req.Enabled

	device, err := c.ascomEngine.GetTelescopeDevice(deviceID, "telescope")
	if err != nil {
		c.publishResponse("control/track/response", map[string]interface{}{
			"success": false,
			"error":   "Telescope not found",
		})
		return
	}

	client := c.ascomEngine.GetClient()
	err = client.SetTracking(ctx, device, tracking)
	if err != nil {
		c.GetLogger().Error("Failed to set tracking", zap.Error(err))
		c.publishResponse("control/track/response", map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.publishResponse("control/track/response", map[string]interface{}{
		"success": true,
	})

	c.GetLogger().Info("Telescope tracking set",
		zap.String("device_id", deviceID),
		zap.Bool("tracking", tracking))
}

// handleAbortSlew aborts a telescope slew.
func (c *TelescopeCoordinator) handleAbortSlew(ctx context.Context, payload []byte) {
	var req struct {
		DeviceID    string `json:"device_id"`
		TelescopeID string `json:"telescope_id"` // deprecated, use device_id
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal abort request", zap.Error(err))
		c.publishResponse("control/abort/response", map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Support both device_id and telescope_id for backward compatibility
	deviceID := req.DeviceID
	if deviceID == "" {
		deviceID = req.TelescopeID
	}

	device, err := c.ascomEngine.GetTelescopeDevice(deviceID, "telescope")
	if err != nil {
		c.publishResponse("control/abort/response", map[string]interface{}{
			"success": false,
			"error":   "Telescope not found",
		})
		return
	}

	client := c.ascomEngine.GetClient()
	err = client.AbortSlew(ctx, device)
	if err != nil {
		c.GetLogger().Error("Failed to abort slew", zap.Error(err))
		c.publishResponse("control/abort/response", map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.publishResponse("control/abort/response", map[string]interface{}{
		"success": true,
	})

	c.GetLogger().Info("Telescope slew aborted", zap.String("device_id", deviceID))
}

// handleGetStatus gets telescope status.
func (c *TelescopeCoordinator) handleGetStatus(ctx context.Context, payload []byte) {
	var req struct {
		DeviceID    string `json:"device_id"`
		TelescopeID string `json:"telescope_id"` // deprecated, use device_id
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal status request", zap.Error(err))
		c.publishResponse("status/get/response", map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Support both device_id and telescope_id for backward compatibility
	deviceID := req.DeviceID
	if deviceID == "" {
		deviceID = req.TelescopeID
	}

	device, err := c.ascomEngine.GetTelescopeDevice(deviceID, "telescope")
	if err != nil {
		c.publishResponse("status/get/response", map[string]interface{}{
			"success": false,
			"error":   "Telescope not found",
		})
		return
	}

	client := c.ascomEngine.GetClient()
	status, err := client.GetTelescopeStatus(ctx, device)
	if err != nil {
		c.GetLogger().Error("Failed to get telescope status", zap.Error(err))
		c.publishResponse("status/get/response", map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.publishResponse("status/get/response", map[string]interface{}{
		"success": true,
		"status":  status,
	})
}

// handleStartSession starts a telescope session.
func (c *TelescopeCoordinator) handleStartSession(ctx context.Context, payload []byte) {
	var req struct {
		ConfigID    string `json:"config_id"`     // telescope configuration ID
		TelescopeID string `json:"telescope_id"`  // deprecated, use config_id
		UserID      string `json:"user_id"`
		SessionName string `json:"session_name"`  // alias for session_type
		SessionType string `json:"session_type"`
		Notes       string `json:"notes"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal start session request", zap.Error(err))
		c.publishResponse("session/start/response", map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Support both config_id and telescope_id
	telescopeID := req.ConfigID
	if telescopeID == "" {
		telescopeID = req.TelescopeID
	}

	// Support both session_name and session_type
	sessionType := req.SessionType
	if sessionType == "" {
		sessionType = req.SessionName
	}

	// Validate and normalize session type to match database constraint
	// Valid values: 'manual', 'automated', 'maintenance'
	if sessionType != "manual" && sessionType != "automated" && sessionType != "maintenance" {
		sessionType = "manual" // default to manual for any other value
	}

	sessionID := uuid.New().String()
	query := `
		INSERT INTO telescope_sessions (id, telescope_id, user_id, started_at, status, session_type, notes)
		VALUES ($1, $2, $3, NOW(), 'active', $4, $5)
	`

	_, err := c.db.Exec(ctx, query, sessionID, telescopeID, req.UserID, sessionType, req.Notes)
	if err != nil {
		c.GetLogger().Error("Failed to start session", zap.Error(err))
		c.publishResponse("session/start/response", map[string]interface{}{
			"success": false,
			"error":   "Failed to start session",
		})
		return
	}

	c.publishResponse("session/start/response", map[string]interface{}{
		"success":    true,
		"session_id": sessionID,
	})

	c.GetLogger().Info("Telescope session started",
		zap.String("session_id", sessionID),
		zap.String("telescope_id", telescopeID),
		zap.String("user_id", req.UserID))
}

// handleEndSession ends a telescope session.
func (c *TelescopeCoordinator) handleEndSession(ctx context.Context, payload []byte) {
	var req struct {
		SessionID string `json:"session_id"`
		Status    string `json:"status"`
		Notes     string `json:"notes"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal end session request", zap.Error(err))
		c.publishResponse("session/end/response", map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	query := `
		UPDATE telescope_sessions
		SET ended_at = NOW(), status = $2, notes = $3
		WHERE id = $1 AND ended_at IS NULL
	`

	result, err := c.db.Exec(ctx, query, req.SessionID, req.Status, req.Notes)
	if err != nil {
		c.GetLogger().Error("Failed to end session", zap.Error(err))
		c.publishResponse("session/end/response", map[string]interface{}{
			"success": false,
			"error":   "Failed to end session",
		})
		return
	}

	if result.RowsAffected() == 0 {
		c.publishResponse("session/end/response", map[string]interface{}{
			"success": false,
			"error":   "Session not found or already ended",
		})
		return
	}

	c.publishResponse("session/end/response", map[string]interface{}{
		"success": true,
	})

	c.GetLogger().Info("Telescope session ended", zap.String("session_id", req.SessionID))
}

// publishResponse publishes a response to an MQTT topic.
func (c *TelescopeCoordinator) publishResponse(subtopic string, payload interface{}) {
	topic := fmt.Sprintf("bigskies/coordinator/telescope/response/%s", subtopic)

	data, err := json.Marshal(payload)
	if err != nil {
		c.GetLogger().Error("Failed to marshal response", zap.Error(err))
		return
	}

	mqttClient := c.GetMQTTClient()
	if mqttClient == nil {
		c.GetLogger().Debug("MQTT client not available, skipping publish")
		return
	}

	if err := mqttClient.Publish(topic, 1, false, data); err != nil {
		c.GetLogger().Error("Failed to publish response",
			zap.String("topic", topic),
			zap.Error(err))
	}
}

