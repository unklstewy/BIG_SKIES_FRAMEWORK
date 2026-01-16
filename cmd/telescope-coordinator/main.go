// Package main is the entry point for the telescope coordinator service.
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
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
	"go.uber.org/zap"
)

func main() {
	// Parse command line flags
	brokerURL := flag.String("broker-url", "tcp://localhost:1883", "MQTT broker URL")
	databaseURL := flag.String("database-url", "postgresql://localhost:5432/bigskies", "PostgreSQL database URL")
	discoveryPort := flag.Int("discovery-port", 32227, "ASCOM Alpaca discovery port")
	healthInterval := flag.Duration("health-interval", 30*time.Second, "Health check interval")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Initialize logger
	var logger *zap.Logger
	var err error

	switch *logLevel {
	case "debug":
		logger, err = zap.NewDevelopment()
	default:
		logger, err = zap.NewProduction()
	}

	if err != nil {
		panic("failed to create logger: " + err.Error())
	}
	defer logger.Sync()

	logger.Info("Starting BIG SKIES Telescope Coordinator",
		zap.String("broker", *brokerURL),
		zap.String("database", maskDatabaseURL(*databaseURL)),
		zap.Int("discovery_port", *discoveryPort))

	// Create coordinator configuration
	config := &coordinators.TelescopeConfig{
		DatabaseURL:         *databaseURL,
		DiscoveryPort:       *discoveryPort,
		HealthCheckInterval: *healthInterval,
	}
	config.Name = "telescope-coordinator"
	config.LogLevel = *logLevel
	config.MQTTConfig = &mqtt.Config{
		BrokerURL:            *brokerURL,
		ClientID:             "telescope-coordinator",
		ConnectTimeout:       5 * time.Second,
		KeepAlive:            60 * time.Second,
		AutoReconnect:        true,
		MaxReconnectInterval: 60 * time.Second,
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		logger.Fatal("Invalid configuration", zap.Error(err))
	}

	// Create telescope coordinator
	coordinator, err := coordinators.NewTelescopeCoordinator(config, logger)
	if err != nil {
		logger.Fatal("Failed to create telescope coordinator", zap.Error(err))
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start coordinator
	if err := coordinator.Start(ctx); err != nil {
		logger.Fatal("Failed to start coordinator", zap.Error(err))
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("Telescope coordinator running, press Ctrl+C to stop")

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutdown signal received")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := coordinator.Stop(shutdownCtx); err != nil {
		logger.Error("Error during shutdown", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Telescope coordinator stopped successfully")
}

// validateConfig validates the coordinator configuration.
func validateConfig(config *coordinators.TelescopeConfig) error {
	if config.DatabaseURL == "" {
		return fmt.Errorf("database URL is required")
	}
	if config.DiscoveryPort == 0 {
		config.DiscoveryPort = 32227
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
	}
	if config.MQTTConfig == nil {
		config.MQTTConfig = &mqtt.Config{
			BrokerURL:            "tcp://localhost:1883",
			ClientID:             "telescope-coordinator",
			ConnectTimeout:       5 * time.Second,
			KeepAlive:            60 * time.Second,
			AutoReconnect:        true,
			MaxReconnectInterval: 60 * time.Second,
		}
	}
	return nil
}

// maskDatabaseURL masks sensitive information in database URL for logging.
func maskDatabaseURL(url string) string {
	// Simple masking - in production use proper URL parsing
	if len(url) > 20 {
		return url[:10] + "***" + url[len(url)-7:]
	}
	return "***"
}
