// Package coordinators provides coordinator implementations.
package coordinators

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/engines/security"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/models"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
	"go.uber.org/zap"
)

// SecurityCoordinator manages security operations including auth, RBAC, and TLS.
type SecurityCoordinator struct {
	*BaseCoordinator
	appSecEngine     *security.AppSecurityEngine
	accountSecEngine *security.AccountSecurityEngine
	tlsSecEngine     *security.TLSSecurityEngine
	db               *pgxpool.Pool
	config           *SecurityConfig
}

// SecurityConfig holds configuration for the security coordinator.
type SecurityConfig struct {
	BaseConfig
	DatabaseURL      string                 `json:"database_url"`
	JWTSecret        string                 `json:"jwt_secret"`
	TokenDuration    time.Duration          `json:"token_duration"`
	TLSConfig        *security.TLSConfig    `json:"tls_config"`
}

// NewSecurityCoordinator creates a new security coordinator instance.
func NewSecurityCoordinator(config *SecurityConfig, logger *zap.Logger) (*SecurityCoordinator, error) {
	if config.Name == "" {
		config.Name = "security"
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
	mqttClient, err := mqtt.NewClient(config.MQTTConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MQTT client: %w", err)
	}

	baseCoord := NewBaseCoordinator(config.Name, mqttClient, logger)

	// Initialize security engines
	appSecEngine := security.NewAppSecurityEngine(config.JWTSecret, config.TokenDuration, logger)
	accountSecEngine := security.NewAccountSecurityEngine(db, logger)
	tlsSecEngine := security.NewTLSSecurityEngine(db, config.TLSConfig, logger)

	coord := &SecurityCoordinator{
		BaseCoordinator:  baseCoord,
		appSecEngine:     appSecEngine,
		accountSecEngine: accountSecEngine,
		tlsSecEngine:     tlsSecEngine,
		db:               db,
		config:           config,
	}

	// Register health checks for engines
	coord.RegisterHealthCheck(appSecEngine)
	coord.RegisterHealthCheck(accountSecEngine)
	coord.RegisterHealthCheck(tlsSecEngine)

	// Register shutdown function for database
	coord.RegisterShutdownFunc(func(ctx context.Context) error {
		db.Close()
		logger.Info("Closed database connection")
		return nil
	})

	// Register shutdown function for TLS engine
	coord.RegisterShutdownFunc(func(ctx context.Context) error {
		tlsSecEngine.Stop()
		return nil
	})

	return coord, nil
}

// Start begins coordinator operations and subscribes to MQTT topics.
func (c *SecurityCoordinator) Start(ctx context.Context) error {
	if err := c.BaseCoordinator.Start(ctx); err != nil {
		return err
	}

	// Start TLS engine renewal monitoring
	c.tlsSecEngine.Start(ctx)

	// Subscribe to security topics
	topics := []string{
		mqtt.NewTopicBuilder().Component("security").Action("auth").Resource("login").Build(),
		mqtt.NewTopicBuilder().Component("security").Action("auth").Resource("validate").Build(),
		mqtt.NewTopicBuilder().Component("security").Action("user").Resource("create").Build(),
		mqtt.NewTopicBuilder().Component("security").Action("user").Resource("update").Build(),
		mqtt.NewTopicBuilder().Component("security").Action("user").Resource("delete").Build(),
		mqtt.NewTopicBuilder().Component("security").Action("role").Resource("assign").Build(),
		mqtt.NewTopicBuilder().Component("security").Action("permission").Resource("check").Build(),
		mqtt.NewTopicBuilder().Component("security").Action("cert").Resource("request").Build(),
		mqtt.NewTopicBuilder().Component("security").Action("cert").Resource("renew").Build(),
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

	// Publish health status
	go c.publishHealthStatus(ctx)

	c.GetLogger().Info("Security coordinator started")
	return nil
}

// handleMessageWrapper wraps handleMessage to satisfy MessageHandler signature.
func (c *SecurityCoordinator) handleMessageWrapper(topic string, payload []byte) error {
	c.handleMessage(topic, payload)
	return nil
}

// handleMessage routes MQTT messages to appropriate handlers.
func (c *SecurityCoordinator) handleMessage(topic string, payload []byte) {
	c.GetLogger().Debug("Received message",
		zap.String("topic", topic),
		zap.Int("payload_size", len(payload)))

	ctx := context.Background()

	// Route based on topic - using string comparison
	switch topic {
	case "bigskies/coordinator/security/auth/login":
		c.handleLogin(ctx, payload)
	case "bigskies/coordinator/security/auth/validate":
		c.handleValidateToken(ctx, payload)
	case "bigskies/coordinator/security/user/create":
		c.handleCreateUser(ctx, payload)
	case "bigskies/coordinator/security/user/update":
		c.handleUpdateUser(ctx, payload)
	case "bigskies/coordinator/security/user/delete":
		c.handleDeleteUser(ctx, payload)
	case "bigskies/coordinator/security/role/assign":
		c.handleAssignRole(ctx, payload)
	case "bigskies/coordinator/security/permission/check":
		c.handleCheckPermission(ctx, payload)
	case "bigskies/coordinator/security/cert/request":
		c.handleRequestCertificate(ctx, payload)
	case "bigskies/coordinator/security/cert/renew":
		c.handleRenewCertificate(ctx, payload)
	default:
		c.GetLogger().Warn("Unhandled topic", zap.String("topic", topic))
	}
}

// handleLogin processes login requests.
func (c *SecurityCoordinator) handleLogin(ctx context.Context, payload []byte) {
	var req models.AuthRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal auth request", zap.Error(err))
		return
	}

	// Authenticate user
	user, err := c.accountSecEngine.AuthenticateUser(ctx, req.Username, req.Password)
	if err != nil {
		c.GetLogger().Warn("Authentication failed",
			zap.String("username", req.Username),
			zap.Error(err))
		c.publishResponse("auth/login/response", map[string]interface{}{
			"success": false,
			"error":   "Authentication failed",
		})
		return
	}

	// Generate JWT token
	token, expiresAt, err := c.appSecEngine.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		c.GetLogger().Error("Failed to generate token", zap.Error(err))
		c.publishResponse("auth/login/response", map[string]interface{}{
			"success": false,
			"error":   "Failed to generate token",
		})
		return
	}

	// Send response
	response := models.AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	}

	c.publishResponse("auth/login/response", response)
	c.GetLogger().Info("User logged in successfully", zap.String("username", req.Username))
}

