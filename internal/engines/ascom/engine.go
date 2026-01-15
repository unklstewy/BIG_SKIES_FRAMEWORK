// Package ascom provides ASCOM Alpaca protocol engine and client implementation.
package ascom

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/models"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/healthcheck"
	"go.uber.org/zap"
)

// Engine manages ASCOM device lifecycle for telescope configurations.
// It handles device discovery, connection pooling, health monitoring,
// and coordinated operations across multiple devices.
type Engine struct {
	client      *Client
	logger      *zap.Logger
	devices     map[string]*managedDevice // key: device_id
	telescopes  map[string]*telescopePool // key: telescope_id
	mu          sync.RWMutex
	healthCheck time.Duration
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// managedDevice represents a managed ASCOM device with connection state.
type managedDevice struct {
	device       *models.AlpacaDevice
	connected    bool
	lastHealthy  time.Time
	failCount    int
	mu           sync.RWMutex
}

// telescopePool manages all devices for a single telescope configuration.
type telescopePool struct {
	telescopeID string
	devices     map[string]*managedDevice // key: device_role (telescope, camera, dome, etc.)
	mu          sync.RWMutex
}

// NewEngine creates a new ASCOM engine instance.
// healthCheckInterval determines how often device health is checked (0 to disable).
func NewEngine(logger *zap.Logger, healthCheckInterval time.Duration) *Engine {
	if logger == nil {
		logger = zap.NewNop()
	}

	if healthCheckInterval == 0 {
		healthCheckInterval = 30 * time.Second
	}

	return &Engine{
		client:      NewClient(logger),
		logger:      logger.With(zap.String("component", "ascom_engine")),
		devices:     make(map[string]*managedDevice),
		telescopes:  make(map[string]*telescopePool),
		healthCheck: healthCheckInterval,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the engine's background operations (health checks, etc.).
func (e *Engine) Start(ctx context.Context) {
	e.logger.Info("Starting ASCOM engine",
		zap.Duration("health_check_interval", e.healthCheck))

	e.wg.Add(1)
	go e.healthCheckLoop(ctx)
}

// Stop shuts down the engine and disconnects all devices.
func (e *Engine) Stop() {
	e.logger.Info("Stopping ASCOM engine")
	close(e.stopCh)
	e.wg.Wait()

	// Disconnect all devices
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, md := range e.devices {
		if md.connected {
			if err := e.client.Disconnect(context.Background(), md.device); err != nil {
				e.logger.Warn("Failed to disconnect device",
					zap.String("device_id", md.device.DeviceID),
					zap.Error(err))
			}
		}
	}

	e.logger.Info("ASCOM engine stopped")
}

// DiscoverDevices performs discovery and returns available devices.
func (e *Engine) DiscoverDevices(ctx context.Context, port int) ([]*models.AlpacaDevice, error) {
	e.logger.Info("Starting device discovery", zap.Int("port", port))
	devices, err := e.client.DiscoverDevices(ctx, port)
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	e.logger.Info("Discovery complete", zap.Int("device_count", len(devices)))
	return devices, nil
}

// RegisterDevice adds a device to the engine's management.
// The device is not automatically connected.
func (e *Engine) RegisterDevice(device *models.AlpacaDevice) error {
	if device == nil {
		return fmt.Errorf("device cannot be nil")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.devices[device.DeviceID]; exists {
		return fmt.Errorf("device %s already registered", device.DeviceID)
	}

	e.devices[device.DeviceID] = &managedDevice{
		device:      device,
		connected:   false,
		lastHealthy: time.Now(),
		failCount:   0,
	}

	e.logger.Info("Device registered",
		zap.String("device_id", device.DeviceID),
		zap.String("device_type", device.DeviceType))

	return nil
}

// UnregisterDevice removes a device from management and disconnects it.
func (e *Engine) UnregisterDevice(ctx context.Context, deviceID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	md, exists := e.devices[deviceID]
	if !exists {
		return fmt.Errorf("device %s not registered", deviceID)
	}

	// Disconnect if connected
	if md.connected {
		if err := e.client.Disconnect(ctx, md.device); err != nil {
			e.logger.Warn("Failed to disconnect device during unregister",
				zap.String("device_id", deviceID),
				zap.Error(err))
		}
	}

	delete(e.devices, deviceID)

	e.logger.Info("Device unregistered", zap.String("device_id", deviceID))
	return nil
}

// ConnectDevice establishes connection to a registered device.
func (e *Engine) ConnectDevice(ctx context.Context, deviceID string) error {
	e.mu.RLock()
	md, exists := e.devices[deviceID]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("device %s not registered", deviceID)
	}

	md.mu.Lock()
	defer md.mu.Unlock()

	if md.connected {
		return nil // Already connected
	}

	if err := e.client.Connect(ctx, md.device); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	md.connected = true
	md.device.Connected = true
	md.lastHealthy = time.Now()
	md.failCount = 0

	e.logger.Info("Device connected", zap.String("device_id", deviceID))
	return nil
}

// DisconnectDevice disconnects a device.
func (e *Engine) DisconnectDevice(ctx context.Context, deviceID string) error {
	e.mu.RLock()
	md, exists := e.devices[deviceID]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("device %s not registered", deviceID)
	}

	md.mu.Lock()
	defer md.mu.Unlock()

	if !md.connected {
		return nil // Already disconnected
	}

	if err := e.client.Disconnect(ctx, md.device); err != nil {
		return fmt.Errorf("failed to disconnect: %w", err)
	}

	md.connected = false
	md.device.Connected = false

	e.logger.Info("Device disconnected", zap.String("device_id", deviceID))
	return nil
}

// IsDeviceConnected checks if a device is currently connected.
func (e *Engine) IsDeviceConnected(deviceID string) (bool, error) {
	e.mu.RLock()
	md, exists := e.devices[deviceID]
	e.mu.RUnlock()

	if !exists {
		return false, fmt.Errorf("device %s not registered", deviceID)
	}

	md.mu.RLock()
	defer md.mu.RUnlock()

	return md.connected, nil
}

// GetDevice returns a registered device by ID.
func (e *Engine) GetDevice(deviceID string) (*models.AlpacaDevice, error) {
	e.mu.RLock()
	md, exists := e.devices[deviceID]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("device %s not registered", deviceID)
	}

	md.mu.RLock()
	defer md.mu.RUnlock()

	return md.device, nil
}

// ListDevices returns all registered devices.
func (e *Engine) ListDevices() []*models.AlpacaDevice {
	e.mu.RLock()
	defer e.mu.RUnlock()

	devices := make([]*models.AlpacaDevice, 0, len(e.devices))
	for _, md := range e.devices {
		md.mu.RLock()
		devices = append(devices, md.device)
		md.mu.RUnlock()
	}

	return devices
}

// RegisterTelescopeDevices registers all devices for a telescope configuration.
// devices is a map of device_role -> device_id.
func (e *Engine) RegisterTelescopeDevices(telescopeID string, devices map[string]string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	pool := &telescopePool{
		telescopeID: telescopeID,
		devices:     make(map[string]*managedDevice),
	}

	for role, deviceID := range devices {
		md, exists := e.devices[deviceID]
		if !exists {
			return fmt.Errorf("device %s not registered", deviceID)
		}
		pool.devices[role] = md
	}

	e.telescopes[telescopeID] = pool

	e.logger.Info("Telescope devices registered",
		zap.String("telescope_id", telescopeID),
		zap.Int("device_count", len(devices)))

	return nil
}

// UnregisterTelescope removes a telescope configuration from management.
func (e *Engine) UnregisterTelescope(telescopeID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.telescopes, telescopeID)

	e.logger.Info("Telescope unregistered", zap.String("telescope_id", telescopeID))
}

