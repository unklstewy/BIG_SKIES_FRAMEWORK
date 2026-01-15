// Package coordinators implements the UI element coordinator.
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

// UIElementCoordinator manages UI element tracking and provisioning from plugins.
type UIElementCoordinator struct {
	*BaseCoordinator
	config   *UIElementCoordinatorConfig
	registry *UIElementRegistry
}

// UIElementCoordinatorConfig holds configuration for the UI element coordinator.
type UIElementCoordinatorConfig struct {
	BaseConfig
	// BrokerURL is the MQTT broker URL
	BrokerURL string `json:"broker_url"`
	// ScanInterval for periodic UI API scanning
	ScanInterval time.Duration `json:"scan_interval"`
}

// UIElementRegistry maintains a registry of UI elements from plugins.
type UIElementRegistry struct {
	elements map[string]*UIElement
	mu       sync.RWMutex
}

// UIElement represents a UI element provided by a plugin.
type UIElement struct {
	// ID is the unique element identifier
	ID string `json:"id"`
	// PluginGUID is the plugin that provides this element
	PluginGUID string `json:"plugin_guid"`
	// Type is the element type (menu, panel, widget, etc.)
	Type UIElementType `json:"type"`
	// Title is the display title
	Title string `json:"title"`
	// APIEndpoint is the plugin API endpoint for this element
	APIEndpoint string `json:"api_endpoint"`
	// Order is the display order
	Order int `json:"order"`
	// Enabled indicates if the element is active
	Enabled bool `json:"enabled"`
	// RegisteredAt is when the element was registered
	RegisteredAt time.Time `json:"registered_at"`
	// Metadata contains element-specific information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UIElementType represents the type of UI element.
type UIElementType string

const (
	// UIElementTypeMenu represents a menu item
	UIElementTypeMenu UIElementType = "menu"
	// UIElementTypePanel represents a panel/page
	UIElementTypePanel UIElementType = "panel"
	// UIElementTypeWidget represents a widget
	UIElementTypeWidget UIElementType = "widget"
	// UIElementTypeTool represents a toolbar item
	UIElementTypeTool UIElementType = "tool"
	// UIElementTypeDialog represents a dialog
	UIElementTypeDialog UIElementType = "dialog"
)

// NewUIElementCoordinator creates a new UI element coordinator instance.
func NewUIElementCoordinator(config *UIElementCoordinatorConfig, logger *zap.Logger) (*UIElementCoordinator, error) {
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
		ClientID:             "uielement-coordinator",
		KeepAlive:            30 * time.Second,
		ConnectTimeout:       10 * time.Second,
		AutoReconnect:        true,
		MaxReconnectInterval: 5 * time.Minute,
	}
	
	mqttClient, err := mqtt.NewClient(mqttConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MQTT client: %w", err)
	}
	
	base := NewBaseCoordinator(mqtt.CoordinatorUIElement, mqttClient, logger)
	
	uec := &UIElementCoordinator{
		BaseCoordinator: base,
		config:          config,
		registry: &UIElementRegistry{
			elements: make(map[string]*UIElement),
		},
	}
	
	// Register self health check
	uec.RegisterHealthCheck(uec)
	
	return uec, nil
}

// Start begins UI element coordinator operations.
func (uec *UIElementCoordinator) Start(ctx context.Context) error {
	uec.GetLogger().Info("Starting UI element coordinator")
	
	// Start base coordinator
	if err := uec.BaseCoordinator.Start(ctx); err != nil {
		return err
	}
	
	// Subscribe to UI element topics
	if err := uec.subscribeUITopics(); err != nil {
		return fmt.Errorf("failed to subscribe to UI topics: %w", err)
	}
	
	// Start UI API scanner
	go uec.scanUIElements(ctx)

	// Start health status publishing
	go uec.StartHealthPublishing(ctx)

	uec.GetLogger().Info("UI element coordinator started successfully")
	return nil
}

// Stop shuts down the UI element coordinator.
func (uec *UIElementCoordinator) Stop(ctx context.Context) error {
	uec.GetLogger().Info("Stopping UI element coordinator")
	return uec.BaseCoordinator.Stop(ctx)
}

// subscribeUITopics subscribes to UI element registration topics.
func (uec *UIElementCoordinator) subscribeUITopics() error {
	registerTopic := mqtt.NewTopicBuilder().
		Component(mqtt.CoordinatorUIElement).
		Action(mqtt.ActionEvent).
		Resource("register").
		Build()
	
	if err := uec.GetMQTTClient().Subscribe(registerTopic, 1, uec.handleUIElementRegistration); err != nil {
		return err
	}
	
	unregisterTopic := mqtt.NewTopicBuilder().
		Component(mqtt.CoordinatorUIElement).
		Action(mqtt.ActionEvent).
		Resource("unregister").
		Build()
	
	return uec.GetMQTTClient().Subscribe(unregisterTopic, 1, uec.handleUIElementUnregistration)
}

// handleUIElementRegistration processes UI element registration messages.
func (uec *UIElementCoordinator) handleUIElementRegistration(topic string, payload []byte) error {
	var msg mqtt.Message
	if err := msg.UnmarshalPayload(&payload); err != nil {
		uec.GetLogger().Error("Failed to unmarshal registration message", zap.Error(err))
		return err
	}
	
	var reg struct {
		ID          string                 `json:"id"`
		PluginGUID  string                 `json:"plugin_guid"`
		Type        UIElementType          `json:"type"`
		Title       string                 `json:"title"`
		APIEndpoint string                 `json:"api_endpoint"`
		Order       int                    `json:"order"`
		Metadata    map[string]interface{} `json:"metadata"`
	}
	
	if err := msg.UnmarshalPayload(&reg); err != nil {
		uec.GetLogger().Error("Failed to unmarshal registration data", zap.Error(err))
		return err
	}
	
	element := &UIElement{
		ID:           reg.ID,
		PluginGUID:   reg.PluginGUID,
		Type:         reg.Type,
		Title:        reg.Title,
		APIEndpoint:  reg.APIEndpoint,
		Order:        reg.Order,
		Enabled:      true,
		RegisteredAt: time.Now(),
		Metadata:     reg.Metadata,
	}
	
	uec.RegisterUIElement(element)
	
	uec.GetLogger().Info("UI element registered",
		zap.String("id", element.ID),
		zap.String("plugin_guid", element.PluginGUID),
		zap.String("type", string(element.Type)),
		zap.String("title", element.Title))
	
	return nil
}

