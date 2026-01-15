// Package main is the entry point for the data store coordinator service.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/coordinators"
	"go.uber.org/zap"
)

func main() {
	// Parse command line flags
	databaseURL := flag.String("database-url", "postgres://localhost:5432/bigskies", "PostgreSQL connection URL")
	maxConns := flag.Int("max-connections", 25, "Maximum database connections")
	minConns := flag.Int("min-connections", 5, "Minimum database connections")
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

	logger.Info("Starting BIG SKIES Data Store Coordinator",
		zap.Int("max_connections", *maxConns),
		zap.Int("min_connections", *minConns))

	// Create coordinator configuration
	config := &coordinators.DataStoreCoordinatorConfig{
		DatabaseURL:       *databaseURL,
		MaxConnections:    *maxConns,
		MinConnections:    *minConns,
		ConnectionTimeout: 10 * time.Second,
	}
	config.Name = "datastore-coordinator"
	config.LogLevel = *logLevel

	// Validate configuration
	if err := validateConfig(config); err != nil {
		logger.Fatal("Invalid configuration", zap.Error(err))
	}

	// Create data store coordinator
	coordinator, err := coordinators.NewDataStoreCoordinator(config, logger)
	if err != nil {
		logger.Fatal("Failed to create data store coordinator", zap.Error(err))
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

	logger.Info("Data store coordinator running, press Ctrl+C to stop")

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

	logger.Info("Data store coordinator stopped successfully")
}

// validateConfig validates the coordinator configuration.
func validateConfig(config *coordinators.DataStoreCoordinatorConfig) error {
	if config.DatabaseURL == "" {
		config.DatabaseURL = "postgres://localhost:5432/bigskies"
	}
	if config.MaxConnections == 0 {
		config.MaxConnections = 25
	}
	if config.MinConnections == 0 {
		config.MinConnections = 5
	}
	if config.ConnectionTimeout == 0 {
		config.ConnectionTimeout = 10 * time.Second
	}
	return nil
}
