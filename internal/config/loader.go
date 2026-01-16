// Package config provides database-driven configuration loading for coordinators.
//
// This package implements the database-driven configuration pattern documented in
// docs/architecture/COORDINATOR_ENGINE_ARCHITECTURE.md. Configuration values are
// stored in the coordinator_config table and loaded at runtime, supporting hot-reload
// via MQTT notifications.
package config

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConfigValue represents a single configuration value from the database.
type ConfigValue struct {
	ID              string
	CoordinatorName string
	ConfigKey       string
	ConfigValue     json.RawMessage // Raw JSONB value
	ConfigType      string          // Type hint: string, int, bool, float, duration, object
	Description     string
	IsSecret        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CoordinatorConfig holds all configuration values for a coordinator.
type CoordinatorConfig struct {
	CoordinatorName string
	Values          map[string]ConfigValue
}

// Loader loads coordinator configuration from the database.
type Loader struct {
	pool *pgxpool.Pool
}

// NewLoader creates a new configuration loader with the given database pool.
func NewLoader(pool *pgxpool.Pool) *Loader {
	return &Loader{
		pool: pool,
	}
}

// LoadCoordinatorConfig loads all configuration values for a specific coordinator.
//
// Returns CoordinatorConfig with all key-value pairs from the database.
// If no configuration exists for the coordinator, returns empty config (not an error).
func (l *Loader) LoadCoordinatorConfig(ctx context.Context, coordinatorName string) (*CoordinatorConfig, error) {
	query := `
		SELECT id, coordinator_name, config_key, config_value, config_type, 
		       description, is_secret, created_at, updated_at
		FROM coordinator_config
		WHERE coordinator_name = $1
		ORDER BY config_key
	`

	rows, err := l.pool.Query(ctx, query, coordinatorName)
	if err != nil {
		return nil, fmt.Errorf("failed to query coordinator config: %w", err)
	}
	defer rows.Close()

	config := &CoordinatorConfig{
		CoordinatorName: coordinatorName,
		Values:          make(map[string]ConfigValue),
	}

	for rows.Next() {
		var cv ConfigValue
		err := rows.Scan(
			&cv.ID,
			&cv.CoordinatorName,
			&cv.ConfigKey,
			&cv.ConfigValue,
			&cv.ConfigType,
			&cv.Description,
			&cv.IsSecret,
			&cv.CreatedAt,
			&cv.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan config value: %w", err)
		}
		config.Values[cv.ConfigKey] = cv
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating config rows: %w", err)
	}

	return config, nil
}

// GetString retrieves a string configuration value.
//
// Returns the value if found, or defaultValue if the key doesn't exist.
func (c *CoordinatorConfig) GetString(key string, defaultValue string) (string, error) {
	cv, exists := c.Values[key]
	if !exists {
		return defaultValue, nil
	}

	if cv.ConfigType != "string" {
		return "", fmt.Errorf("config key %s is type %s, not string", key, cv.ConfigType)
	}

	var value string
	if err := json.Unmarshal(cv.ConfigValue, &value); err != nil {
		return "", fmt.Errorf("failed to unmarshal string value for %s: %w", key, err)
	}

	return value, nil
}

// GetInt retrieves an integer configuration value.
//
// Returns the value if found, or defaultValue if the key doesn't exist.
func (c *CoordinatorConfig) GetInt(key string, defaultValue int) (int, error) {
	cv, exists := c.Values[key]
	if !exists {
		return defaultValue, nil
	}

	if cv.ConfigType != "int" {
		return 0, fmt.Errorf("config key %s is type %s, not int", key, cv.ConfigType)
	}

	var value int
	if err := json.Unmarshal(cv.ConfigValue, &value); err != nil {
		return 0, fmt.Errorf("failed to unmarshal int value for %s: %w", key, err)
	}

	return value, nil
}

// GetBool retrieves a boolean configuration value.
//
// Returns the value if found, or defaultValue if the key doesn't exist.
func (c *CoordinatorConfig) GetBool(key string, defaultValue bool) (bool, error) {
	cv, exists := c.Values[key]
	if !exists {
		return defaultValue, nil
	}

	if cv.ConfigType != "bool" {
		return false, fmt.Errorf("config key %s is type %s, not bool", key, cv.ConfigType)
	}

	var value bool
	if err := json.Unmarshal(cv.ConfigValue, &value); err != nil {
		return false, fmt.Errorf("failed to unmarshal bool value for %s: %w", key, err)
	}

	return value, nil
}

// GetFloat retrieves a float64 configuration value.
//
// Returns the value if found, or defaultValue if the key doesn't exist.
func (c *CoordinatorConfig) GetFloat(key string, defaultValue float64) (float64, error) {
	cv, exists := c.Values[key]
	if !exists {
		return defaultValue, nil
	}

	if cv.ConfigType != "float" {
		return 0, fmt.Errorf("config key %s is type %s, not float", key, cv.ConfigType)
	}

	var value float64
	if err := json.Unmarshal(cv.ConfigValue, &value); err != nil {
		return 0, fmt.Errorf("failed to unmarshal float value for %s: %w", key, err)
	}

	return value, nil
}

// GetDuration retrieves a time.Duration configuration value.
//
// The value is stored in the database as an integer representing seconds.
// Returns the value if found, or defaultValue if the key doesn't exist.
func (c *CoordinatorConfig) GetDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	cv, exists := c.Values[key]
	if !exists {
		return defaultValue, nil
	}

	if cv.ConfigType != "duration" && cv.ConfigType != "int" {
		return 0, fmt.Errorf("config key %s is type %s, not duration/int", key, cv.ConfigType)
	}

	var seconds int
	if err := json.Unmarshal(cv.ConfigValue, &seconds); err != nil {
		return 0, fmt.Errorf("failed to unmarshal duration value for %s: %w", key, err)
	}

	return time.Duration(seconds) * time.Second, nil
}