// handleUIElementUnregistration processes UI element unregistration messages.
func (uec *UIElementCoordinator) handleUIElementUnregistration(topic string, payload []byte) error {
	var msg mqtt.Message
	if err := msg.UnmarshalPayload(&payload); err != nil {
		return err
	}
	
	var unreg struct {
		ID string `json:"id"`
	}
	
	if err := msg.UnmarshalPayload(&unreg); err != nil {
		return err
	}
	
	uec.GetLogger().Info("Unregistering UI element", zap.String("id", unreg.ID))
	uec.UnregisterUIElement(unreg.ID)
	return nil
}

// RegisterUIElement adds a UI element to the registry.
func (uec *UIElementCoordinator) RegisterUIElement(element *UIElement) {
	uec.registry.mu.Lock()
	defer uec.registry.mu.Unlock()
	
	uec.registry.elements[element.ID] = element
}

// UnregisterUIElement removes a UI element from the registry.
func (uec *UIElementCoordinator) UnregisterUIElement(id string) {
	uec.registry.mu.Lock()
	defer uec.registry.mu.Unlock()
	
	delete(uec.registry.elements, id)
	uec.GetLogger().Info("UI element unregistered", zap.String("id", id))
}

// GetUIElement returns a UI element by ID.
func (uec *UIElementCoordinator) GetUIElement(id string) (*UIElement, bool) {
	uec.registry.mu.RLock()
	defer uec.registry.mu.RUnlock()
	
	element, exists := uec.registry.elements[id]
	return element, exists
}

// ListUIElements returns all registered UI elements.
func (uec *UIElementCoordinator) ListUIElements() []*UIElement {
	uec.registry.mu.RLock()
	defer uec.registry.mu.RUnlock()
	
	elements := make([]*UIElement, 0, len(uec.registry.elements))
	for _, element := range uec.registry.elements {
		elements = append(elements, element)
	}
	return elements
}

// ListUIElementsByPlugin returns UI elements for a specific plugin.
func (uec *UIElementCoordinator) ListUIElementsByPlugin(pluginGUID string) []*UIElement {
	uec.registry.mu.RLock()
	defer uec.registry.mu.RUnlock()
	
	elements := make([]*UIElement, 0)
	for _, element := range uec.registry.elements {
		if element.PluginGUID == pluginGUID {
			elements = append(elements, element)
		}
	}
	return elements
}

// ListUIElementsByType returns UI elements of a specific type.
func (uec *UIElementCoordinator) ListUIElementsByType(elementType UIElementType) []*UIElement {
	uec.registry.mu.RLock()
	defer uec.registry.mu.RUnlock()
	
	elements := make([]*UIElement, 0)
	for _, element := range uec.registry.elements {
		if element.Type == elementType {
			elements = append(elements, element)
		}
	}
	return elements
}

// scanUIElements periodically scans plugins for UI elements.
func (uec *UIElementCoordinator) scanUIElements(ctx context.Context) {
	ticker := time.NewTicker(uec.config.ScanInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			uec.scanPluginAPIs()
		}
	}
}

// scanPluginAPIs scans plugin APIs for UI element definitions.
func (uec *UIElementCoordinator) scanPluginAPIs() {
	// TODO: Query plugin coordinator for active plugins
	// TODO: Scan each plugin's API for UI element definitions
	uec.GetLogger().Debug("Scanning plugin APIs for UI elements")
}

// Check implements healthcheck.Checker interface.
func (uec *UIElementCoordinator) Check(ctx context.Context) *healthcheck.Result {
	status := healthcheck.StatusHealthy
	message := "UI element coordinator is healthy"
	details := make(map[string]interface{})
	
	uec.registry.mu.RLock()
	elementCount := len(uec.registry.elements)
	enabledCount := 0
	for _, element := range uec.registry.elements {
		if element.Enabled {
			enabledCount++
		}
	}
	uec.registry.mu.RUnlock()
	
	details["total_elements"] = elementCount
	details["enabled_elements"] = enabledCount
	
	return &healthcheck.Result{
		ComponentName: "uielement-coordinator",
		Status:        status,
		Message:       message,
		Timestamp:     time.Now(),
		Details:       details,
	}
}

// Name returns the coordinator name.
func (uec *UIElementCoordinator) Name() string {
	return "uielement-coordinator"
}

// LoadConfig loads configuration.
func (uec *UIElementCoordinator) LoadConfig(config interface{}) error {
	cfg, ok := config.(*UIElementCoordinatorConfig)
	if !ok {
		return fmt.Errorf("invalid config type")
	}
	
	uec.config = cfg
	return uec.BaseCoordinator.LoadConfig(config)
}

// ValidateConfig validates the configuration.
func (uec *UIElementCoordinator) ValidateConfig() error {
	if uec.config == nil {
		return fmt.Errorf("config is nil")
	}
	if uec.config.ScanInterval <= 0 {
		return fmt.Errorf("scan_interval must be positive")
	}
	return nil
}
