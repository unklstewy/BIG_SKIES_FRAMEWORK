// Package ascom provides ASCOM protocol engines and bridges.
package ascom

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
)

// SecurityMiddleware provides JWT authentication and telescope authorization for ASCOM HTTP API.
// It validates tokens via the security-coordinator and checks telescope permissions against
// the telescope_permissions table to enforce multi-tenant RBAC.
//
// The middleware operates in two layers:
// 1. AuthenticateRequest - Validates JWT token and extracts user context
// 2. AuthorizeTelescope - Checks user permissions for specific telescope access
type SecurityMiddleware struct {
	// mqttClient communicates with security-coordinator for token validation
	mqttClient *mqtt.Client

	// db provides database access for telescope permission queries
	db *pgxpool.Pool

	// logger provides structured logging
	logger *zap.Logger

	// config holds security configuration
	config *SecurityConfig

	// pendingValidations tracks outstanding token validation requests
	// Key: request_id, Value: response channel
	pendingValidations sync.Map

	// validationTimeout is how long to wait for security-coordinator responses
	validationTimeout time.Duration
}

// SecurityConfig holds configuration for the security middleware.
type SecurityConfig struct {
	// RequireAuth enables/disables authentication requirement
	// Default: true. Set to false for backward compatibility during migration.
	RequireAuth bool

	// AllowAnonymousRead allows unauthenticated read-only operations (GET requests)
	// Only applicable when RequireAuth is true. Default: false.
	AllowAnonymousRead bool

	// SessionTimeout defines how long sessions remain active without activity
	SessionTimeout time.Duration

	// TokenValidationTimeout defines timeout for JWT validation via MQTT
	TokenValidationTimeout time.Duration
}

// TokenValidationRequest represents a JWT validation request to security-coordinator.
type TokenValidationRequest struct {
	RequestID string `json:"request_id"`
	Token     string `json:"token"`
}

// TokenValidationResponse represents a JWT validation response from security-coordinator.
type TokenValidationResponse struct {
	RequestID string `json:"request_id"`
	Valid     bool   `json:"valid"`
	UserID    string `json:"user_id,omitempty"`
	Username  string `json:"username,omitempty"`
	Email     string `json:"email,omitempty"`
	Error     string `json:"error,omitempty"`
}

// UserContext holds authenticated user information in gin.Context.
type UserContext struct {
	UserID   string
	Username string
	Email    string
}

const (
	// ContextKeyUser is the key for user context in gin.Context
	ContextKeyUser = "ascom_user"

	// ContextKeyDeviceType is the key for device type in gin.Context
	ContextKeyDeviceType = "ascom_device_type"

	// ContextKeyDeviceNumber is the key for device number in gin.Context
	ContextKeyDeviceNumber = "ascom_device_number"
)

// NewSecurityMiddleware creates a new security middleware instance.
func NewSecurityMiddleware(
	mqttClient *mqtt.Client,
	db *pgxpool.Pool,
	config *SecurityConfig,
	logger *zap.Logger,
) (*SecurityMiddleware, error) {
	if mqttClient == nil {
		return nil, fmt.Errorf("MQTT client is required")
	}
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	// Set config defaults
	if config == nil {
		config = &SecurityConfig{}
	}
	if config.SessionTimeout == 0 {
		config.SessionTimeout = 1 * time.Hour
	}
	if config.TokenValidationTimeout == 0 {
		config.TokenValidationTimeout = 5 * time.Second
	}

	middleware := &SecurityMiddleware{
		mqttClient:        mqttClient,
		db:                db,
		logger:            logger.With(zap.String("component", "ascom_security_middleware")),
		config:            config,
		validationTimeout: config.TokenValidationTimeout,
	}

	// Subscribe to token validation responses
	responseTopic := "bigskies/coordinator/security/auth/validate/response"
	if err := mqttClient.Subscribe(responseTopic, 1, middleware.handleValidationResponse); err != nil {
		return nil, fmt.Errorf("failed to subscribe to validation response topic: %w", err)
	}

	middleware.logger.Info("Security middleware initialized",
		zap.Bool("require_auth", config.RequireAuth),
		zap.Bool("allow_anonymous_read", config.AllowAnonymousRead))

	return middleware, nil
}