// GetObject retrieves a complex object configuration value.
//
// The value is unmarshaled into the provided interface{} pointer.
// Returns error if key doesn't exist or unmarshal fails.
func (c *CoordinatorConfig) GetObject(key string, target interface{}) error {
	cv, exists := c.Values[key]
	if !exists {
		return fmt.Errorf("config key %s not found", key)
	}

	if cv.ConfigType != "object" {
		return fmt.Errorf("config key %s is type %s, not object", key, cv.ConfigType)
	}

	if err := json.Unmarshal(cv.ConfigValue, target); err != nil {
		return fmt.Errorf("failed to unmarshal object value for %s: %w", key, err)
	}

	return nil
}

// UpdateConfigValue updates a single configuration value in the database.
//
// The value is marshaled to JSON and stored. The updated_by field tracks who made the change.
// Configuration history is automatically tracked via database trigger.
func (l *Loader) UpdateConfigValue(ctx context.Context, coordinatorName, key string, value interface{}, configType string, updatedBy *string) error {
	// Marshal value to JSON
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal config value: %w", err)
	}

	query := `
		UPDATE coordinator_config
		SET config_value = $1, updated_at = NOW(), updated_by = $2
		WHERE coordinator_name = $3 AND config_key = $4
	`

	var updatedByUUID *string
	if updatedBy != nil {
		updatedByUUID = updatedBy
	}

	result, err := l.pool.Exec(ctx, query, jsonValue, updatedByUUID, coordinatorName, key)
	if err != nil {
		return fmt.Errorf("failed to update config value: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("config key %s not found for coordinator %s", key, coordinatorName)
	}

	return nil
}

