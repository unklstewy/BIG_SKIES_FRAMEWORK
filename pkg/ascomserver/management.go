package ascomserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ManagementAPI contains handlers for ASCOM Alpaca management endpoints.
// These endpoints provide server-level information and are not device-specific.
//
// Management API endpoints defined by ASCOM Alpaca:
//   - GET /management/v1/description - Server description
//   - GET /management/v1/configureddevices - List of configured devices
//   - GET /management/apiversions - Supported API versions
type ManagementAPI struct {
	server *Server
}

// NewManagementAPI creates a new management API handler.
func NewManagementAPI(server *Server) *ManagementAPI {
	return &ManagementAPI{
		server: server,
	}
}

// RegisterRoutes registers all management API routes with the Gin router.
// This sets up the HTTP endpoints for management operations.
//
// Parameters:
//   - router: Gin router group where routes should be registered
func (m *ManagementAPI) RegisterRoutes(router *gin.RouterGroup) {
	// Create a sub-group for management endpoints
	management := router.Group("/management")
	{
		// API version list endpoint
		// This endpoint is at /management/apiversions (not versioned)
		// Returns: array of supported API version numbers
		management.GET("/apiversions", m.handleAPIVersions)

		// Version 1 management endpoints
		v1 := management.Group("/v1")
		{
			// Server description endpoint
			// Returns: detailed information about the server
			v1.GET("/description", m.handleDescription)

			// Configured devices endpoint
			// Returns: list of all devices this server provides
			v1.GET("/configureddevices", m.handleConfiguredDevices)
		}
	}
}

// handleAPIVersions handles GET /management/apiversions
//
// This endpoint returns a list of Alpaca API versions supported by this server.
// Currently, only version 1 is defined by the ASCOM Alpaca specification.
//
// Response format:
//
//	{
//	  "Value": [1],
//	  "ClientTransactionID": <client_id>,
//	  "ServerTransactionID": <server_id>,
//	  "ErrorNumber": 0,
//	  "ErrorMessage": ""
//	}
func (m *ManagementAPI) handleAPIVersions(c *gin.Context) {
	m.server.logger.Debug("Handling API versions request")

	// Currently, only API version 1 is supported.
	// Future versions would be added to this array.
	versions := []int{AlpacaAPIVersion}

	// Return the list of supported versions
	// Note: This endpoint doesn't require transaction IDs, but we include them
	// for consistency with other ASCOM responses.
	c.JSON(http.StatusOK, NewSuccessResponse(
		versions,
		getClientTransactionID(c),
		getServerTransactionID(c)))
}

// handleDescription handles GET /management/v1/description
//
// This endpoint returns detailed information about the server.
// Clients use this to display server information in their UI and to
// understand the capabilities and identity of the server.
//
// Response format:
//
//	{
//	  "Value": {
//	    "ServerName": "BigSkies ASCOM Reflector",
//	    "Manufacturer": "BigSkies Framework",
//	    "ManufacturerVersion": "1.0.0",
//	    "Location": "Observatory"
//	  },
//	  "ClientTransactionID": <client_id>,
//	  "ServerTransactionID": <server_id>,
//	  "ErrorNumber": 0,
//	  "ErrorMessage": ""
//	}
func (m *ManagementAPI) handleDescription(c *gin.Context) {
	m.server.logger.Debug("Handling server description request")

	// Build the server description from configuration
	description := map[string]interface{}{
		// ServerName is the name of this server instance.
		// This appears in client software when browsing available servers.
		"ServerName": m.server.config.Server.ServerName,

		// Manufacturer is the name of the organization/person who created this server.
		"Manufacturer": m.server.config.Server.Manufacturer,

		// ManufacturerVersion is the version of the server software.
		// This helps with debugging and compatibility checking.
		"ManufacturerVersion": m.server.config.Server.ManufacturerVersion,

		// Location is a human-readable description of where the server is located.
		// Example: "Backyard Observatory", "Remote Site Alpha", etc.
		"Location": m.server.config.Server.Location,
	}

	// Return the server description
	c.JSON(http.StatusOK, NewSuccessResponse(
		description,
		getClientTransactionID(c),
		getServerTransactionID(c)))
}

// handleConfiguredDevices handles GET /management/v1/configureddevices
//
// This endpoint returns a list of all ASCOM devices provided by this server.
// Clients use this information to discover what devices are available and
// to display them in their device selection UI.
//
// Each device in the list includes:
//   - DeviceName: Human-readable device name
//   - DeviceType: ASCOM device type (telescope, camera, dome, etc.)
//   - DeviceNumber: Device number (0-based index for this device type)
//   - UniqueID: Globally unique identifier for this device instance
//
// Response format:
//
//	{
//	  "Value": [
//	    {
//	      "DeviceName": "Primary Telescope",
//	      "DeviceType": "telescope",
//	      "DeviceNumber": 0,
//	      "UniqueID": "12345678-1234-1234-1234-123456789012"
//	    },
//	    ...
//	  ],
//	  "ClientTransactionID": <client_id>,
//	  "ServerTransactionID": <server_id>,
//	  "ErrorNumber": 0,
//	  "ErrorMessage": ""
//	}
func (m *ManagementAPI) handleConfiguredDevices(c *gin.Context) {
	m.server.logger.Debug("Handling configured devices request")

	// Build the list of configured devices from the server's device registry.
	// Each virtual device is exposed as a separate device to clients.
	devices := make([]map[string]interface{}, 0, len(m.server.devices))

	for _, device := range m.server.devices {
		// Create a device entry following the ASCOM Alpaca specification
		deviceInfo := map[string]interface{}{
			// DeviceName is the human-readable name shown in client UIs.
			// Example: "Primary Telescope", "Main Camera", "Dome Controller"
			"DeviceName": device.Name,

			// DeviceType is the ASCOM device type string.
			// Must be one of the standard ASCOM types:
			// "telescope", "camera", "dome", "focuser", "filterwheel",
			// "rotator", "switch", "safetymonitor", "observingconditions", "covercalibrator"
			"DeviceType": device.DeviceType,

			// DeviceNumber is the device number for this type.
			// Devices of the same type are numbered sequentially starting from 0.
			// Example: First telescope is 0, second telescope is 1, etc.
			"DeviceNumber": device.DeviceNumber,

			// UniqueID is a globally unique identifier for this device instance.
			// This should remain stable across server restarts if possible.
			// Format is typically a GUID/UUID, but can be any unique string.
			"UniqueID": device.UniqueID,
		}

		devices = append(devices, deviceInfo)
	}

	m.server.logger.Debug("Returning configured devices",
		m.server.logger.Sugar().String("count", string(rune(len(devices)+'0'))))

	// Return the list of configured devices
	c.JSON(http.StatusOK, NewSuccessResponse(
		devices,
		getClientTransactionID(c),
		getServerTransactionID(c)))
}
