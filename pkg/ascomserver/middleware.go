package ascomserver

import (
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggingMiddleware creates a middleware that logs all HTTP requests and responses.
// This provides detailed information about each API call for debugging and monitoring.
//
// Logged information includes:
//   - HTTP method (GET, POST, PUT, etc.)
//   - Request path and query parameters
//   - Client IP address
//   - Response status code
//   - Response time (latency)
//   - Error messages (if any)
func LoggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record the start time to calculate request duration
		start := time.Now()

		// Get request details before processing
		method := c.Request.Method
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		clientIP := c.ClientIP()

		// Log the incoming request
		logger.Info("Incoming request",
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("client_ip", clientIP))

		// Process the request by calling the next handler in the chain
		c.Next()

		// Calculate request duration
		duration := time.Since(start)

		// Get response details after processing
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		// Log the response
		if statusCode >= 400 {
			// Log errors at warn/error level depending on severity
			if statusCode >= 500 {
				logger.Error("Request failed",
					zap.String("method", method),
					zap.String("path", path),
					zap.Int("status", statusCode),
					zap.Duration("duration", duration),
					zap.String("error", errorMessage))
			} else {
				logger.Warn("Request returned client error",
					zap.String("method", method),
					zap.String("path", path),
					zap.Int("status", statusCode),
					zap.Duration("duration", duration),
					zap.String("error", errorMessage))
			}
		} else {
			// Log successful requests at debug level to reduce noise
			logger.Debug("Request completed",
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", statusCode),
				zap.Duration("duration", duration))
		}
	}
}

// CORSMiddleware creates a middleware that adds CORS (Cross-Origin Resource Sharing) headers.
// This is essential for web-based ASCOM clients that make requests from browsers.
//
// CORS allows web pages from one domain to make requests to your ASCOM server,
// which may be on a different domain. Without CORS, browsers block these requests
// for security reasons.
//
// Parameters:
//   - config: CORS configuration specifying allowed origins, methods, headers, etc.
func CORSMiddleware(config CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set CORS headers based on configuration
		origin := c.Request.Header.Get("Origin")

		// Check if the origin is allowed
		allowedOrigin := ""
		for _, allowed := range config.AllowedOrigins {
			if allowed == "*" || allowed == origin {
				allowedOrigin = allowed
				break
			}
		}

		if allowedOrigin != "" {
			// Set the Access-Control-Allow-Origin header
			// This tells the browser which origins are permitted to make requests
			if allowedOrigin == "*" {
				c.Header("Access-Control-Allow-Origin", "*")
			} else {
				c.Header("Access-Control-Allow-Origin", origin)
			}

			// Set allowed methods (GET, POST, PUT, DELETE, etc.)
			c.Header("Access-Control-Allow-Methods", joinStrings(config.AllowedMethods, ", "))

			// Set allowed headers (Content-Type, Authorization, etc.)
			c.Header("Access-Control-Allow-Headers", joinStrings(config.AllowedHeaders, ", "))

			// Set whether credentials (cookies, auth headers) are allowed
			if config.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}

			// Set how long the preflight response can be cached
			c.Header("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
		}

		// Handle preflight OPTIONS requests.
		// Browsers send OPTIONS requests before the actual request to check CORS permissions.
		if c.Request.Method == "OPTIONS" {
			// Return 204 No Content for preflight requests
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// Continue processing the request
		c.Next()
	}
}