// handleValidateToken validates JWT tokens.
func (c *SecurityCoordinator) handleValidateToken(ctx context.Context, payload []byte) {
	var req models.TokenValidationRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal token validation request", zap.Error(err))
		return
	}

	claims, err := c.appSecEngine.ValidateToken(req.Token)
	if err != nil {
		c.publishResponse("auth/validate/response", models.TokenValidationResponse{
			Valid: false,
			Error: err.Error(),
		})
		return
	}

	c.publishResponse("auth/validate/response", models.TokenValidationResponse{
		Valid:  true,
		UserID: claims.UserID,
	})
}

// handleCreateUser creates a new user.
func (c *SecurityCoordinator) handleCreateUser(ctx context.Context, payload []byte) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal create user request", zap.Error(err))
		return
	}

	user, err := c.accountSecEngine.CreateUser(ctx, req.Username, req.Email, req.Password)
	if err != nil {
		c.GetLogger().Error("Failed to create user", zap.Error(err))
		c.publishResponse("user/create/response", map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.publishResponse("user/create/response", map[string]interface{}{
		"success": true,
		"user":    user,
	})
}

// handleUpdateUser updates user information.
func (c *SecurityCoordinator) handleUpdateUser(ctx context.Context, payload []byte) {
	var user models.User
	if err := json.Unmarshal(payload, &user); err != nil {
		c.GetLogger().Error("Failed to unmarshal update user request", zap.Error(err))
		return
	}

	if err := c.accountSecEngine.UpdateUser(ctx, &user); err != nil {
		c.GetLogger().Error("Failed to update user", zap.Error(err))
		c.publishResponse("user/update/response", map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.publishResponse("user/update/response", map[string]interface{}{
		"success": true,
	})
}

// handleDeleteUser deletes a user.
func (c *SecurityCoordinator) handleDeleteUser(ctx context.Context, payload []byte) {
	var req struct {
		UserID string `json:"user_id"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal delete user request", zap.Error(err))
		return
	}

	if err := c.accountSecEngine.DeleteUser(ctx, req.UserID); err != nil {
		c.GetLogger().Error("Failed to delete user", zap.Error(err))
		c.publishResponse("user/delete/response", map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.publishResponse("user/delete/response", map[string]interface{}{
		"success": true,
	})
}

// handleAssignRole assigns a role to a user.
func (c *SecurityCoordinator) handleAssignRole(ctx context.Context, payload []byte) {
	var req struct {
		UserID string `json:"user_id"`
		RoleID string `json:"role_id"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal assign role request", zap.Error(err))
		return
	}

	if err := c.accountSecEngine.AssignRoleToUser(ctx, req.UserID, req.RoleID); err != nil {
		c.GetLogger().Error("Failed to assign role", zap.Error(err))
		c.publishResponse("role/assign/response", map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.publishResponse("role/assign/response", map[string]interface{}{
		"success": true,
	})
}

// handleCheckPermission checks if a user has permission for a resource/action.
func (c *SecurityCoordinator) handleCheckPermission(ctx context.Context, payload []byte) {
	var req models.PermissionCheckRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal permission check request", zap.Error(err))
		return
	}

	allowed, err := c.accountSecEngine.CheckPermission(ctx, req.UserID, req.Resource, req.Action)
	if err != nil {
		c.GetLogger().Error("Failed to check permission", zap.Error(err))
		c.publishResponse("permission/check/response", models.PermissionCheckResponse{
			Allowed: false,
			Reason:  err.Error(),
		})
		return
	}

	reason := ""
	if !allowed {
		reason = "Permission denied"
	}

	c.publishResponse("permission/check/response", models.PermissionCheckResponse{
		Allowed: allowed,
		Reason:  reason,
	})
}

// handleRequestCertificate requests or generates a certificate.
func (c *SecurityCoordinator) handleRequestCertificate(ctx context.Context, payload []byte) {
	var req models.CertificateRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal certificate request", zap.Error(err))
		return
	}

	var cert *models.TLSCertificate
	var err error

	switch req.Type {
	case "self-signed":
		cert, err = c.tlsSecEngine.GenerateSelfSignedCertificate(ctx, req.Domain, 365)
	case "letsencrypt":
		cert, err = c.tlsSecEngine.RequestLetsEncryptCertificate(ctx, req.Domain)
	default:
		c.publishResponse("cert/request/response", models.CertificateResponse{
			Success: false,
			Error:   "Invalid certificate type",
		})
		return
	}

	if err != nil {
		c.GetLogger().Error("Failed to request certificate", zap.Error(err))
		c.publishResponse("cert/request/response", models.CertificateResponse{
			Success: false,
			Domain:  req.Domain,
			Error:   err.Error(),
		})
		return
	}

	c.publishResponse("cert/request/response", models.CertificateResponse{
		Success:   true,
		Domain:    cert.Domain,
		ExpiresAt: cert.ExpiresAt,
	})
}

// handleRenewCertificate renews an existing certificate.
func (c *SecurityCoordinator) handleRenewCertificate(ctx context.Context, payload []byte) {
	var req struct {
		Domain string `json:"domain"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		c.GetLogger().Error("Failed to unmarshal renew certificate request", zap.Error(err))
		return
	}

	// Get existing certificate
	existingCert, err := c.tlsSecEngine.GetCertificate(ctx, req.Domain)
	if err != nil {
		c.GetLogger().Error("Failed to get existing certificate", zap.Error(err))
		c.publishResponse("cert/renew/response", models.CertificateResponse{
			Success: false,
			Domain:  req.Domain,
			Error:   "Certificate not found",
		})
		return
	}

	var newCert *models.TLSCertificate

	// Renew based on issuer type
	if existingCert.Issuer == "letsencrypt" {
		newCert, err = c.tlsSecEngine.RequestLetsEncryptCertificate(ctx, req.Domain)
	} else {
		newCert, err = c.tlsSecEngine.GenerateSelfSignedCertificate(ctx, req.Domain, 365)
	}

	if err != nil {
		c.GetLogger().Error("Failed to renew certificate", zap.Error(err))
		c.publishResponse("cert/renew/response", models.CertificateResponse{
			Success: false,
			Domain:  req.Domain,
			Error:   err.Error(),
		})
		return
	}

	c.publishResponse("cert/renew/response", models.CertificateResponse{
		Success:   true,
		Domain:    newCert.Domain,
		ExpiresAt: newCert.ExpiresAt,
	})
}

// publishResponse publishes a response to an MQTT topic.
func (c *SecurityCoordinator) publishResponse(subtopic string, payload interface{}) {
	topic := fmt.Sprintf("bigskies/coordinator/security/response/%s", subtopic)

	data, err := json.Marshal(payload)
	if err != nil {
		c.GetLogger().Error("Failed to marshal response", zap.Error(err))
		return
	}

	if err := c.GetMQTTClient().Publish(topic, 1, false, data); err != nil {
		c.GetLogger().Error("Failed to publish response",
			zap.String("topic", topic),
			zap.Error(err))
	}
}

// publishHealthStatus publishes health status periodically.
func (c *SecurityCoordinator) publishHealthStatus(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			health := c.HealthCheck(ctx)
			topic := mqtt.NewTopicBuilder().
				Component("security").
				Action("health").
				Resource("status").
				Build()

			if err := c.GetMQTTClient().PublishJSON(topic, 1, false, health); err != nil {
				c.GetLogger().Error("Failed to publish health status", zap.Error(err))
			}
		}
	}
}