// AuthenticateRequest is a Gin middleware that validates JWT tokens and extracts user context.
// It should be applied to all ASCOM API routes that require authentication.
//
// Usage:
//
//	router.Use(securityMiddleware.AuthenticateRequest())
func (sm *SecurityMiddleware) AuthenticateRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		// If authentication is disabled, allow request to proceed
		if !sm.config.RequireAuth {
			sm.logger.Warn("Authentication disabled - allowing unauthenticated request",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method))
			c.Next()
			return
		}

		// Allow anonymous read operations if configured
		if sm.config.AllowAnonymousRead && c.Request.Method == "GET" {
			sm.logger.Debug("Allowing anonymous read request",
				zap.String("path", c.Request.URL.Path))
			c.Next()
			return
		}

		// Extract JWT token from request
		token, err := sm.extractToken(c)
		if err != nil {
			sm.logger.Warn("Failed to extract token",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path))
			c.JSON(401, gin.H{
				"ErrorNumber":  401,
				"ErrorMessage": "Authentication required: " + err.Error(),
			})
			c.Abort()
			return
		}

		// Validate token via security-coordinator
		userCtx, err := sm.validateTokenViaMQTT(c.Request.Context(), token)
		if err != nil {
			sm.logger.Warn("Token validation failed",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path))
			c.JSON(401, gin.H{
				"ErrorNumber":  401,
				"ErrorMessage": "Invalid or expired token: " + err.Error(),
			})
			c.Abort()
			return
		}

		// Store user context for subsequent handlers
		c.Set(ContextKeyUser, userCtx)

		sm.logger.Debug("Request authenticated",
			zap.String("user_id", userCtx.UserID),
			zap.String("username", userCtx.Username),
			zap.String("path", c.Request.URL.Path))

		c.Next()
	}
}

// AuthorizeTelescope is a Gin middleware that checks telescope permissions for authenticated users.
// It must be applied AFTER AuthenticateRequest middleware.
// It extracts device type and number from URL path and verifies user has permission.
//
// Usage:
//
//	deviceGroup := router.Group("/api/v1/:device_type/:device_number")
//	deviceGroup.Use(securityMiddleware.AuthorizeTelescope())
func (sm *SecurityMiddleware) AuthorizeTelescope() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authorization if authentication is disabled
		if !sm.config.RequireAuth {
			c.Next()
			return
		}

		// Skip authorization for anonymous read requests
		if sm.config.AllowAnonymousRead && c.Request.Method == "GET" {
			_, userExists := c.Get(ContextKeyUser)
			if !userExists {
				// Anonymous read allowed
				c.Next()
				return
			}
		}

		// Get user context from previous middleware
		userCtxRaw, exists := c.Get(ContextKeyUser)
		if !exists {
			sm.logger.Error("User context not found - AuthenticateRequest middleware not applied?")
			c.JSON(500, gin.H{
				"ErrorNumber":  500,
				"ErrorMessage": "Internal error: user context missing",
			})
			c.Abort()
			return
		}
		userCtx := userCtxRaw.(*UserContext)

		// Extract device type and number from URL parameters
		deviceType := c.Param("device_type")
		deviceNumberStr := c.Param("device_number")
		if deviceType == "" || deviceNumberStr == "" {
			// No device parameters in URL - skip telescope authorization
			c.Next()
			return
		}

		// Parse device number
		var deviceNumber int
		if _, err := fmt.Sscanf(deviceNumberStr, "%d", &deviceNumber); err != nil {
			sm.logger.Warn("Invalid device number",
				zap.String("device_number", deviceNumberStr),
				zap.Error(err))
			c.JSON(400, gin.H{
				"ErrorNumber":  400,
				"ErrorMessage": "Invalid device number",
			})
			c.Abort()
			return
		}

		// Store device info in context for session tracking
		c.Set(ContextKeyDeviceType, deviceType)
		c.Set(ContextKeyDeviceNumber, deviceNumber)

		// Check telescope permission
		allowed, err := sm.checkTelescopePermission(
			c.Request.Context(),
			userCtx.UserID,
			deviceType,
			deviceNumber,
			sm.mapHTTPMethodToAction(c.Request.Method),
		)
		if err != nil {
			sm.logger.Error("Failed to check telescope permission",
				zap.Error(err),
				zap.String("user_id", userCtx.UserID),
				zap.String("device_type", deviceType),
				zap.Int("device_number", deviceNumber))
			c.JSON(500, gin.H{
				"ErrorNumber":  500,
				"ErrorMessage": "Permission check failed: " + err.Error(),
			})
			c.Abort()
			return
		}

		if !allowed {
			sm.logger.Warn("Telescope access denied",
				zap.String("user_id", userCtx.UserID),
				zap.String("username", userCtx.Username),
				zap.String("device_type", deviceType),
				zap.Int("device_number", deviceNumber))
			c.JSON(403, gin.H{
				"ErrorNumber":  403,
				"ErrorMessage": "Access denied: insufficient permissions for this telescope",
			})
			c.Abort()
			return
		}

		sm.logger.Debug("Telescope access authorized",
			zap.String("user_id", userCtx.UserID),
			zap.String("device_type", deviceType),
			zap.Int("device_number", deviceNumber))

		c.Next()
	}
}

