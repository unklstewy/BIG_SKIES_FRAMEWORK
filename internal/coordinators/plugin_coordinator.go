// Package coordinators implements the plugin coordinator.
package coordinators

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/healthcheck"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
	"go.uber.org/zap"
)

// PluginCoordinator manages plugin lifecycle operations.
type PluginCoordinator struct {
	*BaseCoordinator
	config   *PluginCoordinatorConfig
	registry *PluginRegistry
}

// PluginCoordinatorConfig holds configuration for the plugin coordinator.
type PluginCoordinatorConfig struct {
	BaseConfig
	// BrokerURL is the MQTT broker URL
	BrokerURL string `json:"broker_url"`
	// PluginDir is the directory for plugin storage
	PluginDir string `json:"plugin_dir"`
	// ScanInterval for periodic plugin scanning
	ScanInterval time.Duration `json:"scan_interval"`
}

// PluginRegistry maintains a registry of installed plugins.
type PluginRegistry struct {
	plugins map[string]*PluginEntry
	mu      sync.RWMutex
}

// PluginEntry represents an installed plugin.
type PluginEntry struct {
	// GUID is the unique plugin identifier
	GUID string `json:"guid"`
	// Name is the plugin name
	Name string `json:"name"`
	// Version is the plugin version
	Version string `json:"version"`
	// Status is the current plugin status
	Status PluginStatus `json:"status"`
	// InstalledAt is when the plugin was installed
	InstalledAt time.Time `json:"installed_at"`
	// LastVerified is when the plugin was last verified
	LastVerified time.Time `json:"last_verified"`
	// ContainerID is the Docker container ID
	ContainerID string `json:"container_id,omitempty"`
	// Metadata contains plugin-specific information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// PluginStatus represents the status of a plugin.
type PluginStatus string

const (
	// PluginStatusInstalled indicates the plugin is installed
	PluginStatusInstalled PluginStatus = "installed"
	// PluginStatusRunning indicates the plugin is running
	PluginStatusRunning PluginStatus = "running"
	// PluginStatusStopped indicates the plugin is stopped
	PluginStatusStopped PluginStatus = "stopped"
	// PluginStatusFailed indicates the plugin has failed
	PluginStatusFailed PluginStatus = "failed"
	// PluginStatusUpdating indicates the plugin is being updated
	PluginStatusUpdating PluginStatus = "updating"
)

// NewPluginCoordinator creates a new plugin coordinator instance.
func NewPluginCoordinator(config *PluginCoordinatorConfig, logger *zap.Logger) (*PluginCoordinator, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	brokerURL := config.BrokerURL
	if brokerURL == "" {
		brokerURL = "tcp://mqtt-broker:1883"
	}
	
	// Create MQTT client
	mqttConfig := &mqtt.Config{
		BrokerURL:            brokerURL,
		ClientID:             "plugin-coordinator",
		KeepAlive:            30 * time.Second,
		ConnectTimeout:       10 * time.Second,
		AutoReconnect:        true,
		MaxReconnectInterval: 5 * time.Minute,
	}
	
	mqttClient, err := mqtt.NewClient(mqttConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MQTT client: %w", err)
	}
	
	base := NewBaseCoordinator(mqtt.CoordinatorPlugin, mqttClient, logger)
	
	pc := &PluginCoordinator{
		BaseCoordinator: base,
		config:          config,
		registry: &PluginRegistry{
			plugins: make(map[string]*PluginEntry),
		},
	}
	
	// Register self health check
	pc.RegisterHealthCheck(pc)
	
	return pc, nil
}

// Start begins plugin coordinator operations.
func (pc *PluginCoordinator) Start(ctx context.Context) error {
	pc.GetLogger().Info("Starting plugin coordinator",
		zap.String("plugin_dir", pc.config.PluginDir))
	
	// Start base coordinator
	if err := pc.BaseCoordinator.Start(ctx); err != nil {
		return err
	}
	
	// Subscribe to plugin command topics
	if err := pc.subscribePluginTopics(); err != nil {
		return fmt.Errorf("failed to subscribe to plugin topics: %w", err)
	}
	
	// Start plugin scanner
	go pc.scanPlugins(ctx)

	// Start health status publishing
	go pc.StartHealthPublishing(ctx)

	pc.GetLogger().Info("Plugin coordinator started successfully")
	return nil
}

// Stop shuts down the plugin coordinator.
func (pc *PluginCoordinator) Stop(ctx context.Context) error {
	pc.GetLogger().Info("Stopping plugin coordinator")
	return pc.BaseCoordinator.Stop(ctx)
}

// subscribePluginTopics subscribes to plugin command topics.
func (pc *PluginCoordinator) subscribePluginTopics() error {
	installTopic := mqtt.NewTopicBuilder().
		Component(mqtt.CoordinatorPlugin).
		Action(mqtt.ActionCommand).
		Resource("install").
		Build()
	
	if err := pc.GetMQTTClient().Subscribe(installTopic, 1, pc.handleInstallCommand); err != nil {
		return err
	}
	
	removeTopic := mqtt.NewTopicBuilder().
		Component(mqtt.CoordinatorPlugin).
		Action(mqtt.ActionCommand).
		Resource("remove").
		Build()
	
	return pc.GetMQTTClient().Subscribe(removeTopic, 1, pc.handleRemoveCommand)
}

// handleInstallCommand processes plugin installation commands.
func (pc *PluginCoordinator) handleInstallCommand(topic string, payload []byte) error {
	var msg mqtt.Message
	if err := msg.UnmarshalPayload(&payload); err != nil {
		pc.GetLogger().Error("Failed to unmarshal install command", zap.Error(err))
		return err
	}
	
	var cmd struct {
		GUID     string                 `json:"guid"`
		Name     string                 `json:"name"`
		Version  string                 `json:"version"`
		Source   string                 `json:"source"`
		Metadata map[string]interface{} `json:"metadata"`
	}
	
	if err := msg.UnmarshalPayload(&cmd); err != nil {
		pc.GetLogger().Error("Failed to unmarshal install data", zap.Error(err))
		return err
	}
	
	pc.GetLogger().Info("Installing plugin",
		zap.String("guid", cmd.GUID),
		zap.String("name", cmd.Name),
		zap.String("version", cmd.Version))
	
	// TODO: Actual installation logic with Docker
	entry := &PluginEntry{
		GUID:         cmd.GUID,
		Name:         cmd.Name,
		Version:      cmd.Version,
		Status:       PluginStatusInstalled,
		InstalledAt:  time.Now(),
		LastVerified: time.Now(),
		Metadata:     cmd.Metadata,
	}
	
	pc.RegisterPlugin(entry)
	return nil
}

// handleRemoveCommand processes plugin removal commands.
func (pc *PluginCoordinator) handleRemoveCommand(topic string, payload []byte) error {
	var msg mqtt.Message
	if err := msg.UnmarshalPayload(&payload); err != nil {
		return err
	}
	
	var cmd struct {
		GUID string `json:"guid"`
	}
	
	if err := msg.UnmarshalPayload(&cmd); err != nil {
		return err
	}
	
	pc.GetLogger().Info("Removing plugin", zap.String("guid", cmd.GUID))
	pc.UnregisterPlugin(cmd.GUID)
	return nil
}

// RegisterPlugin adds a plugin to the registry.
func (pc *PluginCoordinator) RegisterPlugin(entry *PluginEntry) {
	pc.registry.mu.Lock()
	defer pc.registry.mu.Unlock()
	
	pc.registry.plugins[entry.GUID] = entry
	pc.GetLogger().Info("Plugin registered",
		zap.String("guid", entry.GUID),
		zap.String("name", entry.Name))
}

// UnregisterPlugin removes a plugin from the registry.
func (pc *PluginCoordinator) UnregisterPlugin(guid string) {
	pc.registry.mu.Lock()
	defer pc.registry.mu.Unlock()
	
	delete(pc.registry.plugins, guid)
	pc.GetLogger().Info("Plugin unregistered", zap.String("guid", guid))
}

// GetPlugin returns a plugin entry by GUID.
func (pc *PluginCoordinator) GetPlugin(guid string) (*PluginEntry, bool) {
	pc.registry.mu.RLock()
	defer pc.registry.mu.RUnlock()
	
	entry, exists := pc.registry.plugins[guid]
	return entry, exists
}

// ListPlugins returns all registered plugins.
func (pc *PluginCoordinator) ListPlugins() []*PluginEntry {
	pc.registry.mu.RLock()
	defer pc.registry.mu.RUnlock()
	
	plugins := make([]*PluginEntry, 0, len(pc.registry.plugins))
	for _, entry := range pc.registry.plugins {
		plugins = append(plugins, entry)
	}
	return plugins
}

// scanPlugins periodically scans for plugin changes.
func (pc *PluginCoordinator) scanPlugins(ctx context.Context) {
	ticker := time.NewTicker(pc.config.ScanInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pc.verifyPlugins()
		}
	}
}

