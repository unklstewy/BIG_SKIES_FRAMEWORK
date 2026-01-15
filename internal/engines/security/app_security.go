// Package security provides security engine implementations.
package security

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/healthcheck"
	"go.uber.org/zap"
)

// AppSecurityEngine manages application-level security including JWT tokens and API keys.
type AppSecurityEngine struct {
	jwtSecret     []byte
	tokenDuration time.Duration
	apiKeys       map[string]*APIKey // keyed by API key string
	blacklistedTokens map[string]time.Time // token ID -> expiry time for cleanup
	mu            sync.RWMutex
	logger        *zap.Logger
}

// APIKey represents an API key with metadata.
type APIKey struct {
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	Enabled   bool      `json:"enabled"`
}

// JWTClaims represents JWT token claims.
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

// NewAppSecurityEngine creates a new application security engine.
func NewAppSecurityEngine(jwtSecret string, tokenDuration time.Duration, logger *zap.Logger) *AppSecurityEngine {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &AppSecurityEngine{
		jwtSecret:         []byte(jwtSecret),
		tokenDuration:     tokenDuration,
		apiKeys:           make(map[string]*APIKey),
		blacklistedTokens: make(map[string]time.Time),
		logger:            logger.With(zap.String("engine", "app_security")),
	}
}

// GenerateToken creates a new JWT token for a user.
func (e *AppSecurityEngine) GenerateToken(userID, username, email string) (string, time.Time, error) {
	expiresAt := time.Now().Add(e.tokenDuration)
	// Generate unique token ID for blacklist tracking
	tokenIDBytes := make([]byte, 16)
	if _, err := rand.Read(tokenIDBytes); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate token ID: %w", err)
	}
	tokenID := hex.EncodeToString(tokenIDBytes)

	claims := &JWTClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "bigskies-security",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(e.jwtSecret)
	if err != nil {
		e.logger.Error("Failed to generate JWT token", zap.Error(err))
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	e.logger.Info("Generated JWT token",
		zap.String("user_id", userID),
		zap.Time("expires_at", expiresAt))

	return tokenString, expiresAt, nil
}

// ValidateToken validates a JWT token and returns the claims.
func (e *AppSecurityEngine) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return e.jwtSecret, nil
	})

	if err != nil {
		e.logger.Warn("Token validation failed", zap.Error(err))
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		// Check if token is blacklisted
		e.mu.RLock()
		_, blacklisted := e.blacklistedTokens[claims.ID]
		e.mu.RUnlock()
		
		if blacklisted {
			e.logger.Warn("Attempted to use blacklisted token", zap.String("user_id", claims.UserID))
			return nil, fmt.Errorf("token has been revoked")
		}
		
		e.logger.Debug("Token validated successfully", zap.String("user_id", claims.UserID))
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// GenerateAPIKey creates a new API key for a user.
func (e *AppSecurityEngine) GenerateAPIKey(name, userID string, expiresAt *time.Time) (*APIKey, error) {
	// Generate random API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}
	keyString := hex.EncodeToString(keyBytes)

	apiKey := &APIKey{
		Key:       keyString,
		Name:      name,
		UserID:    userID,
		CreatedAt: time.Now(),
		Enabled:   true,
	}

	if expiresAt != nil {
		apiKey.ExpiresAt = *expiresAt
	}

	e.mu.Lock()
	e.apiKeys[keyString] = apiKey
	e.mu.Unlock()

	e.logger.Info("Generated API key",
		zap.String("name", name),
		zap.String("user_id", userID))

	return apiKey, nil
}

// ValidateAPIKey validates an API key and returns the associated user ID.
func (e *AppSecurityEngine) ValidateAPIKey(keyString string) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	apiKey, exists := e.apiKeys[keyString]
	if !exists {
		return "", fmt.Errorf("invalid API key")
	}

	if !apiKey.Enabled {
		return "", fmt.Errorf("API key is disabled")
	}

	// Check expiration
	if !apiKey.ExpiresAt.IsZero() && time.Now().After(apiKey.ExpiresAt) {
		return "", fmt.Errorf("API key has expired")
	}

	return apiKey.UserID, nil
}

// RevokeAPIKey disables an API key.
func (e *AppSecurityEngine) RevokeAPIKey(keyString string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	apiKey, exists := e.apiKeys[keyString]
	if !exists {
		return fmt.Errorf("API key not found")
	}

	apiKey.Enabled = false
	e.logger.Info("Revoked API key", zap.String("name", apiKey.Name))

	return nil
}

// ListAPIKeys returns all API keys for a user.
func (e *AppSecurityEngine) ListAPIKeys(userID string) []*APIKey {
	e.mu.RLock()
	defer e.mu.RUnlock()

	keys := make([]*APIKey, 0)
	for _, apiKey := range e.apiKeys {
		if apiKey.UserID == userID {
			keys = append(keys, apiKey)
		}
	}

	return keys
}

// Check returns the health status of the application security engine.
func (e *AppSecurityEngine) Check(ctx context.Context) *healthcheck.Result {
	e.mu.RLock()
	activeKeys := 0
	expiredKeys := 0
	for _, apiKey := range e.apiKeys {
		if apiKey.Enabled {
			if !apiKey.ExpiresAt.IsZero() && time.Now().After(apiKey.ExpiresAt) {
				expiredKeys++
			} else {
				activeKeys++
			}
		}
	}
	e.mu.RUnlock()

	return &healthcheck.Result{
		ComponentName: "app_security_engine",
		Status:        healthcheck.StatusHealthy,
		Message:       "Application security engine is operational",
		Timestamp:     time.Now(),
		Details: map[string]interface{}{
			"active_api_keys":  activeKeys,
			"expired_api_keys": expiredKeys,
			"jwt_configured":   len(e.jwtSecret) > 0,
		},
	}
}

// Name returns the name of the engine.
func (e *AppSecurityEngine) Name() string {
	return "app_security_engine"
}

// RefreshToken generates a new token from an existing valid token.
func (e *AppSecurityEngine) RefreshToken(tokenString string) (string, time.Time, error) {
	claims, err := e.ValidateToken(tokenString)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("cannot refresh invalid token: %w", err)
	}

	return e.GenerateToken(claims.UserID, claims.Username, claims.Email)
}

// RevokeToken adds a token to the blacklist, preventing further use.
func (e *AppSecurityEngine) RevokeToken(tokenString string) error {
	// Parse token to get claims
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return e.jwtSecret, nil
	})

	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return fmt.Errorf("invalid token claims")
	}

	// Add to blacklist with expiry time for cleanup
	e.mu.Lock()
	e.blacklistedTokens[claims.ID] = claims.ExpiresAt.Time
	e.mu.Unlock()

	e.logger.Info("Token revoked",
		zap.String("token_id", claims.ID),
		zap.String("user_id", claims.UserID))

	return nil
}

// CleanupExpiredBlacklistedTokens removes expired tokens from the blacklist.
// Should be called periodically to prevent memory growth.
func (e *AppSecurityEngine) CleanupExpiredBlacklistedTokens() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	count := 0
	for tokenID, expiresAt := range e.blacklistedTokens {
		if now.After(expiresAt) {
			delete(e.blacklistedTokens, tokenID)
			count++
		}
	}

	if count > 0 {
		e.logger.Debug("Cleaned up expired blacklisted tokens", zap.Int("count", count))
	}

	return count
}