// extractToken extracts JWT token from Authorization header or query parameter.
// Supports:
//   - Authorization: Bearer <token>
//   - Authorization: <token>
//   - Query parameter: ?token=<token>
func (sm *SecurityMiddleware) extractToken(c *gin.Context) (string, error) {
	// Try Authorization header first
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		// Remove "Bearer " prefix if present
		token := strings.TrimPrefix(authHeader, "Bearer ")
		token = strings.TrimSpace(token)
		if token != "" {
			return token, nil
		}
	}

	// Try query parameter as fallback (for WebSocket or client compatibility)
	token := c.Query("token")
	if token != "" {
		return token, nil
	}

	return "", fmt.Errorf("no token provided in Authorization header or query parameter")
}

// validateTokenViaMQTT validates a JWT token by sending a request to security-coordinator via MQTT.
func (sm *SecurityMiddleware) validateTokenViaMQTT(ctx context.Context, token string) (*UserContext, error) {
	// Generate unique request ID
	requestID := fmt.Sprintf("ascom-auth-%d", time.Now().UnixNano())

	// Create response channel
	responseChan := make(chan *TokenValidationResponse, 1)
	sm.pendingValidations.Store(requestID, responseChan)
	defer sm.pendingValidations.Delete(requestID)

	// Publish validation request to security-coordinator
	topic := "bigskies/coordinator/security/auth/validate"

	// Wrap request in MQTT message envelope with request_id
	msgPayload := map[string]interface{}{
		"request_id": requestID,
		"token":      token,
	}

	msg, err := mqtt.NewMessage(mqtt.MessageTypeRequest, "ascom-coordinator", msgPayload)
	if err != nil {
		close(responseChan)
		return nil, fmt.Errorf("failed to create MQTT message: %w", err)
	}

	if err := sm.mqttClient.PublishJSON(topic, 1, false, msg); err != nil {
		close(responseChan)
		return nil, fmt.Errorf("failed to publish validation request: %w", err)
	}

	sm.logger.Debug("Published token validation request",
		zap.String("request_id", requestID),
		zap.String("topic", topic))

	// Wait for response with timeout
	select {
	case response, ok := <-responseChan:
		if !ok {
			return nil, fmt.Errorf("validation response channel closed")
		}
		if !response.Valid {
			return nil, fmt.Errorf("token validation failed: %s", response.Error)
		}

		return &UserContext{
			UserID:   response.UserID,
			Username: response.Username,
			Email:    response.Email,
		}, nil

	case <-time.After(sm.validationTimeout):
		close(responseChan)
		return nil, fmt.Errorf("token validation timeout after %v", sm.validationTimeout)

	case <-ctx.Done():
		close(responseChan)
		return nil, fmt.Errorf("context canceled: %w", ctx.Err())
	}
}

// handleValidationResponse processes token validation responses from security-coordinator.
func (sm *SecurityMiddleware) handleValidationResponse(topic string, payload []byte) error {
	sm.logger.Debug("Received validation response",
		zap.String("topic", topic),
		zap.Int("payload_size", len(payload)))

	// Parse MQTT message envelope
	var msgEnvelope struct {
		MessageType string                  `json:"message_type"`
		Payload     TokenValidationResponse `json:"payload"`
	}

	if err := json.Unmarshal(payload, &msgEnvelope); err != nil {
		sm.logger.Error("Failed to unmarshal validation response", zap.Error(err))
		return err
	}

	response := &msgEnvelope.Payload
	requestID := response.RequestID

	// Find pending validation
	if val, ok := sm.pendingValidations.Load(requestID); ok {
		if ch, ok := val.(chan *TokenValidationResponse); ok {
			select {
			case ch <- response:
				sm.logger.Debug("Delivered validation response",
					zap.String("request_id", requestID),
					zap.Bool("valid", response.Valid))
			default:
				sm.logger.Warn("Validation response channel full or closed",
					zap.String("request_id", requestID))
			}
		}
	} else {
		sm.logger.Warn("Received validation response for unknown request",
			zap.String("request_id", requestID))
	}

	return nil
}

