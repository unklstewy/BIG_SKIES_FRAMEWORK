// Package main is the entry point for the plugin coordinator service.
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
	pluginDir := flag.String("plugin-dir", "/var/lib/bigskies/plugins", "Plugin storage directory")
	scanInterval := flag.Duration("scan-interval", 5*time.Minute, "Plugin scan interval")
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

	logger.Info("Starting BIG SKIES Plugin Coordinator",
		zap.String("plugin_dir", *pluginDir),
		zap.Duration("scan_interval", *scanInterval))

	// Create coordinator configuration
	config := &coordinators.PluginCoordinatorConfig{
		BrokerURL:    *brokerURL,
		PluginDir:    *pluginDir,
		ScanInterval: *scanInterval,
	}
	config.Name = "plugin-coordinator"
	config.LogLevel = *logLevel

	// Create plugin coordinator
	coordinator, err := coordinators.NewPluginCoordinator(config, logger)
	if err != nil {
		logger.Fatal("Failed to create plugin coordinator", zap.Error(err))
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

	logger.Info("Plugin coordinator running, press Ctrl+C to stop")

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

	logger.Info("Plugin coordinator stopped successfully")
}
