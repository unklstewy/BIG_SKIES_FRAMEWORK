package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/bootstrap"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/credentials"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Version information (set via build flags)
var (
	version   = "dev"
	gitCommit = "unknown"
	buildTime = "unknown"
)

func main() {
	// Command-line flags
	var (
		// configFile would be used for loading config from file
		// Currently using default config, but keeping the flag for future use
		// configFile     = flag.String("config", "configs/bootstrap.yaml", "Path to bootstrap configuration file")
		pgpassFile     = flag.String("pgpass", "/shared/secrets/.pgpass", "Path to .pgpass file")
		logLevel       = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
		showVersion    = flag.Bool("version", false, "Show version information and exit")
		validateOnly   = flag.Bool("validate", false, "Validate configuration and credentials only")
		skipMigrations = flag.Bool("skip-migrations", false, "Skip database migrations (use with caution)")
		publishOnly    = flag.Bool("publish-only", false, "Only publish credentials, skip migrations")
	)
	flag.Parse()

	// Show version and exit
	if *showVersion {
		fmt.Printf("BIG SKIES Framework Bootstrap Coordinator\n")
		fmt.Printf("Version:    %s\n", version)
		fmt.Printf("Git Commit: %s\n", gitCommit)
		fmt.Printf("Build Time: %s\n", buildTime)
		os.Exit(0)
	}

	// Initialize logger
	logger, err := initLogger(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting BIG SKIES Framework Bootstrap Coordinator",
		zap.String("version", version),
		zap.String("git_commit", gitCommit),
		zap.String("build_time", buildTime),
	)

	// For now, use hardcoded config - TODO: implement config file loading
	config := bootstrap.DefaultBootstrapConfig()
	logger.Info("Using default bootstrap configuration")

	// Phase 1: Load database credentials
	logger.Info("Loading database credentials")
	creds, err := bootstrap.LoadCredentials(&config.Database)
	if err != nil {
		logger.Fatal("Failed to load database credentials", zap.Error(err))
	}

	// Build connection string
	config.Database.Password = creds.Password
	dbURL := config.Database.DatabaseURL()
	safeURL := fmt.Sprintf("postgresql://%s@%s:%d/%s",
		creds.User, creds.Host, creds.Port, creds.Database)
	logger.Info("Database credentials loaded", zap.String("connection", safeURL))

	// If validate-only mode, exit here
	if *validateOnly {
		logger.Info("Validation successful (--validate mode), exiting")
		os.Exit(0)
	}

	// Phase 2: Run database migrations
	ctx := context.Background()
	if !*skipMigrations && !*publishOnly {
		logger.Info("Running database migrations")
		
		// Create database pool
		pool, err := bootstrap.CreateDatabasePool(dbURL, 5)
		if err != nil {
			logger.Fatal("Failed to create database pool", zap.Error(err))
		}
		defer pool.Close()
		
		// Run migrations
		migrationRunner := bootstrap.NewMigrationRunner(pool, &config.Migrations, logger)
		if err := migrationRunner.Run(ctx); err != nil {
			logger.Fatal("Failed to run database migrations", zap.Error(err))
		}

		// Show applied migrations
		applied, err := migrationRunner.GetAppliedMigrations(ctx)
		if err != nil {
			logger.Warn("Failed to get applied migrations", zap.Error(err))
		} else {
			logger.Info("Database migrations completed",
				zap.Int("total_applied", len(applied)),
			)
			for _, m := range applied {
				logger.Debug("Applied migration",
					zap.String("name", m.Name),
					zap.Int("version", m.Version),
				)
			}
		}
	} else if *skipMigrations {
		logger.Warn("Skipping database migrations (--skip-migrations flag set)")
	} else if *publishOnly {
		logger.Info("Skipping database migrations (--publish-only mode)")
	}

	// Phase 3: Connect to MQTT and publish credentials
	logger.Info("Connecting to MQTT broker",
		zap.String("broker", config.MQTT.BrokerURL),
		zap.Int("port", config.MQTT.BrokerPort),
	)

	// Create MQTT config
	mqttConfig := &mqtt.Config{
		BrokerURL:            fmt.Sprintf("%s:%d", config.MQTT.BrokerURL, config.MQTT.BrokerPort),
		ClientID:             config.MQTT.ClientID,
		Username:             config.MQTT.Username,
		Password:             config.MQTT.Password,
		KeepAlive:            30 * time.Second,
		ConnectTimeout:       10 * time.Second,
		AutoReconnect:        true,
		MaxReconnectInterval: 5 * time.Minute,
	}
	
	mqttClient, err := mqtt.NewClient(mqttConfig, logger)
	if err != nil {
		logger.Fatal("Failed to create MQTT client", zap.Error(err))
	}

	if err := mqttClient.Connect(); err != nil {
		logger.Fatal("Failed to connect to MQTT broker", zap.Error(err))
	}
	defer mqttClient.Disconnect()

	logger.Info("Connected to MQTT broker")

	// Subscribe to credential request topic
	requestTopic := "bigskies/coordinator/bootstrap/request"
	logger.Info("Subscribing to credential request topic", zap.String("topic", requestTopic))

	if err := mqttClient.Subscribe(requestTopic, 1, func(topic string, payload []byte) error {
		logger.Info("Received credential request, publishing credentials")
		if err := publishCredentials(mqttClient, *pgpassFile, logger); err != nil {
			logger.Error("Failed to publish credentials on request", zap.Error(err))
			return err
		}
		return nil
	}); err != nil {
		logger.Fatal("Failed to subscribe to request topic", zap.Error(err))
	}

	// Publish credentials immediately on startup
	logger.Info("Publishing credentials to coordinators")
	if err := publishCredentials(mqttClient, *pgpassFile, logger); err != nil {
		logger.Fatal("Failed to publish credentials", zap.Error(err))
	}

	logger.Info("Bootstrap coordinator initialized successfully")
	logger.Info("Waiting for credential requests (press Ctrl+C to exit)...")

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Periodically republish credentials (every 30 seconds)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Wait for shutdown signal
	for {
		select {
		case sig := <-sigChan:
			logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
			logger.Info("Bootstrap coordinator shutdown complete")
			return
		case <-ticker.C:
			logger.Debug("Republishing credentials (periodic)")
			if err := publishCredentials(mqttClient, *pgpassFile, logger); err != nil {
				logger.Error("Failed to republish credentials", zap.Error(err))
			}
		}
	}
}

// publishCredentials publishes the .pgpass file path to MQTT for coordinators to consume
func publishCredentials(client *mqtt.Client, pgpassPath string, logger *zap.Logger) error {
	// Create credential message
	credMsg := credentials.NewCredentialMessage(pgpassPath)

	// Marshal to JSON
	payload, err := json.Marshal(credMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal credential message: %w", err)
	}

	// Publish to topic (QoS 1, retained)
	topic := "bigskies/coordinator/bootstrap/credentials"
	if err := client.Publish(topic, 1, true, payload); err != nil {
		return fmt.Errorf("failed to publish to %s: %w", topic, err)
	}

	logger.Debug("Published credentials",
		zap.String("topic", topic),
		zap.String("path", pgpassPath),
	)

	return nil
}

// initLogger initializes the zap logger with the specified log level
func initLogger(levelStr string) (*zap.Logger, error) {
	// Parse log level
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", levelStr, err)
	}

	// Create logger configuration
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(level)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// Use console encoder for more human-readable output
	config.Encoding = "console"
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	return config.Build()
}