// checkTelescopePermission queries the telescope_permissions table to verify user access.
// It resolves the ASCOM device to a telescope configuration and checks permissions.
func (sm *SecurityMiddleware) checkTelescopePermission(
	ctx context.Context,
	userID string,
	deviceType string,
	deviceNumber int,
	action string,
) (bool, error) {
	// Resolve ASCOM device to telescope configuration
	telescopeID, err := sm.resolveDeviceToTelescope(ctx, deviceType, deviceNumber)
	if err != nil {
		return false, fmt.Errorf("failed to resolve device to telescope: %w", err)
	}
	if telescopeID == "" {
		// No telescope association - allow access (device not linked to telescope config)
		sm.logger.Debug("Device not linked to telescope configuration - allowing access",
			zap.String("device_type", deviceType),
			zap.Int("device_number", deviceNumber))
		return true, nil
	}

	// Query telescope_permissions table
	query := `
		SELECT permission_level
		FROM telescope_permissions
		WHERE telescope_id = $1
		  AND principal_type = 'user'
		  AND principal_id = $2::UUID
		LIMIT 1
	`

	var permissionLevel string
	err = sm.db.QueryRow(ctx, query, telescopeID, userID).Scan(&permissionLevel)

	if err != nil {
		// No explicit permission found - check group permissions
		groupQuery := `
			SELECT tp.permission_level
			FROM telescope_permissions tp
			JOIN user_groups ug ON ug.group_id = tp.principal_id::UUID
			WHERE tp.telescope_id = $1
			  AND tp.principal_type = 'group'
			  AND ug.user_id = $2::UUID
			ORDER BY
				CASE tp.permission_level
					WHEN 'admin' THEN 1
					WHEN 'control' THEN 2
					WHEN 'write' THEN 3
					WHEN 'read' THEN 4
					ELSE 5
				END
			LIMIT 1
		`

		err = sm.db.QueryRow(ctx, groupQuery, telescopeID, userID).Scan(&permissionLevel)
		if err != nil {
			// No permission found - deny access
			sm.logger.Debug("No telescope permissions found for user",
				zap.String("user_id", userID),
				zap.String("telescope_id", telescopeID))
			return false, nil
		}
	}

	// Check if permission level allows the requested action
	allowed := sm.checkPermissionLevel(permissionLevel, action)

	sm.logger.Debug("Telescope permission check complete",
		zap.String("user_id", userID),
		zap.String("telescope_id", telescopeID),
		zap.String("permission_level", permissionLevel),
		zap.String("action", action),
		zap.Bool("allowed", allowed))

	return allowed, nil
}

// resolveDeviceToTelescope maps an ASCOM device type/number to a telescope configuration ID.
func (sm *SecurityMiddleware) resolveDeviceToTelescope(
	ctx context.Context,
	deviceType string,
	deviceNumber int,
) (string, error) {
	query := `
		SELECT telescope_config_id
		FROM ascom_devices
		WHERE device_type = $1
		  AND device_number = $2
		  AND enabled = true
		LIMIT 1
	`

	var telescopeID *string
	err := sm.db.QueryRow(ctx, query, deviceType, deviceNumber).Scan(&telescopeID)

	if err != nil {
		return "", fmt.Errorf("failed to query ascom_devices: %w", err)
	}

	if telescopeID == nil {
		return "", nil
	}

	return *telescopeID, nil
}

// mapHTTPMethodToAction maps HTTP methods to permission actions.
func (sm *SecurityMiddleware) mapHTTPMethodToAction(httpMethod string) string {
	switch httpMethod {
	case "GET":
		return "read"
	case "PUT", "POST":
		return "write"
	case "DELETE":
		return "delete"
	default:
		return "read"
	}
}

// checkPermissionLevel evaluates if a permission level allows an action.
// Permission hierarchy: admin > control > write > read
func (sm *SecurityMiddleware) checkPermissionLevel(level string, action string) bool {
	switch level {
	case "admin":
		return true // Admin can do everything
	case "control":
		return action == "read" || action == "write" // Control can read and write
	case "write":
		return action == "read" || action == "write" // Write can read and write
	case "read":
		return action == "read" // Read can only read
	default:
		return false
	}
}

// Stop cleans up middleware resources.
func (sm *SecurityMiddleware) Stop() {
	sm.logger.Info("Stopping security middleware")

	// Cancel all pending validations
	sm.pendingValidations.Range(func(key, value interface{}) bool {
		if ch, ok := value.(chan *TokenValidationResponse); ok {
			close(ch)
		}
		sm.pendingValidations.Delete(key)
		return true
	})

	sm.logger.Info("Security middleware stopped")
}
