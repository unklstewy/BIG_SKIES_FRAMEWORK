package handlers

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/ascomserver/proxy"
)

// DeviceHandler defines the interface for ASCOM device-specific handlers.
// Each device type (telescope, camera, dome, etc.) implements this interface
// to provide type-specific API endpoint handling.
//
// The handler layer sits between the HTTP server and the proxy layer,
// translating REST API calls into proxy operations and handling ASCOM
// protocol requirements (transaction IDs, error codes, response formatting).
type DeviceHandler interface {
	// RegisterRoutes registers all HTTP routes for this device type.
	// This should register handlers for all ASCOM API endpoints supported
	// by the device type.
	RegisterRoutes(router *gin.RouterGroup)

	// GetDeviceType returns the ASCOM device type name.
	// Examples: "telescope", "camera", "dome", "focuser"
	GetDeviceType() string

	// GetDeviceNumber returns the device instance number.
	GetDeviceNumber() int

	// GetInterfaceVersion returns the ASCOM interface version supported.
	GetInterfaceVersion() int

	// GetSupportedActions returns a list of custom actions supported.
	// This corresponds to the ASCOM SupportedActions property.
	GetSupportedActions() []string

	// Shutdown gracefully shuts down the handler and releases resources.
	Shutdown(ctx context.Context) error
}

// BaseHandler provides common functionality for all device handlers.
// Device-specific handlers should embed this to inherit common behavior
// like transaction ID management, error handling, and response formatting.
type BaseHandler struct {
	// deviceType is the ASCOM device type
	deviceType string

	// deviceNumber is the device instance number
	deviceNumber int

	// interfaceVersion is the ASCOM interface version
	interfaceVersion int

	// proxy is the backend proxy for device communication
	proxy proxy.DeviceProxy

	// logger provides structured logging
	logger *zap.Logger

	// serverTransactionID is an atomically incrementing counter
	// for generating unique server transaction IDs
	serverTransactionID atomic.Uint32

	// name is the device name
	name string

	// description is the device description
	description string

	// driverInfo is information about the driver
	driverInfo string

	// driverVersion is the driver version
	driverVersion string
}

// NewBaseHandler creates a new base handler instance.
//
// Parameters:
//   - deviceType: ASCOM device type (e.g., "telescope")
//   - deviceNumber: Device instance number (usually 0)
//   - interfaceVersion: ASCOM interface version
//   - proxy: Backend proxy for device communication
//   - logger: Structured logger
//
// Returns a configured BaseHandler.
func NewBaseHandler(
	deviceType string,
	deviceNumber int,
	interfaceVersion int,
	proxy proxy.DeviceProxy,
	logger *zap.Logger,
) *BaseHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &BaseHandler{
		deviceType:       deviceType,
		deviceNumber:     deviceNumber,
		interfaceVersion: interfaceVersion,
		proxy:            proxy,
		logger: logger.With(
			zap.String("handler", deviceType),
			zap.Int("device_number", deviceNumber),
		),
		name:          fmt.Sprintf("%s Simulator", deviceType),
		description:   fmt.Sprintf("BigSkies ASCOM %s", deviceType),
		driverInfo:    "BigSkies ASCOM Reflector/Proxy",
		driverVersion: "1.0.0",
	}
}

// GetDeviceType returns the device type.
func (b *BaseHandler) GetDeviceType() string {
	return b.deviceType
}

// GetDeviceNumber returns the device number.
func (b *BaseHandler) GetDeviceNumber() int {
	return b.deviceNumber
}

// GetInterfaceVersion returns the interface version.
func (b *BaseHandler) GetInterfaceVersion() int {
	return b.interfaceVersion
}

// GetSupportedActions returns supported custom actions.
// Base implementation returns an empty list.
func (b *BaseHandler) GetSupportedActions() []string {
	return []string{}
}

// Shutdown performs cleanup for the handler.
func (b *BaseHandler) Shutdown(ctx context.Context) error {
	b.logger.Info("Shutting down handler")
	// Proxy shutdown is handled at the server level
	return nil
}

