// Package main provides the entry point for the security coordinator.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/coordinators"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/engines/security"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Command-line flags
	brokerHost := flag.String("broker-host", "localhost", "MQTT broker host")
	brokerPort := flag.Int("broker-port", 1883, "MQTT broker port")
	clientID := flag.String("client-id", "security-coordinator", "MQTT client ID")
	databaseURL := flag.String("database-url", "postgres://postgres:postgres@localhost:5432/bigskies", "PostgreSQL database URL")
	jwtSecret := flag.String("jwt-secret", "", "JWT signing secret (required)")
	tokenDuration := flag.Duration("token-duration", 24*time.Hour, "JWT token duration")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	acmeDirectory := flag.String("acme-directory", "", "ACME directory URL for Let's Encrypt")
	acmeEmail := flag.String("acme-email", "", "Email for ACME notifications")
	acmeCacheDir := flag.String("acme-cache-dir", "./certs", "Directory for ACME certificate cache")
	flag.Parse()

	// Validate required flags
	if *jwtSecret == "" {
		fmt.Fprintf(os.Stderr, "Error: --jwt-secret is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize logger
	logger, err := createLogger(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting Security Coordinator",
		zap.String("broker_host", *brokerHost),
		zap.Int("broker_port", *brokerPort),
		zap.String("database_url", maskPassword(*databaseURL)),
		zap.Duration("token_duration", *tokenDuration))

	// Create MQTT configuration
	mqttConfig := &mqtt.Config{
		BrokerURL:            fmt.Sprintf("tcp://%s:%d", *brokerHost, *brokerPort),
		ClientID:             *clientID,
		Username:             os.Getenv("MQTT_USERNAME"),
		Password:             os.Getenv("MQTT_PASSWORD"),
		KeepAlive:            30 * time.Second,
		ConnectTimeout:       10 * time.Second,
		AutoReconnect:        true,
		MaxReconnectInterval: 5 * time.Minute,
	}

	// Create TLS configuration if ACME settings provided
	var tlsConfig *security.TLSConfig
	if *acmeDirectory != "" {
		tlsConfig = &security.TLSConfig{
			ACMEDirectoryURL: *acmeDirectory,
			Email:            *acmeEmail,
			CacheDir:         *acmeCacheDir,
			Domains:          []string{}, // Will be populated from certificate requests
		}
		logger.Info("ACME configuration enabled",
			zap.String("directory", *acmeDirectory),
			zap.String("email", *acmeEmail))
	}

	// Create coordinator configuration
	config := &coordinators.SecurityConfig{
		BaseConfig: coordinators.BaseConfig{
			Name:                "security",
			MQTTConfig:          mqttConfig,
			HealthCheckInterval: 30 * time.Second,
			LogLevel:            *logLevel,
		},
		DatabaseURL:   *databaseURL,
		JWTSecret:     *jwtSecret,
		TokenDuration: *tokenDuration,
		TLSConfig:     tlsConfig,
	}

	// Create coordinator
	coord, err := coordinators.NewSecurityCoordinator(config, logger)
	if err != nil {
		logger.Fatal("Failed to create security coordinator", zap.Error(err))
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start coordinator
	if err := coord.Start(ctx); err != nil {
		logger.Fatal("Failed to start coordinator", zap.Error(err))
	}

	logger.Info("Security coordinator running")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutdown signal received")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := coord.Stop(shutdownCtx); err != nil {
		logger.Error("Error during shutdown", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Security coordinator stopped")
}

// createLogger creates a zap logger with the specified log level.
func createLogger(level string) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		return nil, fmt.Errorf("invalid log level: %s", level)
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return config.Build()
}

// maskPassword masks password in database URL for logging.
func maskPassword(dbURL string) string {
	// Simple password masking for logging
	// In production, use a more robust URL parser
	return dbURL // For now, return as-is. Consider using url.Parse for proper masking
}
