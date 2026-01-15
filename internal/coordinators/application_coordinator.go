// Package coordinators implements the application service coordinator.
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

// ApplicationCoordinator tracks and monitors application microservices.
type ApplicationCoordinator struct {
	*BaseCoordinator
	config   *ApplicationCoordinatorConfig
	registry *ServiceRegistry
}

// ApplicationCoordinatorConfig holds configuration for the application coordinator.
type ApplicationCoordinatorConfig struct {
	BaseConfig
	// BrokerURL is the MQTT broker URL
	BrokerURL string `json:"broker_url"`
	// RegistryCheckInterval for periodic service health checks
	RegistryCheckInterval time.Duration `json:"registry_check_interval"`
	// ServiceTimeout for service responsiveness checks
	ServiceTimeout time.Duration `json:"service_timeout"`
}

// ServiceRegistry maintains a registry of active microservices.
type ServiceRegistry struct {
	services map[string]*ServiceEntry
	mu       sync.RWMutex
}

// ServiceEntry represents a registered microservice.
type ServiceEntry struct {
	// ID is the unique service identifier
	ID string `json:"id"`
	// Name is the service name
	Name string `json:"name"`
	// Status is the current service status
	Status healthcheck.Status `json:"status"`
	// Endpoint is the service endpoint URL
	Endpoint string `json:"endpoint"`
	// RegisteredAt is when the service was registered
	RegisteredAt time.Time `json:"registered_at"`
	// LastHeartbeat is the last time the service reported health
	LastHeartbeat time.Time `json:"last_heartbeat"`
	// Metadata contains service-specific information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewApplicationCoordinator creates a new application coordinator instance.
func NewApplicationCoordinator(config *ApplicationCoordinatorConfig, logger *zap.Logger) (*ApplicationCoordinator, error) {
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
		ClientID:             "application-coordinator",
		KeepAlive:            30 * time.Second,
		ConnectTimeout:       10 * time.Second,
		AutoReconnect:        true,
		MaxReconnectInterval: 5 * time.Minute,
	}
	
	mqttClient, err := mqtt.NewClient(mqttConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MQTT client: %w", err)
	}
	
	base := NewBaseCoordinator(mqtt.CoordinatorApplication, mqttClient, logger)
	
	ac := &ApplicationCoordinator{
		BaseCoordinator: base,
		config:          config,
		registry: &ServiceRegistry{
			services: make(map[string]*ServiceEntry),
		},
	}
	
	// Register self health check
	ac.RegisterHealthCheck(ac)
	
	return ac, nil
}

// Start begins application coordinator operations.
func (ac *ApplicationCoordinator) Start(ctx context.Context) error {
	ac.GetLogger().Info("Starting application coordinator")
	
	// Start base coordinator
	if err := ac.BaseCoordinator.Start(ctx); err != nil {
		return err
	}
	
	// Subscribe to service registration topics
	if err := ac.subscribeServiceTopics(); err != nil {
		return fmt.Errorf("failed to subscribe to service topics: %w", err)
	}
	
	// Start registry monitoring
	go ac.monitorServices(ctx)

	// Start health status publishing
	go ac.StartHealthPublishing(ctx)

	ac.GetLogger().Info("Application coordinator started successfully")
	return nil
}

// Stop shuts down the application coordinator.
func (ac *ApplicationCoordinator) Stop(ctx context.Context) error {
	ac.GetLogger().Info("Stopping application coordinator")
	return ac.BaseCoordinator.Stop(ctx)
}

// subscribeServiceTopics subscribes to service registration and heartbeat topics.
func (ac *ApplicationCoordinator) subscribeServiceTopics() error {
	topic := mqtt.NewTopicBuilder().
		Component("service").
		Action(mqtt.ActionEvent).
		Resource("register").
		Build()
	
	if err := ac.GetMQTTClient().Subscribe(topic, 1, ac.handleServiceRegistration); err != nil {
		return err
	}
	
	heartbeatTopic := mqtt.NewTopicBuilder().
		Component("service").
		Action(mqtt.ActionEvent).
		Resource("heartbeat").
		Build()
	
	return ac.GetMQTTClient().Subscribe(heartbeatTopic, 1, ac.handleServiceHeartbeat)
}

// handleServiceRegistration processes service registration messages.
func (ac *ApplicationCoordinator) handleServiceRegistration(topic string, payload []byte) error {
	var msg mqtt.Message
	if err := msg.UnmarshalPayload(&payload); err != nil {
		ac.GetLogger().Error("Failed to unmarshal registration message", zap.Error(err))
		return err
	}
	
	var registration struct {
		ID       string                 `json:"id"`
		Name     string                 `json:"name"`
		Endpoint string                 `json:"endpoint"`
		Metadata map[string]interface{} `json:"metadata"`
	}
	
	if err := msg.UnmarshalPayload(&registration); err != nil {
		ac.GetLogger().Error("Failed to unmarshal registration data", zap.Error(err))
		return err
	}
	
	ac.RegisterService(&ServiceEntry{
		ID:            registration.ID,
		Name:          registration.Name,
		Status:        healthcheck.StatusHealthy,
		Endpoint:      registration.Endpoint,
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		Metadata:      registration.Metadata,
	})
	
	ac.GetLogger().Info("Service registered",
		zap.String("id", registration.ID),
		zap.String("name", registration.Name))
	
	return nil
}

