// Package coordinators implements the data store coordinator.
package coordinators

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/healthcheck"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
	"go.uber.org/zap"
)

// DataStoreCoordinator manages PostgreSQL database connections and operations.
type DataStoreCoordinator struct {
	*BaseCoordinator
	config *DataStoreCoordinatorConfig
	pool   *pgxpool.Pool
}

// DataStoreCoordinatorConfig holds configuration for the data store coordinator.
type DataStoreCoordinatorConfig struct {
	BaseConfig
	// BrokerURL is the MQTT broker URL
	BrokerURL string `json:"broker_url"`
	// DatabaseURL is the PostgreSQL connection string
	DatabaseURL string `json:"database_url"`
	// MaxConnections is the maximum number of connections in the pool
	MaxConnections int `json:"max_connections"`
	// MinConnections is the minimum number of connections in the pool
	MinConnections int `json:"min_connections"`
	// ConnectionTimeout for establishing connections
	ConnectionTimeout time.Duration `json:"connection_timeout"`
}

// NewDataStoreCoordinator creates a new data store coordinator instance.
func NewDataStoreCoordinator(config *DataStoreCoordinatorConfig, logger *zap.Logger) (*DataStoreCoordinator, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	brokerURL := config.BrokerURL
	if brokerURL == "" {
		brokerURL = "tcp://mqtt-broker:1883"
	}
	
	// Create MQTT client for coordination messages
	mqttConfig := &mqtt.Config{
		BrokerURL:            brokerURL,
		ClientID:             "datastore-coordinator",
		KeepAlive:            30 * time.Second,
		ConnectTimeout:       10 * time.Second,
		AutoReconnect:        true,
		MaxReconnectInterval: 5 * time.Minute,
	}
	
	mqttClient, err := mqtt.NewClient(mqttConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MQTT client: %w", err)
	}
	
	base := NewBaseCoordinator("datastore-coordinator", mqttClient, logger)
	
	dsc := &DataStoreCoordinator{
		BaseCoordinator: base,
		config:          config,
	}
	
	// Register self health check
	dsc.RegisterHealthCheck(dsc)
	
	return dsc, nil
}

// Start begins data store coordinator operations.
func (dsc *DataStoreCoordinator) Start(ctx context.Context) error {
	dsc.GetLogger().Info("Starting data store coordinator",
		zap.String("database_url", maskDatabaseURL(dsc.config.DatabaseURL)))
	
	// Create database pool configuration
	poolConfig, err := pgxpool.ParseConfig(dsc.config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}
	
	poolConfig.MaxConns = int32(dsc.config.MaxConnections)
	poolConfig.MinConns = int32(dsc.config.MinConnections)
	poolConfig.MaxConnLifetime = 1 * time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	
	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}
	
	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}
	
	dsc.pool = pool
	
	// Register cleanup function
	dsc.RegisterShutdownFunc(func(ctx context.Context) error {
		if dsc.pool != nil {
			dsc.pool.Close()
		}
		return nil
	})
	
	// Start base coordinator
	if err := dsc.BaseCoordinator.Start(ctx); err != nil {
		pool.Close()
		return err
	}
	
	dsc.GetLogger().Info("Data store coordinator started successfully")
	return nil
}

// Stop shuts down the data store coordinator.
func (dsc *DataStoreCoordinator) Stop(ctx context.Context) error {
	dsc.GetLogger().Info("Stopping data store coordinator")
	return dsc.BaseCoordinator.Stop(ctx)
}

// GetPool returns the database connection pool.
func (dsc *DataStoreCoordinator) GetPool() *pgxpool.Pool {
	return dsc.pool
}

// Check implements healthcheck.Checker interface.
func (dsc *DataStoreCoordinator) Check(ctx context.Context) *healthcheck.Result {
	status := healthcheck.StatusHealthy
	message := "Data store coordinator is healthy"
	details := make(map[string]interface{})
	
	// Check if pool is available
	if dsc.pool == nil {
		status = healthcheck.StatusUnhealthy
		message = "Database pool not initialized"
		details["pool_initialized"] = false
	} else {
		details["pool_initialized"] = true
		
		// Try to ping database
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		
		if err := dsc.pool.Ping(ctx); err != nil {
			status = healthcheck.StatusUnhealthy
			message = "Database ping failed: " + err.Error()
			details["ping_error"] = err.Error()
		} else {
			// Get pool stats
			stats := dsc.pool.Stat()
			details["total_conns"] = stats.TotalConns()
			details["idle_conns"] = stats.IdleConns()
			details["acquired_conns"] = stats.AcquiredConns()
			
			// Check if we're running low on connections
			if stats.AcquiredConns() >= int32(dsc.config.MaxConnections)-1 {
				status = healthcheck.StatusDegraded
				message = "Database connection pool near capacity"
			}
		}
	}
	
	return &healthcheck.Result{
		ComponentName: "datastore-coordinator",
		Status:        status,
		Message:       message,
		Timestamp:     time.Now(),
		Details:       details,
	}
}

// Name returns the coordinator name.
func (dsc *DataStoreCoordinator) Name() string {
	return "datastore-coordinator"
}

// LoadConfig loads configuration.
func (dsc *DataStoreCoordinator) LoadConfig(config interface{}) error {
	cfg, ok := config.(*DataStoreCoordinatorConfig)
	if !ok {
		return fmt.Errorf("invalid config type")
	}
	
	dsc.config = cfg
	return dsc.BaseCoordinator.LoadConfig(config)
}

// ValidateConfig validates the configuration.
func (dsc *DataStoreCoordinator) ValidateConfig() error {
	if dsc.config == nil {
		return fmt.Errorf("config is nil")
	}
	if dsc.config.DatabaseURL == "" {
		return fmt.Errorf("database_url is required")
	}
	if dsc.config.MaxConnections <= 0 {
		return fmt.Errorf("max_connections must be positive")
	}
	if dsc.config.MinConnections < 0 {
		return fmt.Errorf("min_connections cannot be negative")
	}
	if dsc.config.MinConnections > dsc.config.MaxConnections {
		return fmt.Errorf("min_connections cannot exceed max_connections")
	}
	return nil
}

// maskDatabaseURL masks sensitive information in the database URL.
func maskDatabaseURL(url string) string {
	// Simple masking - replace password
	// In production, use a proper URL parser
	return "postgres://***:***@..."
}
