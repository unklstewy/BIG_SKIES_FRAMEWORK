package handlers

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/ascomserver/proxy"
)

// TelescopeHandler handles ASCOM Telescope interface endpoints.
// This implements the ASCOM Telescope v3 interface specification.
//
// The Telescope interface provides control over motorized telescope mounts,
// including slewing, tracking, parking, and alignment operations.
//
// Reference: https://ascom-standards.org/Help/Developer/html/T_ASCOM_DeviceInterface_ITelescopeV3.htm
type TelescopeHandler struct {
	*BaseHandler
}

// NewTelescopeHandler creates a new telescope handler instance.
//
// Parameters:
//   - deviceNumber: Device instance number (usually 0)
//   - proxy: Backend proxy for telescope communication
//   - logger: Structured logger
//
// Returns a configured TelescopeHandler ready to register routes.
func NewTelescopeHandler(deviceNumber int, proxy proxy.DeviceProxy, logger *zap.Logger) *TelescopeHandler {
	return &TelescopeHandler{
		BaseHandler: NewBaseHandler("telescope", deviceNumber, 3, proxy, logger),
	}
}

// RegisterRoutes registers all telescope-specific HTTP routes.
// This implements the DeviceHandler interface.
func (t *TelescopeHandler) RegisterRoutes(router *gin.RouterGroup) {
	t.logger.Info("Registering telescope routes")

	// Register common routes (connected, description, etc.)
	t.registerCommonRoutes(router)

	// Telescope-specific properties (GET only)
	router.GET("/alignmentmode", func(c *gin.Context) {
		t.handleGet(c, "alignmentmode")
	})

	router.GET("/altitude", func(c *gin.Context) {
		t.handleGet(c, "altitude")
	})

	router.GET("/aperturearea", func(c *gin.Context) {
		t.handleGet(c, "aperturearea")
	})

	router.GET("/aperturediameter", func(c *gin.Context) {
		t.handleGet(c, "aperturediameter")
	})

	router.GET("/athome", func(c *gin.Context) {
		t.handleGet(c, "athome")
	})

	router.GET("/atpark", func(c *gin.Context) {
		t.handleGet(c, "atpark")
	})

	router.GET("/azimuth", func(c *gin.Context) {
		t.handleGet(c, "azimuth")
	})

	router.GET("/canfindhome", func(c *gin.Context) {
		t.handleGet(c, "canfindhome")
	})

	router.GET("/canpark", func(c *gin.Context) {
		t.handleGet(c, "canpark")
	})

	router.GET("/canpulseguide", func(c *gin.Context) {
		t.handleGet(c, "canpulseguide")
	})

	router.GET("/cansetdeclinationrate", func(c *gin.Context) {
		t.handleGet(c, "cansetdeclinationrate")
	})

	router.GET("/cansetguiderates", func(c *gin.Context) {
		t.handleGet(c, "cansetguiderates")
	})

	router.GET("/cansetpark", func(c *gin.Context) {
		t.handleGet(c, "cansetpark")
	})

	router.GET("/cansetpierside", func(c *gin.Context) {
		t.handleGet(c, "cansetpierside")
	})

	router.GET("/cansetrightascensionrate", func(c *gin.Context) {
		t.handleGet(c, "cansetrightascensionrate")
	})

	router.GET("/cansettracking", func(c *gin.Context) {
		t.handleGet(c, "cansettracking")
	})

	router.GET("/canslew", func(c *gin.Context) {
		t.handleGet(c, "canslew")
	})

	router.GET("/canslewaltaz", func(c *gin.Context) {
		t.handleGet(c, "canslewaltaz")
	})

	router.GET("/canslewaltazasync", func(c *gin.Context) {
		t.handleGet(c, "canslewaltazasync")
	})

	router.GET("/canslewasync", func(c *gin.Context) {
		t.handleGet(c, "canslewasync")
	})

	router.GET("/cansync", func(c *gin.Context) {
		t.handleGet(c, "cansync")
	})

	router.GET("/cansyncaltaz", func(c *gin.Context) {
		t.handleGet(c, "cansyncaltaz")
	})

	router.GET("/canunpark", func(c *gin.Context) {
		t.handleGet(c, "canunpark")
	})

	router.GET("/declination", func(c *gin.Context) {
		t.handleGet(c, "declination")
	})

	router.GET("/declinationrate", func(c *gin.Context) {
		t.handleGet(c, "declinationrate")
	})
	router.PUT("/declinationrate", func(c *gin.Context) {
		t.handlePut(c, "declinationrate")
	})

	router.GET("/doesrefraction", func(c *gin.Context) {
		t.handleGet(c, "doesrefraction")
	})
	router.PUT("/doesrefraction", func(c *gin.Context) {
		t.handlePut(c, "doesrefraction")
	})

	router.GET("/equatorialsystem", func(c *gin.Context) {
		t.handleGet(c, "equatorialsystem")
	})

	router.GET("/focallength", func(c *gin.Context) {
		t.handleGet(c, "focallength")
	})

	router.GET("/guideratedeclination", func(c *gin.Context) {
		t.handleGet(c, "guideratedeclination")
	})
	router.PUT("/guideratedeclination", func(c *gin.Context) {
		t.handlePut(c, "guideratedeclination")
	})

	router.GET("/guideraterightascension", func(c *gin.Context) {
		t.handleGet(c, "guideraterightascension")
	})
	router.PUT("/guideraterightascension", func(c *gin.Context) {
		t.handlePut(c, "guideraterightascension")
	})

	router.GET("/ispulseguiding", func(c *gin.Context) {
		t.handleGet(c, "ispulseguiding")
	})

	router.GET("/rightascension", func(c *gin.Context) {
		t.handleGet(c, "rightascension")
	})

	router.GET("/rightascensionrate", func(c *gin.Context) {
		t.handleGet(c, "rightascensionrate")
	})
	router.PUT("/rightascensionrate", func(c *gin.Context) {
		t.handlePut(c, "rightascensionrate")
	})

	router.GET("/sideofpier", func(c *gin.Context) {
		t.handleGet(c, "sideofpier")
	})
	router.PUT("/sideofpier", func(c *gin.Context) {
		t.handlePut(c, "sideofpier")
	})

	router.GET("/siderealtime", func(c *gin.Context) {
		t.handleGet(c, "siderealtime")
	})

	router.GET("/siteelevation", func(c *gin.Context) {
		t.handleGet(c, "siteelevation")
	})
	router.PUT("/siteelevation", func(c *gin.Context) {
		t.handlePut(c, "siteelevation")
	})

	router.GET("/sitelatitude", func(c *gin.Context) {
		t.handleGet(c, "sitelatitude")
	})
	router.PUT("/sitelatitude", func(c *gin.Context) {
		t.handlePut(c, "sitelatitude")
	})

	router.GET("/sitelongitude", func(c *gin.Context) {
		t.handleGet(c, "sitelongitude")
	})
	router.PUT("/sitelongitude", func(c *gin.Context) {
		t.handlePut(c, "sitelongitude")
	})

	router.GET("/slewing", func(c *gin.Context) {
		t.handleGet(c, "slewing")
	})

	router.GET("/slewsettletime", func(c *gin.Context) {
		t.handleGet(c, "slewsettletime")
	})
	router.PUT("/slewsettletime", func(c *gin.Context) {
		t.handlePut(c, "slewsettletime")
	})

	router.GET("/targetdeclination", func(c *gin.Context) {
		t.handleGet(c, "targetdeclination")
	})
	router.PUT("/targetdeclination", func(c *gin.Context) {
		t.handlePut(c, "targetdeclination")
	})

	router.GET("/targetrightascension", func(c *gin.Context) {
		t.handleGet(c, "targetrightascension")
	})
	router.PUT("/targetrightascension", func(c *gin.Context) {
		t.handlePut(c, "targetrightascension")
	})

	router.GET("/tracking", func(c *gin.Context) {
		t.handleGet(c, "tracking")
	})
	router.PUT("/tracking", func(c *gin.Context) {
		t.handlePut(c, "tracking")
	})

	router.GET("/trackingrate", func(c *gin.Context) {
		t.handleGet(c, "trackingrate")
	})
	router.PUT("/trackingrate", func(c *gin.Context) {
		t.handlePut(c, "trackingrate")
	})

	router.GET("/trackingrates", func(c *gin.Context) {
		t.handleGet(c, "trackingrates")
	})

	router.GET("/utcdate", func(c *gin.Context) {
		t.handleGet(c, "utcdate")
	})
	router.PUT("/utcdate", func(c *gin.Context) {
		t.handlePut(c, "utcdate")
	})

	// Telescope methods (PUT only)
	router.PUT("/abortslew", func(c *gin.Context) {
		t.handlePut(c, "abortslew")
	})

	router.PUT("/axisrates", func(c *gin.Context) {
		t.handlePut(c, "axisrates")
	})

	router.PUT("/canmoveaxis", func(c *gin.Context) {
		t.handlePut(c, "canmoveaxis")
	})

	router.PUT("/destinationsideofpier", func(c *gin.Context) {
		t.handlePut(c, "destinationsideofpier")
	})

	router.PUT("/findhome", func(c *gin.Context) {
		t.handlePut(c, "findhome")
	})

	router.PUT("/moveaxis", func(c *gin.Context) {
		t.handlePut(c, "moveaxis")
	})

	router.PUT("/park", func(c *gin.Context) {
		t.handlePut(c, "park")
	})

	router.PUT("/pulseguide", func(c *gin.Context) {
		t.handlePut(c, "pulseguide")
	})

	router.PUT("/setpark", func(c *gin.Context) {
		t.handlePut(c, "setpark")
	})

	router.PUT("/slewtoaltaz", func(c *gin.Context) {
		t.handlePut(c, "slewtoaltaz")
	})

	router.PUT("/slewtoaltazasync", func(c *gin.Context) {
		t.handlePut(c, "slewtoaltazasync")
	})

	router.PUT("/slewtocoordinates", func(c *gin.Context) {
		t.handlePut(c, "slewtocoordinates")
	})

	router.PUT("/slewtocoordinatesasync", func(c *gin.Context) {
		t.handlePut(c, "slewtocoordinatesasync")
	})

	router.PUT("/slewtotarget", func(c *gin.Context) {
		t.handlePut(c, "slewtotarget")
	})

	router.PUT("/slewtotargetasync", func(c *gin.Context) {
		t.handlePut(c, "slewtotargetasync")
	})

	router.PUT("/synctoaltaz", func(c *gin.Context) {
		t.handlePut(c, "synctoaltaz")
	})

	router.PUT("/synctocoordinates", func(c *gin.Context) {
		t.handlePut(c, "synctocoordinates")
	})

	router.PUT("/synctotarget", func(c *gin.Context) {
		t.handlePut(c, "synctotarget")
	})

	router.PUT("/unpark", func(c *gin.Context) {
		t.handlePut(c, "unpark")
	})

	t.logger.Info("Telescope routes registered successfully")
}