// InsertConfigValue inserts a new configuration value into the database.
//
// If the key already exists, returns an error. Use UpdateConfigValue to modify existing values.
func (l *Loader) InsertConfigValue(ctx context.Context, coordinatorName, key string, value interface{}, configType, description string, isSecret bool) error {
	// Marshal value to JSON
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal config value: %w", err)
	}

	query := `
		INSERT INTO coordinator_config (coordinator_name, config_key, config_value, config_type, description, is_secret)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = l.pool.Exec(ctx, query, coordinatorName, key, jsonValue, configType, description, isSecret)
	if err != nil {
		// Check for unique constraint violation
		if pgErr, ok := err.(*pgx.PgError); ok && pgErr.Code == "23505" {
			return fmt.Errorf("config key %s already exists for coordinator %s", key, coordinatorName)
		}
		return fmt.Errorf("failed to insert config value: %w", err)
	}

	return nil
}

// DeleteConfigValue removes a configuration value from the database.
//
// Returns error if the key doesn't exist.
func (l *Loader) DeleteConfigValue(ctx context.Context, coordinatorName, key string) error {
	query := `
		DELETE FROM coordinator_config
		WHERE coordinator_name = $1 AND config_key = $2
	`

	result, err := l.pool.Exec(ctx, query, coordinatorName, key)
	if err != nil {
		return fmt.Errorf("failed to delete config value: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("config key %s not found for coordinator %s", key, coordinatorName)
	}

	return nil
}

// GetConfigHistory retrieves the change history for a specific configuration key.
//
// Returns a list of changes ordered by most recent first.
func (l *Loader) GetConfigHistory(ctx context.Context, coordinatorName, key string, limit int) ([]ConfigHistoryEntry, error) {
	query := `
		SELECT h.id, h.config_id, h.coordinator_name, h.config_key, 
		       h.old_value, h.new_value, h.changed_at, h.changed_by
		FROM coordinator_config_history h
		JOIN coordinator_config c ON h.config_id = c.id
		WHERE c.coordinator_name = $1 AND c.config_key = $2
		ORDER BY h.changed_at DESC
		LIMIT $3
	`

	rows, err := l.pool.Query(ctx, query, coordinatorName, key, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query config history: %w", err)
	}
	defer rows.Close()

	var history []ConfigHistoryEntry
	for rows.Next() {
		var entry ConfigHistoryEntry
		err := rows.Scan(
			&entry.ID,
			&entry.ConfigID,
			&entry.CoordinatorName,
			&entry.ConfigKey,
			&entry.OldValue,
			&entry.NewValue,
			&entry.ChangedAt,
			&entry.ChangedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan history entry: %w", err)
		}
		history = append(history, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating history rows: %w", err)
	}

	return history, nil
}

// ConfigHistoryEntry represents a single change in configuration history.
type ConfigHistoryEntry struct {
	ID              string
	ConfigID        string
	CoordinatorName string
	ConfigKey       string
	OldValue        *json.RawMessage
	NewValue        json.RawMessage
	ChangedAt       time.Time
	ChangedBy       *string
}

// String returns a human-readable representation of the history entry.
func (e *ConfigHistoryEntry) String() string {
	oldVal := "null"
	if e.OldValue != nil {
		oldVal = string(*e.OldValue)
	}
	changedBy := "system"
	if e.ChangedBy != nil {
		changedBy = *e.ChangedBy
	}
	return fmt.Sprintf("[%s] %s.%s: %s → %s (by %s)",
		e.ChangedAt.Format(time.RFC3339),
		e.CoordinatorName,
		e.ConfigKey,
		oldVal,
		string(e.NewValue),
		changedBy,
	)
}

// ValidateConfigType validates that a value matches the expected config type.
func ValidateConfigType(value interface{}, configType string) error {
	switch configType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "int":
		switch v := value.(type) {
		case int, int32, int64:
			// Valid
		case float64:
			// JSON numbers unmarshal as float64, check if it's an integer
			if v != float64(int(v)) {
				return fmt.Errorf("expected int, got float with decimal: %v", v)
			}
		default:
			return fmt.Errorf("expected int, got %T", value)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected bool, got %T", value)
		}
	case "float":
		switch value.(type) {
		case float32, float64, int, int32, int64:
			// Valid - numbers are acceptable
		default:
			return fmt.Errorf("expected float, got %T", value)
		}
	case "duration":
		// Duration is stored as int (seconds)
		switch value.(type) {
		case int, int32, int64, float64:
			// Valid
		default:
			return fmt.Errorf("expected duration (int seconds), got %T", value)
		}
	case "object":
		// Objects must be map or struct-like
		switch value.(type) {
		case map[string]interface{}, []interface{}:
			// Valid
		default:
			// Could be a struct, which is also valid
		}
	default:
		return fmt.Errorf("unknown config type: %s", configType)
	}
	return nil
}

// ParseConfigValueString parses a string value into the appropriate type based on configType.
//
// This is useful for parsing configuration values from environment variables or command-line flags.
func ParseConfigValueString(valueStr string, configType string) (interface{}, error) {
	switch configType {
	case "string":
		return valueStr, nil
	case "int":
		val, err := strconv.Atoi(valueStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse int: %w", err)
		}
		return val, nil
	case "bool":
		val, err := strconv.ParseBool(valueStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse bool: %w", err)
		}
		return val, nil
	case "float":
		val, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse float: %w", err)
		}
		return val, nil
	case "duration":
		// Parse as integer seconds
		val, err := strconv.Atoi(valueStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse duration (seconds): %w", err)
		}
		return val, nil
	case "object":
		// Parse as JSON
		var val interface{}
		if err := json.Unmarshal([]byte(valueStr), &val); err != nil {
			return nil, fmt.Errorf("failed to parse object JSON: %w", err)
		}
		return val, nil
	default:
		return nil, fmt.Errorf("unknown config type: %s", configType)
	}
}