// generateTransactionID generates a unique server transaction ID.
func (b *BaseHandler) generateTransactionID() uint32 {
	return b.serverTransactionID.Add(1)
}

// extractClientTransactionID extracts the client transaction ID from the request.
// If not present or invalid, returns 0.
func (b *BaseHandler) extractClientTransactionID(c *gin.Context) uint32 {
	clientTxnStr := c.Query("ClientTransactionID")
	if clientTxnStr == "" {
		clientTxnStr = c.PostForm("ClientTransactionID")
	}

	if clientTxnStr == "" {
		return 0
	}

	clientTxn, err := strconv.ParseUint(clientTxnStr, 10, 32)
	if err != nil {
		return 0
	}

	return uint32(clientTxn)
}

// extractClientID extracts the client ID from the request.
// If not present or invalid, returns 0.
func (b *BaseHandler) extractClientID(c *gin.Context) uint32 {
	clientIDStr := c.Query("ClientID")
	if clientIDStr == "" {
		clientIDStr = c.PostForm("ClientID")
	}

	if clientIDStr == "" {
		return 0
	}

	clientID, err := strconv.ParseUint(clientIDStr, 10, 32)
	if err != nil {
		return 0
	}

	return uint32(clientID)
}

// buildParams builds a parameter map for proxy calls from the request.
// This extracts both query parameters and form values.
func (b *BaseHandler) buildParams(c *gin.Context) map[string]string {
	params := make(map[string]string)

	// Add query parameters
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	// Add form parameters (for PUT requests)
	if c.Request.Method == "PUT" {
		if err := c.Request.ParseForm(); err == nil {
			for key, values := range c.Request.PostForm {
				if len(values) > 0 {
					params[key] = values[0]
				}
			}
		}
	}

	return params
}

// handleGet processes a GET request through the proxy.
// This is used for reading device properties.
func (b *BaseHandler) handleGet(c *gin.Context, method string) {
	clientTxnID := b.extractClientTransactionID(c)
	serverTxnID := b.generateTransactionID()

	b.logger.Debug("Handling GET request",
		zap.String("method", method),
		zap.Uint32("client_transaction_id", clientTxnID),
		zap.Uint32("server_transaction_id", serverTxnID))

	// Build parameters
	params := b.buildParams(c)

	// Execute proxy request
	value, err := b.proxy.Get(c.Request.Context(), method, params)
	if err != nil {
		b.logger.Error("Proxy GET failed",
			zap.String("method", method),
			zap.Error(err))
		b.sendErrorResponse(c, clientTxnID, serverTxnID, err)
		return
	}

	// Send success response
	b.sendSuccessResponse(c, clientTxnID, serverTxnID, value)
}

// handlePut processes a PUT request through the proxy.
// This is used for setting device properties or executing commands.
func (b *BaseHandler) handlePut(c *gin.Context, method string) {
	clientTxnID := b.extractClientTransactionID(c)
	serverTxnID := b.generateTransactionID()

	b.logger.Debug("Handling PUT request",
		zap.String("method", method),
		zap.Uint32("client_transaction_id", clientTxnID),
		zap.Uint32("server_transaction_id", serverTxnID))

	// Build parameters
	params := b.buildParams(c)

	// Execute proxy request
	value, err := b.proxy.Put(c.Request.Context(), method, params)
	if err != nil {
		b.logger.Error("Proxy PUT failed",
			zap.String("method", method),
			zap.Error(err))
		b.sendErrorResponse(c, clientTxnID, serverTxnID, err)
		return
	}

	// Send success response
	b.sendSuccessResponse(c, clientTxnID, serverTxnID, value)
}

// sendSuccessResponse sends a successful ASCOM response.
func (b *BaseHandler) sendSuccessResponse(c *gin.Context, clientTxnID, serverTxnID uint32, value interface{}) {
	response := gin.H{
		"Value":               value,
		"ClientTransactionID": clientTxnID,
		"ServerTransactionID": serverTxnID,
		"ErrorNumber":         0,
		"ErrorMessage":        "",
	}

	c.JSON(200, response)
}

