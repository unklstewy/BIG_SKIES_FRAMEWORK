// Package main is the entry point for the UI element coordinator service.
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
	brokerURL := flag.String("broker-url", "tcp://mqtt-broker:1883", "MQTT broker URL")
	scanInterval := flag.Duration("scan-interval", 10*time.Minute, "UI API scan interval")
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

	logger.Info("Starting BIG SKIES UI Element Coordinator",
		zap.Duration("scan_interval", *scanInterval))

	// Create coordinator configuration
	config := &coordinators.UIElementCoordinatorConfig{
		BrokerURL:    *brokerURL,
		ScanInterval: *scanInterval,
	}
	config.Name = "uielement-coordinator"
	config.LogLevel = *logLevel

	// Create UI element coordinator
	coordinator, err := coordinators.NewUIElementCoordinator(config, logger)
	if err != nil {
		logger.Fatal("Failed to create UI element coordinator", zap.Error(err))
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

	logger.Info("UI element coordinator running, press Ctrl+C to stop")

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

	logger.Info("UI element coordinator stopped successfully")
}