// AuthMiddleware creates a middleware that enforces HTTP Basic Authentication.
// If authentication is enabled in the config, this middleware requires clients
// to provide valid credentials with each request.
//
// Parameters:
//   - config: Authentication configuration including username, password, and realm
//
// The middleware checks the Authorization header and validates credentials.
// If authentication fails, it returns HTTP 401 Unauthorized with a WWW-Authenticate header.
func AuthMiddleware(config AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication if not enabled
		if !config.Enabled {
			c.Next()
			return
		}

		// Get credentials from the Authorization header
		username, password, hasAuth := c.Request.BasicAuth()

		// Check if credentials were provided and are valid
		if !hasAuth || username != config.Username || password != config.Password {
			// Authentication failed - return 401 Unauthorized
			// The WWW-Authenticate header tells the client to use Basic auth
			c.Header("WWW-Authenticate", `Basic realm="`+config.Realm+`"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, NewErrorResponse(
				getClientTransactionID(c),
				getServerTransactionID(c),
				ErrorCodeUnspecifiedError,
				"Authentication required"))
			return
		}

		// Authentication successful - continue processing
		c.Next()
	}
}

// TransactionMiddleware creates a middleware that tracks transaction IDs.
// ASCOM Alpaca requires that every API response include both the client's
// transaction ID (echoed back) and a unique server transaction ID.
//
// Transaction IDs are used to:
//   - Correlate requests with responses
//   - Debug issues by tracking specific API calls
//   - Detect duplicate or retried requests
//
// Parameters:
//   - counter: Atomic counter for generating unique server transaction IDs
func TransactionMiddleware(counter *int32) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the client transaction ID from query parameters or form data.
		// The ASCOM Alpaca spec requires clients to include "ClientTransactionID"
		// with every request. We need to echo this back in the response.
		clientTxnIDStr := c.Query("ClientTransactionID")
		if clientTxnIDStr == "" {
			clientTxnIDStr = c.PostForm("ClientTransactionID")
		}

		// Parse the client transaction ID as an integer.
		// If parsing fails or no ID is provided, default to 0.
		clientTxnID := int32(0)
		if clientTxnIDStr != "" {
			if val, err := strconv.ParseInt(clientTxnIDStr, 10, 32); err == nil {
				clientTxnID = int32(val)
			}
		}

		// Generate a unique server transaction ID.
		// We use an atomic counter to ensure uniqueness across concurrent requests.
		// The counter wraps around at MaxTransactionID to stay within int32 bounds.
		serverTxnID := atomic.AddInt32(counter, 1)
		if serverTxnID > MaxTransactionID {
			// Wrap around to 1 (not 0, to distinguish from uninitialized)
			atomic.StoreInt32(counter, 1)
			serverTxnID = 1
		}

		// Store transaction IDs in the Gin context so handlers can access them.
		// This allows handlers to include transaction IDs in their responses.
		c.Set("ClientTransactionID", clientTxnID)
		c.Set("ServerTransactionID", serverTxnID)

		// Continue processing the request
		c.Next()
	}
}

// ErrorHandlerMiddleware creates a middleware that catches panics and converts them
// to proper ASCOM error responses. This prevents the server from crashing when
// an unexpected error occurs in a handler.
//
// Any panic that occurs during request processing is caught, logged, and returned
// as an HTTP 500 error with an ASCOM-compliant error response.
func ErrorHandlerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// A panic occurred - log it with full details
				logger.Error("Panic recovered in request handler",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method))

				// Return an ASCOM error response
				c.JSON(http.StatusInternalServerError, NewErrorResponse(
					getClientTransactionID(c),
					getServerTransactionID(c),
					ErrorCodeUnspecifiedError,
					"Internal server error"))

				// Abort further processing
				c.Abort()
			}
		}()

		// Process the request
		c.Next()
	}
}

// Helper function to get the client transaction ID from the Gin context.
// Returns 0 if not set.
func getClientTransactionID(c *gin.Context) int32 {
	if val, exists := c.Get("ClientTransactionID"); exists {
		if txnID, ok := val.(int32); ok {
			return txnID
		}
	}
	return 0
}

// Helper function to get the server transaction ID from the Gin context.
// Returns 0 if not set.
func getServerTransactionID(c *gin.Context) int32 {
	if val, exists := c.Get("ServerTransactionID"); exists {
		if txnID, ok := val.(int32); ok {
			return txnID
		}
	}
	return 0
}

// Helper function to join a slice of strings with a delimiter.
// This is used for building comma-separated lists in CORS headers.
func joinStrings(strs []string, delimiter string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += delimiter + strs[i]
	}
	return result
}