// sendErrorResponse sends an ASCOM error response.
func (b *BaseHandler) sendErrorResponse(c *gin.Context, clientTxnID, serverTxnID uint32, err error) {
	// Map error to ASCOM error code
	errorNumber, errorMessage := b.mapErrorToASCOM(err)

	response := gin.H{
		"Value":               "",
		"ClientTransactionID": clientTxnID,
		"ServerTransactionID": serverTxnID,
		"ErrorNumber":         errorNumber,
		"ErrorMessage":        errorMessage,
	}

	c.JSON(200, response) // ASCOM always returns 200 OK, errors are in the response body
}

// mapErrorToASCOM maps Go errors to ASCOM error codes.
// See: https://ascom-standards.org/Help/Developer/html/T_ASCOM_ErrorCodes.htm
func (b *BaseHandler) mapErrorToASCOM(err error) (int, string) {
	if err == nil {
		return 0, ""
	}

	errStr := err.Error()

	// Check for common proxy errors
	if err == proxy.ErrNotConnected {
		return 0x0407, "Not connected to device"
	}
	if err == proxy.ErrTimeout {
		return 0x0408, "Operation timed out"
	}
	if err == proxy.ErrBackendUnavailable {
		return 0x0500, "Backend unavailable"
	}

	// Check for ASCOM error codes in the error message
	// Format: "ASCOM error XXXX: message"
	var ascomCode int
	if n, _ := fmt.Sscanf(errStr, "ASCOM error %d:", &ascomCode); n == 1 {
		return ascomCode, errStr
	}

	// Default to unspecified error
	return 0x0500, errStr
}

// registerCommonRoutes registers routes common to all ASCOM devices.
// These are the standard ASCOM API endpoints that all devices must implement.
func (b *BaseHandler) registerCommonRoutes(router *gin.RouterGroup) {
	// Common properties (all devices must support these)
	router.GET("/connected", func(c *gin.Context) {
		b.handleGet(c, "connected")
	})
	router.PUT("/connected", func(c *gin.Context) {
		b.handlePut(c, "connected")
	})

	router.GET("/description", func(c *gin.Context) {
		b.sendSuccessResponse(c,
			b.extractClientTransactionID(c),
			b.generateTransactionID(),
			b.description)
	})

	router.GET("/driverinfo", func(c *gin.Context) {
		b.sendSuccessResponse(c,
			b.extractClientTransactionID(c),
			b.generateTransactionID(),
			b.driverInfo)
	})

	router.GET("/driverversion", func(c *gin.Context) {
		b.sendSuccessResponse(c,
			b.extractClientTransactionID(c),
			b.generateTransactionID(),
			b.driverVersion)
	})

	router.GET("/interfaceversion", func(c *gin.Context) {
		b.sendSuccessResponse(c,
			b.extractClientTransactionID(c),
			b.generateTransactionID(),
			b.interfaceVersion)
	})

	router.GET("/name", func(c *gin.Context) {
		b.sendSuccessResponse(c,
			b.extractClientTransactionID(c),
			b.generateTransactionID(),
			b.name)
	})

	router.GET("/supportedactions", func(c *gin.Context) {
		b.sendSuccessResponse(c,
			b.extractClientTransactionID(c),
			b.generateTransactionID(),
			b.GetSupportedActions())
	})

	router.PUT("/action", func(c *gin.Context) {
		b.handlePut(c, "action")
	})

	router.PUT("/commandblind", func(c *gin.Context) {
		b.handlePut(c, "commandblind")
	})

	router.PUT("/commandbool", func(c *gin.Context) {
		b.handlePut(c, "commandbool")
	})

	router.PUT("/commandstring", func(c *gin.Context) {
		b.handlePut(c, "commandstring")
	})
}
