// Package main is the entry point for the application coordinator service.
package main

import (
	"context"
	"flag"
	"fmt"
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
	// Default to postgres:5432 for Docker, but allow override via flag or env var
	defaultDBURL := "postgresql://bigskies:bigskies_dev_password@postgres:5432/bigskies?sslmode=disable"
	if envURL := os.Getenv("DATABASE_URL"); envURL != "" {
		defaultDBURL = envURL
	}
	databaseURL := flag.String("database-url", defaultDBURL, "PostgreSQL connection string")
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

	logger.Info("Starting BIG SKIES Application Coordinator")

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
	coordConfig, err := configLoader.LoadCoordinatorConfig(ctx, "application-coordinator")
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
	registryCheckInterval, err := coordConfig.GetDuration("registry_check_interval", 60*time.Second)
	if err != nil {
		logger.Fatal("Failed to parse registry_check_interval", zap.Error(err))
	}
	serviceTimeout, err := coordConfig.GetDuration("service_timeout", 180*time.Second)
	if err != nil {
		logger.Fatal("Failed to parse service_timeout", zap.Error(err))
	}

	// Construct full broker URL
	fullBrokerURL := fmt.Sprintf("%s:%d", brokerURL, brokerPort)

	logger.Info("Loaded configuration from database",
		zap.String("broker_url", fullBrokerURL),
		zap.Duration("registry_check_interval", registryCheckInterval),
		zap.Duration("service_timeout", serviceTimeout))

	// Create coordinator configuration struct
	cfg := &coordinators.ApplicationCoordinatorConfig{
		BrokerURL:             fullBrokerURL,
		RegistryCheckInterval: registryCheckInterval,
		ServiceTimeout:        serviceTimeout,
	}
	cfg.Name = "application-coordinator"
	cfg.LogLevel = *logLevel

	// Create application coordinator
	coordinator, err := coordinators.NewApplicationCoordinator(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create application coordinator", zap.Error(err))
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

	logger.Info("Application coordinator running, press Ctrl+C to stop")

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

	logger.Info("Application coordinator stopped successfully")
}
