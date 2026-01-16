// Package main provides configuration management service for ASCOM Alpaca Simulator plugin.
package main

import "time"

// ConfigRequest represents a configuration load request from MQTT.
type ConfigRequest struct {
	Command   string `json:"command"`    // load_config, list_configs, get_status
	Model     string `json:"model"`      // s30, s30-pro, s50 (for load_config)
	MountType string `json:"mount_type"` // altaz, equatorial, german-equatorial (for load_config)
	RequestID string `json:"request_id"` // Optional request ID for tracking
}

// ConfigResponse represents a response to a configuration request.
type ConfigResponse struct {
	RequestID string                 `json:"request_id,omitempty"`
	Command   string                 `json:"command"`
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ConfigStatus represents the current configuration state.
type ConfigStatus struct {
	Model      string    `json:"model"`
	MountType  string    `json:"mount_type"`
	LoadedAt   time.Time `json:"loaded_at"`
	LoadedBy   string    `json:"loaded_by,omitempty"`
	ConfigPath string    `json:"config_path"`
}

// ConfigEvent represents a configuration change event.
type ConfigEvent struct {
	EventType string       `json:"event_type"` // config_loaded, config_failed, config_backup_created
	Status    ConfigStatus `json:"status,omitempty"`
	Message   string       `json:"message"`
	Timestamp time.Time    `json:"timestamp"`
}

// AvailableConfig represents a single available configuration.
type AvailableConfig struct {
	Model       string   `json:"model"`
	MountTypes  []string `json:"mount_types"`
	Description string   `json:"description"`
}

// Valid models and their descriptions
var validModels = map[string]string{
	"s30":     "Seestar S30 (30mm f/5, 150mm FL, Sony IMX662 1920×1080)",
	"s30-pro": "Seestar S30 Pro (30mm f/5.3, 160mm FL, Sony IMX585 3840×2160 4K)",
	"s50":     "Seestar S50 (50mm f/5, 250mm FL, Sony IMX462 1920×1080)",
}

// Valid mount types and their descriptions
var validMountTypes = map[string]string{
	"altaz":             "Altitude-Azimuth mount",
	"equatorial":        "Equatorial mount (polar-aligned)",
	"german-equatorial": "German Equatorial mount (with meridian flip)",
}

// ValidateModel checks if the provided model is valid.
func ValidateModel(model string) bool {
	_, ok := validModels[model]
	return ok
}

// ValidateMountType checks if the provided mount type is valid.
func ValidateMountType(mountType string) bool {
	_, ok := validMountTypes[mountType]
	return ok
}

// GetModelDescription returns the description for a model.
func GetModelDescription(model string) string {
	if desc, ok := validModels[model]; ok {
		return desc
	}
	return ""
}

// GetMountTypeDescription returns the description for a mount type.
func GetMountTypeDescription(mountType string) string {
	if desc, ok := validMountTypes[mountType]; ok {
		return desc
	}
	return ""
}

// GetAvailableConfigs returns all available configurations.
func GetAvailableConfigs() []AvailableConfig {
	configs := make([]AvailableConfig, 0, len(validModels))

	// Get mount types as slice
	mountTypes := make([]string, 0, len(validMountTypes))
	for mt := range validMountTypes {
		mountTypes = append(mountTypes, mt)
	}

	for model, desc := range validModels {
		configs = append(configs, AvailableConfig{
			Model:       model,
			MountTypes:  mountTypes,
			Description: desc,
		})
	}

	return configs
}