// handleServiceHeartbeat processes service heartbeat messages.
func (ac *ApplicationCoordinator) handleServiceHeartbeat(topic string, payload []byte) error {
	var msg mqtt.Message
	if err := msg.UnmarshalPayload(&payload); err != nil {
		return err
	}
	
	var heartbeat struct {
		ID     string              `json:"id"`
		Status healthcheck.Status  `json:"status"`
	}
	
	if err := msg.UnmarshalPayload(&heartbeat); err != nil {
		return err
	}
	
	ac.UpdateServiceHeartbeat(heartbeat.ID, heartbeat.Status)
	return nil
}

// RegisterService adds a service to the registry.
func (ac *ApplicationCoordinator) RegisterService(entry *ServiceEntry) {
	ac.registry.mu.Lock()
	defer ac.registry.mu.Unlock()
	
	ac.registry.services[entry.ID] = entry
}

// UnregisterService removes a service from the registry.
func (ac *ApplicationCoordinator) UnregisterService(serviceID string) {
	ac.registry.mu.Lock()
	defer ac.registry.mu.Unlock()
	
	delete(ac.registry.services, serviceID)
	ac.GetLogger().Info("Service unregistered", zap.String("id", serviceID))
}

// UpdateServiceHeartbeat updates the last heartbeat time for a service.
func (ac *ApplicationCoordinator) UpdateServiceHeartbeat(serviceID string, status healthcheck.Status) {
	ac.registry.mu.Lock()
	defer ac.registry.mu.Unlock()
	
	if entry, exists := ac.registry.services[serviceID]; exists {
		entry.LastHeartbeat = time.Now()
		entry.Status = status
	}
}

// GetService returns a service entry by ID.
func (ac *ApplicationCoordinator) GetService(serviceID string) (*ServiceEntry, bool) {
	ac.registry.mu.RLock()
	defer ac.registry.mu.RUnlock()
	
	entry, exists := ac.registry.services[serviceID]
	return entry, exists
}

// ListServices returns all registered services.
func (ac *ApplicationCoordinator) ListServices() []*ServiceEntry {
	ac.registry.mu.RLock()
	defer ac.registry.mu.RUnlock()
	
	services := make([]*ServiceEntry, 0, len(ac.registry.services))
	for _, entry := range ac.registry.services {
		services = append(services, entry)
	}
	return services
}

// monitorServices periodically checks service health.
func (ac *ApplicationCoordinator) monitorServices(ctx context.Context) {
	ticker := time.NewTicker(ac.config.RegistryCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ac.checkServiceHealth()
		}
	}
}

// checkServiceHealth checks if services are still responsive.
func (ac *ApplicationCoordinator) checkServiceHealth() {
	ac.registry.mu.Lock()
	defer ac.registry.mu.Unlock()
	
	now := time.Now()
	timeout := ac.config.ServiceTimeout
	
	for id, entry := range ac.registry.services {
		if now.Sub(entry.LastHeartbeat) > timeout {
			ac.GetLogger().Warn("Service missed heartbeat",
				zap.String("id", id),
				zap.String("name", entry.Name),
				zap.Duration("since", now.Sub(entry.LastHeartbeat)))
			
			entry.Status = healthcheck.StatusUnhealthy
		}
	}
}

// Check implements healthcheck.Checker interface.
func (ac *ApplicationCoordinator) Check(ctx context.Context) *healthcheck.Result {
	status := healthcheck.StatusHealthy
	message := "Application coordinator is healthy"
	details := make(map[string]interface{})
	
	ac.registry.mu.RLock()
	serviceCount := len(ac.registry.services)
	unhealthyCount := 0
	for _, entry := range ac.registry.services {
		if entry.Status == healthcheck.StatusUnhealthy {
			unhealthyCount++
		}
	}
	ac.registry.mu.RUnlock()
	
	details["total_services"] = serviceCount
	details["unhealthy_services"] = unhealthyCount
	
	if unhealthyCount > 0 {
		status = healthcheck.StatusDegraded
		message = fmt.Sprintf("%d service(s) unhealthy", unhealthyCount)
	}
	
	return &healthcheck.Result{
		ComponentName: "application-coordinator",
		Status:        status,
		Message:       message,
		Timestamp:     time.Now(),
		Details:       details,
	}
}

// Name returns the coordinator name.
func (ac *ApplicationCoordinator) Name() string {
	return "application-coordinator"
}

// LoadConfig loads configuration.
func (ac *ApplicationCoordinator) LoadConfig(config interface{}) error {
	cfg, ok := config.(*ApplicationCoordinatorConfig)
	if !ok {
		return fmt.Errorf("invalid config type")
	}
	
	ac.config = cfg
	return ac.BaseCoordinator.LoadConfig(config)
}

// ValidateConfig validates the configuration.
func (ac *ApplicationCoordinator) ValidateConfig() error {
	if ac.config == nil {
		return fmt.Errorf("config is nil")
	}
	if ac.config.RegistryCheckInterval <= 0 {
		return fmt.Errorf("registry_check_interval must be positive")
	}
	if ac.config.ServiceTimeout <= 0 {
		return fmt.Errorf("service_timeout must be positive")
	}
	return nil
}