// verifyPlugins verifies installed plugins.
func (pc *PluginCoordinator) verifyPlugins() {
	pc.registry.mu.Lock()
	defer pc.registry.mu.Unlock()
	
	now := time.Now()
	for _, entry := range pc.registry.plugins {
		// TODO: Actual verification logic
		entry.LastVerified = now
		pc.GetLogger().Debug("Plugin verified",
			zap.String("guid", entry.GUID),
			zap.String("name", entry.Name))
	}
}

// Check implements healthcheck.Checker interface.
func (pc *PluginCoordinator) Check(ctx context.Context) *healthcheck.Result {
	status := healthcheck.StatusHealthy
	message := "Plugin coordinator is healthy"
	details := make(map[string]interface{})
	
	pc.registry.mu.RLock()
	pluginCount := len(pc.registry.plugins)
	failedCount := 0
	for _, entry := range pc.registry.plugins {
		if entry.Status == PluginStatusFailed {
			failedCount++
		}
	}
	pc.registry.mu.RUnlock()
	
	details["total_plugins"] = pluginCount
	details["failed_plugins"] = failedCount
	
	if failedCount > 0 {
		status = healthcheck.StatusDegraded
		message = fmt.Sprintf("%d plugin(s) failed", failedCount)
	}
	
	return &healthcheck.Result{
		ComponentName: "plugin-coordinator",
		Status:        status,
		Message:       message,
		Timestamp:     time.Now(),
		Details:       details,
	}
}

// Name returns the coordinator name.
func (pc *PluginCoordinator) Name() string {
	return "plugin-coordinator"
}

// LoadConfig loads configuration.
func (pc *PluginCoordinator) LoadConfig(config interface{}) error {
	cfg, ok := config.(*PluginCoordinatorConfig)
	if !ok {
		return fmt.Errorf("invalid config type")
	}
	
	pc.config = cfg
	return pc.BaseCoordinator.LoadConfig(config)
}

// ValidateConfig validates the configuration.
func (pc *PluginCoordinator) ValidateConfig() error {
	if pc.config == nil {
		return fmt.Errorf("config is nil")
	}
	if pc.config.PluginDir == "" {
		return fmt.Errorf("plugin_dir is required")
	}
	if pc.config.ScanInterval <= 0 {
		return fmt.Errorf("scan_interval must be positive")
	}
	return nil
}
