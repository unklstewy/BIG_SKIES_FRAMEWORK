// Package main is the entry point for the message coordinator service.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/config"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/coordinators"
	"go.uber.org/zap"
)

func main() {
	// Parse command line flags
	databaseURL := flag.String("database-url", "postgresql://bigskies:bigskies@localhost:5432/bigskies?sslmode=disable", "PostgreSQL connection string")
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

	logger.Info("Starting BIG SKIES Message Coordinator")

	// Create database connection pool for configuration loading
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, *databaseURL)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer dbPool.Close()

	// Create configuration loader
	configLoader := config.NewLoader(dbPool)

	// Load coordinator configuration from database
	coordConfig, err := configLoader.LoadCoordinatorConfig(ctx, "message-coordinator")
	if err != nil {
		logger.Fatal("Failed to load configuration from database", zap.Error(err))
	}

	// Parse configuration values with defaults
	brokerURL, err := coordConfig.GetString("broker_url", "localhost")
	if err != nil {
		logger.Fatal("Failed to parse broker_url", zap.Error(err))
	}
	brokerPort, err := coordConfig.GetInt("broker_port", 1883)
	if err != nil {
		logger.Fatal("Failed to parse broker_port", zap.Error(err))
	}
	monitorInterval, err := coordConfig.GetDuration("monitor_interval", 30*time.Second)
	if err != nil {
		logger.Fatal("Failed to parse monitor_interval", zap.Error(err))
	}
	maxReconnectAttempts, err := coordConfig.GetInt("max_reconnect_attempts", 5)
	if err != nil {
		logger.Fatal("Failed to parse max_reconnect_attempts", zap.Error(err))
	}

	logger.Info("Loaded configuration from database",
		zap.String("broker", brokerURL),
		zap.Int("port", brokerPort),
		zap.Duration("monitor_interval", monitorInterval),
		zap.Int("max_reconnect_attempts", maxReconnectAttempts))

	// Create coordinator configuration struct
	cfg := &coordinators.MessageCoordinatorConfig{
		BrokerURL:            brokerURL,
		BrokerPort:           brokerPort,
		MonitorInterval:      monitorInterval,
		MaxReconnectAttempts: maxReconnectAttempts,
	}
	cfg.Name = "message-coordinator"
	cfg.LogLevel = *logLevel

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		logger.Fatal("Invalid configuration", zap.Error(err))
	}

	// Create message coordinator
	coordinator, err := coordinators.NewMessageCoordinator(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create message coordinator", zap.Error(err))
	}

	// Inject config loader for runtime updates
	coordinator.SetConfigLoader(configLoader)

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

	logger.Info("Message coordinator running, press Ctrl+C to stop")

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

	logger.Info("Message coordinator stopped successfully")
}

// validateConfig validates the coordinator configuration.
func validateConfig(config *coordinators.MessageCoordinatorConfig) error {
	if config.BrokerURL == "" {
		config.BrokerURL = "localhost"
	}
	if config.BrokerPort == 0 {
		config.BrokerPort = 1883
	}
	if config.MonitorInterval == 0 {
		config.MonitorInterval = 30 * time.Second
	}
	if config.MaxReconnectAttempts == 0 {
		config.MaxReconnectAttempts = 10
	}
	return nil
}
