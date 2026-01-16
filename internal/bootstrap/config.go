// Package bootstrap provides infrastructure for bootstrapping the BIG SKIES Framework.
//
// The bootstrap coordinator is responsible for:
//   - Loading credentials securely from .pgpass or environment
//   - Initializing database connections
//   - Running database migrations
//   - Starting coordinators in dependency order
//   - Monitoring coordinator health during startup
//   - Handling graceful shutdown
package bootstrap

import (
	"fmt"
	"time"
)

// BootstrapConfig holds configuration for the bootstrap coordinator.
type BootstrapConfig struct {
	// Database configuration
	Database DatabaseConfig `yaml:"database" json:"database"`

	// MQTT broker configuration
	MQTT MQTTConfig `yaml:"mqtt" json:"mqtt"`

	// Coordinator management configuration
	Coordinators CoordinatorConfig `yaml:"coordinators" json:"coordinators"`

	// Migrations configuration
	Migrations MigrationConfig `yaml:"migrations" json:"migrations"`
}

// DatabaseConfig holds database connection parameters.
type DatabaseConfig struct {
	// Host is the database server hostname
	Host string `yaml:"host" json:"host"`

	// Port is the database server port
	Port int `yaml:"port" json:"port"`

	// Name is the database name
	Name string `yaml:"name" json:"name"`

	// User is the database username
	User string `yaml:"user" json:"user"`

	// Password is loaded from .pgpass or environment (not in config file)
	Password string `yaml:"-" json:"-"`

	// SSLMode specifies the SSL mode (disable, require, verify-ca, verify-full)
	SSLMode string `yaml:"ssl_mode" json:"ssl_mode"`

	// MaxConnections is the maximum number of connections in the pool
	MaxConnections int `yaml:"max_connections" json:"max_connections"`

	// MinConnections is the minimum number of connections in the pool
	MinConnections int `yaml:"min_connections" json:"min_connections"`
}

// MQTTConfig holds MQTT broker configuration.
type MQTTConfig struct {
	// BrokerURL is the MQTT broker hostname
	BrokerURL string `yaml:"broker_url" json:"broker_url"`

	// BrokerPort is the MQTT broker port
	BrokerPort int `yaml:"broker_port" json:"broker_port"`

	// ClientID is the MQTT client identifier
	ClientID string `yaml:"client_id" json:"client_id"`

	// Username for MQTT authentication (optional)
	Username string `yaml:"username" json:"username"`

	// Password for MQTT authentication (optional, loaded from credentials)
	Password string `yaml:"-" json:"-"`
}

// CoordinatorConfig holds coordinator management configuration.
type CoordinatorConfig struct {
	// BinPath is the path to coordinator binaries
	BinPath string `yaml:"bin_path" json:"bin_path"`

	// StartupTimeout is the maximum time to wait for a coordinator to start
	StartupTimeout time.Duration `yaml:"startup_timeout" json:"startup_timeout"`

	// HealthCheckInterval is how often to check coordinator health during startup
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval"`

	// MaxStartupRetries is the maximum number of times to retry starting a failed coordinator
	MaxStartupRetries int `yaml:"max_startup_retries" json:"max_startup_retries"`

	// Order is the startup order for coordinators (respects dependencies)
	Order []string `yaml:"order" json:"order"`

	// FailFast determines if startup should abort on first coordinator failure
	FailFast bool `yaml:"fail_fast" json:"fail_fast"`
}

// MigrationConfig holds database migration configuration.
type MigrationConfig struct {
	// SchemaPath is the directory containing SQL schema files
	SchemaPath string `yaml:"schema_path" json:"schema_path"`

	// AutoMigrate determines if migrations should run automatically on startup
	AutoMigrate bool `yaml:"auto_migrate" json:"auto_migrate"`

	// Order is the migration execution order
	Order []string `yaml:"order" json:"order"`
}

// DefaultBootstrapConfig returns a bootstrap configuration with sensible defaults.
func DefaultBootstrapConfig() *BootstrapConfig {
	return &BootstrapConfig{
		Database: DatabaseConfig{
			Host:           "localhost",
			Port:           5432,
			Name:           "bigskies",
			User:           "bigskies",
			SSLMode:        "disable",
			MaxConnections: 20,
			MinConnections: 5,
		},
		MQTT: MQTTConfig{
			BrokerURL:  "localhost",
			BrokerPort: 1883,
			ClientID:   "bootstrap-coordinator",
		},
		Coordinators: CoordinatorConfig{
			BinPath:             "./bin",
			StartupTimeout:      30 * time.Second,
			HealthCheckInterval: 2 * time.Second,
			MaxStartupRetries:   3,
			FailFast:            true,
			Order: []string{
				"datastore-coordinator",
				"security-coordinator",
				"message-coordinator",
				"application-coordinator",
				"plugin-coordinator",
				"telescope-coordinator",
				"uielement-coordinator",
			},
		},
		Migrations: MigrationConfig{
			SchemaPath:  "configs/sql",
			AutoMigrate: true,
			Order: []string{
				"coordinator_config_schema.sql",
				"security_schema.sql",
				"telescope_schema.sql",
			},
		},
	}
}

// Validate checks if the bootstrap configuration is valid.
func (c *BootstrapConfig) Validate() error {
	// Validate database config
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Database.Port)
	}
	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("database password is required (load from .pgpass or environment)")
	}

	// Validate MQTT config
	if c.MQTT.BrokerURL == "" {
		return fmt.Errorf("MQTT broker URL is required")
	}
	if c.MQTT.BrokerPort <= 0 || c.MQTT.BrokerPort > 65535 {
		return fmt.Errorf("invalid MQTT broker port: %d", c.MQTT.BrokerPort)
	}

	// Validate coordinator config
	if c.Coordinators.BinPath == "" {
		return fmt.Errorf("coordinator bin path is required")
	}
	if c.Coordinators.StartupTimeout <= 0 {
		return fmt.Errorf("coordinator startup timeout must be positive")
	}
	if len(c.Coordinators.Order) == 0 {
		return fmt.Errorf("at least one coordinator must be specified in order")
	}

	// Validate migration config
	if c.Migrations.AutoMigrate && c.Migrations.SchemaPath == "" {
		return fmt.Errorf("migration schema path is required when auto_migrate is enabled")
	}

	return nil
}

// DatabaseURL returns a PostgreSQL connection URL.
func (c *DatabaseConfig) DatabaseURL() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Name,
		c.SSLMode,
	)
}

// BrokerURL returns the full MQTT broker URL.
func (c *MQTTConfig) BrokerURLFull() string {
	return fmt.Sprintf("tcp://%s:%d", c.BrokerURL, c.BrokerPort)
}