// GetTelescopeDevice returns a specific device for a telescope by role.
func (e *Engine) GetTelescopeDevice(telescopeID, deviceRole string) (*models.AlpacaDevice, error) {
	e.mu.RLock()
	pool, exists := e.telescopes[telescopeID]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("telescope %s not registered", telescopeID)
	}

	pool.mu.RLock()
	md, exists := pool.devices[deviceRole]
	pool.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("device role %s not found for telescope %s", deviceRole, telescopeID)
	}

	md.mu.RLock()
	defer md.mu.RUnlock()

	return md.device, nil
}

// GetClient returns the underlying ASCOM client for direct operations.
func (e *Engine) GetClient() *Client {
	return e.client
}

// healthCheckLoop periodically checks device health.
func (e *Engine) healthCheckLoop(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(e.healthCheck)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.performHealthChecks(ctx)
		}
	}
}

// Name implements healthcheck.Checker interface.
func (e *Engine) Name() string {
	return "ascom_engine"
}

// performHealthChecks checks all connected devices.
func (e *Engine) performHealthChecks(ctx context.Context) {
	e.mu.RLock()
	devicesToCheck := make([]*managedDevice, 0, len(e.devices))
	for _, md := range e.devices {
		md.mu.RLock()
		if md.connected {
			devicesToCheck = append(devicesToCheck, md)
		}
		md.mu.RUnlock()
	}
	e.mu.RUnlock()

	for _, md := range devicesToCheck {
		e.checkDeviceHealth(ctx, md)
	}
}

