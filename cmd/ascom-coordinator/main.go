// Package main is the entry point for the ASCOM coordinator service.
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
	httpAddress := flag.String("http-address", "0.0.0.0:11111", "HTTP server listen address for ASCOM API")
	discoveryPort := flag.Int("discovery-port", 32227, "ASCOM Alpaca UDP discovery port")
	healthInterval := flag.Duration("health-interval", 30*time.Second, "Health check interval")
	enableCORS := flag.Bool("enable-cors", true, "Enable CORS for HTTP API")
	serverName := flag.String("server-name", "BigSkies ASCOM Alpaca Server", "ASCOM server name")
	manufacturer := flag.String("manufacturer", "BigSkies", "ASCOM manufacturer name")
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

	logger.Info("Starting BIG SKIES ASCOM Coordinator",
		zap.String("broker", *brokerURL),
		zap.String("database", maskDatabaseURL(*databaseURL)),
		zap.String("http_address", *httpAddress),
		zap.Int("discovery_port", *discoveryPort),
		zap.String("server_name", *serverName))

	// Create coordinator configuration
	config := &coordinators.ASCOMConfig{
		DatabaseURL:         *databaseURL,
		HTTPListenAddress:   *httpAddress,
		DiscoveryPort:       *discoveryPort,
		HealthCheckInterval: *healthInterval,
		ReadTimeout:         30 * time.Second,
		WriteTimeout:        30 * time.Second,
		IdleTimeout:         60 * time.Second,
		EnableCORS:          *enableCORS,
		ServerName:          *serverName,
		ServerDescription:   "ASCOM Alpaca interface for BigSkies Framework",
		Manufacturer:        *manufacturer,
	}
	config.Name = "ascom-coordinator"
	config.LogLevel = *logLevel
	config.MQTTConfig = &mqtt.Config{
		BrokerURL:            *brokerURL,
		ClientID:             mqtt.CoordinatorASCOM,
		ConnectTimeout:       5 * time.Second,
		KeepAlive:            60 * time.Second,
		AutoReconnect:        true,
		MaxReconnectInterval: 60 * time.Second,
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		logger.Fatal("Invalid configuration", zap.Error(err))
	}

	// Create ASCOM coordinator
	coordinator, err := coordinators.NewASCOMCoordinator(config, logger)
	if err != nil {
		logger.Fatal("Failed to create ASCOM coordinator", zap.Error(err))
	}

	// Setup context with cancellation
	startCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start coordinator
	if err := coordinator.Start(startCtx); err != nil {
		logger.Fatal("Failed to start coordinator", zap.Error(err))
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("ASCOM coordinator running",
		zap.String("ascom_api", "http://"+*httpAddress),
		zap.String("discovery", fmt.Sprintf("UDP port %d", *discoveryPort)),
		zap.String("shutdown", "Press Ctrl+C to stop"))

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

	logger.Info("ASCOM coordinator stopped successfully")
}

// validateConfig validates the coordinator configuration.
func validateConfig(config *coordinators.ASCOMConfig) error {
	if config.DatabaseURL == "" {
		return fmt.Errorf("database URL is required")
	}
	if config.HTTPListenAddress == "" {
		config.HTTPListenAddress = "0.0.0.0:11111"
	}
	if config.DiscoveryPort == 0 {
		config.DiscoveryPort = 32227
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
	}
	if config.ServerName == "" {
		config.ServerName = "BigSkies ASCOM Alpaca Server"
	}
	if config.Manufacturer == "" {
		config.Manufacturer = "BigSkies"
	}
	if config.MQTTConfig == nil {
		config.MQTTConfig = &mqtt.Config{
			BrokerURL:            "tcp://localhost:1883",
			ClientID:             mqtt.CoordinatorASCOM,
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
