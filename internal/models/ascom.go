// Package models provides data structures for the BIG SKIES Framework.
package models

import (
	"time"
)

// AlpacaDevice represents a discovered ASCOM Alpaca device.
type AlpacaDevice struct {
	DeviceID      string    `json:"device_id" db:"device_id"`           // Unique device identifier
	DeviceType    string    `json:"device_type" db:"device_type"`       // telescope, camera, dome, etc.
	DeviceNumber  int       `json:"device_number" db:"device_number"`   // ASCOM device number
	Name          string    `json:"name" db:"name"`                     // Device name
	Description   string    `json:"description" db:"description"`       // Device description
	DriverInfo    string    `json:"driver_info" db:"driver_info"`       // Driver information
	DriverVersion string    `json:"driver_version" db:"driver_version"` // Driver version
	ServerURL     string    `json:"server_url" db:"server_url"`         // Alpaca server base URL
	UUID          string    `json:"uuid" db:"uuid"`                     // Device UUID
	Connected     bool      `json:"connected" db:"connected"`           // Connection status
	LastSeen      time.Time `json:"last_seen" db:"last_seen"`           // Last discovery time
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// AlpacaResponse is the standard response wrapper for all ASCOM Alpaca API calls.
type AlpacaResponse struct {
	ClientTransactionID int    `json:"ClientTransactionID"` // Client transaction ID
	ServerTransactionID int    `json:"ServerTransactionID"` // Server transaction ID
	ErrorNumber         int    `json:"ErrorNumber"`         // 0 = success, non-zero = error
	ErrorMessage        string `json:"ErrorMessage"`        // Error message if ErrorNumber != 0
}

// AlpacaValueResponse wraps a value with standard Alpaca response fields.
type AlpacaValueResponse struct {
	AlpacaResponse
	Value interface{} `json:"Value"` // The actual returned value
}

// DiscoveryResponse represents the response from Alpaca discovery protocol.
type DiscoveryResponse struct {
	AlpacaPort int `json:"AlpacaPort"` // Port number for Alpaca API
}

// TelescopeConfig represents a telescope configuration.
type TelescopeConfig struct {
	ID                string    `json:"id" db:"id"`
	Name              string    `json:"name" db:"name"`
	Description       string    `json:"description" db:"description"`
	TelescopeDevice   string    `json:"telescope_device" db:"telescope_device"`     // Device ID for telescope
	CameraDevice      string    `json:"camera_device" db:"camera_device"`           // Device ID for camera
	DomeDevice        string    `json:"dome_device" db:"dome_device"`               // Device ID for dome
	FocuserDevice     string    `json:"focuser_device" db:"focuser_device"`         // Device ID for focuser
	FilterWheelDevice string    `json:"filterwheel_device" db:"filterwheel_device"` // Device ID for filter wheel
	RotatorDevice     string    `json:"rotator_device" db:"rotator_device"`         // Device ID for rotator
	MountType         string    `json:"mount_type" db:"mount_type"`                 // altaz, equatorial, etc.
	Latitude          float64   `json:"latitude" db:"latitude"`                     // Observatory latitude
	Longitude         float64   `json:"longitude" db:"longitude"`                   // Observatory longitude
	Elevation         float64   `json:"elevation" db:"elevation"`                   // Observatory elevation (meters)
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// TelescopeStatus represents the current status of a telescope.
type TelescopeStatus struct {
	Connected      bool    `json:"connected"`
	Tracking       bool    `json:"tracking"`
	Slewing        bool    `json:"slewing"`
	AtPark         bool    `json:"at_park"`
	AtHome         bool    `json:"at_home"`
	RightAscension float64 `json:"right_ascension"` // Hours
	Declination    float64 `json:"declination"`     // Degrees
	Altitude       float64 `json:"altitude"`        // Degrees
	Azimuth        float64 `json:"azimuth"`         // Degrees
	SiderealTime   float64 `json:"sidereal_time"`   // Hours
	UTCDate        string  `json:"utc_date"`        // ISO 8601 format
}

// CameraStatus represents the current status of a camera.
type CameraStatus struct {
	Connected        bool    `json:"connected"`
	CameraState      string  `json:"camera_state"`    // Idle, Waiting, Exposing, Reading, Download, Error
	CCDTemperature   float64 `json:"ccd_temperature"` // Celsius
	CoolerOn         bool    `json:"cooler_on"`
	CoolerPower      float64 `json:"cooler_power"` // Percentage
	ImageReady       bool    `json:"image_ready"`
	PercentCompleted int     `json:"percent_completed"` // Exposure completion percentage
}

// DomeStatus represents the current status of a dome.
type DomeStatus struct {
	Connected     bool    `json:"connected"`
	AtHome        bool    `json:"at_home"`
	AtPark        bool    `json:"at_park"`
	Slewing       bool    `json:"slewing"`
	Azimuth       float64 `json:"azimuth"`        // Degrees
	ShutterStatus string  `json:"shutter_status"` // Open, Closed, Opening, Closing, Error
}

// FocuserStatus represents the current status of a focuser.
type FocuserStatus struct {
	Connected   bool    `json:"connected"`
	IsMoving    bool    `json:"is_moving"`
	Position    int     `json:"position"`    // Current position
	MaxStep     int     `json:"max_step"`    // Maximum position
	TempComp    bool    `json:"temp_comp"`   // Temperature compensation enabled
	Temperature float64 `json:"temperature"` // Celsius
}

// FilterWheelStatus represents the current status of a filter wheel.
type FilterWheelStatus struct {
	Connected bool     `json:"connected"`
	Position  int      `json:"position"` // Current position (0-based)
	Names     []string `json:"names"`    // Filter names
}

// RotatorStatus represents the current status of a rotator.
type RotatorStatus struct {
	Connected          bool    `json:"connected"`
	IsMoving           bool    `json:"is_moving"`
	Position           float64 `json:"position"`            // Degrees
	MechanicalPosition float64 `json:"mechanical_position"` // Degrees
}

// SwitchStatus represents the current status of a switch device.
type SwitchStatus struct {
	Connected    bool           `json:"connected"`
	MaxSwitch    int            `json:"max_switch"`    // Number of switches
	SwitchStates map[int]bool   `json:"switch_states"` // State of each switch
	SwitchNames  map[int]string `json:"switch_names"`  // Name of each switch
}

// SafetyMonitorStatus represents the current status of a safety monitor.
type SafetyMonitorStatus struct {
	Connected bool `json:"connected"`
	IsSafe    bool `json:"is_safe"` // True if conditions are safe
}

// ObservingConditionsData represents weather/environmental data.
type ObservingConditionsData struct {
	Connected      bool    `json:"connected"`
	CloudCover     float64 `json:"cloud_cover"`     // Percentage
	DewPoint       float64 `json:"dew_point"`       // Celsius
	Humidity       float64 `json:"humidity"`        // Percentage
	Pressure       float64 `json:"pressure"`        // hPa
	RainRate       float64 `json:"rain_rate"`       // mm/hour
	SkyBrightness  float64 `json:"sky_brightness"`  // Lux
	SkyQuality     float64 `json:"sky_quality"`     // mag/arcsec²
	SkyTemperature float64 `json:"sky_temperature"` // Celsius
	StarFWHM       float64 `json:"star_fwhm"`       // arcseconds
	Temperature    float64 `json:"temperature"`     // Celsius
	WindDirection  float64 `json:"wind_direction"`  // Degrees
	WindGust       float64 `json:"wind_gust"`       // m/s
	WindSpeed      float64 `json:"wind_speed"`      // m/s
}

// CoverCalibratorStatus represents the current status of a cover calibrator.
type CoverCalibratorStatus struct {
	Connected       bool   `json:"connected"`
	CoverState      string `json:"cover_state"`      // NotPresent, Closed, Moving, Open, Unknown, Error
	CalibratorState string `json:"calibrator_state"` // NotPresent, Off, NotReady, Ready, Unknown, Error
	Brightness      int    `json:"brightness"`       // Calibrator brightness level
}

// DeviceCommand represents a command to execute on a device.
type DeviceCommand struct {
	DeviceID   string                 `json:"device_id"`
	DeviceType string                 `json:"device_type"`
	Method     string                 `json:"method"`     // ASCOM method name
	Parameters map[string]interface{} `json:"parameters"` // Method parameters
}

// DeviceCommandResponse represents the response from a device command.
type DeviceCommandResponse struct {
	Success bool        `json:"success"`
	Value   interface{} `json:"value,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SimulatorInstance represents a running simulator instance.
type SimulatorInstance struct {
	InstanceID    string     `json:"instance_id" db:"instance_id"`
	PluginGUID    string     `json:"plugin_guid" db:"plugin_guid"`
	ContainerID   string     `json:"container_id" db:"container_id"`
	ServerURL     string     `json:"server_url" db:"server_url"`
	APIPort       int        `json:"api_port" db:"api_port"`
	DiscoveryPort int        `json:"discovery_port" db:"discovery_port"`
	Status        string     `json:"status" db:"status"` // running, stopped, error
	StartedAt     time.Time  `json:"started_at" db:"started_at"`
	StoppedAt     *time.Time `json:"stopped_at,omitempty" db:"stopped_at"`
}