// checkDeviceHealth performs a health check on a single device.
func (e *Engine) checkDeviceHealth(ctx context.Context, md *managedDevice) {
	md.mu.Lock()
	defer md.mu.Unlock()

	// Check if still connected
	connected, err := e.client.IsConnected(ctx, md.device)
	if err != nil {
		md.failCount++
		e.logger.Warn("Device health check failed",
			zap.String("device_id", md.device.DeviceID),
			zap.Int("fail_count", md.failCount),
			zap.Error(err))

		// Mark as disconnected after 3 consecutive failures
		if md.failCount >= 3 {
			md.connected = false
			md.device.Connected = false
			e.logger.Error("Device marked as disconnected after health check failures",
				zap.String("device_id", md.device.DeviceID))
		}
		return
	}

	if !connected {
		md.connected = false
		md.device.Connected = false
		e.logger.Warn("Device reported as disconnected",
			zap.String("device_id", md.device.DeviceID))
		return
	}

	// Device is healthy
	md.lastHealthy = time.Now()
	md.failCount = 0
}

// Check implements healthcheck.Checker interface.
func (e *Engine) Check(ctx context.Context) *healthcheck.Result {
	e.mu.RLock()
	defer e.mu.RUnlock()

	totalDevices := len(e.devices)
	connectedDevices := 0
	healthyDevices := 0

	for _, md := range e.devices {
		md.mu.RLock()
		if md.connected {
			connectedDevices++
			if md.failCount == 0 {
				healthyDevices++
			}
		}
		md.mu.RUnlock()
	}

	status := healthcheck.StatusHealthy
	message := fmt.Sprintf("ASCOM engine healthy: %d/%d devices connected, %d healthy",
		connectedDevices, totalDevices, healthyDevices)

	if totalDevices > 0 && connectedDevices == 0 {
		status = healthcheck.StatusUnhealthy
		message = "No devices connected"
	} else if connectedDevices > 0 && healthyDevices < connectedDevices {
		status = healthcheck.StatusDegraded
		message = fmt.Sprintf("Some devices unhealthy: %d/%d", healthyDevices, connectedDevices)
	}

	return &healthcheck.Result{
		ComponentName: "ascom_engine",
		Status:        status,
		Message:       message,
		Timestamp:     time.Now(),
		Details: map[string]interface{}{
			"total_devices":     totalDevices,
			"connected_devices": connectedDevices,
			"healthy_devices":   healthyDevices,
			"telescope_count":   len(e.telescopes),
		},
	}
}
